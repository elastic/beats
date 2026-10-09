// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration

// Package serverless runs a packaged Beat's setup, export and run commands
// against an Elastic Cloud serverless project. The project is provisioned
// outside the tests, by .buildkite/scripts/serverless.sh, and reached through
// the same environment variables as the ECH tests.
package serverless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/go-elasticsearch/v8"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/testing/go-ech"
)

// beatConfigs holds the Beat specific part of the configuration for each
// supported Beat. They must produce events without anything from the Beat's
// home directory, since ech.RunSmokeTest runs the Beat with an empty one.
var beatConfigs = map[string]string{
	"metricbeat": `
metricbeat.modules:
  - module: system
    metricsets: [cpu, memory]
    period: 10s
`,
	"filebeat": `
filebeat.inputs:
  - type: filestream
    id: serverless-test
    paths:
      - {{.LogFile}}
`,
	"auditbeat": `
auditbeat.modules:

- module: file_integrity
  paths:
  - /bin
  - /usr/bin
  - /sbin
  - /usr/sbin
  - /etc
`,
	"packetbeat": ``,
}

const outputConfig = `
output.elasticsearch:
  hosts: ["${ES_HOST}"]
  api_key: ${ES_API_KEY}
  allow_older_versions: true
# setup.kibana inherits the output's API key.
setup.kibana:
  host: ${KIBANA_HOST}
`

// Run runs the suite for beatName, packaged and extracted in $BEAT_HOME,
// against the project described by the ES_* and KIBANA_* environment variables.
func Run(t *testing.T, beatName string) {
	t.Helper()
	_, ok := beatConfigs[beatName]
	require.True(t, ok, "unsupported beat %q", beatName)
	suite.Run(t, &BeatRunner{beatName: beatName})
}

// BeatRunner is the suite.
type BeatRunner struct {
	suite.Suite
	beatName string

	// home is the extracted Beat package, homeDirs the directories in it.
	home     string
	homeDirs []string
	config   string

	esClient     *elasticsearch.Client
	kibanaClient *kibana.Client
}

func (r *BeatRunner) binary() string { return filepath.Join(r.home, r.beatName) }

// SetupSuite connects to the project and renders the Beat configuration.
func (r *BeatRunner) SetupSuite() {
	t := r.T()
	ech.VerifyEnvVars(t)
	for _, name := range []string{"ES_API_KEY", "KIBANA_HOST", "KIBANA_USER", "KIBANA_PASS", "BEAT_HOME"} {
		require.NotEmpty(t, os.Getenv(name), "expected env var %s to be not-empty", name)
	}
	r.home = os.Getenv("BEAT_HOME")

	entries, err := os.ReadDir(r.home)
	require.NoError(t, err, "could not read the Beat package in BEAT_HOME")
	for _, e := range entries {
		if e.IsDir() {
			r.homeDirs = append(r.homeDirs, e.Name())
		}
	}

	r.esClient, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{os.Getenv("ES_HOST")},
		Username:  os.Getenv("ES_USER"),
		Password:  os.Getenv("ES_PASS"),
	})
	require.NoError(t, err, "could not create the Elasticsearch client")

	r.kibanaClient, err = kibana.NewClientWithConfig(&kibana.ClientConfig{
		Host:     os.Getenv("KIBANA_HOST"),
		Username: os.Getenv("KIBANA_USER"),
		Password: os.Getenv("KIBANA_PASS"),
	}, "", "", "", "")
	require.NoError(t, err, "could not create the Kibana client")

	logFile := filepath.Join(t.TempDir(), "input.log")
	integration.WriteLogFile(t, logFile, 100, false)

	tmpl, err := template.New("config").Parse(outputConfig + beatConfigs[r.beatName])
	require.NoError(t, err, "could not parse the Beat configuration template")
	var rendered bytes.Buffer
	require.NoError(t, tmpl.Execute(&rendered, map[string]string{"LogFile": logFile}), "could not render the Beat configuration")
	r.config = rendered.String()
}

// newBeat returns a configured Beat process whose home links to the
// packaged Beat's directories, such as its dashboards and modules.
func (r *BeatRunner) newBeat() *integration.BeatProc {
	t := r.T()
	proc := integration.NewStandardBeat(t, r.beatName, r.binary())
	for _, dir := range r.homeDirs {
		err := os.Symlink(filepath.Join(r.home, dir), filepath.Join(proc.TempDir(), dir))
		require.NoError(t, err, "could not link %s into the Beat's home", dir)
	}
	proc.WriteConfigFile(r.config)
	return proc
}

// run runs the Beat with args until it exits. It returns the process, to
// inspect its output and logs, and the exit error.
func (r *BeatRunner) run(args ...string) (*integration.BeatProc, error) {
	proc := r.newBeat()
	proc.Start(args...)
	return proc, proc.Cmd.Wait()
}

// stdout returns what the exited Beat process wrote to stdout.
func (r *BeatRunner) stdout(proc *integration.BeatProc) string {
	out, err := proc.ReadStdout()
	r.Require().NoError(err, "could not read the Beat's stdout")
	return out
}

// stderr returns what the exited Beat process wrote to stderr.
func (r *BeatRunner) stderr(proc *integration.BeatProc) string {
	out, err := os.ReadFile(filepath.Join(proc.TempDir(), "stderr"))
	r.Require().NoError(err, "could not read the Beat's stderr")
	return string(out)
}

// run the beat, ensure data is ingested
func (r *BeatRunner) TestRunAndCheckData() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()

	// in case there's already a running template, delete it, forcing the beat to re-install
	r.CleanupTemplates(ctx)

	ech.RunSmokeTest(r.T(), r.beatName, r.binary(), r.config)
}

// tests the [beat] setup --dashboards command
func (r *BeatRunner) TestSetupDashboards() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute*3) //dashboards seem to take a while
	defer cancel()

	proc, err := r.run("setup", "--dashboards")
	r.NoError(err, "setup --dashboards failed")
	r.Require().Contains(r.stdout(proc), "Loaded dashboards")

	dashList, err := GetDashboards(ctx, r.kibanaClient)
	r.Require().NoError(err)

	// interesting hack in cases where we don't have a clean environment
	// check to see if any of the dashboards were created recently
	found := false
	for _, dash := range dashList {
		if time.Since(dash.UpdatedAt) < time.Minute*5 {
			found = true
			break
		}
	}
	r.Require().True(found, "could not find dashboard newer than 5 minutes, out of %d dashboards", len(dashList))

	r.Run("export dashboards", r.SubtestExportDashboards)
	// cleanup
	for _, dash := range dashList {
		if err := DeleteDashboard(ctx, r.kibanaClient, dash.ID); err != nil {
			r.T().Logf("WARNING: could not delete dashboards after test: %s", err)
			break
		}
	}
}

// tests the [beat] export dashboard command
func (r *BeatRunner) SubtestExportDashboards() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute*2)
	defer cancel()
	outDir := r.T().TempDir()

	dashlist, err := GetDashboards(ctx, r.kibanaClient)
	r.Require().NoError(err)
	r.Require().NotEmpty(dashlist)

	_, err = r.run("export", "dashboard", "--folder", outDir, "--id", dashlist[0].ID)
	r.NoError(err, "export dashboard failed")

	// The folder matches the major version of Kibana, so we read it from the API
	dashboardFolder := fmt.Sprintf("/_meta/kibana/%d/dashboard", r.kibanaClient.GetVersion().Major)
	inFolder, err := os.ReadDir(filepath.Join(outDir, dashboardFolder))
	r.Require().NoError(err)
	r.T().Logf("got log contents: %#v", inFolder)
	r.Require().NotEmpty(inFolder)
}

// NOTE for the below tests: all tests share one project, so a previous test may have
// already done setup. Tests clean up after themselves and pre-delete what they need to be absent.

// tests the [beat] setup --pipelines command
func (r *BeatRunner) TestSetupPipelines() {
	if r.beatName != "filebeat" {
		r.T().Skip("pipelines only available on filebeat")
	}
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()

	defer func() {
		if err := estools.DeletePipelines(ctx, r.esClient, "*filebeat*"); err != nil {
			r.T().Logf("WARNING: could not clean up pipelines: %s", err)
		}
	}()

	// need to actually enable something that has pipelines
	_, err := r.run("setup", "--pipelines", "--modules", "apache", "-M", "apache.error.enabled=true", "-M", "apache.access.enabled=true")
	r.NoError(err, "setup --pipelines failed")

	pipelines, err := estools.GetPipelines(ctx, r.esClient, "*filebeat*")
	r.Require().NoError(err)
	r.Require().NotEmpty(pipelines)
}

// test beat setup --index-management with ILM disabled
func (r *BeatRunner) TestIndexManagementNoILM() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	proc, err := r.run("setup", "--index-management", "--E=setup.ilm.enabled=false")
	r.NoError(err, "setup --index-management failed")
	// we should not print a warning if we've explicitly disabled ILM
	r.False(proc.LogContains("not supported"), "unexpected warning about ILM not being supported")

	tmpls, err := estools.GetIndexTemplatesForPattern(ctx, r.esClient, fmt.Sprintf("*%s*", r.beatName))
	r.Require().NoError(err)
	for _, tmpl := range tmpls.IndexTemplates {
		r.T().Logf("got template: %s", tmpl.Name)
	}
	r.Require().NotEmpty(tmpls.IndexTemplates)

	r.Run("export templates", r.SubtestExportTemplates)
	r.Run("export index patterns", r.SubtestExportIndexPatterns)
}

// tests setup with all default settings
func (r *BeatRunner) TestWithAllDefaults() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	// pre-delete in case something else missed cleanup
	r.CleanupTemplates(ctx)

	_, err := r.run("setup", "--index-management")
	r.Require().NoError(err)

	streams, err := estools.GetDataStreamsForPattern(ctx, r.esClient, fmt.Sprintf("%s*", r.beatName))
	r.Require().NoError(err)
	r.Require().NotEmpty(streams.DataStreams)
}

// test the setup process with mismatching template and DSL names
func (r *BeatRunner) TestCustomBadNames() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	proc, err := r.run("setup", "--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.data_stream_pattern='custom-bad-name'", "--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'")
	r.Require().NoError(err)

	r.Require().True(proc.LogContains("Additional updates & overwrites to this config will not work."), "expected a warning about the mismatching names")
}

func (r *BeatRunner) TestOverwriteWithCustomName() {
	//an updated policy that has a different value than the default of 7d
	updatedPolicy := mapstr.M{
		"data_retention": "1d",
	}
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	lifecyclePath := r.writePolicy(updatedPolicy)

	r.CleanupTemplates(ctx)

	_, err := r.run("setup", "--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.data_stream_pattern='custom-name'", "--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'")
	r.Require().NoError(err)

	r.CheckDSLPolicy(ctx, "*custom-name*", "7d")

	_, err = r.run("setup", "--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.overwrite=true", "--E=setup.dsl.data_stream_pattern='custom-name'",
		"--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'", fmt.Sprintf("--E=setup.dsl.policy_file=%s", lifecyclePath))
	r.Require().NoError(err)

	r.CheckDSLPolicy(ctx, "*custom-name*", "1d")
}

// TestWithCustomLifecyclePolicy uploads a custom DSL policy
func (r *BeatRunner) TestWithCustomLifecyclePolicy() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	lifecyclePath := r.writePolicy(mapstr.M{"data_retention": "1d"})

	r.CleanupTemplates(ctx)

	_, err := r.run("setup", "--index-management",
		"--E=setup.dsl.enabled=true", fmt.Sprintf("--E=setup.dsl.policy_file=%s", lifecyclePath))
	r.Require().NoError(err)

	r.CheckDSLPolicy(ctx, fmt.Sprintf("%s*", r.beatName), "1d")
}

// tests beat setup --index-management with ILM explicitly set
// On serverless, this should fail.
func (r *BeatRunner) TestIndexManagementILMEnabledFailure() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	r.requireServerless(ctx)

	proc, err := r.run("setup", "--index-management", "--E=setup.ilm.enabled=true", "--E=setup.ilm.overwrite=true")
	r.Require().Error(err)
	r.Contains(r.stderr(proc), "error creating")
}

// tests setup with both ILM and DSL enabled, should fail
func (r *BeatRunner) TestBothLifecyclesEnabled() {
	_, err := r.run("setup", "--index-management", "--E=setup.ilm.enabled=true", "--E=setup.dsl.enabled=true")
	r.Require().Error(err)
}

// disable all lifecycle management, ensure it's actually disabled
func (r *BeatRunner) TestAllLifecyclesDisabled() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	r.CleanupTemplates(ctx)

	_, err := r.run("setup", "--index-management", "--E=setup.ilm.enabled=false", "--E=setup.dsl.enabled=false")
	r.Require().NoError(err)

	// make sure we have data streams, but there's no lifecycles
	streams, err := estools.GetDataStreamsForPattern(ctx, r.esClient, fmt.Sprintf("*%s*", r.beatName))
	r.Require().NoError(err)

	r.Require().NotEmpty(streams.DataStreams, "found no datastreams")
	foundPolicy := false
	for _, stream := range streams.DataStreams {
		if stream.Lifecycle.DataRetention != "" {
			foundPolicy = true
			break
		}
	}
	r.Require().False(foundPolicy, "Found a lifecycle policy despite disabling lifecycles. Found: %#v", streams)
}

// the export command doesn't actually make a network connection,
// so this won't fail
func (r *BeatRunner) TestExport() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	r.requireServerless(ctx)

	proc, err := r.run("export", "ilm-policy", "--E=setup.ilm.enabled=true")
	r.NoError(err, "export ilm-policy failed")
	// check to see if we got a valid output
	policy := map[string]any{}
	r.Require().NoError(json.Unmarshal([]byte(r.stdout(proc)), &policy))

	r.Require().NotEmpty(policy["policy"])
}

// tests beat export with DSL
func (r *BeatRunner) TestExportDSL() {
	proc, err := r.run("export", "ilm-policy", "--E=setup.dsl.enabled=true")
	r.NoError(err, "export ilm-policy failed")
	// check to see if we got a valid output
	policy := map[string]any{}
	r.Require().NoError(json.Unmarshal([]byte(r.stdout(proc)), &policy))

	r.Require().NotEmpty(policy["data_retention"])
}

func (r *BeatRunner) SubtestExportTemplates() {
	outDir := r.T().TempDir()

	_, err := r.run("export", "template", "--dir", outDir)
	r.NoError(err, "export template failed")

	inFolder, err := os.ReadDir(filepath.Join(outDir, "/template"))
	r.Require().NoError(err)
	r.T().Logf("got log contents: %#v", inFolder)
	r.Require().NotEmpty(inFolder)
}

func (r *BeatRunner) SubtestExportIndexPatterns() {
	proc, err := r.run("export", "index-pattern")
	r.NoError(err, "export index-pattern failed")

	idxPattern := map[string]any{}
	r.Require().NoError(json.Unmarshal([]byte(r.stdout(proc)), &idxPattern))
	r.Require().NotNil(idxPattern["attributes"])
}

// requireServerless skips the test unless the cluster is serverless.
func (r *BeatRunner) requireServerless(ctx context.Context) {
	info, err := estools.GetPing(ctx, r.esClient)
	r.Require().NoError(err)
	if info.Version.BuildFlavor != "serverless" {
		r.T().Skip("must run on serverless")
	}
}

// writePolicy writes a DSL policy file and returns its path.
func (r *BeatRunner) writePolicy(policy mapstr.M) string {
	raw, err := json.MarshalIndent(policy, "", " ")
	r.Require().NoError(err)
	path := filepath.Join(r.T().TempDir(), "dsl_policy.json")
	r.Require().NoError(os.WriteFile(path, raw, 0o600))
	return path
}

// CheckDSLPolicy checks if we have a match for the given DSL policy given a template name and policy data_retention
func (r *BeatRunner) CheckDSLPolicy(ctx context.Context, tmpl string, policy string) {
	streams, err := estools.GetDataStreamsForPattern(ctx, r.esClient, tmpl)
	r.Require().NoError(err)

	foundCustom := false
	for _, stream := range streams.DataStreams {
		if stream.Lifecycle.DataRetention == policy {
			foundCustom = true
			break
		}
	}

	r.Require().True(foundCustom, "did not find our lifecycle policy. Found: %#v", streams)
}

// CleanupTemplates removes any existing index
func (r *BeatRunner) CleanupTemplates(ctx context.Context) {
	_ = estools.DeleteIndexTemplatesDataStreams(ctx, r.esClient, fmt.Sprintf("%s*", r.beatName))
	_ = estools.DeleteIndexTemplatesDataStreams(ctx, r.esClient, "*custom-name*")
}

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

// Package beatsuite runs a Beat's setup, export and run commands against an
// Elastic Cloud serverless project.
package beatsuite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/go-elasticsearch/v8"

	serverless "github.com/elastic/beats/v7/testing/go-serverless"
)

const (
	// defaultRegion is used when SERVERLESS_REGION is unset. The client falls
	// back to the first region the API lists if this one isn't offered.
	defaultRegion = "aws-us-east-1"

	// beatStopTimeout is how long a Beat gets to exit after SIGINT before it's killed.
	beatStopTimeout = 30 * time.Second
)

// Config selects the Beat under test.
type Config struct {
	// BeatName is the name of the Beat, e.g. "filebeat". It must be one of
	// the Beats with a configuration in beatConfigs.
	BeatName string
	// HomeDir is the directory with the extracted Beat package: the binary
	// named BeatName and the kibana directory. Defaults to $BEAT_HOME.
	HomeDir string
}

// beatConfigs holds the Beat specific part of the configuration for each supported Beat.
var beatConfigs = map[string]string{
	"metricbeat": `
metricbeat.config.modules:
  path: ${path.config}/modules.d/*.yml
`,
	"filebeat": `
filebeat.modules:
  - module: system
    syslog:
      enabled: true
    auth:
      enabled: true
filebeat.inputs:
  - type: filestream
    id: serverless-test
    paths:
      - {{.log_file}}
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
  hosts: ["{{.es_host}}"]
  api_key: "{{.key_user}}:{{.key_pass}}"
  allow_older_versions: true
setup.kibana:
  host: {{.kb_host}}
processors:
  - add_fields:
      target: host
      fields:
        test-id: {{.test_id}}
`

// Run creates a serverless project, runs the test suite for the Beat
// described by cfg against it and deletes the project afterwards.
// It requires the Cloud API key in EC_API_KEY.
func Run(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.HomeDir == "" {
		cfg.HomeDir = os.Getenv("BEAT_HOME")
	}
	require.NotEmpty(t, cfg.HomeDir, "Config.HomeDir or BEAT_HOME must be set")
	_, ok := beatConfigs[cfg.BeatName]
	require.True(t, ok, "unsupported beat %q", cfg.BeatName)
	apiKey := os.Getenv("EC_API_KEY")
	require.NotEmpty(t, apiKey, "EC_API_KEY must be set to create the serverless project")

	suite.Run(t, &BeatRunner{cfg: cfg, apiKey: apiKey})
}

// BeatRunner is the suite.
type BeatRunner struct {
	suite.Suite
	cfg    Config
	apiKey string

	cloud   *serverless.Client
	project serverless.Project

	esClient     *elasticsearch.Client
	kibanaClient *kibana.Client

	// cfgFile is the Beat configuration, dataDir holds its data and logs.
	cfgFile string
	dataDir string
	testID  string
}

func (r *BeatRunner) binary() string { return filepath.Join(r.cfg.HomeDir, r.cfg.BeatName) }

// SetupSuite creates the serverless project and writes the Beat configuration.
func (r *BeatRunner) SetupSuite() {
	t := r.T()
	r.cloud = serverless.NewClient("", r.apiKey, serverless.ProjectTypeObservability)

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()

	region := os.Getenv("SERVERLESS_REGION")
	if region == "" {
		region = defaultRegion
	}
	region, err := r.cloud.PickRegion(ctx, region)
	require.NoError(t, err, "could not pick a serverless region")

	id, err := uuid.NewV4()
	require.NoError(t, err, "could not generate a UUID")
	r.testID = id.String()
	name := fmt.Sprintf("beats-ci-%s-%s", r.cfg.BeatName, r.testID[:8])
	if build := os.Getenv("BUILDKITE_BUILD_NUMBER"); build != "" {
		name = fmt.Sprintf("beats-ci-%s-%s-%s", r.cfg.BeatName, build, r.testID[:8])
	}

	t.Logf("creating serverless project %s in %s", name, region)
	r.project, err = r.cloud.Create(ctx, name, region)
	if r.project.ID != "" {
		// Register the cleanup before anything else can fail, so we never leak a project.
		t.Cleanup(r.deleteProject)
	}
	require.NoError(t, err, "could not create the serverless project")

	r.project, err = r.cloud.WaitReady(ctx, r.project)
	require.NoError(t, err, "serverless project did not become ready")
	t.Logf("project %s is ready: ES: %s Kibana: %s", r.project.ID, r.project.Endpoints.Elasticsearch, r.project.Endpoints.Kibana)

	r.esClient, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{r.project.Endpoints.Elasticsearch},
		Username:  r.project.Credentials.Username,
		Password:  r.project.Credentials.Password,
	})
	require.NoError(t, err, "could not create the Elasticsearch client")

	r.kibanaClient, err = kibana.NewClientWithConfig(&kibana.ClientConfig{
		Host:     withPort(t, r.project.Endpoints.Kibana), // the client would default to :5601
		Username: r.project.Credentials.Username,
		Password: r.project.Credentials.Password,
	}, "", "", "", "")
	require.NoError(t, err, "could not create the Kibana client")

	r.writeBeatConfig(ctx)
}

func (r *BeatRunner) deleteProject() {
	// Don't derive from the test context: it's cancelled by the time cleanups run.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := r.cloud.Delete(ctx, r.project); err != nil {
		r.T().Logf("WARNING: could not delete serverless project %s, remove it manually: %s", r.project.ID, err)
	}
}

func (r *BeatRunner) writeBeatConfig(ctx context.Context) {
	t := r.T()
	apiKey, err := estools.CreateAPIKey(ctx, r.esClient, estools.APIKeyRequest{Name: "test-api-key", Expiration: "1d"})
	require.NoError(t, err, "could not create an API key")

	r.dataDir = t.TempDir()
	logFile := filepath.Join(r.dataDir, "input.log")
	var lines strings.Builder
	for i := range 100 {
		fmt.Fprintf(&lines, "serverless test line %d\n", i)
	}
	require.NoError(t, os.WriteFile(logFile, []byte(lines.String()), 0o600), "could not write the input log")

	vars := map[string]string{
		"es_host":  withPort(t, r.project.Endpoints.Elasticsearch),
		"kb_host":  withPort(t, r.project.Endpoints.Kibana),
		"key_user": apiKey.ID,
		"key_pass": apiKey.APIKey,
		"test_id":  r.testID,
		"log_file": logFile,
	}
	tmpl, err := template.New("config").Parse(outputConfig + beatConfigs[r.cfg.BeatName])
	require.NoError(t, err, "could not parse the Beat configuration template")
	var rendered bytes.Buffer
	require.NoError(t, tmpl.Execute(&rendered, vars), "could not render the Beat configuration")

	r.cfgFile = filepath.Join(r.dataDir, r.cfg.BeatName+".yml")
	require.NoError(t, os.WriteFile(r.cfgFile, rendered.Bytes(), 0o600), "could not write the Beat configuration")
}

// withPort adds the default HTTPS port to endpoint if it has none. Beats adds
// standard ports to URLs that lack them, and the Cloud API returns endpoints
// without a port.
func withPort(t *testing.T, endpoint string) string {
	t.Helper()
	parsed, err := url.Parse(endpoint)
	require.NoError(t, err, "invalid endpoint %q", endpoint)
	if parsed.Port() == "" {
		return endpoint + ":443"
	}
	return endpoint
}

// beatArgs returns the arguments common to all invocations of the Beat.
func (r *BeatRunner) beatArgs(args ...string) []string {
	common := []string{
		"--path.home", r.cfg.HomeDir,
		"--path.data", filepath.Join(r.dataDir, "data"),
		"--path.logs", filepath.Join(r.dataDir, "logs"),
		"-c", r.cfgFile,
		"--strict.perms=false",
	}
	return append(common, args...)
}

// exec runs the Beat to completion and returns its combined output.
func (r *BeatRunner) exec(ctx context.Context, args ...string) ([]byte, error) {
	//nolint:gosec // G204 can be ignored: the binary path comes from the test configuration
	return exec.CommandContext(ctx, r.binary(), r.beatArgs(args...)...).CombinedOutput()
}

// runAndWaitForData runs the Beat until an event from this test is indexed, then stops it.
func (r *BeatRunner) runAndWaitForData(ctx context.Context) {
	t := r.T()
	var out bytes.Buffer
	//nolint:gosec // G204 can be ignored: the binary path comes from the test configuration
	cmd := exec.CommandContext(ctx, r.binary(), r.beatArgs("-e")...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Start(), "could not start the Beat")

	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	defer func() {
		_ = cmd.Process.Signal(syscall.SIGINT)
		select {
		case <-exited:
		case <-time.After(beatStopTimeout):
			_ = cmd.Process.Kill()
			<-exited
		}
	}()

	query := map[string]any{"match": map[string]any{"host.test-id": r.testID}}
	index := fmt.Sprintf("*%s*", r.cfg.BeatName)
	require.Eventually(t, func() bool {
		docs, err := estools.GetLatestDocumentMatchingQuery(ctx, r.esClient, query, index)
		return err == nil && len(docs.Hits.Hits) > 0
	}, 3*time.Minute, 5*time.Second, "no events from the Beat were indexed")

	select {
	case <-exited:
		t.Fatalf("the Beat exited while running; output:\n%s", out.String())
	default:
	}
}

// run the beat with default metricsets, ensure data is ingested
func (r *BeatRunner) TestRunAndCheckData() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute*5)
	defer cancel()

	// in case there's already a running template, delete it, forcing the beat to re-install
	r.CleanupTemplates(ctx)

	r.runAndWaitForData(ctx)
}

// tests the [beat] setup --dashboards command
func (r *BeatRunner) TestSetupDashboards() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute*3) //dashboards seem to take a while
	defer cancel()

	resp, err := r.exec(ctx, "setup", "--dashboards")
	r.NoError(err, "setup --dashboards failed")
	r.T().Logf("got response from dashboard setup: %s", string(resp))
	r.Require().Contains(string(resp), "Loaded dashboards")

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

	exportOut, err := r.exec(ctx, "export", "dashboard", "--folder", outDir, "--id", dashlist[0].ID)
	r.T().Logf("got output: %s", exportOut)
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
	if r.cfg.BeatName != "filebeat" {
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
	resp, err := r.exec(ctx, "setup", "--pipelines", "--modules", "apache", "-M", "apache.error.enabled=true", "-M", "apache.access.enabled=true")
	r.NoError(err, "setup --pipelines failed")
	r.T().Logf("got response from pipeline setup: %s", string(resp))

	pipelines, err := estools.GetPipelines(ctx, r.esClient, "*filebeat*")
	r.Require().NoError(err)
	r.Require().NotEmpty(pipelines)
}

// test beat setup --index-management with ILM disabled
func (r *BeatRunner) TestIndexManagementNoILM() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	resp, err := r.exec(ctx, "setup", "--index-management", "--E=setup.ilm.enabled=false")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.NoError(err, "setup --index-management failed")
	// we should not print a warning if we've explicitly disabled ILM
	r.NotContains(string(resp), "not supported")

	tmpls, err := estools.GetIndexTemplatesForPattern(ctx, r.esClient, fmt.Sprintf("*%s*", r.cfg.BeatName))
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

	resp, err := r.exec(ctx, "setup", "--index-management")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().NoError(err)

	streams, err := estools.GetDataStreamsForPattern(ctx, r.esClient, fmt.Sprintf("%s*", r.cfg.BeatName))
	r.Require().NoError(err)
	r.Require().NotEmpty(streams.DataStreams)
}

// test the setup process with mismatching template and DSL names
func (r *BeatRunner) TestCustomBadNames() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	resp, err := r.exec(ctx, "-e", "setup", "--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.data_stream_pattern='custom-bad-name'", "--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().NoError(err)

	r.Require().Contains(string(resp), "Additional updates & overwrites to this config will not work.")
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

	resp, err := r.exec(ctx, "setup", "--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.data_stream_pattern='custom-name'", "--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().NoError(err)

	r.CheckDSLPolicy(ctx, "*custom-name*", "7d")

	resp, err = r.exec(ctx, "setup", "--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.overwrite=true", "--E=setup.dsl.data_stream_pattern='custom-name'",
		"--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'", fmt.Sprintf("--E=setup.dsl.policy_file=%s", lifecyclePath))
	r.T().Logf("got response from management setup: %s", string(resp))
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

	resp, err := r.exec(ctx, "setup", "--index-management",
		"--E=setup.dsl.enabled=true", fmt.Sprintf("--E=setup.dsl.policy_file=%s", lifecyclePath))
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().NoError(err)

	r.CheckDSLPolicy(ctx, fmt.Sprintf("%s*", r.cfg.BeatName), "1d")
}

// tests beat setup --index-management with ILM explicitly set
// On serverless, this should fail.
func (r *BeatRunner) TestIndexManagementILMEnabledFailure() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	r.requireServerless(ctx)

	resp, err := r.exec(ctx, "setup", "--index-management", "--E=setup.ilm.enabled=true", "--E=setup.ilm.overwrite=true")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().Error(err)
	r.Contains(string(resp), "error creating")
}

// tests setup with both ILM and DSL enabled, should fail
func (r *BeatRunner) TestBothLifecyclesEnabled() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()

	resp, err := r.exec(ctx, "setup", "--index-management", "--E=setup.ilm.enabled=true", "--E=setup.dsl.enabled=true")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().Error(err)
}

// disable all lifecycle management, ensure it's actually disabled
func (r *BeatRunner) TestAllLifecyclesDisabled() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()
	defer r.CleanupTemplates(ctx)

	r.CleanupTemplates(ctx)

	resp, err := r.exec(ctx, "setup", "--index-management", "--E=setup.ilm.enabled=false", "--E=setup.dsl.enabled=false")
	r.T().Logf("got response from management setup: %s", string(resp))
	r.Require().NoError(err)

	// make sure we have data streams, but there's no lifecycles
	streams, err := estools.GetDataStreamsForPattern(ctx, r.esClient, fmt.Sprintf("*%s*", r.cfg.BeatName))
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

	resp, err := r.exec(ctx, "export", "ilm-policy", "--E=setup.ilm.enabled=true")
	r.T().Logf("got response from export: %s", string(resp))
	r.NoError(err, "export ilm-policy failed")
	// check to see if we got a valid output
	policy := map[string]any{}
	r.Require().NoError(json.Unmarshal(resp, &policy))

	r.Require().NotEmpty(policy["policy"])
}

// tests beat export with DSL
func (r *BeatRunner) TestExportDSL() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute)
	defer cancel()

	resp, err := r.exec(ctx, "export", "ilm-policy", "--E=setup.dsl.enabled=true")
	r.T().Logf("got response from export: %s", string(resp))
	r.NoError(err, "export ilm-policy failed")
	// check to see if we got a valid output
	policy := map[string]any{}
	r.Require().NoError(json.Unmarshal(resp, &policy))

	r.Require().NotEmpty(policy["data_retention"])
}

func (r *BeatRunner) SubtestExportTemplates() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute*2)
	defer cancel()
	outDir := r.T().TempDir()

	_, err := r.exec(ctx, "export", "template", "--dir", outDir)
	r.NoError(err, "export template failed")

	inFolder, err := os.ReadDir(filepath.Join(outDir, "/template"))
	r.Require().NoError(err)
	r.T().Logf("got log contents: %#v", inFolder)
	r.Require().NotEmpty(inFolder)
}

func (r *BeatRunner) SubtestExportIndexPatterns() {
	ctx, cancel := context.WithTimeout(r.T().Context(), time.Minute*2)
	defer cancel()

	rawPattern, err := r.exec(ctx, "export", "index-pattern")
	r.NoError(err, "export index-pattern failed")

	idxPattern := map[string]any{}
	r.Require().NoError(json.Unmarshal(rawPattern, &idxPattern))
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
	_ = estools.DeleteIndexTemplatesDataStreams(ctx, r.esClient, fmt.Sprintf("%s*", r.cfg.BeatName))
	_ = estools.DeleteIndexTemplatesDataStreams(ctx, r.esClient, "*custom-name*")
}

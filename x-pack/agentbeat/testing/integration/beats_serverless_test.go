// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/elastic/elastic-agent-libs/mapstr"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/estools"
)

type BeatRunner struct {
	suite.Suite
	requirementsInfo *define.Info
	agentFixture     *atesting.Fixture

	// connection info
	ESHost  string
	user    string
	pass    string
	kibHost string

	testUuid     string
	testbeatName string

	skipCleanup bool
}

func TestBeatsServerless(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Default,
		OS: []define.OS{
			{Type: define.Linux},
		},
		Stack: &define.Stack{},
		Local: false,
		Sudo:  true,
	})

	suite.Run(t, &BeatRunner{requirementsInfo: info})
}

func (runner *BeatRunner) SetupSuite() {
	runner.skipCleanup = false

	runner.testbeatName = os.Getenv("TEST_BINARY_NAME")
	if runner.testbeatName == "" {
		runner.T().Fatalf("TEST_BINARY_NAME must be set")
	}
	if runner.testbeatName == "elastic-agent" {
		runner.T().Skipf("tests must be run against a beat, not elastic-agent")
	}

	if runner.testbeatName != "filebeat" && runner.testbeatName != "metricbeat" && runner.testbeatName != "auditbeat" && runner.testbeatName != "packetbeat" {
		runner.T().Skip("test only supports metricbeat or filebeat")
	}
	runner.T().Logf("running serverless tests with %s", runner.testbeatName)

	agentFixture, err := define.NewFixtureWithBinary(runner.T(), define.Version(), runner.testbeatName, "/home/ubuntu", atesting.WithRunLength(time.Minute*3), atesting.WithAdditionalArgs([]string{"-E", "output.elasticsearch.allow_older_versions=true"}))
	runner.agentFixture = agentFixture
	require.NoError(runner.T(), err)

	// the require.* code will fail without these, so assume the values are non-nil
	runner.ESHost = os.Getenv("ELASTICSEARCH_HOST")
	runner.user = os.Getenv("ELASTICSEARCH_USERNAME")
	runner.pass = os.Getenv("ELASTICSEARCH_PASSWORD")
	runner.kibHost = os.Getenv("KIBANA_HOST")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	beatOutConfig := `
output.elasticsearch:
  hosts: ["{{.es_host}}"]
  api_key: "{{.key_user}}:{{.key_pass}}"
setup.kibana:
  host: {{.kb_host}}
processors:
  - add_fields:
      target: host
      fields:
        test-id: {{.test_id}}
{{.beat_cfg}}
`

	mbCfg := `
metricbeat.config.modules:
  path: ${path.config}/modules.d/*.yml
`

	fbCfg := `
filebeat.modules:
  - module: system
    syslog:
      enabled: true
    auth:
      enabled: true
filebeat.config.modules:
  - modules: system
    syslog:
      enabled: true
    auth:
      enabled: true
`
	auditbeatCfg := `
auditbeat.modules:

- module: file_integrity
  paths:
  - /bin
  - /usr/bin
  - /sbin
  - /usr/sbin
  - /etc
`

	packetbeatCfg := `
`

	tmpl, err := template.New("config").Parse(beatOutConfig)
	require.NoError(runner.T(), err)

	apiResp, err := estools.CreateAPIKey(ctx, runner.requirementsInfo.ESClient, estools.APIKeyRequest{Name: "test-api-key", Expiration: "1d"})
	require.NoError(runner.T(), err)

	// beats likes to add standard ports to URLs that don't have them, and ESS will sometimes return a URL without a port, assuming :443
	// so try to fix that here
	fixedKibanaHost := runner.kibHost
	parsedKibana, err := url.Parse(runner.kibHost)
	require.NoError(runner.T(), err)
	if parsedKibana.Port() == "" {
		fixedKibanaHost = fmt.Sprintf("%s:443", fixedKibanaHost)
	}

	fixedESHost := runner.ESHost
	parsedES, err := url.Parse(runner.ESHost)
	require.NoError(runner.T(), err)
	if parsedES.Port() == "" {
		fixedESHost = fmt.Sprintf("%s:443", fixedESHost)
	}

	runner.T().Logf("configuring beats with %s / %s", fixedESHost, fixedKibanaHost)

	testUuid, err := uuid.NewV4()
	require.NoError(runner.T(), err)
	runner.testUuid = testUuid.String()

	additionalCfg := mbCfg
	if runner.testbeatName == "filebeat" {
		additionalCfg = fbCfg
	} else if runner.testbeatName == "auditbeat" {
		additionalCfg = auditbeatCfg
	} else if runner.testbeatName == "packetbeat" {
		additionalCfg = packetbeatCfg
	}

	tmpl_map := map[string]string{"es_host": fixedESHost, "key_user": apiResp.Id, "key_pass": apiResp.APIKey, "kb_host": fixedKibanaHost, "test_id": testUuid.String(), "beat_cfg": additionalCfg}
	parsedCfg := bytes.Buffer{}
	err = tmpl.Execute(&parsedCfg, tmpl_map)
	require.NoError(runner.T(), err)

	err = runner.agentFixture.WriteFileToWorkDir(ctx, parsedCfg.String(), fmt.Sprintf("%s.yml", runner.testbeatName))
	require.NoError(runner.T(), err)
}

// run the beat with default metricsets, ensure no errors in logs + data is ingested
func (runner *BeatRunner) TestRunAndCheckData() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	// in case there's already a running template, delete it, forcing the beat to re-install
	runner.CleanupTemplates(ctx)

	err := runner.agentFixture.RunBeat(ctx)
	require.NoError(runner.T(), err)

	docs, err := estools.GetLatestDocumentMatchingQuery(ctx, runner.requirementsInfo.ESClient, map[string]interface{}{
		"match": map[string]interface{}{
			"host.test-id": runner.testUuid,
		},
	}, fmt.Sprintf("*%s*", runner.testbeatName))
	require.NoError(runner.T(), err)
	require.NotEmpty(runner.T(), docs.Hits.Hits)
}

// tests the [beat] setup --dashboards command
func (runner *BeatRunner) TestSetupDashboards() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3) //dashboards seem to take a while
	defer cancel()

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home", runner.agentFixture.WorkDir(), "setup", "--dashboards"})
	assert.NoError(runner.T(), err)
	runner.T().Logf("got response from dashboard setup: %s", string(resp))
	require.True(runner.T(), strings.Contains(string(resp), "Loaded dashboards"))

	dashList, err := tools.GetDashboards(ctx, runner.requirementsInfo.KibanaClient)
	require.NoError(runner.T(), err)

	// interesting hack in cases where we don't have a clean environment
	// check to see if any of the dashboards were created recently
	found := false
	for _, dash := range dashList {
		if time.Since(dash.UpdatedAt) < time.Minute*5 {
			found = true
			break
		}
	}
	require.True(runner.T(), found, fmt.Sprintf("could not find dashboard newer than 5 minutes, out of %d dashboards", len(dashList)))

	runner.Run("export dashboards", runner.SubtestExportDashboards)
	// cleanup
	if !runner.skipCleanup {
		for _, dash := range dashList {
			err = tools.DeleteDashboard(ctx, runner.requirementsInfo.KibanaClient, dash.ID)
			if err != nil {
				runner.T().Logf("WARNING: could not delete dashboards after test: %s", err)
				break
			}
		}
	}
}

// tests the [beat] export dashboard command
func (runner *BeatRunner) SubtestExportDashboards() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	outDir := runner.T().TempDir()

	dashlist, err := tools.GetDashboards(ctx, runner.requirementsInfo.KibanaClient)
	require.NoError(runner.T(), err)
	require.NotEmpty(runner.T(), dashlist)

	exportOut, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"export",
		"dashboard", "--folder", outDir, "--id", dashlist[0].ID})

	runner.T().Logf("got output: %s", exportOut)
	assert.NoError(runner.T(), err)

	// The folder matches the major version of Kibana, so we read it from the API
	dashboardFolder := fmt.Sprintf("/_meta/kibana/%d/dashboard", runner.requirementsInfo.KibanaClient.GetVersion().Major)
	inFolder, err := os.ReadDir(filepath.Join(outDir, dashboardFolder))
	require.NoError(runner.T(), err)
	runner.T().Logf("got log contents: %#v", inFolder)
	require.NotEmpty(runner.T(), inFolder)
}

// NOTE for the below tests: the testing framework doesn't guarantee a new stack instance each time,
// which means we might be running against a stack where a previous test has already done setup.
// perhaps CI should run `mage integration:clean` first?

// tests the [beat] setup --pipelines command
func (runner *BeatRunner) TestSetupPipelines() {
	if runner.testbeatName != "filebeat" {
		runner.T().Skip("pipelines only available on filebeat")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	defer func() {
		// cleanup
		if !runner.skipCleanup {
			err := estools.DeletePipelines(ctx, runner.requirementsInfo.ESClient, "*filebeat*")
			if err != nil {
				runner.T().Logf("WARNING: could not clean up pipelines: %s", err)
			}
		}

	}()

	// need to actually enable something that has pipelines
	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home", runner.agentFixture.WorkDir(),
		"setup", "--pipelines", "--modules", "apache", "-M", "apache.error.enabled=true", "-M", "apache.access.enabled=true"})
	assert.NoError(runner.T(), err)

	runner.T().Logf("got response from pipeline setup: %s", string(resp))

	pipelines, err := estools.GetPipelines(ctx, runner.requirementsInfo.ESClient, "*filebeat*")
	require.NoError(runner.T(), err)
	require.NotEmpty(runner.T(), pipelines)

}

// test beat setup --index-management with ILM disabled
func (runner *BeatRunner) TestIndexManagementNoILM() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer func() {
		runner.CleanupTemplates(ctx)
	}()

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.ilm.enabled=false"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	assert.NoError(runner.T(), err)
	// we should not print a warning if we've explicitly disabled ILM
	assert.NotContains(runner.T(), string(resp), "not supported")

	tmpls, err := estools.GetIndexTemplatesForPattern(ctx, runner.requirementsInfo.ESClient, fmt.Sprintf("*%s*", runner.testbeatName))
	require.NoError(runner.T(), err)
	for _, tmpl := range tmpls.IndexTemplates {
		runner.T().Logf("got template: %s", tmpl.Name)
	}
	require.NotEmpty(runner.T(), tmpls.IndexTemplates)

	runner.Run("export templates", runner.SubtestExportTemplates)
	runner.Run("export index patterns", runner.SubtestExportIndexPatterns)

}

// tests setup with all default settings
func (runner *BeatRunner) TestWithAllDefaults() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer func() {
		runner.CleanupTemplates(ctx)
	}()

	// pre-delete in case something else missed cleanup
	runner.CleanupTemplates(ctx)

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.NoError(runner.T(), err)

	streams, err := estools.GetDataStreamsForPattern(ctx, runner.requirementsInfo.ESClient, fmt.Sprintf("%s*", runner.testbeatName))
	require.NoError(runner.T(), err)

	require.NotEmpty(runner.T(), streams.DataStreams)

}

// test the setup process with mismatching template and DSL names
func (runner *BeatRunner) TestCustomBadNames() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	defer func() {
		runner.CleanupTemplates(ctx)
	}()

	resp, err := runner.agentFixture.Exec(ctx, []string{"-e", "--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.data_stream_pattern='custom-bad-name'", "--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.NoError(runner.T(), err)

	require.True(runner.T(), strings.Contains(string(resp), "Additional updates & overwrites to this config will not work."))

}

func (runner *BeatRunner) TestOverwriteWithCustomName() {
	//an updated policy that has a different value than the default of 7d
	updatedPolicy := mapstr.M{
		"data_retention": "1d",
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer func() {
		runner.CleanupTemplates(ctx)
	}()

	lctemp := runner.T().TempDir()
	raw, err := json.MarshalIndent(updatedPolicy, "", " ")
	require.NoError(runner.T(), err)

	lifecyclePath := filepath.Join(lctemp, "dsl_policy.json")

	err = os.WriteFile(lifecyclePath, raw, 0o744)
	require.NoError(runner.T(), err)

	runner.CleanupTemplates(ctx)

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.data_stream_pattern='custom-name'", "--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.NoError(runner.T(), err)

	runner.CheckDSLPolicy(ctx, "*custom-name*", "7d")

	resp, err = runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.dsl.enabled=true", "--E=setup.dsl.overwrite=true", "--E=setup.dsl.data_stream_pattern='custom-name'",
		"--E=setup.template.name='custom-name'", "--E=setup.template.pattern='custom-name'", fmt.Sprintf("--E=setup.dsl.policy_file=%s", lifecyclePath)})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.NoError(runner.T(), err)

	runner.CheckDSLPolicy(ctx, "*custom-name*", "1d")

}

// TestWithCustomLifecyclePolicy uploads a custom DSL policy
func (runner *BeatRunner) TestWithCustomLifecyclePolicy() {
	//create a custom policy file
	dslPolicy := mapstr.M{
		"data_retention": "1d",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer func() {
		runner.CleanupTemplates(ctx)
	}()

	lctemp := runner.T().TempDir()
	raw, err := json.MarshalIndent(dslPolicy, "", " ")
	require.NoError(runner.T(), err)

	lifecyclePath := filepath.Join(lctemp, "dsl_policy.json")

	err = os.WriteFile(lifecyclePath, raw, 0o744)
	require.NoError(runner.T(), err)

	runner.CleanupTemplates(ctx)

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.dsl.enabled=true", fmt.Sprintf("--E=setup.dsl.policy_file=%s", lifecyclePath)})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.NoError(runner.T(), err)

	runner.CheckDSLPolicy(ctx, fmt.Sprintf("%s*", runner.testbeatName), "1d")

}

// tests beat setup --index-management with ILM explicitly set
// On serverless, this should fail.
func (runner *BeatRunner) TestIndexManagementILMEnabledFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	info, err := estools.GetPing(ctx, runner.requirementsInfo.ESClient)
	require.NoError(runner.T(), err)

	if info.Version.BuildFlavor != "serverless" {
		runner.T().Skip("must run on serverless")
	}

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.ilm.enabled=true", "--E=setup.ilm.overwrite=true"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.Error(runner.T(), err)
	assert.Contains(runner.T(), string(resp), "error creating")
}

// tests setup with both ILM and DSL enabled, should fail
func (runner *BeatRunner) TestBothLifecyclesEnabled() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.ilm.enabled=true", "--E=setup.dsl.enabled=true"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.Error(runner.T(), err)
}

// disable all lifecycle management, ensure it's actually disabled
func (runner *BeatRunner) TestAllLifecyclesDisabled() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer func() {
		runner.CleanupTemplates(ctx)
	}()

	runner.CleanupTemplates(ctx)

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"setup",
		"--index-management",
		"--E=setup.ilm.enabled=false", "--E=setup.dsl.enabled=false"})
	runner.T().Logf("got response from management setup: %s", string(resp))
	require.NoError(runner.T(), err)

	// make sure we have data streams, but there's no lifecycles
	streams, err := estools.GetDataStreamsForPattern(ctx, runner.requirementsInfo.ESClient, fmt.Sprintf("*%s*", runner.testbeatName))
	require.NoError(runner.T(), err)

	require.NotEmpty(runner.T(), streams.DataStreams, "found no datastreams")
	foundPolicy := false
	for _, stream := range streams.DataStreams {
		if stream.Lifecycle.DataRetention != "" {
			foundPolicy = true
			break
		}
	}
	require.False(runner.T(), foundPolicy, "Found a lifecycle policy despite disabling lifecycles. Found: %#v", streams)
}

// the export command doesn't actually make a network connection,
// so this won't fail
func (runner *BeatRunner) TestExport() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	info, err := estools.GetPing(ctx, runner.requirementsInfo.ESClient)
	require.NoError(runner.T(), err)

	if info.Version.BuildFlavor != "serverless" {
		runner.T().Skip("must run on serverless")
	}

	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"export", "ilm-policy", "--E=setup.ilm.enabled=true"})
	runner.T().Logf("got response from export: %s", string(resp))
	assert.NoError(runner.T(), err)
	// check to see if we got a valid output
	policy := map[string]interface{}{}
	err = json.Unmarshal(resp, &policy)
	require.NoError(runner.T(), err)

	require.NotEmpty(runner.T(), policy["policy"])
}

// tests beat export with DSL
func (runner *BeatRunner) TestExportDSL() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	resp, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"export", "ilm-policy", "--E=setup.dsl.enabled=true"})
	runner.T().Logf("got response from export: %s", string(resp))
	assert.NoError(runner.T(), err)
	// check to see if we got a valid output
	policy := map[string]interface{}{}
	err = json.Unmarshal(resp, &policy)
	require.NoError(runner.T(), err)

	require.NotEmpty(runner.T(), policy["data_retention"])
}

func (runner *BeatRunner) SubtestExportTemplates() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	outDir := runner.T().TempDir()

	_, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"export",
		"template", "--dir", outDir})
	assert.NoError(runner.T(), err)

	inFolder, err := os.ReadDir(filepath.Join(outDir, "/template"))
	require.NoError(runner.T(), err)
	runner.T().Logf("got log contents: %#v", inFolder)
	require.NotEmpty(runner.T(), inFolder)
}

func (runner *BeatRunner) SubtestExportIndexPatterns() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	rawPattern, err := runner.agentFixture.Exec(ctx, []string{"--path.home",
		runner.agentFixture.WorkDir(),
		"export",
		"index-pattern"})
	assert.NoError(runner.T(), err)

	idxPattern := map[string]interface{}{}

	err = json.Unmarshal(rawPattern, &idxPattern)
	require.NoError(runner.T(), err)
	require.NotNil(runner.T(), idxPattern["attributes"])
}

// CheckDSLPolicy checks if we have a match for the given DSL policy given a template name and policy data_retention
func (runner *BeatRunner) CheckDSLPolicy(ctx context.Context, tmpl string, policy string) {
	streams, err := estools.GetDataStreamsForPattern(ctx, runner.requirementsInfo.ESClient, tmpl)
	require.NoError(runner.T(), err)

	foundCustom := false
	for _, stream := range streams.DataStreams {
		if stream.Lifecycle.DataRetention == policy {
			foundCustom = true
			break
		}
	}

	require.True(runner.T(), foundCustom, "did not find our lifecycle policy. Found: %#v", streams)
}

// CleanupTemplates removes any existing index
func (runner *BeatRunner) CleanupTemplates(ctx context.Context) {
	if !runner.skipCleanup {
		_ = estools.DeleteIndexTemplatesDataStreams(ctx, runner.requirementsInfo.ESClient, fmt.Sprintf("%s*", runner.testbeatName))
		_ = estools.DeleteIndexTemplatesDataStreams(ctx, runner.requirementsInfo.ESClient, "*custom-name*")
	}
}

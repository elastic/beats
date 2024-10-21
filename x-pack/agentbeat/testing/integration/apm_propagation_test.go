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
	"io"
	"net/http"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/go-elasticsearch/v8"

	"github.com/elastic/elastic-agent/pkg/control/v2/client"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
)

const agentConfigTemplateString = `
outputs:
  default:
    type: fake-output
inputs:
  - id: fake-apm
    type: fake-apm
agent.monitoring:
  traces: true
  apm:
    hosts:
      - {{ .host }}
    environment: {{ .environment }}
    secret_token: {{ .secret_token }}
    global_labels:
      test_name: TestAPMConfig
      test_type: Agent integration test
    tls:
      skip_verify: true
`

func TestAPMConfig(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Default,
		Stack: &define.Stack{},
	})
	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	deadline := time.Now().Add(10 * time.Minute)
	ctx, cancel := testcontext.WithDeadline(t, context.Background(), deadline)
	defer cancel()

	err = f.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	name := "fake-apm"
	environment := info.Namespace

	agentConfig := generateAgentConfigForAPM(t, agentConfigTemplateString, info, environment)
	t.Logf("Rendered agent config:\n%s", agentConfig)

	testAPMTraces := func(ctx context.Context) error {
		state, err := f.Client().State(ctx)
		require.NoError(t, err)

		t.Logf("agent state: %+v", state)

		// test that APM traces are being sent using initial configuration
		require.Eventually(t, func() bool {
			count, errCount := countAPMTraces(ctx, t, info.ESClient, name, environment)
			if errCount != nil {
				t.Logf("Error retrieving APM traces count for service %q and environment %q: %s", name, environment, errCount)
				return false
			}
			return count > 0
		}, 1*time.Minute, time.Second)

		// change the configuration with a new environment and check that the update has been processed
		environment = environment + "-changed"
		modifiedAgentConfig := generateAgentConfigForAPM(t, agentConfigTemplateString, info, environment)
		t.Logf("Rendered agent modified config:\n%s", modifiedAgentConfig)
		err = f.Client().Configure(ctx, modifiedAgentConfig)
		require.NoError(t, err, "error updating agent config with a new APM environment")

		// check that we receive traces with the new environment string
		require.Eventually(t, func() bool {
			count, errCount := countAPMTraces(ctx, t, info.ESClient, name, environment)
			if errCount != nil {
				t.Logf("Error retrieving APM traces count for service %q and environment %q: %s", name, environment, errCount)
				return false
			}
			return count > 0
		}, 1*time.Minute, time.Second)

		return nil
	}

	err = f.Run(ctx, atesting.State{
		Configure:  agentConfig,
		AgentState: atesting.NewClientState(client.Healthy),
		Components: map[string]atesting.ComponentState{
			"fake-apm-default": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					atesting.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-apm-default"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					atesting.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-apm-default-fake-apm"}: {
						State: atesting.NewClientState(client.Healthy),
					},
				},
			},
		},
		After: testAPMTraces,
	})

	require.NoError(t, err)

}

func countAPMTraces(ctx context.Context, t *testing.T, esClient *elasticsearch.Client, serviceName, environment string) (int, error) {
	queryRaw := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"service.name": map[string]interface{}{
								"value": serviceName,
							},
						},
					},
					{
						"term": map[string]interface{}{
							"service.environment": map[string]interface{}{
								"value": environment,
							},
						},
					},
				},
			},
		},
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(queryRaw)
	if err != nil {
		return 0, fmt.Errorf("error encoding query: %w", err)
	}

	count := esClient.Count

	response, err := count(
		count.WithContext(ctx),
		count.WithIndex("traces-apm-default"),
		count.WithBody(buf),
	)
	if err != nil {
		return 0, fmt.Errorf("error executing query: %w", err)
	}

	defer response.Body.Close()

	var body struct {
		Count int
	}

	// decoder := json.NewDecoder(response.Body)
	// err = decoder.Decode(&body)
	bodyBytes, _ := io.ReadAll(response.Body)

	t.Logf("received ES response: %s", bodyBytes)
	err = json.Unmarshal(bodyBytes, &body)

	return body.Count, err
}

// types to correctly parse the APM config we get from kibana API
type apmConfigResponse struct {
	CloudStandaloneSetup CloudStandaloneSetup `json:"cloudStandaloneSetup,omitempty"`
	IsFleetEnabled       bool                 `json:"isFleetEnabled,omitempty"`
	FleetAgents          []FleetAgents        `json:"fleetAgents,omitempty"`
}
type CloudStandaloneSetup struct {
	ApmServerURL string `json:"apmServerUrl,omitempty"`
	SecretToken  string `json:"secretToken,omitempty"`
}
type FleetAgents struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	ApmServerURL string `json:"apmServerUrl,omitempty"`
	SecretToken  string `json:"secretToken,omitempty"`
}

func generateAgentConfigForAPM(t *testing.T, configTemplate string, info *define.Info, environment string) string {
	t.Helper()
	apmConfigData := getAPMConfigFromKibana(t, info.KibanaClient)

	configT, err := template.New("test config").Parse(configTemplate)
	require.NoErrorf(t, err, "Error parsing agent config template\n%s", configTemplate)

	buf := new(strings.Builder)
	templateData := map[string]any{
		"environment":  environment,
		"secret_token": apmConfigData.SecretToken,
		"host":         apmConfigData.ApmServerURL,
	}
	err = configT.Execute(buf, templateData)
	require.NoErrorf(t, err, "Error rendering template\n%s\nwith data %v", configTemplate, templateData)
	return buf.String()
}

func getAPMConfigFromKibana(t *testing.T, kc *kibana.Client) CloudStandaloneSetup {
	t.Helper()
	response, err := kc.Send(http.MethodGet, "/internal/apm/fleet/agents", nil, nil, nil)
	require.NoError(t, err, "Error getting APM connection params from kibana")
	defer response.Body.Close()

	responseBytes, err := io.ReadAll(response.Body)
	require.NoError(t, err, "Error reading data from http response")
	apmConfig := new(apmConfigResponse)
	err = json.Unmarshal(responseBytes, apmConfig)
	require.NoError(t, err, "Error unmarshalling apm config")
	require.NotEmpty(t, apmConfig.CloudStandaloneSetup.ApmServerURL, "APM config URL is empty")
	require.NotEmpty(t, apmConfig.CloudStandaloneSetup.SecretToken, "APM config token is empty")

	return apmConfig.CloudStandaloneSetup
}

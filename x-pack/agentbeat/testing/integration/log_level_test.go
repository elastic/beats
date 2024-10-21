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

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent/pkg/control/v2/cproto"
	"github.com/elastic/elastic-agent/pkg/core/logger"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/pkg/utils"
)

func TestSetLogLevelFleetManaged(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Fleet,
		Stack: &define.Stack{},
		Sudo:  true,
	})

	deadline := time.Now().Add(10 * time.Minute)
	ctx, cancel := testcontext.WithDeadline(t, context.Background(), deadline)
	defer cancel()

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err, "failed creating agent fixture")

	policyResp, enrollmentTokenResp := createPolicyAndEnrollmentToken(ctx, t, info.KibanaClient, createBasicPolicy())
	t.Logf("Created policy %+v", policyResp.AgentPolicy)

	t.Log("Getting default Fleet Server URL...")
	fleetServerURL, err := fleettools.DefaultURL(ctx, info.KibanaClient)
	require.NoError(t, err, "failed getting Fleet Server URL")

	installOutput, err := f.Install(ctx, &atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
		EnrollOpts: atesting.EnrollOpts{
			URL:             fleetServerURL,
			EnrollmentToken: enrollmentTokenResp.APIKey,
		},
	})

	assert.NoErrorf(t, err, "Error installing agent. Install output:\n%s\n", string(installOutput))

	require.Eventuallyf(t, func() bool {
		return waitForAgentAndFleetHealthy(ctx, t, f)
	}, time.Minute, time.Second, "agent never became healthy or connected to Fleet")

	// get the agent ID
	agentID, err := getAgentID(ctx, f)
	require.NoError(t, err, "error getting the agent ID")

	testLogLevelSetViaFleet(ctx, f, agentID, t, info, policyResp)
}

func testLogLevelSetViaFleet(ctx context.Context, f *atesting.Fixture, agentID string, t *testing.T, info *define.Info, policyResp kibana.PolicyResponse) {

	// Step 0: get the initial log level reported by agent
	initialLogLevel, err := getLogLevelFromInspectOutput(ctx, f)
	require.NoError(t, err, "error retrieving agent log level")
	assert.Equal(t, logger.DefaultLogLevel.String(), initialLogLevel, "unexpected default log level at agent startup")

	// Step 1: set a different log level in Fleet policy
	policyLogLevel := logp.ErrorLevel

	t.Logf("Setting policy log level to %q", policyLogLevel.String())
	// make sure we are changing something
	require.NotEqualf(t, logger.DefaultLogLevel, policyLogLevel, "Policy log level %s should be different than agent default log level", policyLogLevel)
	// set policy log level and verify that eventually the agent sets it
	err = updatePolicyLogLevel(ctx, t, info.KibanaClient, policyResp.AgentPolicy, policyLogLevel.String())
	require.NoError(t, err, "error updating policy log level")

	// assert `elastic-agent inspect` eventually reports the new log level
	// TODO re-enable inspect assertion after https://github.com/elastic/elastic-agent/issues/4870 is solved
	//assert.Eventuallyf(t, func() bool {
	//	agentLogLevel, err := getLogLevelFromInspectOutput(ctx, f)
	//	if err != nil {
	//		t.Logf("error getting log level from agent: %v", err)
	//		return false
	//	}
	//	t.Logf("Agent log level: %q policy log level: %q", agentLogLevel, policyLogLevel)
	//	return agentLogLevel == policyLogLevel.String()
	//}, 30*time.Second, time.Second, "agent never received expected log level %q", policyLogLevel)

	// assert Fleet eventually receives the new log level from agent through checkin
	assert.Eventuallyf(t, func() bool {
		fleetMetadataLogLevel, err := getLogLevelFromFleetMetadata(ctx, t, info.KibanaClient, agentID)
		if err != nil {
			t.Logf("error getting log level for agent %q from Fleet metadata: %v", agentID, err)
			return false
		}
		t.Logf("Fleet metadata log level for agent %q: %q policy log level: %q", agentID, fleetMetadataLogLevel, policyLogLevel)
		return fleetMetadataLogLevel == policyLogLevel.String()
	}, 30*time.Second, time.Second, "agent never communicated policy log level %q to Fleet", policyLogLevel)

	// Step 2: set a different log level for the specific agent using Settings action
	// set agent log level and verify that it takes precedence over the policy one
	agentLogLevel := logp.DebugLevel.String()

	t.Logf("Setting agent log level to %q", agentLogLevel)

	err = updateAgentLogLevel(ctx, t, info.KibanaClient, agentID, agentLogLevel)
	require.NoError(t, err, "error updating agent log level")

	// TODO re-enable inspect assertion after https://github.com/elastic/elastic-agent/issues/4870 is solved
	//assert.Eventuallyf(t, func() bool {
	//	actualAgentLogLevel, err := getLogLevelFromInspectOutput(ctx, f)
	//	if err != nil {
	//		t.Logf("error getting log level from agent: %v", err)
	//		return false
	//	}
	//	t.Logf("Agent log level: %q, expected level: %q", actualAgentLogLevel, agentLogLevel)
	//	return actualAgentLogLevel == agentLogLevel
	//}, 2*time.Minute, time.Second, "agent never received agent-specific log level %q", agentLogLevel)

	// assert Fleet eventually receives the new log level from agent through checkin
	assert.Eventuallyf(t, func() bool {
		fleetMetadataLogLevel, err := getLogLevelFromFleetMetadata(ctx, t, info.KibanaClient, agentID)
		if err != nil {
			t.Logf("error getting log level for agent %q from Fleet metadata: %v", agentID, err)
			return false
		}
		t.Logf("Fleet metadata log level for agent %q: %q agent log level: %q", agentID, fleetMetadataLogLevel, agentLogLevel)
		return fleetMetadataLogLevel == agentLogLevel
	}, 30*time.Second, time.Second, "agent never communicated agent-specific log level %q to Fleet", agentLogLevel)

	// Step 3: Clear the agent-specific log level override, verify that we revert to policy log level
	t.Logf("Clearing agent log level, expecting log level to revert back to %q", policyLogLevel)
	err = updateAgentLogLevel(ctx, t, info.KibanaClient, agentID, "")
	require.NoError(t, err, "error clearing agent log level")

	// assert `elastic-agent inspect` eventually reports the new log level
	// TODO re-enable inspect assertion after https://github.com/elastic/elastic-agent/issues/4870 is solved
	//assert.Eventuallyf(t, func() bool {
	//	actualAgentLogLevel, err := getLogLevelFromInspectOutput(ctx, f)
	//	if err != nil {
	//		t.Logf("error getting log level from agent: %v", err)
	//		return false
	//	}
	//	t.Logf("Agent log level: %q policy log level: %q", actualAgentLogLevel, policyLogLevel)
	//	return actualAgentLogLevel == policyLogLevel.String()
	//}, 30*time.Second, time.Second, "agent never reverted to policy log level %q", policyLogLevel)

	// assert Fleet eventually receives the new log level from agent through checkin
	assert.Eventuallyf(t, func() bool {
		fleetMetadataLogLevel, err := getLogLevelFromFleetMetadata(ctx, t, info.KibanaClient, agentID)
		if err != nil {
			t.Logf("error getting log level for agent %q from Fleet metadata: %v", agentID, err)
			return false
		}
		t.Logf("Fleet metadata log level for agent %q: %q policy log level: %q", agentID, fleetMetadataLogLevel, policyLogLevel)
		return fleetMetadataLogLevel == policyLogLevel.String()
	}, 30*time.Second, time.Second, "agent never communicated reverting to policy log level %q to Fleet", policyLogLevel)

	// Step 4: Clear the log level in policy and verify that agent reverts to the initial log level
	t.Logf("Clearing policy log level, expecting log level to revert back to %q", initialLogLevel)
	err = updatePolicyLogLevel(ctx, t, info.KibanaClient, policyResp.AgentPolicy, "")
	require.NoError(t, err, "error clearing policy log level")

	// assert `elastic-agent inspect` eventually reports the initial log level
	// TODO re-enable inspect assertion after https://github.com/elastic/elastic-agent/issues/4870 is solved
	//assert.Eventuallyf(t, func() bool {
	//	actualAgentLogLevel, err := getLogLevelFromInspectOutput(ctx, f)
	//	if err != nil {
	//		t.Logf("error getting log level from agent: %v", err)
	//		return false
	//	}
	//	t.Logf("Agent log level: %q initial log level: %q", actualAgentLogLevel, initialLogLevel)
	//	return actualAgentLogLevel == initialLogLevel
	//}, 2*time.Minute, time.Second, "agent never reverted to initial log level %q", initialLogLevel)

	// assert Fleet eventually receives the new log level from agent through checkin
	assert.Eventuallyf(t, func() bool {
		fleetMetadataLogLevel, err := getLogLevelFromFleetMetadata(ctx, t, info.KibanaClient, agentID)
		if err != nil {
			t.Logf("error getting log level for agent %q from Fleet metadata: %v", agentID, err)
			return false
		}
		t.Logf("Fleet metadata log level for agent %q: %q initial log level: %q", agentID, fleetMetadataLogLevel, initialLogLevel)
		return fleetMetadataLogLevel == initialLogLevel
	}, 30*time.Second, time.Second, "agent never communicated initial log level %q to Fleet", initialLogLevel)
}

func waitForAgentAndFleetHealthy(ctx context.Context, t *testing.T, f *atesting.Fixture) bool {
	status, err := f.ExecStatus(ctx)
	if err != nil {
		t.Logf("error fetching agent status: %v", err)
		return false
	}

	statusBuffer := new(strings.Builder)
	err = json.NewEncoder(statusBuffer).Encode(status)
	if err != nil {
		t.Logf("error marshaling agent status: %v", err)
	} else {
		t.Logf("agent status: %v", statusBuffer.String())
	}

	return status.State == int(cproto.State_HEALTHY) && status.FleetState == int(cproto.State_HEALTHY)
}

func updateAgentLogLevel(ctx context.Context, t *testing.T, kibanaClient *kibana.Client, agentID string, logLevel string) error {
	updateLogLevelTemplateString := `{
		"action": {
			"type": "SETTINGS",
			"data": {
				"log_level": {{ .logLevel }}
			}
		}
	}`
	updateLogLevelTemplate, err := template.New("updatePolicyLogLevel").Parse(updateLogLevelTemplateString)
	if err != nil {
		return fmt.Errorf("error parsing update log level request template: %w", err)
	}

	buf := new(bytes.Buffer)
	templateData := map[string]string{}
	if logLevel != "" {
		templateData["logLevel"] = `"` + logLevel + `"`
	} else {
		templateData["logLevel"] = "null"
	}

	err = updateLogLevelTemplate.Execute(buf, templateData)
	t.Logf("Updating agent-specific log level to %q", logLevel)
	_, err = kibanaClient.SendWithContext(ctx, http.MethodPost, "/api/fleet/agents/"+agentID+"/actions", nil, nil, buf)
	if err != nil {
		return fmt.Errorf("error executing fleet request: %w", err)
	}

	// The log below is a bit spammy but it can be useful for debugging
	//respDump, err := httputil.DumpResponse(fleetResp, true)
	//if err != nil {
	//	t.Logf("Error dumping Fleet response to updating agent-specific log level: %v", err)
	//} else {
	//	t.Logf("Fleet response to updating agent-specific log level:\n----- BEGIN RESPONSE DUMP -----\n%s\n----- END RESPONSE DUMP -----\n", string(respDump))
	//}

	return nil
}

func updatePolicyLogLevel(ctx context.Context, t *testing.T, kibanaClient *kibana.Client, policy kibana.AgentPolicy, newPolicyLogLevel string) error {
	// The request we would need is the one below, but at the time of writing there is no way to set overrides with fleet api definition in elastic-agent-libs, need to update
	// info.KibanaClient.UpdatePolicy(ctx, policyResp.ID, kibana.AgentPolicyUpdateRequest{})
	// Let's do a generic HTTP request

	updateLogLevelTemplateString := `{
	   "name": "{{ .policyName }}",
	   "namespace": "{{ .namespace }}",
	   "advanced_settings": {
		"agent_logging_level": {{ .logLevel }}	
	   }
	}`
	updateLogLevelTemplate, err := template.New("updatePolicyLogLevel").Parse(updateLogLevelTemplateString)
	if err != nil {
		return fmt.Errorf("error parsing update log level request template: %w", err)
	}

	buf := new(bytes.Buffer)
	templateData := map[string]string{"policyName": policy.Name, "namespace": policy.Namespace}
	if newPolicyLogLevel == "" {
		// to reset the log level we have to set it to null
		templateData["logLevel"] = "null"
	} else {
		templateData["logLevel"] = `"` + newPolicyLogLevel + `"`
	}

	err = updateLogLevelTemplate.Execute(buf, templateData)
	if err != nil {
		return fmt.Errorf("error rendering policy update template: %w", err)
	}

	_, err = kibanaClient.SendWithContext(ctx, http.MethodPut, "/api/fleet/agent_policies/"+policy.ID, nil, nil, buf)

	if err != nil {
		return fmt.Errorf("error executing fleet request: %w", err)
	}

	// The log below is a bit spammy but it can be useful for debugging
	//respDump, err := httputil.DumpResponse(fleetResp, true)
	//if err != nil {
	//	t.Logf("Error dumping Fleet response to updating policy log level: %v", err)
	//} else {
	//	t.Logf("Fleet response to updating policy log level:\n----- BEGIN RESPONSE DUMP -----\n%s\n----- END RESPONSE DUMP -----\n", string(respDump))
	//}

	return nil
}

func getAgentID(ctx context.Context, f *atesting.Fixture) (string, error) {
	agentInspectOutput, err := f.ExecInspect(ctx)
	if err != nil {
		return "", fmt.Errorf("executing elastic-agent inspect: %w", err)
	}

	return agentInspectOutput.Agent.ID, nil
}

func getLogLevelFromInspectOutput(ctx context.Context, f *atesting.Fixture) (string, error) {
	agentInspectOutput, err := f.ExecInspect(ctx)
	if err != nil {
		return "", fmt.Errorf("executing elastic-agent inspect: %w", err)
	}

	return agentInspectOutput.Agent.Logging.Level, nil
}

func getLogLevelFromFleetMetadata(ctx context.Context, t *testing.T, kibanaClient *kibana.Client, agentID string) (string, error) {
	// The request we would need is kibanaClient.GetAgent(), but at the time of writing there is no way to get loglevel with fleet api definition in elastic-agent-libs, need to update
	// kibana.AgentCommon struct to pick up log level from `local_metadata`
	// Let's do a generic HTTP request

	response, err := kibanaClient.SendWithContext(ctx, http.MethodGet, "/api/fleet/agents/"+agentID, nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("getting agent from Fleet: %w", err)
	}
	defer response.Body.Close()

	// The log below is a bit spammy but it can be useful for debugging
	//dumpResponse, err := httputil.DumpResponse(response, true)
	//if err != nil {
	//	t.Logf(" error dumping agent metadata fleet response: %v", err)
	//} else {
	//	t.Logf("agent metadata fleet response:\n----- BEGIN RESPONSE DUMP -----\n%s\n----- END RESPONSE DUMP -----", dumpResponse)
	//}

	responseBodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body from Fleet: %w", err)
	}

	rawJson := map[string]any{}
	err = json.Unmarshal(responseBodyBytes, &rawJson)
	if err != nil {
		return "", fmt.Errorf("unmarshalling Fleet response: %w", err)
	}
	rawLogLevel, err := utils.GetNestedMap(rawJson, "item", "local_metadata", "elastic", "agent", "log_level")
	if err != nil {
		return "", fmt.Errorf("looking for item/local_metadata/elastic/agent/log_level key in Fleet response: %w", err)
	}

	if logLevel, ok := rawLogLevel.(string); ok {
		return logLevel, nil
	}
	return "", fmt.Errorf("loglevel from Fleet output is not a string: %T", rawLogLevel)
}

func createPolicyAndEnrollmentToken(ctx context.Context, t *testing.T, kibClient *kibana.Client, policy kibana.AgentPolicy) (kibana.PolicyResponse, kibana.CreateEnrollmentAPIKeyResponse) {
	t.Log("Creating Agent policy...")
	policyResp, err := kibClient.CreatePolicy(ctx, policy)
	require.NoError(t, err, "failed creating policy")

	t.Log("Creating Agent enrollment API key...")
	createEnrollmentApiKeyReq := kibana.CreateEnrollmentAPIKeyRequest{
		PolicyID: policyResp.ID,
	}
	enrollmentToken, err := kibClient.CreateEnrollmentAPIKey(ctx, createEnrollmentApiKeyReq)
	require.NoError(t, err, "failed creating enrollment API key")
	return policyResp, enrollmentToken
}
func createBasicPolicy() kibana.AgentPolicy {
	policyUUID := uuid.Must(uuid.NewV4()).String()
	return kibana.AgentPolicy{
		Name:              "testloglevel-policy-" + policyUUID,
		Namespace:         "default",
		Description:       "Test Log Level Policy " + policyUUID,
		MonitoringEnabled: []kibana.MonitoringEnabledOption{},
	}
}

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

package kibana

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"text/template"

	"github.com/gofrs/uuid/v5"

	"github.com/stretchr/testify/require"
)

func mustGetEnv(t *testing.T) ClientConfig {
	kibUser, foundUser := os.LookupEnv("KIBANA_USERNAME")
	kibPass, foundPass := os.LookupEnv("KIBANA_PASSWORD")
	kibHost, foundHost := os.LookupEnv("KIBANA_HOST")

	if !foundPass || !foundUser || !foundHost {
		t.Skip("test requires a Kibana instance with KIBANA_USERNAME, KIBANA_PASSWORD and KIBANA_HOST set")
	}

	cfg := ClientConfig{
		Host:     kibHost,
		Username: kibUser,
		Password: kibPass,
	}

	return cfg
}

var tests = []struct {
	name      string
	protected bool
}{
	{
		name: "unprotected",
	},
	{
		name:      "protected",
		protected: true,
	},
}

/*
These are a series of integration tests that run some of the fleet API clients against an actual elastic stack.
Just set the env vars listed in mustGetEnv().
*/

func TestGetPolicyKibana(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	basePolicy := AgentPolicy{
		Name:        "TestGetPolicyKibana",
		Namespace:   "defaultttest",
		Description: "original policy",
	}

	for _, tc := range tests {
		testPolicy := basePolicy
		testPolicy.IsProtected = tc.protected

		t.Run(tc.name, func(t *testing.T) {
			respPolicy, err := client.CreatePolicy(ctx, testPolicy)
			require.NoError(t, err)
			t.Logf("Created policy for test with ID %s", respPolicy.ID)

			respGet, err := client.GetPolicy(ctx, respPolicy.ID)
			require.NoError(t, err)

			require.Equal(t, respPolicy.Description, respGet.Description)
			require.Equal(t, respPolicy.IsProtected, respGet.IsProtected)

			err = client.DeletePolicy(ctx, respPolicy.ID)
			require.NoError(t, err)
		})
	}
}

func TestCreatePolicyKibana(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	basePolicy := AgentPolicy{
		Name:      "TestCreatePolicyKibana",
		Namespace: "defaulttest",
	}
	for _, tc := range tests {
		testPolicy := basePolicy
		testPolicy.IsProtected = tc.protected
		t.Run(tc.name, func(t *testing.T) {
			respPolicy, err := client.CreatePolicy(ctx, testPolicy)
			t.Logf("created policy with ID %s", respPolicy.ID)
			require.NoError(t, err)
			require.Equal(t, testPolicy.Name, respPolicy.Name)
			require.Equal(t, testPolicy.Namespace, respPolicy.Namespace)
			require.Equal(t, testPolicy.IsProtected, respPolicy.IsProtected)
			require.NotEmpty(t, respPolicy.ID)
			// delete policy
			err = client.DeletePolicy(ctx, respPolicy.ID)
			require.NoError(t, err)
		})
	}

}

func TestUpdatePolicyKibana(t *testing.T) {
	cfg := mustGetEnv(t)

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	uid, err := uuid.NewV4()
	require.NoError(t, err)

	basePolicy := AgentPolicy{
		Name:        "TestUpdatePolicyKibana-" + uid.String(),
		Namespace:   "defaultttest",
		Description: "original policy",
	}

	for _, tc := range tests {
		testPolicy := basePolicy
		testPolicy.IsProtected = tc.protected
		t.Run(tc.name, func(t *testing.T) {
			testUpdatePolicyKibana(t, client, testPolicy)
		})
	}
}

func testUpdatePolicyKibana(t *testing.T, client *Client, testPolicy AgentPolicy) {
	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	respPolicy, err := client.CreatePolicy(ctx, testPolicy)
	require.NoError(t, err)
	t.Logf("Created policy for test with ID %s", respPolicy.ID)
	require.Empty(t, respPolicy.MonitoringEnabled)

	updatePolicy := AgentPolicyUpdateRequest{
		Name:              testPolicy.Name,
		Namespace:         testPolicy.Namespace,
		MonitoringEnabled: []MonitoringEnabledOption{MonitoringEnabledMetrics},
		IsProtected:       &testPolicy.IsProtected,
	}

	updateResp, err := client.UpdatePolicy(ctx, respPolicy.ID, updatePolicy)
	require.NoError(t, err)
	// make sure our update was applied
	require.Equal(t, []MonitoringEnabledOption{MonitoringEnabledMetrics}, updateResp.MonitoringEnabled)
	// make sure we didn't somehow change something else
	require.Equal(t, respPolicy.InactivityTImeout, updateResp.InactivityTImeout)
	require.Equal(t, respPolicy.Description, updateResp.Description)
	require.Equal(t, respPolicy.IsProtected, updateResp.IsProtected)

	// Enable tamper protection
	updatePolicyTamperProtection := AgentPolicyUpdateRequest{
		Name:        testPolicy.Name,
		Namespace:   testPolicy.Namespace,
		IsProtected: TRUE,
	}

	// Verify that tamper protection is enabled
	updateResp, err = client.UpdatePolicy(ctx, respPolicy.ID, updatePolicyTamperProtection)
	require.NoError(t, err)
	require.Equal(t, *updatePolicyTamperProtection.IsProtected, updateResp.IsProtected)

	// Get uninstall tokens, should be one
	uninstallTokenResp, err := client.GetPolicyUninstallTokens(ctx, respPolicy.ID)
	require.NoError(t, err)
	require.Greater(t, len(uninstallTokenResp.Items), 0, "Expected non-zero number of tokens")
	require.Greater(t, len(uninstallTokenResp.Items[0].Token), 0, "expected non-empty token")

	// Disable tamper protection
	updatePolicyTamperProtection = AgentPolicyUpdateRequest{
		Name:        testPolicy.Name,
		Namespace:   testPolicy.Namespace,
		IsProtected: FALSE,
	}

	// Verify that tamper protection is disabled
	updateResp, err = client.UpdatePolicy(ctx, respPolicy.ID, updatePolicyTamperProtection)
	require.NoError(t, err)
	require.Equal(t, *updatePolicyTamperProtection.IsProtected, updateResp.IsProtected)

	err = client.DeletePolicy(ctx, respPolicy.ID)
	require.NoError(t, err)
}

//go:embed endpoint_security_package.json.tmpl
var endpointPackagePolicyTemplate string

type endpointPackageTemplateVars struct {
	ID       string
	Name     string
	PolicyID string
	Version  string
}

const endpointPackageVersion = "8.9.0"

func TestFleetPackage(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	policyUUID := uuid.Must(uuid.NewV4()).String()
	require.NoError(t, err)

	req := AgentPolicy{
		Name:        "test-policy-" + policyUUID,
		Namespace:   "default",
		Description: "Test policy " + policyUUID,
		MonitoringEnabled: []MonitoringEnabledOption{
			MonitoringEnabledLogs,
			MonitoringEnabledMetrics,
		},
	}

	// Create policy
	res, err := client.CreatePolicy(ctx, req)
	require.NoError(t, err)

	// Install package
	packagePolicyID := uuid.Must(uuid.NewV4()).String()
	packRes, err := installElasticDefendPackage(t, client, res.ID, packagePolicyID)
	require.NoError(t, err)

	// Remove package
	delRes, err := client.DeleteFleetPackage(context.Background(), packRes.Item.ID)
	require.NoError(t, err)
	require.Equal(t, packagePolicyID, delRes.ID)

	// Cleanup
	err = client.DeletePolicy(ctx, res.ID)
	require.NoError(t, err)
}

func installElasticDefendPackage(t *testing.T, client *Client, policyID, packagePolicyID string) (r PackagePolicyResponse, err error) {
	t.Helper()

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	t.Log("Templating endpoint package policy request")
	tmpl, err := template.New("pkgpolicy").Parse(endpointPackagePolicyTemplate)
	if err != nil {
		return r, fmt.Errorf("error creating new template: %w", err)
	}

	var pkgPolicyBuf bytes.Buffer

	// Need unique name for Endpoint integration otherwise on multiple runs on the same instance you get
	// http error response with code 409: {StatusCode:409 Error:Conflict Message:An integration policy with the name Defend-cbomziz4uvn5fov9t1gsrcvdwn2p1s7tefnvgsye already exists. Please rename it or choose a different name.}
	err = tmpl.Execute(&pkgPolicyBuf, endpointPackageTemplateVars{
		ID:       packagePolicyID,
		Name:     "Defend-" + packagePolicyID,
		PolicyID: policyID,
		Version:  endpointPackageVersion,
	})
	if err != nil {
		return r, fmt.Errorf("error executing template: %w", err)
	}

	// Make sure the templated value is actually valid JSON before making the API request.
	// Using json.Unmarshal will give us the actual syntax error, calling json.Valid() would not.
	var packagePolicyReq PackagePolicyRequest
	err = json.Unmarshal(pkgPolicyBuf.Bytes(), &packagePolicyReq)
	if err != nil {
		return r, fmt.Errorf("templated package policy is not valid JSON: %s, %w", pkgPolicyBuf.String(), err)
	}

	pkgResp, err := client.InstallFleetPackage(ctx, packagePolicyReq)
	if err != nil {
		t.Logf("Error installing fleet package: %v", err)
		return r, fmt.Errorf("error installing fleet package: %w", err)
	}
	t.Logf("Endpoint package Policy Response:\n%+v", pkgResp)
	return pkgResp, err
}

func TestCreateEnrollmentAPIKey(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	testPolicy := AgentPolicy{
		Name:        fmt.Sprintf("TestCreateEnrollmentAPIKey-%s", uuid.Must(uuid.NewV4()).String()),
		Namespace:   "defaultttest",
		Description: "original policy",
	}

	policyResp, err := client.CreatePolicy(ctx, testPolicy)
	require.NoError(t, err)
	defer func() {
		err = client.DeletePolicy(ctx, policyResp.ID)
		require.NoError(t, err)
	}()

	enrollKeyResp, err := client.CreateEnrollmentAPIKey(ctx, CreateEnrollmentAPIKeyRequest{
		Name:     "TestEnrollmentKey",
		PolicyID: policyResp.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, enrollKeyResp.APIKey)
}

func TestListAgents(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	listResp, err := client.ListAgents(ctx, ListAgentsRequest{})
	require.NoError(t, err)
	require.Greater(t, len(listResp.Items), 0)
}

func TestGetAgent(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	// Test get non-existent Agent
	nonExistentAgentID := uuid.Must(uuid.NewV4()).String()
	agentResp, err := client.GetAgent(ctx, GetAgentRequest{ID: nonExistentAgentID})

	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("Agent %s not found", nonExistentAgentID), err.Error())
	require.Empty(t, agentResp.ID)

	// Get the list of Agent and then get the Agent by ID from that list
	listResp, err := client.ListAgents(ctx, ListAgentsRequest{})
	require.NoError(t, err)
	require.Greater(t, len(listResp.Items), 0)

	agentResp, err = client.GetAgent(ctx, GetAgentRequest{ID: listResp.Items[0].ID})
	require.NoError(t, err)
	require.NotEmpty(t, agentResp.ID)
}

func TestUnenrollAgent(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	// Test unenroll non-existent Agent
	nonExistentAgentID := uuid.Must(uuid.NewV4()).String()
	_, err = client.UnEnrollAgent(ctx, UnEnrollAgentRequest{ID: nonExistentAgentID})

	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("Agent %s not found", nonExistentAgentID), err.Error())
}

func TestListFleetServerHosts(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	resp, err := client.ListFleetServerHosts(ctx, ListFleetServerHostsRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Items)
}

func TestGetFleetServerHost(t *testing.T) {
	cfg := mustGetEnv(t)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	listResp, err := client.ListFleetServerHosts(ctx, ListFleetServerHostsRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, listResp.Items)

	resp, err := client.GetFleetServerHost(ctx, GetFleetServerHostRequest{listResp.Items[0].ID})
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)
}

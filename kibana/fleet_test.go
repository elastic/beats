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
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
)

var (
	//go:embed testdata/fleet_list_agents_response.json
	fleetListAgentsResponse []byte

	//go:embed testdata/fleet_get_agent_response.json
	fleetGetAgentResponse []byte

	//go:embed testdata/fleet_create_policy_response.json
	fleetCreatePolicyResponse []byte

	//go:embed testdata/fleet_get_policy_response.json
	fleetGetPolicyResponse []byte

	//go:embed testdata/fleet_update_policy_response.json
	fleetUpdatePolicyResponse []byte

	//go:embed testdata/fleet_create_enrollment_api_key_response.json
	fleetCreateEnrollmentAPIKeyResponse []byte

	//go:embed testdata/fleet_list_fleet_server_hosts_response.json
	fleetListServerHostsResponse []byte

	//go:embed testdata/fleet_get_fleet_server_host_response.json
	fleetGetFleetServerHostResponse []byte
)

func TestFleetCreatePolicy(t *testing.T) {
	const (
		policyName        = "test policy"
		policyDescription = "a policy used for testing"
	)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fleetAgentPoliciesAPI:
			_, _ = w.Write(fleetCreatePolicyResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := AgentPolicy{
		Name:        policyName,
		Description: policyDescription,
		MonitoringEnabled: []MonitoringEnabledOption{
			MonitoringEnabledLogs,
			MonitoringEnabledMetrics,
		},
	}
	resp, err := client.CreatePolicy(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, resp.ID, "a580c680-ea40-11ed-aae7-4b4fd4906b3d")
	require.Equal(t, resp.Name, policyName)
	require.Equal(t, resp.Description, policyDescription)
	require.Equal(t, resp.Namespace, "default")
	// require.Equal(t, resp.Status, "active")
	// require.Equal(t, resp.IsManaged, false)
	require.Equal(t, resp.MonitoringEnabled, []MonitoringEnabledOption{MonitoringEnabledLogs, MonitoringEnabledMetrics})
}

func TestFleetGetPolicy(t *testing.T) {
	const id = "elastic-agent-managed-ep"

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf(fleetAgentPolicyAPI, id):
			_, _ = w.Write(fleetGetPolicyResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	resp, err := client.GetPolicy(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, id, resp.ID)
	require.Equal(t, "Elastic-Agent (elastic-package)", resp.Name)
	require.Equal(t, "default", resp.Namespace)
	require.Equal(t, "", resp.Description)
	require.Equal(t, "fleet-custom-fleet-server-host", resp.FleetServerHostID)
	require.Equal(t, []MonitoringEnabledOption{MonitoringEnabledLogs}, resp.MonitoringEnabled)
}

func TestFleetUpdatePolicy(t *testing.T) {
	const (
		id         = "b4cd25b0-f040-11ed-a1b3-373f5d648cd4"
		policyName = "test-fqdn"
	)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf(fleetAgentPolicyAPI, id):
			_, _ = w.Write(fleetUpdatePolicyResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	agentFeatures := []map[string]interface{}{
		{
			"name":    "fqdn",
			"enabled": true,
		},
	}
	req := AgentPolicyUpdateRequest{

		Name: policyName,
		MonitoringEnabled: []MonitoringEnabledOption{
			MonitoringEnabledLogs,
			MonitoringEnabledMetrics,
		},
		AgentFeatures: agentFeatures,
	}

	resp, err := client.UpdatePolicy(ctx, id, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, id, resp.ID)
	require.Equal(t, policyName, resp.Name)
	require.Equal(t, "default", resp.Namespace)
	// require.Equal(t, "active", resp.Status)
	// require.Equal(t, false, resp.IsManaged)
	require.Equal(t, []MonitoringEnabledOption{MonitoringEnabledLogs, MonitoringEnabledMetrics}, resp.MonitoringEnabled)
	require.Equal(t, agentFeatures, resp.AgentFeatures)
}

func TestFleetCreateEnrollmentAPIKey(t *testing.T) {
	const (
		id       = "880c7460-a7e4-43df-8fc3-6a9593c6d555"
		name     = "test"
		policyID = "a580c680-ea40-11ed-aae7-4b4fd4906b3d"
	)

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fleetEnrollmentAPIKeysAPI:
			_, _ = w.Write(fleetCreateEnrollmentAPIKeyResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := CreateEnrollmentAPIKeyRequest{
		Name:     name,
		PolicyID: policyID,
	}
	resp, err := client.CreateEnrollmentAPIKey(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, resp.ID, id)
	require.Equal(t, resp.Name, fmt.Sprintf("%s (%s)", name, id))
	require.Equal(t, resp.APIKey, "XxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXxXx==")
	require.Equal(t, resp.PolicyID, policyID)
	require.True(t, resp.Active)
}

func TestFleetListAgents(t *testing.T) {
	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fleetAgentsAPI:
			_, _ = w.Write(fleetListAgentsResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := ListAgentsRequest{}
	resp, err := client.ListAgents(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Items, 1)
	item := resp.Items[0]
	require.Equal(t, "eba58282-ec1c-4d9e-aac0-2b29f754b437", item.Agent.ID)
	require.Equal(t, "8.8.0", item.Agent.Version)
	require.Equal(t, "c75d66b1dac5", item.LocalMetadata.Host.Hostname)
	require.Equal(t, true, item.LocalMetadata.Elastic.Agent.FIPS)
}

func TestFleetGetAgent(t *testing.T) {
	const id = "26802301-8996-457a-ab6a-8ea955ef2723"

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf(fleetAgentAPI, id):
			_, _ = w.Write(fleetGetAgentResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := GetAgentRequest{
		ID: id,
	}
	resp, err := client.GetAgent(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, id, resp.ID)
	require.True(t, resp.Active)
	require.Equal(t, "online", resp.Status)
	require.Equal(t, id, resp.Agent.ID)
	require.Equal(t, "8.7.1", resp.Agent.Version)
	require.Equal(t, "Shaunaks-MBP.attlocal.net", resp.LocalMetadata.Host.Hostname)
	require.Equal(t, "8196af30-f041-11ed-a1b3-373f5d648cd4", resp.PolicyID)
	require.Equal(t, 4, resp.PolicyRevision)
}

func TestFleetUnEnrollAgent(t *testing.T) {
	const agentID = "f512f36f-bf78-4285-aff0-baeafbcdf21e"

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf(fleetUnEnrollAgentAPI, agentID):
			_, _ = w.Write([]byte(`{}`))
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := UnEnrollAgentRequest{
		ID:     agentID,
		Revoke: true,
	}
	resp, err := client.UnEnrollAgent(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestFleetUpgradeAgent(t *testing.T) {
	const agentID = "f512f36f-bf78-4285-aff0-baeafbcdf21e"

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf(fleetUpgradeAgentAPI, agentID):
			_, _ = w.Write([]byte(`{}`))
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := UpgradeAgentRequest{
		ID:      agentID,
		Version: "8.8.0",
	}
	resp, err := client.UpgradeAgent(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestFleetListFleetServerHosts(t *testing.T) {
	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fleetFleetServerHostsAPI:
			_, _ = w.Write(fleetListServerHostsResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := ListFleetServerHostsRequest{}
	resp, err := client.ListFleetServerHosts(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Items, 1)
	item := resp.Items[0]
	require.Equal(t, "fleet-default-fleet-server-host", item.ID)
	require.Equal(t, "Default", item.Name)
	require.True(t, item.IsDefault)
	require.Equal(t, []string{"https://fleet-server:8220"}, item.HostURLs)
	require.True(t, item.IsPreconfigured)
}

func TestFleetGetFleetServerHost(t *testing.T) {
	const id = "fleet-default-fleet-server-host"

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fleetFleetServerHostsAPI + "/" + id:
			_, _ = w.Write(fleetGetFleetServerHostResponse)
		}
	}

	client, err := createTestServerAndClient(handler)
	require.NoError(t, err)
	require.NotNil(t, client)

	req := GetFleetServerHostRequest{
		ID: id,
	}
	resp, err := client.GetFleetServerHost(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, id, resp.ID)
	require.Equal(t, "Default", resp.Name)
	require.True(t, resp.IsDefault)
	require.Equal(t, []string{"https://fleet-server:8220"}, resp.HostURLs)
	require.True(t, resp.IsPreconfigured)
}

func createTestServerAndClient(handler http.HandlerFunc) (*Client, error) {
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case statusAPI:
			_, _ = w.Write([]byte(`{"version":{"number":"1.2.3-beta","build_snapshot":true}}`))
		default:
			handler(w, r)
		}
	}))

	cfg := fmt.Sprintf(`
protocol: http
host: %s
`, kibanaTS.Listener.Addr().String())
	return NewKibanaClient(config.MustNewConfigFrom(cfg), binaryName, v, commit, buildTime)
}

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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	fleetAgentPoliciesAPI     = "/api/fleet/agent_policies"
	fleetAgentPolicyAPI       = "/api/fleet/agent_policies/%s"
	fleetAgentsDeleteAPI      = "/api/fleet/agent_policies/delete"
	fleetEnrollmentAPIKeysAPI = "/api/fleet/enrollment_api_keys" //nolint:gosec // no API key being leaked here
	fleetAgentsAPI            = "/api/fleet/agents"
	fleetAgentAPI             = "/api/fleet/agents/%s"
	fleetUnEnrollAgentAPI     = "/api/fleet/agents/%s/unenroll"
	fleetUpgradeAgentAPI      = "/api/fleet/agents/%s/upgrade"
	fleetFleetServerHostsAPI  = "/api/fleet/fleet_server_hosts"
	fleetFleetServerHostAPI   = "/api/fleet/fleet_server_hosts/%s"
	fleetPackagePoliciesAPI   = "/api/fleet/package_policies"
	fleetUninstallTokensAPI   = "/api/fleet/uninstall_tokens" //nolint:gosec // NOT the "Potential hardcoded credentials"
)

//
// Create Policy
//

// MonitoringEnabledOption is a Kibana JSON value that specifies the various monitoring option types
type MonitoringEnabledOption string

const (
	// MonitoringEnabledLogs specifies log monitoring
	MonitoringEnabledLogs MonitoringEnabledOption = "logs"
	// MonitoringEnabledMetrics specifies metrics monitoring
	MonitoringEnabledMetrics MonitoringEnabledOption = "metrics"
)

// AgentPolicy is the JSON that represents a agent policy. These fields are used by both the create policy request, and the GET request for an agent policy.
// see: https://github.com/elastic/kibana/blob/v8.8.0/x-pack/plugins/fleet/common/openapi/components/schemas/agent_policy_create_request.yaml
// and https://github.com/elastic/kibana/blob/v8.8.0/x-pack/plugins/fleet/common/openapi/components/schemas/agent_policy.yaml
type AgentPolicy struct {
	ID string `json:"id,omitempty"`
	// Name of the policy. Required to create a policy.
	Name string `json:"name"`
	// Namespace of the policy. Required to create a policy.
	Namespace          string                    `json:"namespace"`
	Description        string                    `json:"description,omitempty"`
	MonitoringEnabled  []MonitoringEnabledOption `json:"monitoring_enabled,omitempty"`
	DataOutputID       string                    `json:"data_output_id,omitempty"`
	MonitoringOutputID string                    `json:"monitoring_output_id,omitempty"`
	FleetServerHostID  string                    `json:"fleet_server_host_id,omitempty"`
	DownloadSourceID   string                    `json:"download_source_id,omitempty"`
	UnenrollTimeout    int                       `json:"unenroll_timeout,omitempty"`
	InactivityTImeout  int                       `json:"inactivity_timeout,omitempty"`
	AgentFeatures      []map[string]interface{}  `json:"agent_features,omitempty"`
	IsProtected        bool                      `json:"is_protected"`
}

// PolicyResponse is the response JSON from a policy request
// This is returned on a GET request for a policy, and on a policy create request
// See https://github.com/elastic/kibana/blob/v8.8.0/x-pack/plugins/fleet/common/openapi/paths/agent_policies.yaml
type PolicyResponse struct {
	AgentPolicy     `json:",inline"`
	UpdatedOn       time.Time                `json:"updated_on"`
	UpdatedBy       string                   `json:"updated_by"`
	Revision        int                      `json:"revision"`
	IsProtected     bool                     `json:"is_protected"`
	PackagePolicies []map[string]interface{} `json:"package_policies"`
}

// AgentPolicyUpdateRequest is the JSON object for requesting an updated policy
// Unlike the Agent create and response structures, the update request does not contain an ID field.
// See https://github.com/elastic/kibana/blob/v8.9.0/x-pack/plugins/fleet/common/openapi/components/schemas/agent_policy_update_request.yaml
type AgentPolicyUpdateRequest struct {
	// Name of the policy. Required in an update request.
	Name string `json:"name"`
	// Namespace of the policy. Required in an update request.
	Namespace          string                    `json:"namespace"`
	Description        string                    `json:"description,omitempty"`
	MonitoringEnabled  []MonitoringEnabledOption `json:"monitoring_enabled,omitempty"`
	DataOutputID       string                    `json:"data_output_id,omitempty"`
	MonitoringOutputID string                    `json:"monitoring_output_id,omitempty"`
	FleetServerHostID  string                    `json:"fleet_server_host_id,omitempty"`
	DownloadSourceID   string                    `json:"download_source_id,omitempty"`
	UnenrollTimeout    int                       `json:"unenroll_timeout,omitempty"`
	InactivityTImeout  int                       `json:"inactivity_timeout,omitempty"`
	AgentFeatures      []map[string]interface{}  `json:"agent_features,omitempty"`
	IsProtected        *bool                     `json:"is_protected,omitempty"` // Optional bool for compatibility with the older pre 8.9.0 stack
}

// Constant booleans for convenience for dealing with *bool
var (
	bt bool = true
	bf bool = false

	TRUE  *bool = &bt
	FALSE *bool = &bf
)

type policyResp struct {
	Item PolicyResponse `json:"item"`
}

// CreatePolicy creates a new agent policy with the given config
func (client *Client) CreatePolicy(ctx context.Context, request AgentPolicy) (r PolicyResponse, err error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return r, fmt.Errorf("unable to marshal create policy request into JSON: %w", err)
	}

	resp, err := client.Connection.SendWithContext(ctx, http.MethodPost, fleetAgentPoliciesAPI, nil, nil, bytes.NewReader(reqBody))
	if err != nil {
		return r, fmt.Errorf("error calling create policy API: %w", err)
	}
	defer resp.Body.Close()
	var polResp policyResp
	err = readJSONResponse(resp, &polResp)
	return polResp.Item, err
}

// GetPolicy returns the requested ID
func (client *Client) GetPolicy(ctx context.Context, id string) (r PolicyResponse, err error) {
	apiURL := fmt.Sprintf(fleetAgentPolicyAPI, id)
	resp, err := client.Connection.SendWithContext(ctx, http.MethodGet, apiURL, nil, nil, nil)
	if err != nil {
		return r, fmt.Errorf("error calling get policy API: %w", err)
	}
	defer resp.Body.Close()
	var polResp policyResp
	err = readJSONResponse(resp, &polResp)
	return polResp.Item, err
}

// UpdatePolicy updates an existing agent policy.
func (client *Client) UpdatePolicy(ctx context.Context, id string, request AgentPolicyUpdateRequest) (r PolicyResponse, err error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return r, fmt.Errorf("unable to marshal update policy request into JSON: %w", err)
	}

	apiURL := fmt.Sprintf(fleetAgentPolicyAPI, id)
	resp, err := client.Connection.SendWithContext(ctx, http.MethodPut, apiURL, nil, nil, bytes.NewReader(reqBody))
	if err != nil {
		return r, fmt.Errorf("error calling update policy API: %w", err)
	}
	defer resp.Body.Close()
	var polResp policyResp
	err = readJSONResponse(resp, &polResp)
	return polResp.Item, err
}

// DeletePolicy deletes the policy with the given ID
func (client *Client) DeletePolicy(ctx context.Context, id string) error {
	var delRequest = struct {
		AgentPolicyID string `json:"agentPolicyId"`
	}{
		AgentPolicyID: id,
	}

	reqBody, err := json.Marshal(delRequest)
	if err != nil {
		return fmt.Errorf("unable to marshal delete policy request into JSON: %w", err)
	}

	resp, err := client.Connection.SendWithContext(ctx, http.MethodPost, fleetAgentsDeleteAPI, nil, nil, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("error calling delete policy API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to delete policy; API returned status code [%d] and error reading response: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("unable to delete policy; API returned status code [%d] and body [%s]", resp.StatusCode, string(respBody))
	}
	return nil
}

//
// Create Enrollment API Key
//

// CreateEnrollmentAPIKeyRequest is the JSON object for requesting an enrollment API key
type CreateEnrollmentAPIKeyRequest struct {
	Name     string `json:"name"`
	PolicyID string `json:"policy_id"`
}

// CreateEnrollmentAPIKeyResponse is the JSON response the an enrollment key request
type CreateEnrollmentAPIKeyResponse struct {
	Active   bool   `json:"active"`
	APIKey   string `json:"api_key"`
	APIKeyID string `json:"api_key_id"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	PolicyID string `json:"policy_id"`
}

// CreateEnrollmentAPIKey creates an enrollment API key
func (client *Client) CreateEnrollmentAPIKey(ctx context.Context, request CreateEnrollmentAPIKeyRequest) (r CreateEnrollmentAPIKeyResponse, err error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return r, fmt.Errorf("unable to marshal create enrollment API key request into JSON: %w", err)
	}

	resp, err := client.Connection.SendWithContext(ctx, http.MethodPost, fleetEnrollmentAPIKeysAPI, nil, nil, bytes.NewReader(reqBody))
	if err != nil {
		return r, fmt.Errorf("error calling create enrollment API key API: %w", err)
	}
	defer resp.Body.Close()

	var enrollResp struct {
		Item CreateEnrollmentAPIKeyResponse `json:"item"`
	}
	err = readJSONResponse(resp, &enrollResp)
	return enrollResp.Item, err
}

//
// List Agents
//

// AgentCommon represents common agent data used across APIs
type AgentCommon struct {
	Active bool   `json:"active"`
	Status string `json:"status"`
	Agent  struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"agent"`
	LocalMetadata struct {
		Host struct {
			Hostname string `json:"hostname"`
		} `json:"host"`
	} `json:"local_metadata"`
	PolicyID       string `json:"policy_id"`
	PolicyRevision int    `json:"policy_revision"`
}

// AgentExisting is the data structure for an existing agent
type AgentExisting struct {
	ID          string `json:"id"`
	AgentCommon `json:",inline"`
}

// ListAgentsRequest is currently unused
type ListAgentsRequest struct {
	// For future use
}

// ListAgentsResponse is a list of agents returned by the API
type ListAgentsResponse struct {
	Items []AgentExisting `json:"items"`
}

// ListAgents returns a list of agents known to Kibana
func (client *Client) ListAgents(ctx context.Context, _ ListAgentsRequest) (r ListAgentsResponse, err error) {
	resp, err := client.Connection.SendWithContext(ctx, http.MethodGet, fleetAgentsAPI, nil, nil, nil)
	if err != nil {
		return r, fmt.Errorf("error calling list agents API: %w", err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)
	return r, err
}

//
// Get Agent
//

// GetAgentRequest contains the ID used for fetching agent data
type GetAgentRequest struct {
	ID string
}

// GetAgentResponse is the JSON response for GetAgent
type GetAgentResponse AgentExisting

// GetAgent fetches data for an agent
func (client *Client) GetAgent(ctx context.Context, request GetAgentRequest) (r GetAgentResponse, err error) {
	apiURL := fmt.Sprintf(fleetAgentAPI, request.ID)

	resp, err := client.Connection.SendWithContext(ctx, http.MethodGet, apiURL, nil, nil, nil)
	if err != nil {
		return r, fmt.Errorf("error calling get agent API: %w", err)
	}
	defer resp.Body.Close()

	var agentResp struct {
		Item GetAgentResponse `json:"item"`
	}
	err = readJSONResponse(resp, &agentResp)
	return agentResp.Item, err
}

//
// Unenroll Agent
//

// UnEnrollAgentRequest is the JSON request for unenrolling an agent
type UnEnrollAgentRequest struct {
	ID     string `json:"-"` // ID is not part of the request body send to the Fleet API
	Revoke bool   `json:"revoke"`
}

// UnEnrollAgentResponse is currently unused
type UnEnrollAgentResponse struct {
	// For future use
}

// UnEnrollAgent removes the agent from fleet
func (client *Client) UnEnrollAgent(ctx context.Context, request UnEnrollAgentRequest) (r UnEnrollAgentResponse, err error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return r, fmt.Errorf("unable to marshal unenroll agent request into JSON: %w", err)
	}
	apiURL := fmt.Sprintf(fleetUnEnrollAgentAPI, request.ID)

	resp, err := client.Connection.SendWithContext(ctx, http.MethodPost, apiURL, nil, nil, bytes.NewReader(reqBody))
	if err != nil {
		return r, fmt.Errorf("error calling unenroll agent API: %w", err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)
	return r, err
}

//
// Upgrade Agent
//

// UpgradeAgentRequest is the JSON request for an agent upgrade
type UpgradeAgentRequest struct {
	ID      string `json:"-"` // ID is not part of the request body send to the Fleet API
	Version string `json:"version"`
}

// UpgradeAgentResponse is currently unused
type UpgradeAgentResponse struct {
	// For future use
}

// UpgradeAgent upgrades the requested agent
func (client *Client) UpgradeAgent(ctx context.Context, request UpgradeAgentRequest) (r UpgradeAgentResponse, err error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return r, fmt.Errorf("unable to marshal upgrade agent request into JSON: %w", err)
	}

	apiURL := fmt.Sprintf(fleetUpgradeAgentAPI, request.ID)
	resp, err := client.Connection.SendWithContext(ctx, http.MethodPost, apiURL, nil, nil, bytes.NewReader(reqBody))
	if err != nil {
		return r, fmt.Errorf("error calling upgrade agent API: %w", err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)
	return r, err
}

//
// List Fleet Server Hosts
//

// FleetServerHost handles JSON data for fleet server info
type FleetServerHost struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	IsDefault       bool     `json:"is_default"`
	HostURLs        []string `json:"host_urls"`
	IsPreconfigured bool     `json:"is_preconfigured"`
}

// ListFleetServerHostsRequest is currently unused
type ListFleetServerHostsRequest struct {
	// For future use
}

// ListFleetServerHostsResponse is the JSON response for ListFleetServerHosts
type ListFleetServerHostsResponse struct {
	Items []FleetServerHost `json:"items"`
}

// ListFleetServerHosts returns a list of fleet server hosts
func (client *Client) ListFleetServerHosts(ctx context.Context, _ ListFleetServerHostsRequest) (r ListFleetServerHostsResponse, err error) {
	resp, err := client.Connection.SendWithContext(ctx, http.MethodGet, fleetFleetServerHostsAPI, nil, nil, nil)
	if err != nil {
		return r, fmt.Errorf("error calling list fleet server hosts API: %w", err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)

	return r, err
}

//
// Get Fleet Server Host
//

// GetFleetServerHostRequest is the ID for a request via GetFleetServerHost
type GetFleetServerHostRequest struct {
	ID string
}

// GetFleetServerHostResponse is the JSON respose from GetFleetServerHost
type GetFleetServerHostResponse FleetServerHost

// GetFleetServerHost returns data on a fleet server
func (client *Client) GetFleetServerHost(ctx context.Context, request GetFleetServerHostRequest) (r GetFleetServerHostResponse, err error) {
	apiURL := fmt.Sprintf(fleetFleetServerHostAPI, request.ID)

	resp, err := client.Connection.SendWithContext(ctx, http.MethodGet, apiURL, nil, nil, nil)
	if err != nil {
		return r, fmt.Errorf("error calling get fleet server hosts API: %w", err)
	}
	defer resp.Body.Close()

	var fleetResp struct {
		Item GetFleetServerHostResponse `json:"item"`
	}
	err = readJSONResponse(resp, &fleetResp)

	return fleetResp.Item, err
}

//
// Fleet Package Policy
//

// https://www.elastic.co/guide/en/fleet/8.8/fleet-apis.html#createPackagePolicy
// request https://www.elastic.co/guide/en/fleet/8.8/fleet-apis.html#package_policy_request
type PackagePolicyRequest struct {
	ID        string                      `json:"id,omitempty"`
	Name      string                      `json:"name"`
	Namespace string                      `json:"namespace"`
	PolicyID  string                      `json:"policy_id"`
	Package   PackagePolicyRequestPackage `json:"package"`
	Vars      map[string]interface{}      `json:"vars"`
	Inputs    []map[string]interface{}    `json:"inputs"`
	Force     bool                        `json:"force"`
}

type PackagePolicyRequestPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// https://www.elastic.co/guide/en/fleet/8.8/fleet-apis.html#create_package_policy_200_response
type PackagePolicyResponse struct {
	Item PackagePolicy `json:"item"`
}

// https://www.elastic.co/guide/en/fleet/8.8/fleet-apis.html#package_policy
type PackagePolicy struct {
	ID          string                      `json:"id,omitempty"`
	Revision    int                         `json:"revision"`
	Enabled     bool                        `json:"enabled"`
	Inputs      []map[string]interface{}    `json:"inputs"`
	Package     PackagePolicyRequestPackage `json:"package"`
	Namespace   string                      `json:"namespace"`
	OutputID    string                      `json:"output_id"`
	PolicyID    string                      `json:"policy_id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
}

// https://www.elastic.co/guide/en/fleet/8.8/fleet-apis.html#delete_package_policy_200_response
type DeletePackagePolicyResponse struct {
	ID string `json:"id"`
}

// InstallFleetPackage uses the Fleet package policies API install an integration package as specified in the request.
// Note that the package policy ID and Name must be globally unique across all installed packages.
func (client *Client) InstallFleetPackage(ctx context.Context, req PackagePolicyRequest) (r PackagePolicyResponse, err error) {
	reqBytes, err := json.Marshal(&req)
	if err != nil {
		return r, fmt.Errorf("marshalling request json: %w", err)
	}

	resp, err := client.Connection.SendWithContext(ctx,
		http.MethodPost,
		fleetPackagePoliciesAPI,
		nil,
		nil,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return r, fmt.Errorf("posting %s: %w", fleetPackagePoliciesAPI, err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)

	return r, err
}

// DeleteFleetPackage deletes integration with packagePolicyID from the policy ID
func (client *Client) DeleteFleetPackage(ctx context.Context, packagePolicyID string) (r DeletePackagePolicyResponse, err error) {
	u, err := url.JoinPath(fleetPackagePoliciesAPI, packagePolicyID)
	if err != nil {
		return r, err
	}

	resp, err := client.Connection.SendWithContext(ctx,
		http.MethodDelete,
		u,
		nil,
		nil,
		nil,
	)

	if err != nil {
		return r, fmt.Errorf("DELETE %s: %w", u, err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)

	return r, err
}

// UninstallTokenResponse uninstall tokens response with resolved token values
type UninstallTokenResponse struct {
	Items   []UninstallTokenItem `json:"items"`
	Total   int                  `json:"total"`
	Page    int                  `json:"page"`
	PerPage int                  `json:"perPage"`
}

type UninstallTokenItem struct {
	ID        string `json:"id"`
	PolicyID  string `json:"policy_id"`
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
}

type uninstallTokenValueResponse struct {
	Item UninstallTokenItem `json:"item"`
}

// GetPolicyUninstallTokens Retrieves the the policy uninstall tokens
func (client *Client) GetPolicyUninstallTokens(ctx context.Context, policyID string) (r UninstallTokenResponse, err error) {
	// Fetch uninstall token for the policy
	// /api/fleet/uninstall_tokens?policyId={policyId}&page=1&perPage=1000
	q := make(url.Values)
	q.Add("policyId", policyID)
	q.Add("page", "1")
	q.Add("perPage", "1000")

	resp, err := client.Connection.SendWithContext(ctx,
		http.MethodGet,
		fleetUninstallTokensAPI,
		q,
		nil,
		nil,
	)
	if err != nil {
		return r, fmt.Errorf("getting %s, policyID %s: %w", fleetUninstallTokensAPI, policyID, err)
	}
	defer resp.Body.Close()

	err = readJSONResponse(resp, &r)
	if err != nil {
		return r, err
	}

	// Resolve token values for token ID
	for i := 0; i < len(r.Items); i++ {
		tokRes, err := client.GetUninstallToken(ctx, r.Items[i].ID)
		if err != nil {
			return r, err
		}
		r.Items[i] = tokRes
	}

	return r, nil
}

// GetUninstallToken return uninstall token value for the given token ID
func (client *Client) GetUninstallToken(ctx context.Context, tokenID string) (r UninstallTokenItem, err error) {
	u, err := url.JoinPath(fleetUninstallTokensAPI, tokenID)
	if err != nil {
		return r, err
	}

	resp, err := client.Connection.SendWithContext(ctx,
		http.MethodGet,
		u,
		nil,
		nil,
		nil,
	)

	if err != nil {
		return r, fmt.Errorf("getting %s: %w", u, err)
	}
	defer resp.Body.Close()

	var res uninstallTokenValueResponse
	err = readJSONResponse(resp, &res)
	if err != nil {
		return r, err
	}

	return res.Item, nil
}

func readJSONResponse(r *http.Response, v any) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if r.StatusCode != http.StatusOK {
		return extractError(b)
	}

	err = json.Unmarshal(b, v)
	if err != nil {
		return fmt.Errorf("unmarshalling response json: %w", err)
	}
	return nil
}

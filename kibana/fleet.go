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

	"github.com/elastic/elastic-agent-libs/upgrade/details"
)

// The full documentation for the Kibana API can be found on
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json
const (
	fleetAgentAPI                = "/api/fleet/agents/%s"
	fleetAgentPoliciesAPI        = "/api/fleet/agent_policies"
	fleetAgentPolicyAPI          = "/api/fleet/agent_policies/%s"
	fleetAgentsAPI               = "/api/fleet/agents"
	fleetAgentsDeleteAPI         = "/api/fleet/agent_policies/delete"
	fleetEnrollmentAPIKeysAPI    = "/api/fleet/enrollment_api_keys" //nolint:gosec // no API key being leaked here
	fleetFleetServerHostsAPI     = "/api/fleet/fleet_server_hosts"
	fleetPackagePoliciesAPI      = "/api/fleet/package_policies"
	fleetUnEnrollAgentAPI        = "/api/fleet/agents/%s/unenroll"
	fleetUninstallTokensAPI      = "/api/fleet/uninstall_tokens" //nolint:gosec // NOT the "Potential hardcoded credentials"
	fleetUpgradeAgentAPI         = "/api/fleet/agents/%s/upgrade"
	fleetAgentDownloadSourcesAPI = "/api/fleet/agent_download_sources"
	fleetProxiesAPI              = "/api/fleet/proxies"
)

// Constant booleans for convenience for dealing with *bool
var (
	bt bool = true
	bf bool = false

	TRUE  *bool = &bt
	FALSE *bool = &bf
)

//
// Agent Policies
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Elastic%20Agent%20policies
//

const (
	// MonitoringEnabledLogs specifies log monitoring
	MonitoringEnabledLogs MonitoringEnabledOption = "logs"
	// MonitoringEnabledMetrics specifies metrics monitoring
	MonitoringEnabledMetrics MonitoringEnabledOption = "metrics"
)

type policyResp struct {
	Item PolicyResponse `json:"item"`
}

// MonitoringEnabledOption is a Kibana JSON value that specifies the various monitoring option types
type MonitoringEnabledOption string

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
	Overrides          map[string]interface{}    `json:"overrides,omitempty"`
	IsProtected        bool                      `json:"is_protected"`
}

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
	Overrides          map[string]interface{}    `json:"overrides,omitempty"`
	IsProtected        *bool                     `json:"is_protected,omitempty"` // Optional bool for compatibility with the older pre 8.9.0 stack
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

type DownloadSource struct {
	Name      string      `json:"name"`
	Host      string      `json:"host"`
	IsDefault bool        `json:"is_default"`
	ProxyID   interface{} `json:"proxy_id"`
}

type DownloadSourceResponse struct {
	Item struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Host      string `json:"host"`
		IsDefault bool   `json:"is_default"`
		ProxyID   string `json:"proxy_id"`
	} `json:"item"`
}

func (client *Client) CreateDownloadSource(ctx context.Context, source DownloadSource) (DownloadSourceResponse, error) {
	reqBody, err := json.Marshal(source)
	if err != nil {
		return DownloadSourceResponse{},
			fmt.Errorf("unable to marshal DownloadSource into JSON: %w", err)
	}

	resp, err := client.Connection.SendWithContext(
		ctx,
		http.MethodPost,
		fleetAgentDownloadSourcesAPI,
		nil,
		nil,
		bytes.NewReader(reqBody))
	if err != nil {
		return DownloadSourceResponse{},
			fmt.Errorf("error calling Agent Binary Download Sources API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var respBody string
		if bs, err := io.ReadAll(resp.Body); err != nil {
			respBody = "could not read response body"
		} else {
			respBody = string(bs)
		}

		client.log.Errorw(
			"could not create download source, kibana returned "+resp.Status,
			"http.response.body.content", respBody)
		return DownloadSourceResponse{},
			fmt.Errorf("could not create download source, kibana returned %s. response body: %s: %w",
				resp.Status, respBody, err)
	}

	body := DownloadSourceResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return DownloadSourceResponse{},
			fmt.Errorf("failed parsing download source response: %w", err)
	}

	return body, nil
}

// GetPolicy returns the policy with 'policy_id' id.
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
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Fleet%20enrollment%20API%20keys
//

type CreateEnrollmentAPIKeyRequest struct {
	Name     string `json:"name"`
	PolicyID string `json:"policy_id"`
}

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
// Elastic Agents
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Elastic%20Agents
//

// AgentCommon represents common agent data used across Agent APIs
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
		Elastic struct {
			Agent struct {
				FIPS bool `json:"fips"`
			} `json:"agent"`
		} `json:"elastic"`
	} `json:"local_metadata"`
	PolicyID       string               `json:"policy_id"`
	PolicyRevision int                  `json:"policy_revision"`
	UpgradeDetails *AgentUpgradeDetails `json:"upgrade_details"`
}

type AgentUpgradeDetails struct {
	TargetVersion string `json:"target_version"`
	State         string `json:"state"`
	ActionID      string `json:"action_id"`
	Metadata      struct {
		ScheduledAt     *time.Time           `json:"scheduled_at"`
		DownloadPercent float64              `json:"download_percent"`
		DownloadRate    details.DownloadRate `json:"download_rate"`
		FailedState     string               `json:"failed_state"`
		ErrorMsg        string               `json:"error_msg"`
	} `json:"metadata"`
}

type AgentExisting struct {
	ID          string `json:"id"`
	AgentCommon `json:",inline"`
}

type ListAgentsRequest struct {
	// For future use
}

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

type GetAgentRequest struct {
	ID string
}

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
// Elastic Agent actions
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Elastic%20Agent%20actions
//

type UnEnrollAgentRequest struct {
	ID     string `json:"-"` // ID is not part of the request body send to the Fleet API
	Revoke bool   `json:"revoke"`
}

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

type UpgradeAgentRequest struct {
	ID        string `json:"-"` // ID is not part of the request body send to the Fleet API
	Version   string `json:"version"`
	SourceURI string `json:"source_uri"`
	Force     bool   `json:"force"`
}

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
// Fleet Server Hosts
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Fleet%20Server%20hosts
//

type FleetServerHost struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	HostURLs        []string `json:"host_urls"`
	IsDefault       bool     `json:"is_default"`
	IsInternal      bool     `json:"is_internal"`
	IsPreconfigured bool     `json:"is_preconfigured"`
	ProxyID         string   `json:"proxy_id"`
}

type ListFleetServerHostsRequest struct {
	HostURLs   []string `json:"host_urls"`
	ID         string   `json:"id"`
	IsDefault  bool     `json:"is_default"`
	IsInternal bool     `json:"is_internal"`
	Name       string   `json:"name"`
	ProxyID    string   `json:"proxy_id"`
}

type ListFleetServerHostsResponse struct {
	Items []FleetServerHost `json:"items"`
}

type FleetServerHostsResponse struct {
	Item struct {
		ID              string   `json:"id"`
		HostUrls        []string `json:"host_urls"`
		IsDefault       bool     `json:"is_default"`
		IsInternal      bool     `json:"is_internal"`
		IsPreconfigured bool     `json:"is_preconfigured"`
		Name            string   `json:"name"`
		ProxyID         string   `json:"proxy_id"`
	} `json:"item"`
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

// CreateFleetServerHosts creates a new Fleet Server host
func (client *Client) CreateFleetServerHosts(ctx context.Context, req ListFleetServerHostsRequest) (FleetServerHostsResponse, error) {
	bs, err := json.Marshal(req)
	if err != nil {
		return FleetServerHostsResponse{}, fmt.Errorf("could not marshal ListFleetServerHostsRequest: %w", err)
	}

	resp, err := client.Connection.SendWithContext(ctx, http.MethodPost,
		fleetFleetServerHostsAPI,
		nil, nil, bytes.NewReader(bs))
	if err != nil {
		return FleetServerHostsResponse{}, fmt.Errorf("error calling new fleet server hosts API: %w", err)
	}
	defer resp.Body.Close()

	var fleetResp FleetServerHostsResponse
	err = readJSONResponse(resp, &fleetResp)
	return fleetResp, err
}

type GetFleetServerHostRequest struct {
	ID string
}

type GetFleetServerHostResponse FleetServerHost

func (client *Client) GetFleetServerHost(ctx context.Context, request GetFleetServerHostRequest) (r GetFleetServerHostResponse, err error) {
	apiURL := fleetFleetServerHostsAPI + "/" + request.ID

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
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Fleet%20package%20policies
//

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

type PackagePolicyResponse struct {
	Item PackagePolicy `json:"item"`
}

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

//
// Fleet Proxies
// see full documentation on:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/elastic/kibana/main/oas_docs/bundle.json#/Fleet%20proxies
//

type ProxiesRequest struct {
	Certificate            string            `json:"certificate"`
	CertificateAuthorities string            `json:"certificate_authorities"`
	CertificateKey         string            `json:"certificate_key"`
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	ProxyHeaders           map[string]string `json:"proxy_headers"`
	URL                    string            `json:"url"`
}

type ProxiesResponse struct {
	Item struct {
		Certificate            string            `json:"certificate"`
		CertificateAuthorities string            `json:"certificate_authorities"`
		CertificateKey         string            `json:"certificate_key"`
		ID                     string            `json:"id"`
		IsPreconfigured        bool              `json:"is_preconfigured"`
		Name                   string            `json:"name"`
		ProxyHeaders           map[string]string `json:"proxy_headers"`
		URL                    string            `json:"url"`
	} `json:"item"`
}

// CreateFleetProxy creates a new proxy
func (client *Client) CreateFleetProxy(ctx context.Context, req ProxiesRequest) (ProxiesResponse, error) {
	// if `proxy_headers` is `null` 8.x kibana/Fleet will return a 400 - Bad request
	if req.ProxyHeaders == nil {
		req.ProxyHeaders = map[string]string{}
	}
	bs, err := json.Marshal(req)
	if err != nil {
		return ProxiesResponse{}, fmt.Errorf("could not marshal ListFleetServerHostsRequest: %w", err)
	}

	r, err := client.Connection.SendWithContext(ctx, http.MethodPost,
		fleetProxiesAPI, nil, nil,
		bytes.NewReader(bs),
	)
	if err != nil {
		return ProxiesResponse{}, err
	}
	defer r.Body.Close()

	resp := ProxiesResponse{}
	err = readJSONResponse(r, &resp)
	return resp, err
}

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

// GetPolicyUninstallTokens Retrieves the policy uninstall tokens
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

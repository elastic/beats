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

package ess

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateDeploymentRequest struct {
	Name    string `json:"name"`
	Region  string `json:"region"`
	Version string `json:"version"`
	Tags    []Tag  `json:"tags"`
}

type CreateDeploymentResponse struct {
	ID string `json:"id"`

	ElasticsearchEndpoint string
	KibanaEndpoint        string

	Username string
	Password string
}

type GetDeploymentResponse struct {
	Elasticsearch struct {
		Status     DeploymentStatus
		ServiceUrl string
	}
	Kibana struct {
		Status     DeploymentStatus
		ServiceUrl string
	}
	IntegrationsServer struct {
		Status     DeploymentStatus
		ServiceUrl string
	}
}

type DeploymentStatus string

func (d *DeploymentStatus) UnmarshalJSON(data []byte) error {
	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	switch status {
	case string(DeploymentStatusInitializing), string(DeploymentStatusReconfiguring), string(DeploymentStatusStarted):
		*d = DeploymentStatus(status)
	default:
		return fmt.Errorf("unknown status: [%s]", status)
	}

	return nil
}

func (d *DeploymentStatus) String() string {
	return string(*d)
}

const (
	DeploymentStatusInitializing  DeploymentStatus = "initializing"
	DeploymentStatusReconfiguring DeploymentStatus = "reconfiguring"
	DeploymentStatusStarted       DeploymentStatus = "started"
)

type DeploymentStatusResponse struct {
	Overall DeploymentStatus

	Elasticsearch      DeploymentStatus
	Kibana             DeploymentStatus
	IntegrationsServer DeploymentStatus
}

// CreateDeployment creates the deployment with the specified configuration.
func (c *Client) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*CreateDeploymentResponse, error) {
	reqBodyBytes, err := generateCreateDeploymentRequestBody(req)
	if err != nil {
		return nil, err
	}

	createResp, err := c.doPost(
		ctx,
		"deployments",
		"application/json",
		bytes.NewReader(reqBodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("error calling deployment creation API: %w", err)
	}
	defer createResp.Body.Close()

	var createRespBody struct {
		ID        string `json:"id"`
		Resources []struct {
			Kind        string `json:"kind"`
			Credentials struct {
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"credentials"`
		} `json:"resources"`
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(createResp.Body).Decode(&createRespBody); err != nil {
		return nil, fmt.Errorf("error parsing deployment creation API response: %w", err)
	}

	if len(createRespBody.Errors) > 0 {
		return nil, fmt.Errorf("failed to create: (%s) %s", createRespBody.Errors[0].Code, createRespBody.Errors[0].Message)
	}

	r := CreateDeploymentResponse{
		ID: createRespBody.ID,
	}

	for _, resource := range createRespBody.Resources {
		if resource.Kind == "elasticsearch" {
			r.Username = resource.Credentials.Username
			r.Password = resource.Credentials.Password
			break
		}
	}

	// Get Elasticsearch and Kibana endpoint URLs
	getResp, err := c.getDeployment(ctx, r.ID)
	if err != nil {
		return nil, fmt.Errorf("error calling deployment retrieval API: %w", err)
	}
	defer getResp.Body.Close()

	var getRespBody struct {
		Resources struct {
			Elasticsearch []struct {
				Info struct {
					Metadata struct {
						ServiceUrl string `json:"service_url"`
					} `json:"metadata"`
				} `json:"info"`
			} `json:"elasticsearch"`
			Kibana []struct {
				Info struct {
					Metadata struct {
						ServiceUrl string `json:"service_url"`
					} `json:"metadata"`
				} `json:"info"`
			} `json:"kibana"`
		} `json:"resources"`
	}

	if err := json.NewDecoder(getResp.Body).Decode(&getRespBody); err != nil {
		return nil, fmt.Errorf("error parsing deployment retrieval API response: %w", err)
	}

	r.ElasticsearchEndpoint = getRespBody.Resources.Elasticsearch[0].Info.Metadata.ServiceUrl
	r.KibanaEndpoint = getRespBody.Resources.Kibana[0].Info.Metadata.ServiceUrl

	return &r, nil
}

// ShutdownDeployment attempts to shut down the ESS deployment with the specified ID.
func (c *Client) ShutdownDeployment(ctx context.Context, deploymentID string) error {
	u, err := url.JoinPath("deployments", deploymentID, "_shutdown")
	if err != nil {
		return fmt.Errorf("unable to create deployment shutdown API URL: %w", err)
	}

	res, err := c.doPost(ctx, u, "", nil)
	if err != nil {
		return fmt.Errorf("error calling deployment shutdown API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("got unexpected response code [%d] from deployment shutdown API: %s", res.StatusCode, string(resBytes))
	}

	return nil
}

// DeploymentStatus returns the overall status of the deployment as well as statuses of every component.
func (c *Client) DeploymentStatus(ctx context.Context, deploymentID string) (*DeploymentStatusResponse, error) {
	getResp, err := c.getDeployment(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("error calling deployment retrieval API: %w", err)
	}
	defer getResp.Body.Close()

	var getRespBody struct {
		Resources struct {
			Elasticsearch []struct {
				Info struct {
					Status DeploymentStatus `json:"status"`
				} `json:"info"`
			} `json:"elasticsearch"`
			Kibana []struct {
				Info struct {
					Status DeploymentStatus `json:"status"`
				} `json:"info"`
			} `json:"kibana"`
			IntegrationsServer []struct {
				Info struct {
					Status DeploymentStatus `json:"status"`
				} `json:"info"`
			} `json:"integrations_server"`
		} `json:"resources"`
	}

	if err := json.NewDecoder(getResp.Body).Decode(&getRespBody); err != nil {
		return nil, fmt.Errorf("error parsing deployment retrieval API response: %w", err)
	}

	s := DeploymentStatusResponse{
		Elasticsearch:      getRespBody.Resources.Elasticsearch[0].Info.Status,
		Kibana:             getRespBody.Resources.Kibana[0].Info.Status,
		IntegrationsServer: getRespBody.Resources.IntegrationsServer[0].Info.Status,
	}
	s.Overall = overallStatus(s.Elasticsearch, s.Kibana, s.IntegrationsServer)

	return &s, nil
}

// DeploymentIsReady returns true when the deployment is ready, checking its status
// every `tick` until `waitFor` duration.
func (c *Client) DeploymentIsReady(ctx context.Context, deploymentID string, tick time.Duration) (bool, error) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	var errs error
	statusCh := make(chan DeploymentStatus, 1)
	for {
		select {
		case <-ctx.Done():
			return false, errors.Join(errs, ctx.Err())
		case <-ticker.C:
			go func() {
				statusCtx, statusCancel := context.WithTimeout(ctx, tick)
				defer statusCancel()
				status, err := c.DeploymentStatus(statusCtx, deploymentID)
				if err != nil {
					errs = errors.Join(errs, err)
					return
				}
				statusCh <- status.Overall
			}()
		case status := <-statusCh:
			if status == DeploymentStatusStarted {
				return true, nil
			}
		}
	}
}

func (c *Client) getDeployment(ctx context.Context, deploymentID string) (*http.Response, error) {
	u, err := url.JoinPath("deployments", deploymentID)
	if err != nil {
		return nil, fmt.Errorf("unable to create deployment retrieval API URL: %w", err)
	}

	return c.doGet(ctx, u)
}

func overallStatus(statuses ...DeploymentStatus) DeploymentStatus {
	// The overall status is started if every component's status is started. Otherwise,
	// we take the non-started components' statuses and pick the first one as the overall
	// status.
	statusMap := map[DeploymentStatus]struct{}{}
	for _, status := range statuses {
		statusMap[status] = struct{}{}
	}

	if len(statusMap) == 1 {
		if _, allStarted := statusMap[DeploymentStatusStarted]; allStarted {
			return DeploymentStatusStarted
		}
	}

	var overallStatus DeploymentStatus
	for _, status := range statuses {
		if status != DeploymentStatusStarted {
			overallStatus = status
			break
		}
	}

	return overallStatus
}

//go:embed create_deployment_request.tmpl.json
var createDeploymentRequestTemplate string

//go:embed create_deployment_csp_configuration.yaml
var cloudProviderSpecificValues []byte

func generateCreateDeploymentRequestBody(req CreateDeploymentRequest) ([]byte, error) {
	var csp string
	// Special case: AWS us-east-1 region is just called
	// us-east-1 (instead of aws-us-east-1)!
	if req.Region == "us-east-1" {
		csp = "aws"
	} else {
		regionParts := strings.Split(req.Region, "-")
		if len(regionParts) < 2 {
			return nil, fmt.Errorf("unable to parse CSP out of region [%s]", req.Region)
		}

		csp = regionParts[0]
	}
	templateContext, err := createDeploymentTemplateContext(csp, req)
	if err != nil {
		return nil, fmt.Errorf("creating request template context: %w", err)
	}

	tpl, err := template.New("create_deployment_request").
		Funcs(template.FuncMap{"json": jsonMarshal}).
		Parse(createDeploymentRequestTemplate)
	if err != nil {
		return nil, fmt.Errorf("unable to parse deployment creation template: %w", err)
	}

	var bBuf bytes.Buffer
	err = tpl.Execute(&bBuf, templateContext)
	if err != nil {
		return nil, fmt.Errorf("rendering create deployment request template with context %v : %w", templateContext, err)
	}
	return bBuf.Bytes(), nil
}

func jsonMarshal(in any) (string, error) {
	jsonBytes, err := json.Marshal(in)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func createDeploymentTemplateContext(csp string, req CreateDeploymentRequest) (map[string]any, error) {
	cspSpecificContext, err := loadCspValues(csp)
	if err != nil {
		return nil, fmt.Errorf("loading csp-specific values for %q: %w", csp, err)
	}

	cspSpecificContext["request"] = req

	return cspSpecificContext, nil
}

func loadCspValues(csp string) (map[string]any, error) {
	var cspValues map[string]map[string]any

	err := yaml.Unmarshal(cloudProviderSpecificValues, &cspValues)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling error: %w", err)
	}
	values, supportedCSP := cspValues[csp]
	if !supportedCSP {
		return nil, fmt.Errorf("csp %s not supported", csp)
	}

	return values, nil
}

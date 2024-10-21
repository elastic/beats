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
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"io"
	"net/http"
	"strings"
	"time"
)

var serverlessURL = "https://cloud.elastic.co"

// ServerlessClient is the handler the serverless ES instance
type ServerlessClient struct {
	region      string
	projectType string
	api         string
	proj        Project
	log         common.Logger
}

// ServerlessRequest contains the data needed for a new serverless instance
type ServerlessRequest struct {
	Name     string `json:"name"`
	RegionID string `json:"region_id"`
}

// Project represents a serverless project
type Project struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Type   string `json:"type"`
	Region string `json:"region_id"`

	Credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"credentials"`

	Endpoints struct {
		Elasticsearch string `json:"elasticsearch"`
		Kibana        string `json:"kibana"`
		Fleet         string `json:"fleet,omitempty"`
		APM           string `json:"apm,omitempty"`
	} `json:"endpoints"`
}

// CredResetResponse contains the new auth details for a
// stack credential reset
type CredResetResponse struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

// NewServerlessClient creates a new instance of the serverless client
func NewServerlessClient(region, projectType, api string, logger common.Logger) *ServerlessClient {
	return &ServerlessClient{
		region:      region,
		api:         api,
		projectType: projectType,
		log:         logger,
	}
}

// DeployStack creates a new serverless elastic stack
func (srv *ServerlessClient) DeployStack(ctx context.Context, req ServerlessRequest) (Project, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return Project{}, fmt.Errorf("error marshaling JSON request %w", err)
	}
	urlPath := fmt.Sprintf("%s/api/v1/serverless/projects/%s", serverlessURL, srv.projectType)

	httpHandler, err := http.NewRequestWithContext(ctx, "POST", urlPath, bytes.NewReader(reqBody))
	if err != nil {
		return Project{}, fmt.Errorf("error creating new httpRequest: %w", err)
	}

	httpHandler.Header.Set("Content-Type", "application/json")
	httpHandler.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", srv.api))

	resp, err := http.DefaultClient.Do(httpHandler)
	if err != nil {
		return Project{}, fmt.Errorf("error performing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		p, _ := io.ReadAll(resp.Body)
		return Project{}, fmt.Errorf("Non-201 status code returned by server: %d, body: %s", resp.StatusCode, string(p))
	}

	serverlessHandle := Project{}
	err = json.NewDecoder(resp.Body).Decode(&serverlessHandle)
	if err != nil {
		return Project{}, fmt.Errorf("error decoding JSON response: %w", err)
	}
	srv.proj = serverlessHandle

	// as of 8/8-ish, the serverless ESS cloud no longer provides credentials on the first POST request, we must send an additional POST
	// to reset the credentials
	updated, err := srv.ResetCredentials(ctx)
	if err != nil {
		return serverlessHandle, fmt.Errorf("error resetting credentials: %w", err)
	}
	srv.proj.Credentials.Username = updated.Username
	srv.proj.Credentials.Password = updated.Password

	return serverlessHandle, nil
}

// DeploymentIsReady returns true when the serverless deployment is healthy and ready
func (srv *ServerlessClient) DeploymentIsReady(ctx context.Context) (bool, error) {
	err := srv.WaitForEndpoints(ctx)
	if err != nil {
		return false, fmt.Errorf("error waiting for endpoints to become available: %w", err)
	}
	srv.log.Logf("Endpoints available: ES: %s Fleet: %s Kibana: %s", srv.proj.Endpoints.Elasticsearch, srv.proj.Endpoints.Fleet, srv.proj.Endpoints.Kibana)
	err = srv.WaitForElasticsearch(ctx)
	if err != nil {
		return false, fmt.Errorf("error waiting for ES to become available: %w", err)
	}
	srv.log.Logf("Elasticsearch healthy...")
	err = srv.WaitForKibana(ctx)
	if err != nil {
		return false, fmt.Errorf("error waiting for Kibana to become available: %w", err)
	}
	srv.log.Logf("Kibana healthy...")

	return true, nil
}

// DeleteDeployment deletes the deployment
func (srv *ServerlessClient) DeleteDeployment(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/api/v1/serverless/projects/%s/%s", serverlessURL, srv.proj.Type, srv.proj.ID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", srv.api))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing delete request: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d from %s: %s", resp.StatusCode, req.URL, errBody)
	}
	return nil
}

// WaitForEndpoints polls the API and waits until fleet/ES endpoints are available
func (srv *ServerlessClient) WaitForEndpoints(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/api/v1/serverless/projects/%s/%s", serverlessURL, srv.proj.Type, srv.proj.ID)
	httpHandler, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	httpHandler.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", srv.api))

	readyFunc := func(resp *http.Response) bool {
		project := &Project{}
		err = json.NewDecoder(resp.Body).Decode(project)
		resp.Body.Close()
		if err != nil {
			srv.log.Logf("response decoding error: %v", err)
			return false
		}
		if project.Endpoints.Elasticsearch != "" {
			// fake out the fleet URL, set to ES url
			if project.Endpoints.Fleet == "" {
				project.Endpoints.Fleet = strings.Replace(project.Endpoints.Elasticsearch, "es.eks", "fleet.eks", 1)
			}

			srv.proj.Endpoints = project.Endpoints
			return true
		}
		return false
	}

	err = srv.waitForRemoteState(ctx, httpHandler, time.Second*5, readyFunc)
	if err != nil {
		return fmt.Errorf("error waiting for remote instance to start: %w", err)
	}

	return nil
}

// WaitForElasticsearch waits until the ES endpoint is healthy
func (srv *ServerlessClient) WaitForElasticsearch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", srv.proj.Endpoints.Elasticsearch, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.SetBasicAuth(srv.proj.Credentials.Username, srv.proj.Credentials.Password)

	// _cluster/health no longer works on serverless, just check response code
	readyFunc := func(resp *http.Response) bool {
		return resp.StatusCode == 200
	}

	err = srv.waitForRemoteState(ctx, req, time.Second*5, readyFunc)
	if err != nil {
		return fmt.Errorf("error waiting for ES to become healthy: %w", err)
	}
	return nil
}

// WaitForKibana waits until the kibana endpoint is healthy
func (srv *ServerlessClient) WaitForKibana(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", srv.proj.Endpoints.Kibana+"/api/status", nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.SetBasicAuth(srv.proj.Credentials.Username, srv.proj.Credentials.Password)

	readyFunc := func(resp *http.Response) bool {
		var status struct {
			Status struct {
				Overall struct {
					Level string `json:"level"`
				} `json:"overall"`
			} `json:"status"`
		}
		err = json.NewDecoder(resp.Body).Decode(&status)
		if err != nil {
			srv.log.Logf("response decoding error: %v", err)
			return false
		}
		resp.Body.Close()
		return status.Status.Overall.Level == "available"
	}

	err = srv.waitForRemoteState(ctx, req, time.Second*5, readyFunc)
	if err != nil {
		return fmt.Errorf("error waiting for ES to become healthy: %w", err)
	}
	return nil
}

// ResetCredentials resets the credentials for the given ESS instance
func (srv *ServerlessClient) ResetCredentials(ctx context.Context) (CredResetResponse, error) {
	resetURL := fmt.Sprintf("%s/api/v1/serverless/projects/%s/%s/_reset-internal-credentials", serverlessURL, srv.projectType, srv.proj.ID)

	resetHandler, err := http.NewRequestWithContext(ctx, "POST", resetURL, nil)
	if err != nil {
		return CredResetResponse{}, fmt.Errorf("error creating new httpRequest: %w", err)
	}

	resetHandler.Header.Set("Content-Type", "application/json")
	resetHandler.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", srv.api))

	resp, err := http.DefaultClient.Do(resetHandler)
	if err != nil {
		return CredResetResponse{}, fmt.Errorf("error performing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p, _ := io.ReadAll(resp.Body)
		return CredResetResponse{}, fmt.Errorf("Non-200 status code returned by server: %d, body: %s", resp.StatusCode, string(p))
	}

	updated := CredResetResponse{}
	err = json.NewDecoder(resp.Body).Decode(&updated)
	if err != nil {
		return CredResetResponse{}, fmt.Errorf("error decoding JSON response: %w", err)
	}

	return updated, nil
}

func (srv *ServerlessClient) waitForRemoteState(ctx context.Context, httpHandler *http.Request, tick time.Duration, isReady func(*http.Response) bool) error {
	timer := time.NewTimer(time.Millisecond)
	// in cases where we get a timeout, also return the last error returned via HTTP
	var lastErr error
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("got context done; Last HTTP Error: %w", lastErr)
		case <-timer.C:
		}

		resp, err := http.DefaultClient.Do(httpHandler)
		if err != nil {
			errMsg := fmt.Errorf("request error: %w", err)
			// Logger interface doesn't have a debug level and we don't want to auto-log these;
			// as most of the time it's just spam.
			//srv.log.Logf(errMsg.Error())
			lastErr = errMsg
			timer.Reset(time.Second * 5)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			errBody, _ := io.ReadAll(resp.Body)
			errMsg := fmt.Errorf("unexpected status code %d in request to %s, body: %s", resp.StatusCode, httpHandler.URL.String(), string(errBody))
			//srv.log.Logf(errMsg.Error())
			lastErr = errMsg
			resp.Body.Close()
			timer.Reset(time.Second * 5)
			continue
		}

		if isReady(resp) {
			return nil
		}
		timer.Reset(tick)
	}
}

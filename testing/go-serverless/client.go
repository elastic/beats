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

// Package serverless creates and destroys Elastic Cloud serverless projects
// for integration tests.
package serverless

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultCloudURL is the Elastic Cloud API endpoint used when none is configured.
const DefaultCloudURL = "https://cloud.elastic.co"

// ProjectTypeObservability is the project type the Beats tests run against.
const ProjectTypeObservability = "observability"

const defaultPollInterval = 5 * time.Second

// Project is a serverless project as returned by the Cloud API.
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
	} `json:"endpoints"`
}

// Region is an entry of the serverless regions API.
type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Client talks to the serverless part of the Elastic Cloud API.
type Client struct {
	baseURL     string
	apiKey      string
	projectType string
	http        *http.Client

	pollInterval time.Duration
}

// NewClient returns a Client authenticating with the given Cloud API key.
// An empty baseURL selects DefaultCloudURL.
func NewClient(baseURL, apiKey, projectType string) *Client {
	if baseURL == "" {
		baseURL = DefaultCloudURL
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiKey:      apiKey,
		projectType: projectType,
		http:        &http.Client{Timeout: 30 * time.Second},

		pollInterval: defaultPollInterval,
	}
}

// PickRegion returns preferred if the serverless API offers it, otherwise the
// first region the API lists. The serverless and regular Cloud APIs use
// different region IDs, so a region configured for the latter may not exist here.
func (c *Client) PickRegion(ctx context.Context, preferred string) (string, error) {
	var regions []Region
	if err := c.do(ctx, http.MethodGet, "/api/v1/serverless/regions", nil, http.StatusOK, &regions); err != nil {
		return "", fmt.Errorf("listing serverless regions: %w", err)
	}
	if len(regions) == 0 {
		return "", errors.New("the serverless API returned no regions")
	}
	for _, r := range regions {
		if r.ID == preferred {
			return preferred, nil
		}
	}
	return regions[0].ID, nil
}

// Create creates a project and resets its credentials, since the API doesn't
// return usable ones on creation. It doesn't wait for the project to be ready,
// see WaitReady.
func (c *Client) Create(ctx context.Context, name, region string) (Project, error) {
	body, err := json.Marshal(map[string]string{"name": name, "region_id": region})
	if err != nil {
		return Project{}, fmt.Errorf("encoding create request: %w", err)
	}

	var proj Project
	path := fmt.Sprintf("/api/v1/serverless/projects/%s", c.projectType)
	if err := c.do(ctx, http.MethodPost, path, body, http.StatusCreated, &proj); err != nil {
		return Project{}, fmt.Errorf("creating project: %w", err)
	}

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	path = fmt.Sprintf("/api/v1/serverless/projects/%s/%s/_reset-internal-credentials", c.projectType, proj.ID)
	if err := c.do(ctx, http.MethodPost, path, nil, http.StatusOK, &creds); err != nil {
		return proj, fmt.Errorf("resetting credentials of project %s: %w", proj.ID, err)
	}
	proj.Credentials.Username = creds.Username
	proj.Credentials.Password = creds.Password
	if proj.Type == "" {
		proj.Type = c.projectType
	}
	return proj, nil
}

// WaitReady blocks until the endpoints of proj are published and both
// Elasticsearch and Kibana answer, then returns the updated project.
func (c *Client) WaitReady(ctx context.Context, proj Project) (Project, error) {
	path := fmt.Sprintf("/api/v1/serverless/projects/%s/%s", c.projectType, proj.ID)
	err := c.poll(ctx, func() error {
		var got Project
		if err := c.do(ctx, http.MethodGet, path, nil, http.StatusOK, &got); err != nil {
			return err
		}
		if got.Endpoints.Elasticsearch == "" || got.Endpoints.Kibana == "" {
			return errors.New("endpoints not published yet")
		}
		proj.Endpoints = got.Endpoints
		return nil
	})
	if err != nil {
		return proj, fmt.Errorf("waiting for endpoints of project %s: %w", proj.ID, err)
	}

	err = c.poll(ctx, func() error {
		return c.checkStatus(ctx, proj, proj.Endpoints.Elasticsearch, http.StatusOK)
	})
	if err != nil {
		return proj, fmt.Errorf("waiting for Elasticsearch of project %s: %w", proj.ID, err)
	}

	err = c.poll(ctx, func() error {
		return c.checkKibana(ctx, proj)
	})
	if err != nil {
		return proj, fmt.Errorf("waiting for Kibana of project %s: %w", proj.ID, err)
	}
	return proj, nil
}

// Delete deletes the project.
func (c *Client) Delete(ctx context.Context, proj Project) error {
	path := fmt.Sprintf("/api/v1/serverless/projects/%s/%s", c.projectType, proj.ID)
	if err := c.do(ctx, http.MethodDelete, path, nil, http.StatusOK, nil); err != nil {
		return fmt.Errorf("deleting project %s: %w", proj.ID, err)
	}
	return nil
}

func (c *Client) checkStatus(ctx context.Context, proj Project, url string, want int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(proj.Credentials.Username, proj.Credentials.Password)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != want {
		return fmt.Errorf("GET %s: status %d, want %d", url, resp.StatusCode, want)
	}
	return nil
}

func (c *Client) checkKibana(ctx context.Context, proj Project) error {
	url := strings.TrimRight(proj.Endpoints.Kibana, "/") + "/api/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(proj.Credentials.Username, proj.Credentials.Password)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	var status struct {
		Status struct {
			Overall struct {
				Level string `json:"level"`
			} `json:"overall"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("decoding Kibana status: %w", err)
	}
	if level := status.Status.Overall.Level; level != "available" {
		return fmt.Errorf("kibana status is %q", level)
	}
	return nil
}

// do performs an authenticated Cloud API request and decodes the JSON response into out, if non-nil.
func (c *Client) do(ctx context.Context, method, path string, body []byte, wantStatus int, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s %s: status %d, want %d, body: %s", method, path, resp.StatusCode, wantStatus, msg)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s %s: decoding response: %w", method, path, err)
	}
	return nil
}

// poll calls check every c.pollInterval until it succeeds or ctx is done, in
// which case the last check error is returned alongside the context error.
func (c *Client) poll(ctx context.Context, check func() error) error {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		lastErr := check()
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %w)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

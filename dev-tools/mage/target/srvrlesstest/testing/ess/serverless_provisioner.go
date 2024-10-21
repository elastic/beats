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
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"io"
	"net/http"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

const ProvisionerServerless = "serverless"

// ServerlessProvisioner contains
type ServerlessProvisioner struct {
	cfg ProvisionerConfig
	log common.Logger
}

type defaultLogger struct {
	wrapped *logp.Logger
}

// Logf implements the runner.Logger interface
func (log *defaultLogger) Logf(format string, args ...any) {
	if len(args) == 0 {

	} else {
		log.wrapped.Infof(format, args)
	}

}

// ServerlessRegions is the JSON response from the serverless regions API endpoint
type ServerlessRegions struct {
	CSP       string `json:"csp"`
	CSPRegion string `json:"csp_region"`
	ID        string `json:"id"`
	Name      string `json:"name"`
}

// NewServerlessProvisioner creates a new StackProvisioner instance for serverless
func NewServerlessProvisioner(ctx context.Context, cfg ProvisionerConfig) (common.StackProvisioner, error) {
	prov := &ServerlessProvisioner{
		cfg: cfg,
		log: &defaultLogger{wrapped: logp.L()},
	}
	err := prov.CheckCloudRegion(ctx)
	if err != nil {
		return nil, fmt.Errorf("error checking region setting: %w", err)
	}
	return prov, nil
}

func (prov *ServerlessProvisioner) Name() string {
	return ProvisionerServerless
}

// SetLogger sets the logger for the
func (prov *ServerlessProvisioner) SetLogger(l common.Logger) {
	prov.log = l
}

// Create creates a stack.
func (prov *ServerlessProvisioner) Create(ctx context.Context, request common.StackRequest) (common.Stack, error) {
	// allow up to 4 minutes for requests
	createCtx, createCancel := context.WithTimeout(ctx, 4*time.Minute)
	defer createCancel()

	client := NewServerlessClient(prov.cfg.Region, "observability", prov.cfg.APIKey, prov.log)
	srvReq := ServerlessRequest{Name: request.ID, RegionID: prov.cfg.Region}

	prov.log.Logf("Creating serverless stack %s [stack_id: %s]", request.Version, request.ID)
	proj, err := client.DeployStack(createCtx, srvReq)
	if err != nil {
		return common.Stack{}, fmt.Errorf("error deploying stack for request %s: %w", request.ID, err)
	}
	err = client.WaitForEndpoints(createCtx)
	if err != nil {
		return common.Stack{}, fmt.Errorf("error waiting for endpoints to become available for serverless stack %s [stack_id: %s, deployment_id: %s]: %w", request.Version, request.ID, proj.ID, err)
	}
	stack := common.Stack{
		ID:            request.ID,
		Provisioner:   prov.Name(),
		Version:       request.Version,
		Elasticsearch: client.proj.Endpoints.Elasticsearch,
		Kibana:        client.proj.Endpoints.Kibana,
		Username:      client.proj.Credentials.Username,
		Password:      client.proj.Credentials.Password,
		Internal: map[string]interface{}{
			"deployment_id":   proj.ID,
			"deployment_type": proj.Type,
		},
		Ready: false,
	}
	prov.log.Logf("Created serverless stack %s [stack_id: %s, deployment_id: %s]", request.Version, request.ID, proj.ID)
	return stack, nil
}

// WaitForReady should block until the stack is ready or the context is cancelled.
func (prov *ServerlessProvisioner) WaitForReady(ctx context.Context, stack common.Stack) (common.Stack, error) {
	deploymentID, deploymentType, err := prov.getDeploymentInfo(stack)
	if err != nil {
		return stack, fmt.Errorf("failed to get deployment info from the stack: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	client := NewServerlessClient(prov.cfg.Region, "observability", prov.cfg.APIKey, prov.log)
	client.proj.ID = deploymentID
	client.proj.Type = deploymentType
	client.proj.Region = prov.cfg.Region
	client.proj.Endpoints.Elasticsearch = stack.Elasticsearch
	client.proj.Endpoints.Kibana = stack.Kibana
	client.proj.Credentials.Username = stack.Username
	client.proj.Credentials.Password = stack.Password

	prov.log.Logf("Waiting for serverless stack %s to be ready [stack_id: %s, deployment_id: %s]", stack.Version, stack.ID, deploymentID)

	errCh := make(chan error)
	var lastErr error

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			return stack, fmt.Errorf("serverless stack %s [stack_id: %s, deployment_id: %s] never became ready: %w", stack.Version, stack.ID, deploymentID, lastErr)
		case <-ticker.C:
			go func() {
				statusCtx, statusCancel := context.WithTimeout(ctx, 30*time.Second)
				defer statusCancel()
				ready, err := client.DeploymentIsReady(statusCtx)
				if err != nil {
					errCh <- err
				} else if !ready {
					errCh <- fmt.Errorf("serverless stack %s [stack_id: %s, deployment_id: %s] never became ready", stack.Version, stack.ID, deploymentID)
				} else {
					errCh <- nil
				}
			}()
		case err := <-errCh:
			if err == nil {
				stack.Ready = true
				return stack, nil
			}
			lastErr = err
		}
	}
}

// Delete deletes a stack.
func (prov *ServerlessProvisioner) Delete(ctx context.Context, stack common.Stack) error {
	deploymentID, deploymentType, err := prov.getDeploymentInfo(stack)
	if err != nil {
		return fmt.Errorf("failed to get deployment info from the stack: %w", err)
	}

	client := NewServerlessClient(prov.cfg.Region, "observability", prov.cfg.APIKey, prov.log)
	client.proj.ID = deploymentID
	client.proj.Type = deploymentType
	client.proj.Region = prov.cfg.Region
	client.proj.Endpoints.Elasticsearch = stack.Elasticsearch
	client.proj.Endpoints.Kibana = stack.Kibana
	client.proj.Credentials.Username = stack.Username
	client.proj.Credentials.Password = stack.Password

	prov.log.Logf("Destroying serverless stack %s [stack_id: %s, deployment_id: %s]", stack.Version, stack.ID, deploymentID)
	err = client.DeleteDeployment(ctx)
	if err != nil {
		return fmt.Errorf("error removing serverless stack %s [stack_id: %s, deployment_id: %s]: %w", stack.Version, stack.ID, deploymentID, err)
	}
	return nil
}

// CheckCloudRegion checks to see if the provided region is valid for the serverless
// if we have an invalid region, overwrite with a valid one.
// The "normal" and serverless ESS APIs have different regions, hence why we need this.
func (prov *ServerlessProvisioner) CheckCloudRegion(ctx context.Context) error {
	urlPath := fmt.Sprintf("%s/api/v1/serverless/regions", serverlessURL)

	httpHandler, err := http.NewRequestWithContext(ctx, "GET", urlPath, nil)
	if err != nil {
		return fmt.Errorf("error creating new httpRequest: %w", err)
	}

	httpHandler.Header.Set("Content-Type", "application/json")
	httpHandler.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", prov.cfg.APIKey))

	resp, err := http.DefaultClient.Do(httpHandler)
	if err != nil {
		return fmt.Errorf("error performing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Non-201 status code returned by server: %d, body: %s", resp.StatusCode, string(p))
	}
	regions := []ServerlessRegions{}

	err = json.NewDecoder(resp.Body).Decode(&regions)
	if err != nil {
		return fmt.Errorf("error unpacking regions from list: %w", err)
	}
	resp.Body.Close()

	found := false
	for _, region := range regions {
		if region.ID == prov.cfg.Region {
			found = true
		}
	}
	if !found {
		if len(regions) == 0 {
			return fmt.Errorf("no regions found for cloudless API")
		}
		newRegion := regions[0].ID
		prov.log.Logf("WARNING: Region %s is not available for serverless, selecting %s. Other regions are:", prov.cfg.Region, newRegion)
		for _, avail := range regions {
			prov.log.Logf(" %s - %s", avail.ID, avail.Name)
		}
		prov.cfg.Region = newRegion
	}

	return nil
}

func (prov *ServerlessProvisioner) getDeploymentInfo(stack common.Stack) (string, string, error) {
	if stack.Internal == nil {
		return "", "", fmt.Errorf("missing internal information")
	}
	deploymentIDRaw, ok := stack.Internal["deployment_id"]
	if !ok {
		return "", "", fmt.Errorf("missing internal deployment_id")
	}
	deploymentID, ok := deploymentIDRaw.(string)
	if !ok {
		return "", "", fmt.Errorf("internal deployment_id not a string")
	}
	deploymentTypeRaw, ok := stack.Internal["deployment_type"]
	if !ok {
		return "", "", fmt.Errorf("missing internal deployment_type")
	}
	deploymentType, ok := deploymentTypeRaw.(string)
	if !ok {
		return "", "", fmt.Errorf("internal deployment_type is not a string")
	}
	return deploymentID, deploymentType, nil
}

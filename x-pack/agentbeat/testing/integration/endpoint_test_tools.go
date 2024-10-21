// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent/pkg/control/v2/client"
	"github.com/elastic/elastic-agent/pkg/testing/define"
)

//go:embed endpoint_security_package.json.tmpl
var endpointPackagePolicyTemplate string

type endpointPackageTemplateVars struct {
	ID       string
	Name     string
	PolicyID string
	Version  string
}

// TODO: Setup a GitHub Action to update this for each release of https://github.com/elastic/endpoint-package
const endpointPackageVersion = "8.11.0"

func agentAndEndpointAreHealthy(t *testing.T, ctx context.Context, agentClient client.Client) bool {
	t.Helper()

	state, err := agentClient.State(ctx)
	if err != nil {
		t.Logf("Error getting agent state: %s", err)
		return false
	}

	if state.State != client.Healthy {
		t.Logf("local Agent is not Healthy: current state: %+v", state)
		return false
	}

	foundEndpointInputUnit := false
	foundEndpointOutputUnit := false
	for _, comp := range state.Components {
		isEndpointComponent := strings.Contains(comp.Name, "endpoint")
		if comp.State != client.Healthy {
			t.Logf("endpoint component is not Healthy: current state: %+v", comp)
			return false
		}

		for _, unit := range comp.Units {
			if isEndpointComponent {
				if unit.UnitType == client.UnitTypeInput {
					foundEndpointInputUnit = true
				}
				if unit.UnitType == client.UnitTypeOutput {
					foundEndpointOutputUnit = true
				}
			}

			if unit.State != client.Healthy {
				t.Logf("unit %q is not Healthy\n%+v", unit.UnitID, unit)
				return false
			}
		}
	}

	// Ensure both the endpoint input and output units were found and healthy.
	if !foundEndpointInputUnit || !foundEndpointOutputUnit {
		t.Logf("State did not contain endpoint units (input: %v/output: %v) state: %+v. ", foundEndpointInputUnit, foundEndpointOutputUnit, state)
		return false
	}

	return true
}

// Installs the Elastic Defend package to cause the agent to install the endpoint-security service.
func installElasticDefendPackage(t *testing.T, info *define.Info, policyID string) (r kibana.PackagePolicyResponse, err error) {
	t.Helper()

	t.Log("Templating endpoint package policy request")
	tmpl, err := template.New("pkgpolicy").Parse(endpointPackagePolicyTemplate)
	if err != nil {
		return r, fmt.Errorf("error creating new template: %w", err)
	}

	packagePolicyID := uuid.Must(uuid.NewV4()).String()
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
	packagePolicyReq := kibana.PackagePolicyRequest{}
	err = json.Unmarshal(pkgPolicyBuf.Bytes(), &packagePolicyReq)
	if err != nil {
		return r, fmt.Errorf("templated package policy is not valid JSON: %s, %w", pkgPolicyBuf.String(), err)
	}

	t.Log("POST /api/fleet/package_policies")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pkgResp, err := info.KibanaClient.InstallFleetPackage(ctx, packagePolicyReq)
	if err != nil {
		t.Logf("Error installing fleet package: %v", err)
		return r, fmt.Errorf("error installing fleet package: %w", err)
	}
	t.Logf("Endpoint package Policy Response:\n%+v", pkgResp)
	return pkgResp, err
}

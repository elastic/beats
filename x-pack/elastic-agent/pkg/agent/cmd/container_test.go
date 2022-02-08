// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	defaultFleetPolicy = kibanaPolicy{
		ID:     "499b5aa7-d214-5b5d-838b-3cd76469844e",
		Name:   "Default Fleet Server policy",
		Status: "active",
		PackagePolicies: []string{
			"default-fleet-server-agent-policy",
		},
	}
	defaultAgentPolicy = kibanaPolicy{
		ID:     "2016d7cc-135e-5583-9758-3ba01f5a06e5",
		Name:   "Default policy",
		Status: "active",
		PackagePolicies: []string{
			"default-system-policy",
		},
	}
	nondefaultAgentPolicy = kibanaPolicy{
		ID:     "bc634ea6-8460-4925-babd-7540c3e7df24",
		Name:   "Another free policy",
		Status: "active",
		PackagePolicies: []string{
			"3668df9e-f2a3-4b65-9e6c-58ed352f2b63",
		},
	}

	nondefaultFleetPolicy = kibanaPolicy{
		ID:     "7b0093d2-7eab-4862-86c8-63b3dd1db001",
		Name:   "Some kinda dependent policy",
		Status: "active",
		PackagePolicies: []string{
			"63e2f84f-ab11-439c-93fa-531ff5b53e20",
		},
	}
)

var policies kibanaPolicies = kibanaPolicies{
	Items: []kibanaPolicy{
		defaultFleetPolicy,
		defaultAgentPolicy,
		nondefaultAgentPolicy,
		nondefaultFleetPolicy,
	},
}

var PackagePolicies = packagePolicyResponse{
	Fleet: map[string]struct{}{
		"7b0093d2-7eab-4862-86c8-63b3dd1db001": {},
		"499b5aa7-d214-5b5d-838b-3cd76469844e": {},
	},
	NonFleet: map[string]struct{}{
		"bc634ea6-8460-4925-babd-7540c3e7df24": {},
		"2016d7cc-135e-5583-9758-3ba01f5a06e5": {},
	},
}

func TestFindPolicyById(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable:   true,
			PolicyID: "7b0093d2-7eab-4862-86c8-63b3dd1db001",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &nondefaultFleetPolicy, policy)
}

func TestFindPolicyByName(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			TokenPolicyName: "Default policy",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &defaultAgentPolicy, policy)
}

func TestFindPolicyByIdAndName(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			TokenPolicyName: "Default policy",
		},
		FleetServer: fleetServerConfig{
			Enable:   true,
			PolicyID: "7b0093d2-7eab-4862-86c8-63b3dd1db001",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &nondefaultFleetPolicy, policy)
}

func TestFindPolicyDefaultFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: true,
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &defaultFleetPolicy, policy)
}

func TestFindPolicyDefaultNonFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: false,
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &defaultAgentPolicy, policy)
}

func TestFindPolicyNoMatchNonFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: false,
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &packagePolicyResponse{Fleet: PackagePolicies.Fleet})
	require.Error(t, err)
	require.Nil(t, policy)
}

func TestFindPolicyNoMatchFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: true,
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &packagePolicyResponse{NonFleet: PackagePolicies.NonFleet})
	require.Error(t, err)
	require.Nil(t, policy)
}

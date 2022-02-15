// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"

	"github.com/stretchr/testify/require"
)

var (
	defaultFleetPolicy = kibanaPolicy{
		ID:     "fleet-server-policy",
		Name:   "Default Fleet Server policy",
		Status: "active",
	}
	defaultAgentPolicy = kibanaPolicy{
		ID:     "2016d7cc-135e-5583-9758-3ba01f5a06e5",
		Name:   "Default policy",
		Status: "active",
	}
	nondefaultAgentPolicy = kibanaPolicy{
		ID:     "bc634ea6-8460-4925-babd-7540c3e7df24",
		Name:   "Another free policy",
		Status: "active",
	}

	nondefaultFleetPolicy = kibanaPolicy{
		ID:     "7b0093d2-7eab-4862-86c8-63b3dd1db001",
		Name:   "Some kinda dependent policy",
		Status: "active",
	}
)

var streams *cli.IOStreams = cli.NewIOStreams()

var policies kibanaPolicies = kibanaPolicies{
	Items: []kibanaPolicy{
		defaultFleetPolicy,
		defaultAgentPolicy,
		nondefaultAgentPolicy,
		nondefaultFleetPolicy,
	},
}

// Finding policies

func TestFindPolicyById(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable:   true,
			PolicyID: "7b0093d2-7eab-4862-86c8-63b3dd1db001",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.NoError(t, err)
	require.Equal(t, &nondefaultFleetPolicy, policy)
}

func TestFindPolicyByName(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			TokenPolicyName: "Default policy",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.NoError(t, err)
	require.Equal(t, &defaultAgentPolicy, policy)
}

func TestFindPolicyByIdOverName(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			TokenPolicyName: "Default policy",
		},
		FleetServer: fleetServerConfig{
			Enable:   true,
			PolicyID: "7b0093d2-7eab-4862-86c8-63b3dd1db001",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.NoError(t, err)
	require.Equal(t, &nondefaultFleetPolicy, policy)
}

func TestFindPolicyByIdMiss(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable:   true,
			PolicyID: "invalid id",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.Error(t, err)
	require.Nil(t, policy)
}

func TestFindPolicyByNameMiss(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			TokenPolicyName: "invalid name",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.Error(t, err)
	require.Nil(t, policy)
}

func TestFindPolicyDefaultFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable:          true,
			DefaultPolicyID: "fleet-server-policy",
		},
	}

	items := []kibanaPolicy{
		defaultAgentPolicy,
		nondefaultAgentPolicy,
		nondefaultFleetPolicy,
		defaultFleetPolicy,
	}

	policy, err := findPolicy(cfg, items, streams)
	require.NoError(t, err)
	require.Equal(t, &defaultFleetPolicy, policy)
}

func TestFindPolicyAmbiguousNoDefaultFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: true,
		},
	}

	items := []kibanaPolicy{
		defaultAgentPolicy,
		nondefaultAgentPolicy,
		nondefaultFleetPolicy,
		defaultFleetPolicy,
	}

	policy, err := findPolicy(cfg, items, streams)
	require.Error(t, err)
	require.Nil(t, policy)
}

func TestFindPolicyDefaultNonFleet(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			DefaultTokenPolicyName: "Default policy",
		},
		FleetServer: fleetServerConfig{
			Enable: false,
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.NoError(t, err)
	require.Equal(t, &defaultAgentPolicy, policy)
}

func TestFindPolicyNoMatchNonFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: false,
		},
	}

	policy, err := findPolicy(cfg, policies.Items, streams)
	require.Error(t, err)
	require.Nil(t, policy)
}

func TestFindPolicyNoMatchFleet(t *testing.T) {
	cfg := setupConfig{
		FleetServer: fleetServerConfig{
			Enable: true,
		},
	}

	items := []kibanaPolicy{
		defaultAgentPolicy,
		nondefaultAgentPolicy,
		nondefaultFleetPolicy,
	}
	policy, err := findPolicy(cfg, items, streams)
	require.Error(t, err)
	require.Nil(t, policy)
}

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
		ID:     "fleet-server-policy",
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
		"fleet-server-policy":                  {},
	},
	NonFleet: map[string]struct{}{
		"bc634ea6-8460-4925-babd-7540c3e7df24": {},
		"2016d7cc-135e-5583-9758-3ba01f5a06e5": {},
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

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
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

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
	require.Error(t, err)
	require.Nil(t, policy)
}

func TestFindPolicyByNameMiss(t *testing.T) {
	cfg := setupConfig{
		Fleet: fleetConfig{
			TokenPolicyName: "invalid name",
		},
	}

	policy, err := findPolicy(cfg, policies.Items, &PackagePolicies)
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

	policy, err := findPolicy(cfg, items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &defaultFleetPolicy, policy)
}

func TestFindPolicyNoDefaultFleet(t *testing.T) {
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

	policy, err := findPolicy(cfg, items, &PackagePolicies)
	require.NoError(t, err)
	require.Equal(t, &nondefaultFleetPolicy, policy)
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

	items := []kibanaPolicy{
		defaultAgentPolicy,
		nondefaultAgentPolicy,
		nondefaultFleetPolicy,
	}
	policy, err := findPolicy(cfg, items, &packagePolicyResponse{NonFleet: PackagePolicies.NonFleet})
	require.Error(t, err)
	require.Nil(t, policy)
}

// Separating policies by package
var (
	fleetPackage = kibanaPackage{
		Name: "fleet_server",
	}

	nonfleetPackage = kibanaPackage{
		Name: "some_other_package",
	}
)

func generatePackagePolicies(fleetedPolicyIDs []string, nonfleetedPolicyIDs []string) *kibanaPackagePolicies {
	items := []kibanaPackagePolicy{}
	for _, ID := range fleetedPolicyIDs {
		items = append(items, kibanaPackagePolicy{
			PolicyID: ID,
			Package:  fleetPackage,
		})
	}
	for _, ID := range nonfleetedPolicyIDs {
		items = append(items, kibanaPackagePolicy{
			PolicyID: ID,
			Package:  nonfleetPackage,
		})
	}
	return &kibanaPackagePolicies{
		Items: items,
	}
}

func reverse(policies *kibanaPackagePolicies) {
	for i, j := 0, len(policies.Items)-1; i < j; i, j = i+1, j-1 {
		policies.Items[i], policies.Items[j] = policies.Items[j], policies.Items[i]
	}
}

func TestSeparatePackagePolicies(t *testing.T) {
	policies := generatePackagePolicies([]string{"fleeted-id"}, []string{"nonfleeted-id"})
	response := separatePackagePolicies(policies)
	require.Contains(t, response.Fleet, "fleeted-id")
	require.Contains(t, response.NonFleet, "nonfleeted-id")
}

func TestSeparatePackagePoliciesFleetPrecedence(t *testing.T) {
	policies := generatePackagePolicies([]string{"fleeted-id", "multipackage"}, []string{"multipackage"})
	response := separatePackagePolicies(policies)
	require.Contains(t, response.Fleet, "fleeted-id")
	require.Contains(t, response.Fleet, "multipackage")
	require.NotContains(t, response.NonFleet, "multipackage")
}

func TestSeparatePackagePoliciesConflictingNonFleetPackagesFirst(t *testing.T) {
	policies := generatePackagePolicies([]string{"fleeted-id", "multipackage"}, []string{"multipackage"})
	reverse(policies)
	response := separatePackagePolicies(policies)
	require.Contains(t, response.Fleet, "fleeted-id")
	require.Contains(t, response.Fleet, "multipackage")
	require.NotContains(t, response.NonFleet, "multipackage")
}

func TestSeparatePackagePoliciesNonFleetPackagesFirst(t *testing.T) {
	policies := generatePackagePolicies([]string{"fleeted-id"}, []string{"nonfleeted-id"})
	reverse(policies)
	response := separatePackagePolicies(policies)
	require.Contains(t, response.Fleet, "fleeted-id")
	require.Contains(t, response.NonFleet, "nonfleeted-id")
}

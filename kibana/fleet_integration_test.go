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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustGetEnv(t *testing.T) ClientConfig {
	kibUser, foundUser := os.LookupEnv("KIBANA_USERNAME")
	kibPass, foundPass := os.LookupEnv("KIBANA_PASSWORD")
	kibHost, foundHost := os.LookupEnv("KIBANA_HOST")

	if !foundPass || !foundUser || !foundHost {
		t.Skip("test requires a Kibana instance with KIBANA_USERNAME, KIBANA_PASSWORD and KIBANA_HOST set")
	}

	cfg := ClientConfig{
		Host:     kibHost,
		Username: kibUser,
		Password: kibPass,
	}

	return cfg
}

/*
These are a series of integration tests that run some of the fleet API clients against an actual elastic stack.
Just set the env vars listed in mustGetEnv().
*/

func TestGetPolicyKibana(t *testing.T) {
	cfg := mustGetEnv(t)
	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	testPolicy := AgentPolicy{
		Name:        "TestGetPolicyKibana",
		Namespace:   "defaultttest",
		Description: "original policy",
	}

	respPolicy, err := client.CreatePolicy(testPolicy)
	require.NoError(t, err)
	t.Logf("Created policy for test with ID %s", respPolicy.ID)

	respGet, err := client.GetPolicy(respPolicy.ID)
	require.NoError(t, err)

	require.Equal(t, respPolicy.Description, respGet.Description)

	err = client.DeletePolicy(respPolicy.ID)
	require.NoError(t, err)
}

func TestCreatePolicyKibana(t *testing.T) {
	cfg := mustGetEnv(t)

	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	testPolicy := AgentPolicy{
		Name:      "TestCreatePolicyKibana",
		Namespace: "defaulttest",
	}

	respPolicy, err := client.CreatePolicy(testPolicy)
	t.Logf("created policy with ID %s", respPolicy.ID)
	require.NoError(t, err)
	require.Equal(t, testPolicy.Name, respPolicy.Name)
	require.Equal(t, testPolicy.Namespace, respPolicy.Namespace)
	require.NotEmpty(t, respPolicy.ID)
	// delete policy
	err = client.DeletePolicy(respPolicy.ID)
	require.NoError(t, err)
}

func TestUpdatePolicyKibana(t *testing.T) {
	cfg := mustGetEnv(t)
	client, err := NewClientWithConfig(&cfg, "", "", "", "")
	require.NoError(t, err)

	testPolicy := AgentPolicy{
		Name:        "TestUpdatePolicyKibana",
		Namespace:   "defaultttest",
		Description: "original policy",
	}
	respPolicy, err := client.CreatePolicy(testPolicy)
	require.NoError(t, err)
	t.Logf("Created policy for test with ID %s", respPolicy.ID)
	require.Empty(t, respPolicy.MonitoringEnabled)

	updatePolicy := AgentPolicyUpdateRequest{
		Name:              testPolicy.Name,
		Namespace:         testPolicy.Namespace,
		MonitoringEnabled: []MonitoringEnabledOption{MonitoringEnabledMetrics},
	}

	updateResp, err := client.UpdatePolicy(respPolicy.ID, updatePolicy)
	require.NoError(t, err)
	// make sure our update was applied
	require.Equal(t, []MonitoringEnabledOption{MonitoringEnabledMetrics}, updateResp.MonitoringEnabled)
	// make sure we didn't somehow change something else
	require.Equal(t, respPolicy.InactivityTImeout, updateResp.InactivityTImeout)
	require.Equal(t, respPolicy.Description, updateResp.Description)
	err = client.DeletePolicy(respPolicy.ID)
	require.NoError(t, err)
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/kibana"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/go-elasticsearch/v8"
)

func TestFQDN(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: FQDN,
		OS: []define.OS{
			{Type: define.Linux},
		},
		Stack: &define.Stack{},
		Local: false,
		Sudo:  true,
	})

	agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	externalIP, err := getExternalIP()
	require.NoError(t, err)

	// Save original /etc/hosts so we can restore it at the end of each test
	origEtcHosts, err := getEtcHosts()
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Save original hostname so we can restore it at the end of each test
	origHostname, err := getHostname(ctx)
	require.NoError(t, err)

	kibClient := info.KibanaClient

	shortName := strings.ToLower(randStr(6))
	fqdn := shortName + ".baz.io"
	t.Logf("Set FQDN on host to %s", fqdn)
	err = setHostFQDN(ctx, origEtcHosts, externalIP, fqdn, t.Log)
	require.NoError(t, err)

	t.Log("Enroll agent in Fleet with a test policy")
	createPolicyReq := kibana.AgentPolicy{
		Name:        "test-policy-fqdn-" + strings.ReplaceAll(fqdn, ".", "-"),
		Namespace:   info.Namespace,
		Description: fmt.Sprintf("Test policy for FQDN E2E test (%s)", fqdn),
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
		AgentFeatures: []map[string]interface{}{
			{
				"name":    "fqdn",
				"enabled": false,
			},
		},
	}
	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
	}
	policy, err := tools.InstallAgentWithPolicy(ctx, t, installOpts, agentFixture, kibClient, createPolicyReq)
	require.NoError(t, err)

	t.Cleanup(func() {
		// Use a separate context as the one in the test body will have been cancelled at this point.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()

		t.Log("Un-enrolling Elastic Agent...")
		assert.NoError(t, fleettools.UnEnrollAgent(cleanupCtx, info.KibanaClient, policy.ID))

		t.Log("Restoring hostname...")
		err := setHostname(cleanupCtx, origHostname, t.Log)
		require.NoError(t, err)

		t.Log("Restoring original /etc/hosts...")
		err = setEtcHosts(origEtcHosts)
		require.NoError(t, err)
	})

	t.Log("Verify that agent name is short hostname")
	agent := verifyAgentName(ctx, t, policy.ID, shortName, info.KibanaClient)

	t.Log("Verify that hostname in `logs-*` and `metrics-*` is short hostname")
	verifyHostNameInIndices(t, "logs-*", shortName, info.Namespace, info.ESClient)
	verifyHostNameInIndices(t, "metrics-*", shortName, info.Namespace, info.ESClient)

	t.Log("Update Agent policy to enable FQDN")
	policy.AgentFeatures = []map[string]interface{}{
		{
			"name":    "fqdn",
			"enabled": true,
		},
	}
	updatePolicyReq := kibana.AgentPolicyUpdateRequest{
		Name:          policy.Name,
		Namespace:     info.Namespace,
		AgentFeatures: policy.AgentFeatures,
	}
	_, err = kibClient.UpdatePolicy(ctx, policy.ID, updatePolicyReq)
	require.NoError(t, err)

	t.Log("Wait until policy has been applied by Agent")
	expectedAgentPolicyRevision := agent.PolicyRevision + 1
	require.Eventually(
		t,
		tools.IsPolicyRevision(ctx, t, kibClient, agent.ID, expectedAgentPolicyRevision),
		2*time.Minute,
		1*time.Second,
	)

	t.Log("Verify that agent name is FQDN")
	verifyAgentName(ctx, t, policy.ID, fqdn, info.KibanaClient)

	t.Log("Verify that hostname in `logs-*` and `metrics-*` is FQDN")
	verifyHostNameInIndices(t, "logs-*", fqdn, info.Namespace, info.ESClient)
	verifyHostNameInIndices(t, "metrics-*", fqdn, info.Namespace, info.ESClient)

	t.Log("Update Agent policy to disable FQDN")
	policy.AgentFeatures = []map[string]interface{}{
		{
			"name":    "fqdn",
			"enabled": false,
		},
	}
	updatePolicyReq = kibana.AgentPolicyUpdateRequest{
		Name:          policy.Name,
		Namespace:     info.Namespace,
		AgentFeatures: policy.AgentFeatures,
	}
	_, err = kibClient.UpdatePolicy(ctx, policy.ID, updatePolicyReq)
	require.NoError(t, err)

	t.Log("Wait until policy has been applied by Agent")
	expectedAgentPolicyRevision++
	require.Eventually(
		t,
		tools.IsPolicyRevision(ctx, t, kibClient, agent.ID, expectedAgentPolicyRevision),
		2*time.Minute,
		1*time.Second,
	)

	t.Log("Verify that agent name is short hostname again")
	verifyAgentName(ctx, t, policy.ID, shortName, info.KibanaClient)

	// TODO: Re-enable assertion once https://github.com/elastic/elastic-agent/issues/3078 is
	// investigated for root cause and resolved.
	// t.Log("Verify that hostname in `logs-*` and `metrics-*` is short hostname again")
	// verifyHostNameInIndices(t, "logs-*", shortName, info.ESClient)
	// verifyHostNameInIndices(t, "metrics-*", shortName, info.ESClient)
}

func verifyAgentName(ctx context.Context, t *testing.T, policyID, hostname string, kibClient *kibana.Client) *kibana.AgentExisting {
	t.Helper()

	var agent *kibana.AgentExisting
	var err error

	require.Eventually(
		t,
		func() bool {
			agent, err = fleettools.GetAgentByPolicyIDAndHostnameFromList(ctx, kibClient, policyID, hostname)
			return err == nil && agent != nil
		},
		5*time.Minute,
		5*time.Second,
	)

	return agent
}

func verifyHostNameInIndices(t *testing.T, indices, hostname, namespace string, esClient *elasticsearch.Client) {
	queryRaw := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"host.name": map[string]interface{}{
								"value": hostname,
							},
						},
					},
					{
						"term": map[string]interface{}{
							"data_stream.namespace": map[string]interface{}{
								"value": namespace,
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(queryRaw)
	require.NoError(t, err)

	search := esClient.Search

	require.Eventually(
		t,
		func() bool {
			resp, err := search(
				search.WithIndex(indices),
				search.WithSort("@timestamp:desc"),
				search.WithFilterPath("hits.hits"),
				search.WithSize(1),
				search.WithBody(&buf),
			)
			require.NoError(t, err)
			require.False(t, resp.IsError())
			defer resp.Body.Close()

			var body struct {
				Hits struct {
					Hits []struct {
						Source struct {
							Host struct {
								Name string `json:"name"`
							} `json:"host"`
						} `json:"_source"`
					} `json:"hits"`
				} `json:"hits"`
			}
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&body)
			require.NoError(t, err)

			return len(body.Hits.Hits) == 1
		},
		2*time.Minute,
		5*time.Second,
	)
}

func getHostname(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "hostname")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func getEtcHosts() ([]byte, error) {
	filename := string(filepath.Separator) + filepath.Join("etc", "hosts")
	return os.ReadFile(filename)
}

func setHostFQDN(ctx context.Context, etcHosts []byte, externalIP, fqdn string, log func(args ...any)) error {
	filename := string(filepath.Separator) + filepath.Join("etc", "hosts")

	// Add entry for FQDN in /etc/hosts
	parts := strings.Split(fqdn, ".")
	shortName := parts[0]
	line := fmt.Sprintf("%s\t%s %s\n", externalIP, fqdn, shortName)

	etcHosts = append(etcHosts, []byte(line)...)
	err := os.WriteFile(filename, etcHosts, 0o644)
	if err != nil {
		return err
	}

	// Set hostname to FQDN
	cmd := exec.CommandContext(ctx, "hostname", shortName)
	output, err := cmd.Output()
	if err != nil {
		log(string(output))
	}

	return err
}

func setEtcHosts(data []byte) error {
	filename := string(filepath.Separator) + filepath.Join("etc", "hosts")
	return os.WriteFile(filename, data, 0o644)
}

func setHostname(ctx context.Context, hostname string, log func(args ...any)) error {
	cmd := exec.CommandContext(ctx, "hostname", hostname)
	output, err := cmd.Output()
	if err != nil {
		log(string(output))
	}
	return err
}

func getExternalIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

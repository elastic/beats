// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/elastic/elastic-agent-libs/kibana"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/estools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
)

type EndpointMetricsMonRunner struct {
	suite.Suite
	info       *define.Info
	fixture    *atesting.Fixture
	endpointID string
}

func TestEndpointAgentServiceMonitoring(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Fleet,
		Stack: &define.Stack{},
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
		OS: []define.OS{
			{Type: define.Linux},
		},
	})

	// Get path to agent executable.
	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err, "could not create agent fixture")

	runner := &EndpointMetricsMonRunner{
		info:       info,
		fixture:    fixture,
		endpointID: "endpoint-default",
	}

	suite.Run(t, runner)
}

func (runner *EndpointMetricsMonRunner) SetupSuite() {
	deadline := time.Now().Add(10 * time.Minute)
	ctx, cancel := testcontext.WithDeadline(runner.T(), context.Background(), deadline)
	defer cancel()

	runner.T().Log("Enrolling the agent in Fleet")
	policyUUID := uuid.Must(uuid.NewV4()).String()

	createPolicyReq := kibana.AgentPolicy{
		Name:        "test-policy-" + policyUUID,
		Namespace:   "default",
		Description: "Test policy " + policyUUID,
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
	}

	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
		Privileged:     true,
	}

	policy, err := tools.InstallAgentWithPolicy(ctx, runner.T(),
		installOpts, runner.fixture, runner.info.KibanaClient, createPolicyReq)
	require.NoError(runner.T(), err, "failed to install agent with policy")

	runner.T().Log("Installing Elastic Defend")
	pkgPolicyResp, err := installElasticDefendPackage(runner.T(), runner.info, policy.ID)
	require.NoErrorf(runner.T(), err, "Policy Response was: %v", pkgPolicyResp)

	runner.T().Log("Polling for endpoint-security to become Healthy")
	ctx, cancel = context.WithTimeout(ctx, time.Minute*3)
	defer cancel()

	agentClient := runner.fixture.Client()
	err = agentClient.Connect(ctx)
	require.NoError(runner.T(), err, "could not connect to local agent")

	require.Eventually(runner.T(),
		func() bool { return agentAndEndpointAreHealthy(runner.T(), ctx, agentClient) },
		time.Minute*3,
		time.Second,
		"Endpoint component or units are not healthy.",
	)

}

func (runner *EndpointMetricsMonRunner) TestEndpointMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*15)
	defer cancel()

	agentStatus, err := runner.fixture.ExecStatus(ctx)
	require.NoError(runner.T(), err)

	require.Eventually(runner.T(), func() bool {

		query := genESQueryByBinary(agentStatus.Info.ID, runner.endpointID)
		res, err := estools.PerformQueryForRawQuery(ctx, query, "metrics-elastic_agent*", runner.info.ESClient)
		require.NoError(runner.T(), err)
		runner.T().Logf("Fetched metrics for %s, got %d hits", runner.endpointID, res.Hits.Total.Value)
		return res.Hits.Total.Value >= 1
	}, time.Minute*10, time.Second*10, "could not fetch component metricsets for endpoint with ID %s and agent ID %s", runner.endpointID, agentStatus.Info.ID)

}

func (runner *EndpointMetricsMonRunner) TestEndpointMetricsAfterRestart() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*15)
	defer cancel()
	// once we've gotten the first round of metrics,forcably restart endpoint, see if we still get metrics
	// This makes sure that the backend coordinator can deal with properly updating the metrics handlers if there's unexpected state changes

	// confine this to linux; the behavior is platform-agnostic, and this way we have `pgrep`
	if runtime.GOOS != "linux" {
		return
	}

	// kill endpoint
	cmd := exec.Command("pgrep", "-f", "endpoint")
	pgrep, err := cmd.CombinedOutput()
	runner.T().Logf("killing pid: %s", string(pgrep))

	cmd = exec.Command("pkill", "--signal", "SIGKILL", "-f", "endpoint")
	_, err = cmd.CombinedOutput()
	require.NoError(runner.T(), err)

	// wait for endpoint to come back up. We use `pgrep`
	// since the agent health status won't imidately register that the endpoint process itself is gone.
	require.Eventually(runner.T(), func() bool {
		cmd := exec.Command("pgrep", "-f", "endpoint")
		pgrep, err := cmd.CombinedOutput()
		runner.T().Logf("found pid: %s", string(pgrep))
		if err == nil {
			return true
		}
		return false
	}, time.Minute*2, time.Second)

	// make sure agent still says we're healthy
	agentClient := runner.fixture.Client()
	err = agentClient.Connect(ctx)
	require.NoError(runner.T(), err, "could not connect to local agent")

	require.Eventually(runner.T(),
		func() bool { return agentAndEndpointAreHealthy(runner.T(), ctx, agentClient) },
		time.Minute*3,
		time.Second,
		"Endpoint component or units are not healthy.",
	)

	// catch the time endpoint is restarted, so we can filter for documents after a given time
	endpointRestarted := time.Now()

	agentStatus, err := runner.fixture.ExecStatus(ctx)
	require.NoError(runner.T(), err)

	// now query again, but make sure we're getting new metrics
	require.Eventually(runner.T(), func() bool {
		query := genESQueryByDate(agentStatus.Info.ID, runner.endpointID, endpointRestarted.Format(time.RFC3339))
		res, err := estools.PerformQueryForRawQuery(ctx, query, "metrics-elastic_agent*", runner.info.ESClient)
		require.NoError(runner.T(), err)
		runner.T().Logf("Fetched metrics for %s, got %d hits", runner.endpointID, res.Hits.Total.Value)
		return res.Hits.Total.Value >= 1
	}, time.Minute*10, time.Second*10, "could not fetch component metricsets for endpoint with ID %s and agent ID %s", runner.endpointID, agentStatus.Info.ID)
}

func genESQueryByDate(agentID string, componentID string, dateAfter string) map[string]interface{} {
	queryRaw := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"agent.id": agentID,
						},
					},
					{
						"match": map[string]interface{}{
							"component.id": componentID,
						},
					},
					{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte": dateAfter,
							},
						},
					},
					{
						"range": map[string]interface{}{
							"system.process.cpu.total.value": map[string]interface{}{
								"gt": 0,
							},
						},
					},
					{
						"range": map[string]interface{}{
							"system.process.memory.size": map[string]interface{}{
								"gt": 0,
							},
						},
					},
				},
			},
		},
	}

	return queryRaw
}

func genESQueryByBinary(agentID string, componentID string) map[string]interface{} {
	// see https://github.com/elastic/kibana/blob/main/x-pack/plugins/fleet/server/services/agents/agent_metrics.ts
	queryRaw := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"agent.id": agentID,
						},
					},
					{
						"match": map[string]interface{}{
							"component.id": componentID,
						},
					},
					{
						"range": map[string]interface{}{
							"system.process.cpu.total.value": map[string]interface{}{
								"gt": 0,
							},
						},
					},
					{
						"range": map[string]interface{}{
							"system.process.memory.size": map[string]interface{}{
								"gt": 0,
							},
						},
					},
				},
			},
		},
	}

	return queryRaw
}

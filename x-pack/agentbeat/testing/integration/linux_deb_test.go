// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/elastic-agent-libs/kibana"

	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/check"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/testing/upgradetest"

	"github.com/stretchr/testify/require"
)

func TestDebLogIngestFleetManaged(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Deb,
		Stack: &define.Stack{},
		OS: []define.OS{
			{
				Type:   define.Linux,
				Distro: "ubuntu",
			},
		},
		Local: false,
		Sudo:  true,
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), atesting.WithPackageFormat("deb"))
	require.NoError(t, err)

	// 1. Create a policy in Fleet with monitoring enabled.
	// To ensure there are no conflicts with previous test runs against
	// the same ESS stack, we add the current time at the end of the policy
	// name. This policy does not contain any integration.
	t.Log("Enrolling agent in Fleet with a test policy")
	createPolicyReq := kibana.AgentPolicy{
		Name:        fmt.Sprintf("test-policy-enroll-%s", uuid.Must(uuid.NewV4()).String()),
		Namespace:   info.Namespace,
		Description: "test policy for agent enrollment",
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
		AgentFeatures: []map[string]interface{}{
			{
				"name":    "test_enroll",
				"enabled": true,
			},
		},
	}

	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
	}

	// 2. Install the Elastic-Agent with the policy that
	// was just created.
	policy, err := tools.InstallAgentWithPolicy(
		ctx,
		t,
		installOpts,
		agentFixture,
		info.KibanaClient,
		createPolicyReq)
	require.NoError(t, err)
	t.Logf("created policy: %s", policy.ID)
	check.ConnectedToFleet(ctx, t, agentFixture, 5*time.Minute)

	t.Run("Monitoring logs are shipped", func(t *testing.T) {
		testMonitoringLogsAreShipped(t, ctx, info, agentFixture, policy)
	})

	t.Run("Normal logs with flattened data_stream are shipped", func(t *testing.T) {
		testFlattenedDatastreamFleetPolicy(t, ctx, info, policy)
	})
}

func TestDebFleetUpgrade(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Deb,
		Stack: &define.Stack{},
		OS: []define.OS{
			{
				Type:   define.Linux,
				Distro: "ubuntu",
			},
		},
		Local: false,
		Sudo:  true,
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// start from previous minor
	upgradeFromVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	startFixture, err := atesting.NewFixture(
		t,
		upgradeFromVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
		atesting.WithPackageFormat("deb"),
	)
	require.NoError(t, err)

	// end on the current build with deb
	endFixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), atesting.WithPackageFormat("deb"))
	require.NoError(t, err)

	// 1. Create a policy in Fleet with monitoring enabled.
	// To ensure there are no conflicts with previous test runs against
	// the same ESS stack, we add the current time at the end of the policy
	// name. This policy does not contain any integration.
	t.Log("Enrolling agent in Fleet with a test policy")
	createPolicyReq := kibana.AgentPolicy{
		Name:        fmt.Sprintf("test-policy-enroll-%s", uuid.Must(uuid.NewV4()).String()),
		Namespace:   info.Namespace,
		Description: "test policy for agent enrollment",
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
		AgentFeatures: []map[string]interface{}{
			{
				"name":    "test_enroll",
				"enabled": true,
			},
		},
	}

	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
	}

	// 2. Install the Elastic-Agent with the policy that
	// was just created.
	policy, err := tools.InstallAgentWithPolicy(
		ctx,
		t,
		installOpts,
		startFixture,
		info.KibanaClient,
		createPolicyReq)
	require.NoError(t, err)
	t.Logf("created policy: %s", policy.ID)
	check.ConnectedToFleet(ctx, t, startFixture, 5*time.Minute)

	// 3. Upgrade deb to the build version
	srcPackage, err := endFixture.SrcPackage(ctx)
	require.NoError(t, err)
	cmd := exec.CommandContext(ctx, "sudo", "apt-get", "install", "-y", "-qq", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", srcPackage)
	cmd.Env = append(cmd.Env, "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput() // #nosec G204 -- Need to pass in name of package
	require.NoError(t, err, string(out))

	// 4. Wait for version in Fleet to match
	// Fleet will not include the `-SNAPSHOT` in the `GetAgentVersion` result
	noSnapshotVersion := strings.TrimSuffix(define.Version(), "-SNAPSHOT")
	require.Eventually(t, func() bool {
		newVersion, err := fleettools.GetAgentVersion(ctx, info.KibanaClient, policy.ID)
		if err != nil {
			t.Logf("error getting agent version: %v", err)
			return false
		}
		if noSnapshotVersion == newVersion {
			return true
		}
		t.Logf("Got Agent version %s != %s", newVersion, noSnapshotVersion)
		return false
	}, 5*time.Minute, time.Second)
}

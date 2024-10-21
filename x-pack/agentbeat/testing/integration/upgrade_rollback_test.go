// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kardianos/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/internal/pkg/agent/application/paths"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/upgrade/details"
	"github.com/elastic/elastic-agent/internal/pkg/agent/install"
	"github.com/elastic/elastic-agent/pkg/control/v2/cproto"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/pkg/version"
	"github.com/elastic/elastic-agent/testing/upgradetest"
)

const reallyFastWatcherCfg = `
agent.upgrade.watcher:
  grace_period: 1m
  error_check.interval: 5s
`

// TestStandaloneUpgradeRollback tests the scenario where upgrading to a new version
// of Agent fails due to the new Agent binary reporting an unhealthy status. It checks
// that the Agent is rolled back to the previous version.
func TestStandaloneUpgradeRollback(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Upgrade from an old build because the new watcher from the new build will
	// be ran. Otherwise the test will run the old watcher from the old build.
	upgradeFromVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	startFixture, err := atesting.NewFixture(
		t,
		upgradeFromVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err)
	startVersionInfo, err := startFixture.ExecVersion(ctx)
	require.NoError(t, err, "failed to get start agent build version info")

	// Upgrade to the build under test.
	endFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	t.Logf("Testing Elastic Agent upgrade from %s to %s...", upgradeFromVersion, define.Version())

	// We need to use the core version in the condition below because -SNAPSHOT is
	// stripped from the ${agent.version.version} evaluation below.
	endVersion, err := version.ParseVersion(define.Version())
	require.NoError(t, err)

	// Configure Agent with fast watcher configuration and also an invalid
	// input when the Agent version matches the upgraded Agent version. This way
	// the pre-upgrade version of the Agent runs healthy, but the post-upgrade
	// version doesn't.
	preInstallHook := func() error {
		invalidInputPolicy := upgradetest.FastWatcherCfg + fmt.Sprintf(`
outputs:
  default:
    type: elasticsearch
    hosts: [127.0.0.1:9200]

inputs:
  - condition: '${agent.version.version} == "%s"'
    type: invalid
    id: invalid-input
`, endVersion.CoreVersion())
		return startFixture.Configure(ctx, []byte(invalidInputPolicy))
	}

	// Use the post-upgrade hook to bypass the remainder of the PerformUpgrade
	// because we want to do our own checks for the rollback.
	var ErrPostExit = errors.New("post exit")
	postUpgradeHook := func() error {
		return ErrPostExit
	}

	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t,
		upgradetest.WithPreInstallHook(preInstallHook),
		upgradetest.WithPostUpgradeHook(postUpgradeHook))
	if !errors.Is(err, ErrPostExit) {
		require.NoError(t, err)
	}

	// rollback should now occur

	// wait for the agent to be healthy and back at the start version
	err = upgradetest.WaitHealthyAndVersion(ctx, startFixture, startVersionInfo.Binary, 10*time.Minute, 10*time.Second, t)
	if err != nil {
		// agent never got healthy, but we need to ensure the watcher is stopped before continuing (this
		// prevents this test failure from interfering with another test)
		// this kills the watcher instantly and waits for it to be gone before continuing
		watcherErr := upgradetest.WaitForNoWatcher(ctx, 1*time.Minute, time.Second, 100*time.Millisecond)
		if watcherErr != nil {
			t.Logf("failed to kill watcher due to agent not becoming healthy: %s", watcherErr)
		}
	}
	require.NoError(t, err)

	// ensure that upgrade details now show the state as UPG_ROLLBACK. This is only possible with Elastic
	// Agent versions >= 8.12.0.
	startVersion, err := version.ParseVersion(startVersionInfo.Binary.Version)
	require.NoError(t, err)

	if !startVersion.Less(*version.NewParsedSemVer(8, 12, 0, "", "")) {
		client := startFixture.Client()
		err = client.Connect(ctx)
		require.NoError(t, err)

		state, err := client.State(ctx)
		require.NoError(t, err)

		if state.State == cproto.State_UPGRADING {
			if state.UpgradeDetails == nil {
				t.Fatal("upgrade details in the state cannot be nil")
			}
			require.Equal(t, details.StateRollback, details.State(state.UpgradeDetails.State))
		} else {
			t.Logf("rollback finished, status is '%s', cannot check UpgradeDetails", state.State.String())
		}
	}

	// rollback should stop the watcher
	// killTimeout is greater than timeout as the watcher should have been
	// stopped on its own, and we don't want this test to hide that fact
	err = upgradetest.WaitForNoWatcher(ctx, 2*time.Minute, 10*time.Second, 3*time.Minute)
	require.NoError(t, err)

	// now that the watcher has stopped lets ensure that it's still the expected
	// version, otherwise it's possible that it was rolled back to the original version
	err = upgradetest.CheckHealthyAndVersion(ctx, startFixture, startVersionInfo.Binary)
	assert.NoError(t, err)
}

// TestStandaloneUpgradeRollbackOnRestarts tests the scenario where upgrading to a new version
// of Agent fails due to the new Agent binary not starting up. It checks that the Agent is
// rolled back to the previous version.
func TestStandaloneUpgradeRollbackOnRestarts(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Upgrade from an old build because the new watcher from the new build will
	// be ran. Otherwise the test will run the old watcher from the old build.
	upgradeFromVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	startFixture, err := atesting.NewFixture(
		t,
		upgradeFromVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err)
	startVersionInfo, err := startFixture.ExecVersion(ctx)
	require.NoError(t, err, "failed to get start agent build version info")

	// Upgrade to the build under test.
	endFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	t.Logf("Testing Elastic Agent upgrade from %s to %s...", upgradeFromVersion, define.Version())

	// Use the post-upgrade hook to bypass the remainder of the PerformUpgrade
	// because we want to do our own checks for the rollback.
	var ErrPostExit = errors.New("post exit")
	postUpgradeHook := func() error {
		return ErrPostExit
	}

	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t,
		upgradetest.WithPostUpgradeHook(postUpgradeHook),
		upgradetest.WithCustomWatcherConfig(reallyFastWatcherCfg))
	if !errors.Is(err, ErrPostExit) {
		require.NoError(t, err)
	}

	// A few seconds after the upgrade, deliberately restart upgraded Agent a
	// couple of times to simulate Agent crashing.
	for restartIdx := 0; restartIdx < 3; restartIdx++ {
		time.Sleep(10 * time.Second)
		topPath := paths.Top()

		t.Logf("Stopping agent via service to simulate crashing")
		err = install.StopService(topPath, install.DefaultStopTimeout, install.DefaultStopInterval)
		if err != nil && runtime.GOOS == define.Windows && strings.Contains(err.Error(), "The service has not been started.") {
			// Due to the quick restarts every 10 seconds its possible that this is faster than Windows
			// can handle. Decrementing restartIdx means that the loop will occur again.
			t.Logf("Got an allowed error on Windows: %s", err)
			err = nil
		}
		require.NoError(t, err)

		// ensure that it's stopped before starting it again
		var status service.Status
		var statusErr error
		require.Eventuallyf(t, func() bool {
			status, statusErr = install.StatusService(topPath)
			if statusErr != nil {
				return false
			}
			return status != service.StatusRunning
		}, 2*time.Minute, 1*time.Second, "service never fully stopped (status: %v): %s", status, statusErr)
		t.Logf("Stopped agent via service to simulate crashing")

		// start it again
		t.Logf("Starting agent via service to simulate crashing")
		err = install.StartService(topPath)
		require.NoError(t, err)

		// ensure that it's started before next loop
		require.Eventuallyf(t, func() bool {
			status, statusErr = install.StatusService(topPath)
			if statusErr != nil {
				return false
			}
			return status == service.StatusRunning
		}, 2*time.Minute, 1*time.Second, "service never fully started (status: %v): %s", status, statusErr)
		t.Logf("Started agent via service to simulate crashing")
	}

	// wait for the agent to be healthy and back at the start version
	err = upgradetest.WaitHealthyAndVersion(ctx, startFixture, startVersionInfo.Binary, 10*time.Minute, 10*time.Second, t)
	if err != nil {
		// agent never got healthy, but we need to ensure the watcher is stopped before continuing
		// this kills the watcher instantly and waits for it to be gone before continuing
		watcherErr := upgradetest.WaitForNoWatcher(ctx, 1*time.Minute, time.Second, 100*time.Millisecond)
		if watcherErr != nil {
			t.Logf("failed to kill watcher due to agent not becoming healthy: %s", watcherErr)
		}
	}
	require.NoError(t, err)

	// ensure that upgrade details now show the state as UPG_ROLLBACK. This is only possible with Elastic
	// Agent versions >= 8.12.0.
	startVersion, err := version.ParseVersion(startVersionInfo.Binary.Version)
	require.NoError(t, err)

	if !startVersion.Less(*version.NewParsedSemVer(8, 12, 0, "", "")) {
		client := startFixture.Client()
		err = client.Connect(ctx)
		require.NoError(t, err)

		state, err := client.State(ctx)
		require.NoError(t, err)

		require.NotNil(t, state.UpgradeDetails)
		require.Equal(t, details.StateRollback, details.State(state.UpgradeDetails.State))
	}

	// rollback should stop the watcher
	// killTimeout is greater than timeout as the watcher should have been
	// stopped on its own, and we don't want this test to hide that fact
	err = upgradetest.WaitForNoWatcher(ctx, 2*time.Minute, 10*time.Second, 3*time.Minute)
	require.NoError(t, err)

	// now that the watcher has stopped lets ensure that it's still the expected
	// version, otherwise it's possible that it was rolled back to the original version
	err = upgradetest.CheckHealthyAndVersion(ctx, startFixture, startVersionInfo.Binary)
	assert.NoError(t, err)
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/pkg/version"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/testing/upgradetest"
)

func TestStandaloneUpgradeUninstallKillWatcher(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	currentVersion, err := version.ParseVersion(define.Version())
	require.NoError(t, err)
	if currentVersion.Less(*upgradetest.Version_8_11_0_SNAPSHOT) {
		t.Skipf("Version %s is lower than min version %s; test cannot be performed", define.Version(), upgradetest.Version_8_11_0_SNAPSHOT)
	}

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Upgrades to build under test.
	endFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)
	endVersionInfo, err := endFixture.ExecVersion(ctx)
	require.NoError(t, err, "failed to get end agent build version info")

	// Start on a snapshot build, we want this test to upgrade to our
	// build to ensure that the uninstall will kill the watcher.
	// We need a version with a non-matching commit hash to perform the upgrade
	startVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	startFixture, err := atesting.NewFixture(
		t,
		startVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err)

	// Use the post-upgrade hook to bypass the remainder of the PerformUpgrade
	// because we want to do our own checks for the rollback.
	var ErrPostExit = errors.New("post exit")
	postUpgradeHook := func() error {
		return ErrPostExit
	}

	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t, upgradetest.WithPostUpgradeHook(postUpgradeHook))
	if !errors.Is(err, ErrPostExit) {
		require.NoError(t, err)
	}

	// wait for the agent to be healthy and at the new version
	err = upgradetest.WaitHealthyAndVersion(ctx, startFixture, endVersionInfo.Binary, 10*time.Minute, 10*time.Second, t)
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

	// watcher needs to start before uninstall, otherwise you can
	// get a race condition where watcher hasn't started before
	// uninstall does it's PID check to find the watcher.
	watcherErr := upgradetest.WaitForWatcher(ctx, 1*time.Minute, time.Second)
	if watcherErr != nil {
		t.Logf("watcher failed to start: %s", watcherErr)
	}

	// call uninstall now, do not wait for the watcher to finish running
	// 8.11+ should always kill the running watcher (if it doesn't uninstall will fail)
	uninstallCtx, uninstallCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer uninstallCancel()
	output, err := startFixture.Uninstall(uninstallCtx, &atesting.UninstallOpts{Force: true})
	assert.NoError(t, err, "uninstall failed with output:\n%s", string(output))
}

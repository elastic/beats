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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/testing/upgradetest"
)

// TestStandaloneUpgradeFailsWhenUpgradeIsInProgress initiates an upgrade for a
// standalone Elastic Agent and, while that upgrade is still in progress, attempts
// to initiate a second upgrade. The test expects Elastic Agent to not allow
// the second upgrade.
func TestStandaloneUpgradeFailsWhenUpgradeIsInProgress(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// For this test we start with a version of Agent that's two minors older
	// than the current version and upgrade to the current version. Then we attempt
	// upgrading to the current version again, expecting Elastic Agent to disallow
	// this second upgrade.
	upgradeFromVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	startFixture, err := atesting.NewFixture(
		t,
		upgradeFromVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err, "error creating previous agent fixture")

	endFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)
	endVersionInfo, err := endFixture.ExecVersion(ctx)
	require.NoError(t, err)

	// Use the post-upgrade hook to bypass the remainder of the PerformUpgrade
	// because we want to do our own checks for the rollback.
	var ErrPostExit = errors.New("post exit")
	postUpgradeHook := func() error {
		return ErrPostExit
	}

	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t,
		upgradetest.WithPostUpgradeHook(postUpgradeHook))
	if !errors.Is(err, ErrPostExit) {
		require.NoError(t, err)
	}

	// upgrade has occurred and the watcher is running

	// wait for the agent to be healthy and correct version
	// before trying to perform another upgrade
	err = upgradetest.WaitHealthyAndVersion(ctx, startFixture, endVersionInfo.Binary, 2*time.Minute, 10*time.Second, t)
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

	// trigger another upgrade that should fail because the current watcher
	// is still running and another upgrade can be performed until it is complete
	upgradeCmdArgs := []string{"upgrade", endVersionInfo.Binary.String()}
	upgradeOutput, err := startFixture.Exec(ctx, upgradeCmdArgs)
	require.NotNil(t, err)
	require.Contains(t, string(upgradeOutput), "an upgrade is already in progress; please try again later.")

	// killTimeout is greater than timeout as the watcher should have been
	// stopped on its own, and we don't want this test to hide that fact
	err = upgradetest.WaitForNoWatcher(ctx, 2*time.Minute, 10*time.Second, 3*time.Minute)
	require.NoError(t, err)

	// now that the watcher has stopped lets ensure that it's still the expected
	// version, otherwise it's possible that it was rolled back to the original version
	err = upgradetest.CheckHealthyAndVersion(ctx, startFixture, endVersionInfo.Binary)
	assert.NoError(t, err)
}

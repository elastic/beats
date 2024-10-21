// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/internal/pkg/release"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/pkg/version"
	"github.com/elastic/elastic-agent/testing/upgradetest"
)

func TestStandaloneUpgradeWithGPGFallback(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	minVersion := upgradetest.Version_8_10_0_SNAPSHOT
	currentVersion, err := version.ParseVersion(define.Version())
	require.NoError(t, err)

	if currentVersion.Less(*minVersion) {
		t.Skipf("Version %s is lower than min version %s", define.Version(), minVersion)
	}

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Start at the build version as we want to test the retry
	// logic that is in the build.
	startFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)
	startVersionInfo, err := startFixture.ExecVersion(ctx)
	require.NoError(t, err)

	// Upgrade to an old build of the same version.
	endFixture, err := atesting.NewFixture(
		t,
		upgradetest.EnsureSnapshot(define.Version()),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err)

	endVersionInfo, err := endFixture.ExecVersion(ctx)
	require.NoError(t, err)
	if startVersionInfo.Binary.String() == endVersionInfo.Binary.String() &&
		startVersionInfo.Binary.Commit == endVersionInfo.Binary.Commit {
		t.Skipf("Build under test is the same as the build from the artifacts repository (version: %s) [commit: %s]",
			startVersionInfo.Binary.String(), startVersionInfo.Binary.Commit)
	}

	t.Logf("Testing Elastic Agent upgrade from %s to %s...", define.Version(), endVersionInfo.Binary.String())

	defaultPGP := release.PGP()
	firstSeven := string(defaultPGP[:7])
	newPgp := strings.Replace(
		string(defaultPGP),
		firstSeven,
		"abcDEFg",
		1,
	)

	customPGP := upgradetest.CustomPGP{
		PGP: newPgp,
	}

	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t,
		// passing "" as source URI is a hack to disable the --source-uri argument pointing at the endFixture srcPackage location
		// this test needs the agent to download the real thing from artifacts.elastic.co so empty string.
		// We need to download the same file from the same url and  use that as end fixture
		// or we need a way to disable the commit hash check (in this case the upgrade can be verified just with the
		// version string)
		upgradetest.WithSourceURI(""),
		upgradetest.WithCustomPGP(customPGP),
		upgradetest.WithSkipVerify(false))
	require.NoError(t, err, "perform upgrade failed")
}

func TestStandaloneUpgradeWithGPGFallbackOneRemoteFailing(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	minVersion := upgradetest.Version_8_10_0_SNAPSHOT
	currentVersion, err := version.ParseVersion(define.Version())
	require.NoError(t, err)

	if currentVersion.Less(*minVersion) {
		t.Skipf("Version %s is lower than min version %s", define.Version(), minVersion)
	}

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Start at the build version as we want to test the retry
	// logic that is in the build.
	startFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	// Upgrade to an old build.
	// This is probably a way of getting a signed package
	upgradeToVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	var fetcher atesting.Fetcher

	// FIXME: this is a hack, PreviousMinor() uses a version.ParsedSemVer internally and that's what we should use for the snapshot check
	// We need to distinguish between snapshots and released versions and use the appropriate fetcher
	if upgradeToVersion.IsSnapshot() {
		// it's a snapshot, use the artifact fetcher
		fetcher = atesting.ArtifactFetcher()
	} else {
		// it's a released version, use httpFetcher
		// this fetcher will literally pull the package from the default elastic agent download URL
		fetcher = atesting.NewHttpFetcher()
	}

	endFixture, err := atesting.NewFixture(
		t,
		upgradeToVersion.String(),
		atesting.WithFetcher(fetcher),
	)
	require.NoError(t, err)

	t.Logf("Testing Elastic Agent upgrade from %s to %s...", define.Version(), upgradeToVersion)

	defaultPGP := release.PGP()
	firstSeven := string(defaultPGP[:7])
	newPgp := strings.Replace(
		string(defaultPGP),
		firstSeven,
		"abcDEFg",
		1,
	)

	customPGP := upgradetest.CustomPGP{
		PGP:    newPgp,
		PGPUri: "https://127.0.0.1:3456/non/existing/path",
	}

	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t,
		// passing "" as source URI is a hack to disable the --source-uri argument pointing at the endFixture srcPackage location
		// this test needs the agent to download the real thing from artifacts.elastic.co so empty string.
		// We need to download the same file from the same url and  use that as end fixture
		// or we need a way to disable the commit hash check (in this case the upgrade can be verified just with the
		// version string)
		upgradetest.WithSourceURI(""),
		upgradetest.WithCustomPGP(customPGP),
		upgradetest.WithSkipVerify(false))
	require.NoError(t, err, "perform upgrade failed")
}

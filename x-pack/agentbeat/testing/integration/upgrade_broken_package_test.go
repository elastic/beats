// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/testing/upgradetest"
	agtversion "github.com/elastic/elastic-agent/version"
)

func TestUpgradeBrokenPackageVersion(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	t.Skip("This test cannot succeed with a AGENT_PACKAGE_VERSION override. Check contents of .buildkite/scripts/steps/beats_tests.sh")

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Start at the build version as we want to test the retry
	// logic that is in the build.
	startFixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), atesting.WithAdditionalArgs([]string{"-E", "output.elasticsearch.allow_older_versions=true"}))
	require.NoError(t, err)

	// Upgrade to an old build.
	upgradeToVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	endFixture, err := atesting.NewFixture(
		t,
		upgradeToVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err)

	// Pre-upgrade remove the package version files.
	preUpgradeHook := func() error {
		// get rid of the package version files in the installed directory
		return removePackageVersionFiles(t, startFixture)
	}

	t.Logf("Testing Elastic Agent upgrade from %s to %s...", define.Version(), upgradeToVersion)

	err = upgradetest.PerformUpgrade(ctx, startFixture, endFixture, t, upgradetest.WithPreUpgradeHook(preUpgradeHook))
	assert.NoError(t, err)
}

func removePackageVersionFiles(t *testing.T, f *atesting.Fixture) error {
	installFS := os.DirFS(f.WorkDir())
	matches := []string{}

	err := fs.WalkDir(installFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Name() == agtversion.PackageVersionFileName {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	t.Logf("package version files found: %v", matches)

	// the version files should have been removed from the other test, we just make sure
	for _, m := range matches {
		vFile := filepath.Join(f.WorkDir(), m)
		t.Logf("removing package version file %q", vFile)
		err = os.Remove(vFile)
		if err != nil {
			return fmt.Errorf("error removing package version file %q: %w", vFile, err)
		}
	}
	return nil
}

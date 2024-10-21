// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/internal/pkg/agent/application/paths"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/testing/installtest"
)

func TestInstallWithoutBasePath(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		// We require sudo for this test to run
		// `elastic-agent install` (even though it will
		// be installed as non-root).
		Sudo: true,

		// It's not safe to run this test locally as it
		// installs Elastic Agent.
		Local: false,
	})

	// Get path to Elastic Agent executable
	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Prepare the Elastic Agent so the binary is extracted and ready to use.
	err = fixture.Prepare(ctx)
	require.NoError(t, err)

	// Run `elastic-agent install`.  We use `--force` to prevent interactive
	// execution.
	opts := atesting.InstallOpts{Force: true, Privileged: false}
	out, err := fixture.Install(ctx, &opts)
	if err != nil {
		t.Logf("install output: %s", out)
		require.NoError(t, err)
	}

	// Check that Agent was installed in default base path
	topPath := installtest.DefaultTopPath()
	require.NoError(t, installtest.CheckSuccess(ctx, fixture, topPath, &installtest.CheckOpts{Privileged: opts.Privileged}))

	t.Run("check agent package version", testAgentPackageVersion(ctx, fixture, true))
	t.Run("check second agent installs with --develop", testSecondAgentCanInstall(ctx, fixture, "", true, opts))

	// Make sure uninstall from within the topPath fails on Windows
	if runtime.GOOS == "windows" {
		cwd, err := os.Getwd()
		require.NoErrorf(t, err, "GetWd failed: %s", err)
		err = os.Chdir(topPath)
		require.NoErrorf(t, err, "Chdir to topPath failed: %s", err)
		t.Cleanup(func() {
			_ = os.Chdir(cwd)
		})
		out, err = fixture.Uninstall(ctx, &atesting.UninstallOpts{Force: true})
		require.Error(t, err, "uninstall should have failed")
		require.Containsf(t, string(out), "uninstall must be run from outside the installed path", "expected error string not found in: %s err: %s", out, err)
	}
}

func TestInstallWithBasePath(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		// We require sudo for this test to run
		// `elastic-agent install` (even though it will
		// be installed as non-root).
		Sudo: true,

		// It's not safe to run this test locally as it
		// installs Elastic Agent.
		Local: false,
	})

	// Get path to Elastic Agent executable
	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Prepare the Elastic Agent so the binary is extracted and ready to use.
	err = fixture.Prepare(ctx)
	require.NoError(t, err)

	// When installing with unprivileged using a base path the
	// base needs to be accessible by the `elastic-agent-user` user that will be
	// executing the process, but is not created yet. Using a base that exists
	// and is known to be accessible by standard users, ensures this tests
	// works correctly and will not hit a permission issue when spawning the
	// elastic-agent service.
	var basePath string
	switch runtime.GOOS {
	case define.Linux:
		basePath = `/usr`
	case define.Windows:
		basePath = `C:\`
	default:
		// Set up random temporary directory to serve as base path for Elastic Agent
		// installation.
		tmpDir := t.TempDir()
		basePath = filepath.Join(tmpDir, strings.ToLower(randStr(8)))
	}

	// Run `elastic-agent install`.  We use `--force` to prevent interactive
	// execution.
	opts := atesting.InstallOpts{
		BasePath:   basePath,
		Force:      true,
		Privileged: false,
	}
	out, err := fixture.Install(ctx, &opts)
	if err != nil {
		t.Logf("install output: %s", out)
		require.NoError(t, err)
	}

	// Check that Agent was installed in the custom base path
	topPath := filepath.Join(basePath, "Elastic", "Agent")
	require.NoError(t, installtest.CheckSuccess(ctx, fixture, topPath, &installtest.CheckOpts{Privileged: opts.Privileged}))

	t.Run("check agent package version", testAgentPackageVersion(ctx, fixture, true))
	t.Run("check second agent installs with --namespace", testSecondAgentCanInstall(ctx, fixture, basePath, false, opts))

	// Make sure uninstall from within the topPath fails on Windows
	if runtime.GOOS == "windows" {
		cwd, err := os.Getwd()
		require.NoErrorf(t, err, "GetWd failed: %s", err)
		err = os.Chdir(topPath)
		require.NoErrorf(t, err, "Chdir to topPath failed: %s", err)
		t.Cleanup(func() {
			_ = os.Chdir(cwd)
		})
		out, err = fixture.Uninstall(ctx, &atesting.UninstallOpts{Force: true})
		require.Error(t, err, "uninstall should have failed")
		require.Containsf(t, string(out), "uninstall must be run from outside the installed path", "expected error string not found in: %s err: %s", out, err)
	}
}

func TestInstallPrivilegedWithoutBasePath(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		// We require sudo for this test to run
		// `elastic-agent install`.
		Sudo: true,

		// It's not safe to run this test locally as it
		// installs Elastic Agent.
		Local: false,
	})

	// Get path to Elastic Agent executable
	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Prepare the Elastic Agent so the binary is extracted and ready to use.
	err = fixture.Prepare(ctx)
	require.NoError(t, err)

	// Run `elastic-agent install`.  We use `--force` to prevent interactive
	// execution.
	opts := atesting.InstallOpts{Force: true, Privileged: true}
	out, err := fixture.Install(ctx, &opts)
	if err != nil {
		t.Logf("install output: %s", out)
		require.NoError(t, err)
	}

	// Check that Agent was installed in default base path
	require.NoError(t, installtest.CheckSuccess(ctx, fixture, opts.BasePath, &installtest.CheckOpts{Privileged: opts.Privileged}))

	t.Run("check agent package version", testAgentPackageVersion(ctx, fixture, true))
	t.Run("check second agent installs with --namespace", testSecondAgentCanInstall(ctx, fixture, "", false, opts))
}

func TestInstallPrivilegedWithBasePath(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		// We require sudo for this test to run
		// `elastic-agent install`.
		Sudo: true,

		// It's not safe to run this test locally as it
		// installs Elastic Agent.
		Local: false,
	})

	// Get path to Elastic Agent executable
	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Prepare the Elastic Agent so the binary is extracted and ready to use.
	err = fixture.Prepare(ctx)
	require.NoError(t, err)

	// Set up random temporary directory to serve as base path for Elastic Agent
	// installation.
	tmpDir := t.TempDir()
	randomBasePath := filepath.Join(tmpDir, strings.ToLower(randStr(8)))

	// Run `elastic-agent install`.  We use `--force` to prevent interactive
	// execution.
	opts := atesting.InstallOpts{
		BasePath:   randomBasePath,
		Force:      true,
		Privileged: true,
	}
	out, err := fixture.Install(ctx, &opts)
	if err != nil {
		t.Logf("install output: %s", out)
		require.NoError(t, err)
	}

	// Check that Agent was installed in the custom base path
	topPath := filepath.Join(randomBasePath, "Elastic", "Agent")
	require.NoError(t, installtest.CheckSuccess(ctx, fixture, topPath, &installtest.CheckOpts{Privileged: opts.Privileged}))
	t.Run("check agent package version", testAgentPackageVersion(ctx, fixture, true))
	t.Run("check second agent installs with --develop", testSecondAgentCanInstall(ctx, fixture, randomBasePath, true, opts))
}

// Tests that a second agent can be installed in an isolated namespace, using either --develop or --namespace.
func testSecondAgentCanInstall(ctx context.Context, fixture *atesting.Fixture, basePath string, develop bool, installOpts atesting.InstallOpts) func(*testing.T) {
	return func(t *testing.T) {
		// Get path to Elastic Agent executable
		devFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
		require.NoError(t, err)

		// Prepare the Elastic Agent so the binary is extracted and ready to use.
		err = devFixture.Prepare(ctx)
		require.NoError(t, err)

		// If development mode was requested, the namespace will be automatically set to Development after Install().
		// Otherwise, install into a test namespace.
		installOpts.Develop = develop
		if !installOpts.Develop {
			installOpts.Namespace = "Testing"
		}

		devOut, err := devFixture.Install(ctx, &installOpts)
		if err != nil {
			t.Logf("install output: %s", devOut)
			require.NoError(t, err)
		}

		topPath := installtest.NamespaceTopPath(installOpts.Namespace)
		if basePath != "" {
			topPath = filepath.Join(basePath, "Elastic", paths.InstallDirNameForNamespace(installOpts.Namespace))
		}

		require.NoError(t, installtest.CheckSuccess(ctx, fixture, topPath, &installtest.CheckOpts{
			Privileged: installOpts.Privileged,
			Namespace:  installOpts.Namespace,
		}))
	}
}

// TestInstallUninstallAudit will test to make sure that a fleet-managed agent can use the audit/unenroll endpoint when uninstalling itself.
func TestInstallUninstallAudit(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Default,
		Stack: &define.Stack{}, // needs a fleet-server.
		Sudo:  true,
		Local: false,
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	policyResp, enrollmentTokenResp := createPolicyAndEnrollmentToken(ctx, t, info.KibanaClient, createBasicPolicy())
	t.Logf("Created policy %+v", policyResp.AgentPolicy)

	t.Log("Getting default Fleet Server URL...")
	fleetServerURL, err := fleettools.DefaultURL(ctx, info.KibanaClient)
	require.NoError(t, err, "failed getting Fleet Server URL")

	err = fixture.Prepare(ctx)
	require.NoError(t, err)
	// Run `elastic-agent install`.  We use `--force` to prevent interactive
	// execution.
	opts := &atesting.InstallOpts{
		Force: true,
		EnrollOpts: atesting.EnrollOpts{
			URL:             fleetServerURL,
			EnrollmentToken: enrollmentTokenResp.APIKey,
		},
	}
	out, err := fixture.Install(ctx, opts)
	if err != nil {
		t.Logf("install output: %s", out)
		require.NoError(t, err)
	}

	require.Eventuallyf(t, func() bool {
		return waitForAgentAndFleetHealthy(ctx, t, fixture)
	}, time.Minute, time.Second, "agent never became healthy or connected to Fleet")

	agentID, err := getAgentID(ctx, fixture)
	require.NoError(t, err, "error getting the agent inspect output")
	require.NotEmpty(t, agentID, "agent ID empty")

	out, err = fixture.Uninstall(ctx, &atesting.UninstallOpts{Force: true})
	if err != nil {
		t.Logf("uninstall output: %s", out)
		require.NoError(t, err)
	}

	// TODO: replace direct query to ES index with API call to Fleet
	// Blocked on https://github.com/elastic/kibana/issues/194884
	response, err := info.ESClient.Get(".fleet-agents", agentID, info.ESClient.Get.WithContext(ctx))
	require.NoError(t, err)
	defer response.Body.Close()
	p, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, response.StatusCode, "ES status code expected 200, body: %s", p)
	var res struct {
		Source struct {
			AuditUnenrolledReason string `json:"audit_unenrolled_reason"`
		} `json:"_source"`
	}
	err = json.Unmarshal(p, &res)
	require.NoError(t, err)
	require.Equal(t, "uninstall", res.Source.AuditUnenrolledReason)
}

// TestRepeatedInstallUninstall will install then uninstall the agent
// repeatedly.  This test exists because of a number of race
// conditions that have occurred in the uninstall process.  Current
// testing shows each iteration takes around 16 seconds.
func TestRepeatedInstallUninstall(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		// We require sudo for this test to run
		// `elastic-agent install` (even though it will
		// be installed as non-root).
		Sudo: true,

		// It's not safe to run this test locally as it
		// installs Elastic Agent.
		Local: false,
	})

	maxRunTime := 2 * time.Minute
	iterations := 100
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("%s-%d", t.Name(), i), func(t *testing.T) {

			// Get path to Elastic Agent executable
			fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
			require.NoError(t, err)

			ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(maxRunTime))
			defer cancel()

			// Prepare the Elastic Agent so the binary is extracted and ready to use.
			err = fixture.Prepare(ctx)
			require.NoError(t, err)

			// Run `elastic-agent install`.  We use `--force` to prevent interactive
			// execution.
			opts := &atesting.InstallOpts{Force: true}
			out, err := fixture.Install(ctx, opts)
			if err != nil {
				t.Logf("install output: %s", out)
				require.NoError(t, err)
			}

			// Check that Agent was installed in default base path
			require.NoError(t, installtest.CheckSuccess(ctx, fixture, opts.BasePath, &installtest.CheckOpts{Privileged: opts.Privileged}))
			t.Run("check agent package version", testAgentPackageVersion(ctx, fixture, true))
			out, err = fixture.Uninstall(ctx, &atesting.UninstallOpts{Force: true})
			require.NoErrorf(t, err, "uninstall failed: %s", err)
		})
	}
}

func randStr(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	runes := make([]rune, length)
	for i := range runes {
		runes[i] = letters[rand.IntN(len(letters))]
	}

	return string(runes)
}

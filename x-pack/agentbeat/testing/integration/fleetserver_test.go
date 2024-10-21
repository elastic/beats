// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/kibana"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/estools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
)

func fleetPolicy() kibana.AgentPolicy {
	policyUUID := uuid.Must(uuid.NewV4()).String()

	return kibana.AgentPolicy{
		ID:          "test-fleet-policy-" + policyUUID,
		Name:        "test-fleet-policy-" + policyUUID,
		Namespace:   "default",
		Description: "Test fleet policy " + policyUUID,
	}
}

func TestInstallFleetServerBootstrap(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: FleetPrivileged,
		Stack: &define.Stack{},
		Sudo:  true,
		Local: false,
	})

	t.Skip("Skip until the first 8.16.0-SNAPSHOT is available")

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Get path to Elastic Agent executable
	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), atesting.WithAdditionalArgs([]string{"-E", "output.elasticsearch.allow_older_versions=true"}))
	require.NoError(t, err)
	err = fixture.Prepare(ctx)
	require.NoError(t, err)

	t.Log("Ensure base path is clean")
	var defaultBasePath string
	switch runtime.GOOS {
	case "darwin":
		defaultBasePath = `/Library`
	case "linux":
		defaultBasePath = `/opt`
	case "windows":
		defaultBasePath = `C:\Program Files`
	}

	topPath := filepath.Join(defaultBasePath, "Elastic", "Agent")
	err = os.RemoveAll(topPath)
	require.NoError(t, err, "failed to remove %q. The test requires this path not to exist.")

	t.Log("Create fleet-server policy...")
	policyResp, err := info.KibanaClient.CreatePolicy(ctx, fleetPolicy())
	require.NoError(t, err, "failed creating policy")
	policy := policyResp.AgentPolicy
	_, err = tools.InstallPackageFromDefaultFile(ctx, info.KibanaClient, "fleet-server", "1.5.0", "fleet-server.json", uuid.Must(uuid.NewV4()).String(), policy.ID)
	require.NoError(t, err, "failed creating fleet-server integration")

	t.Log("Get fleet-server service token...")
	serviceToken, err := estools.CreateServiceToken(ctx, info.ESClient, "fleet-server")
	require.NoError(t, err, "failed creating service token")

	esHost, ok := os.LookupEnv("ELASTICSEARCH_HOST")
	require.True(t, ok, "environment var ELASTICSEARCH_HOST is empty")
	u, err := url.Parse(esHost)
	require.NoError(t, err, "could not parse %q as a URL", esHost)
	if u.Port() == "" {
		switch u.Scheme {
		case "":
			u.Host += ":80"
			u.Scheme = "http"
		case "http":
			u.Host += ":80"
		case "https":
			u.Host += ":443"
		default:
			require.Failf(t, "elasticsearch host has unknown scheme: %s", u.Scheme)
		}
		esHost = u.String()
	}

	t.Logf("fleet-server will enroll with es host: %q", esHost)

	// Run `elastic-agent install` with fleet-server bootstrap options.
	// We use `--force` to prevent interactive execution.
	opts := &atesting.InstallOpts{
		Force:      true,
		Privileged: true,
		FleetBootstrapOpts: atesting.FleetBootstrapOpts{
			ESHost:       esHost,
			ServiceToken: serviceToken,
			Policy:       policy.ID,
			Port:         8220,
		},
	}
	out, err := fixture.Install(ctx, opts)
	if err != nil {
		t.Logf("Install output: %s", out)
		require.NoError(t, err, "unable to install elastic-agent with fleet-server bootstrap options")
	}

	// checkInstallSuccess(t, fixture, topPath, true) // FIXME fails to build if this is uncommented, but the method is part of install_test.go
	t.Run("check agent package version", testAgentPackageVersion(ctx, fixture, true))

	// elastic-agent will self sign a cert to use with fleet-server if one is not passed
	// in order to interact with the API we need to ignore the cert.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	fleetOK := false
	for i := 0; i < 10; i++ {
		t.Log("Checking fleet-server status")
		resp, err := client.Get("https://localhost:8220/api/status")
		if err != nil {
			t.Logf("fleet-server status check returned error: %v, retry in 10s...", err)
			time.Sleep(10 * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			fleetOK = true
			break
		}
		t.Logf("fleet-server status check returned incorrect status: %d, retry in 10s", resp.StatusCode)
		time.Sleep(10 * time.Second)
		continue
	}
	require.True(t, fleetOK, "expected fleet-server /api/status to return 200")

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

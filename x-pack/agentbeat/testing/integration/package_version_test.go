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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent/pkg/control/v2/client"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/version"
)

func TestPackageVersion(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: true,
	})

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = f.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	t.Run("check package version without the agent running", testAgentPackageVersion(ctx, f, true))

	// run the agent and check the daemon version as well
	t.Run("check package version while the agent is running", testVersionWithRunningAgent(ctx, f))

	// Destructive/mutating tests ahead! If you need to do a normal test on a healthy installation of agent, put it before the tests below

	// change the version in the version file and verify that the agent returns the new value
	t.Run("check package version after updating file", testVersionAfterUpdatingFile(ctx, f))

	// remove the pkg version file and check that we return the default beats version
	t.Run("remove package versions file and test version again", testAfterRemovingPkgVersionFiles(ctx, f))
}

func TestComponentBuildHashInDiagnostics(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Fleet,
		Stack: &define.Stack{},
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})
	ctx := context.Background()

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err, "could not create new fixture")

	err = f.Prepare(ctx)
	require.NoError(t, err, "could not prepare fixture")

	enrollParams, err := fleettools.NewEnrollParams(ctx, info.KibanaClient)
	require.NoError(t, err, "failed preparing Agent enrollment")

	t.Log("Installing Elastic Agent...")
	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
		EnrollOpts: atesting.EnrollOpts{
			URL:             enrollParams.FleetURL,
			EnrollmentToken: enrollParams.EnrollmentToken,
		},
	}
	output, err := f.Install(ctx, &installOpts)
	require.NoError(t, err,
		"failed to install start agent [output: %s]", string(output))

	stateBuff := bytes.Buffer{}
	var status atesting.AgentStatusOutput
	allHealthy := func() bool {
		stateBuff.Reset()

		status, err = f.ExecStatus(ctx)
		if err != nil {
			stateBuff.WriteString(fmt.Sprintf("failed to get agent status: %v",
				err))
			return false
		}

		if client.State(status.State) != client.Healthy {
			stateBuff.WriteString(fmt.Sprintf(
				"agent isn't healthy: %s-%s",
				client.State(status.State), status.Message))
			return false
		}

		if len(status.Components) == 0 {
			stateBuff.WriteString(fmt.Sprintf(
				"healthy but without components: agent status: %s-%s",
				client.State(status.State), status.Message))
			return false
		}

		// the agent might be healthy but waiting its first configuration,
		// in that case, there would be no components yet. Therefore, ensure
		// the agent received the policy with components before proceeding with
		// the test.
		for _, c := range status.Components {
			bs, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				stateBuff.WriteString(fmt.Sprintf(
					"%s not healthy, could not marshal status outptu: %v",
					c.Name, err))
				return false
			}

			state := client.State(c.State)
			if state != client.Healthy {
				stateBuff.WriteString(fmt.Sprintf(
					"%s not health, agent status output: %s",
					c.Name, bs))
				return false
			}

			// there is a rare a race condition unlike to happen on a
			// production scenario where the component is healthy but the
			// version info delays to update. As the Status command and the
			// diagnostics fetch this information in the same way, it guarantees
			// the version info is up-to-date before proceeding with the test.
			if c.VersionInfo.Meta.Commit == "" {
				stateBuff.WriteString(fmt.Sprintf(
					"%s health, but no versionInfo. agent status output: %s",
					c.Name, bs))
				return false
			}
		}

		return true
	}
	require.Eventuallyf(t,
		allHealthy,
		5*time.Minute, 10*time.Second,
		"agent never became healthy. Last status: %v", &stateBuff)
	defer func() {
		if !t.Failed() {
			return
		}

		t.Logf("test failed: last status output: %#v", status)
	}()

	agentbeat := "agentbeat"
	if runtime.GOOS == "windows" {
		agentbeat += ".exe"
	}
	wd := f.WorkDir()
	glob := filepath.Join(wd, "data", "elastic-agent-*", "components", agentbeat)
	compPaths, err := filepath.Glob(glob)
	require.NoErrorf(t, err, "failed to glob agentbeat path pattern %q", glob)
	require.Lenf(t, compPaths, 1,
		"glob pattern \"%s\": found %d paths to agentbeat, can only have 1",
		glob, len(compPaths))

	cmdVer := exec.Command(compPaths[0], "filebeat", "version")
	output, err = cmdVer.CombinedOutput()
	require.NoError(t, err, "failed to get filebeat version")
	outStr := string(output)

	// version output example:
	// filebeat version 8.14.0 (arm64), libbeat 8.14.0 [ab27a657e4f15976c181cf44c529bba6159f2c64 built 2024-04-17 18:13:16 +0000 UTC]
	t.Log("parsing commit hash from filebeat version: ", outStr)
	splits := strings.Split(outStr, "[")
	require.Lenf(t, splits, 2,
		"expected beats output version to be split into 2, it was split into %q",
		strings.Join(splits, "|"))
	splits = strings.Split(splits[1], " built")
	require.Lenf(t, splits, 2,
		"expected split of beats output version to be split into 2, it was split into %q",
		strings.Join(splits, "|"))
	wantBuildHash := splits[0]

	diagZip, err := f.ExecDiagnostics(ctx)
	require.NoError(t, err, "failed collecting diagnostics")

	diag := t.TempDir()
	extractZipArchive(t, diagZip, diag)
	// if the test fails, the diagnostics used is useful for debugging.
	defer func() {
		if !t.Failed() {
			return
		}

		t.Logf("the test failed: trying to save the diagnostics used on the test")
		diagDir, err := f.DiagnosticsDir()
		if err != nil {
			t.Logf("could not get diagnostics directory to save the diagnostics used on the test")
			return
		}

		err = os.Rename(diagZip, filepath.Join(diagDir,
			fmt.Sprintf("TestComponentBuildHashInDiagnostics-used-diag-%d.zip",
				time.Now().Unix())))
		if err != nil {
			t.Logf("could not move diagnostics used in the test to %s: %v",
				diagDir, err)
			return
		}
	}()

	stateFilePath := filepath.Join(diag, "state.yaml")
	stateYAML, err := os.Open(stateFilePath)
	require.NoError(t, err, "could not open diagnostics state.yaml")
	defer func(stateYAML *os.File) {
		err := stateYAML.Close()
		assert.NoErrorf(t, err, "error closing %q", stateFilePath)
	}(stateYAML)

	state := struct {
		Components []struct {
			ID    string `yaml:"id"`
			State struct {
				VersionInfo struct {
					BuildHash string `yaml:"build_hash"`
					Meta      struct {
						BuildTime string `yaml:"build_time"`
						Commit    string `yaml:"commit"`
					} `yaml:"meta"`
					Name string `yaml:"name"`
				} `yaml:"version_info"`
			} `yaml:"state"`
		} `yaml:"components"`
	}{}
	err = yaml.NewDecoder(stateYAML).Decode(&state)
	require.NoError(t, err, "could not parse state.yaml (%s)", stateYAML.Name())

	for _, c := range state.Components {
		assert.Equalf(t, wantBuildHash, c.State.VersionInfo.BuildHash,
			"component %s: VersionInfo.BuildHash mismatch", c.ID)
		assert.Equalf(t, wantBuildHash, c.State.VersionInfo.Meta.Commit,
			"component %s: VersionInfo.Meta.Commit mismatch", c.ID)
	}

	if t.Failed() {
		_, seek := stateYAML.Seek(0, 0)
		if seek != nil {
			t.Logf("could not reset state.yaml offset to print it")
			return
		}
		data, err := io.ReadAll(stateYAML)
		if err != nil {
			t.Logf("could not read state.yaml: %v", err)
		}
		t.Logf("test failed: state.yaml contents: %q", string(data))
	}
}

func testVersionWithRunningAgent(runCtx context.Context, f *atesting.Fixture) func(*testing.T) {

	return func(t *testing.T) {

		testf := func(ctx context.Context) error {
			testAgentPackageVersion(ctx, f, false)
			return nil
		}

		runAgentWithAfterTest(runCtx, f, t, testf)
	}
}

func testVersionAfterUpdatingFile(runCtx context.Context, f *atesting.Fixture) func(*testing.T) {

	return func(t *testing.T) {
		pkgVersionFiles := findPkgVersionFiles(t, f.WorkDir())

		testVersion := "1.2.3-test-abcdef"

		for _, pkgVerFile := range pkgVersionFiles {
			err := os.WriteFile(pkgVerFile, []byte(testVersion), 0o644)
			require.NoError(t, err)
		}

		testf := func(ctx context.Context) error {
			testAgentPackageVersion(ctx, f, false)
			return nil
		}

		runAgentWithAfterTest(runCtx, f, t, testf)
	}
}

func testAfterRemovingPkgVersionFiles(runCtx context.Context, f *atesting.Fixture) func(*testing.T) {
	return func(t *testing.T) {
		matches := findPkgVersionFiles(t, f.WorkDir())

		for _, m := range matches {
			t.Logf("removing package version file %q", m)
			err := os.Remove(m)
			require.NoErrorf(t, err, "error removing package version file %q", m)
		}
		testf := func(ctx context.Context) error {
			// check the version returned by the running agent
			stdout, stderr, processState := getAgentVersionOutput(t, f, ctx, false)

			binaryActualVersion := unmarshalVersionOutput(t, stdout, "binary")
			assert.Equal(t, version.GetDefaultVersion(), binaryActualVersion, "binary version does not return default beat version when the package version file is missing")
			daemonActualVersion := unmarshalVersionOutput(t, stdout, "daemon")
			assert.Equal(t, version.GetDefaultVersion(), daemonActualVersion, "daemon version does not return default beat version when the package version file is missing")
			assert.True(t, processState.Success(), "elastic agent version command should be successful even if the pkg version is not found")

			assert.Contains(t, string(stderr), "Error initializing version information")

			return nil
		}

		runAgentWithAfterTest(runCtx, f, t, testf)
	}

}

func runAgentWithAfterTest(runCtx context.Context, f *atesting.Fixture, t *testing.T, testf func(ctx context.Context) error) {

	err := f.Run(runCtx, atesting.State{
		AgentState: atesting.NewClientState(client.Healthy),
		// we don't really need a config and a state but the testing fwk wants it anyway
		Configure: simpleConfig2,
		Components: map[string]atesting.ComponentState{
			"fake-default": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					{UnitType: client.UnitTypeOutput, UnitID: "fake-default"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					{UnitType: client.UnitTypeInput, UnitID: "fake-default-fake"}: {
						State: atesting.NewClientState(client.Healthy),
					},
				},
			},
		},
		After: testf,
	})

	require.NoError(t, err)

}

type StateComponentVersion struct {
	Components []struct {
		ID    string `yaml:"id"`
		State struct {
			VersionInfo struct {
				Meta struct {
					BuildTime string `yaml:"build_time"`
					Commit    string `yaml:"commit"`
				} `yaml:"meta"`
				Name string `yaml:"name"`
			} `yaml:"version_info"`
		} `yaml:"state"`
	} `yaml:"components"`
}

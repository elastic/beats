// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && linux

package synthexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/security"
)

// TestSyntheticsWorkdirIntegration requires a CLI with --workdir support and its
// Playwright browser installed. SYNTHETICS_WORKDIR_CLI can select a development CLI.
func TestSyntheticsWorkdirIntegration(t *testing.T) {
	if os.Geteuid() == 0 {
		original := security.NodeChildProcCred
		security.NodeChildProcCred = &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{0}}
		t.Cleanup(func() { security.NodeChildProcCred = original })
	}
	cli := os.Getenv("SYNTHETICS_WORKDIR_CLI")
	if cli == "" {
		var err error
		cli, err = exec.LookPath("elastic-synthetics")
		require.NoError(t, err, "install a synthetics CLI supporting --workdir")
	}
	cli, err := filepath.EvalSymlinks(cli)
	require.NoError(t, err, "resolve CLI installation")
	cli, err = filepath.Abs(cli)
	require.NoError(t, err, "resolve absolute CLI path")
	for _, mode := range []string{"inline", "inline-api", "project", "api-project"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			tempRoot := t.TempDir()
			// The root-container case must let the restricted child traverse the test
			// fixtures and create Playwright's own temporary browser profiles.
			if os.Geteuid() == 0 {
				for _, dir := range []string{filepath.Dir(root), root} {
					require.NoError(t, os.Chmod(dir, 0755), "allow restricted child to read fixtures")
				}
				require.NoError(t, os.Chmod(tempRoot, os.ModeSticky|0777), "allow restricted child to create browser profiles")
			}
			t.Setenv("TMPDIR", tempRoot)
			stale := filepath.Join(root, ".synthetics")
			cacheMode := os.FileMode(0500)
			if os.Geteuid() == 0 {
				cacheMode = 0750
			}
			require.NoError(t, os.Mkdir(stale, cacheMode), "simulate unwritable Agent cache")
			t.Cleanup(func() { _ = os.Chmod(stale, 0700) })
			require.NoError(t, os.WriteFile(filepath.Join(root, "synthetics.config.js"), []byte(`module.exports = { params: { marker: 'original-cwd' } };`), 0644), "write config discovered from original cwd")
			script := `step('local page', async () => { await page.setContent('<h1>isolated scratch directory</h1>'); });`
			if mode == "inline-api" {
				script = `step('local API step', async () => { if (2 + 2 !== 4) throw new Error('unexpected arithmetic'); });`
			}
			if mode == "project" || mode == "api-project" {
				require.NoError(t, os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"workdir-test","private":true}`), 0644), "create project manifest")
				binDir := filepath.Join(root, "node_modules", ".bin")
				require.NoError(t, os.MkdirAll(binDir, 0755), "create project binary directory")
				require.NoError(t, os.Symlink(cli, filepath.Join(binDir, "elastic-synthetics")), "use selected CLI for project runs")
				kind, body := "journey", "async () => { await page.setContent('<h1>project screenshot</h1>'); }"
				if mode == "api-project" {
					kind, body = "apiJourney", "async () => { if (2 + 2 !== 4) throw new Error('unexpected arithmetic'); }"
				}
				source := fmt.Sprintf("const { %s, step } = require(%q); %s('scratch isolation', ({ page, params }) => { if (params.marker !== 'original-cwd') throw new Error('config not found'); step('local step', %s); });", kind, filepath.Join(filepath.Dir(cli), "index.js"), kind, body)
				//nolint:gosec // Fixture contents refer to the explicitly selected CLI; the destination is t.TempDir.
				require.NoError(t, os.WriteFile(filepath.Join(root, "test.journey.js"), []byte(source), 0644), "write local test journey")
			}
			// Confirm the fixture reproduces the original permissions failure.
			if mode == "inline" {
				//nolint:gosec // The test explicitly selects the CLI under test.
				baseline := &SynthCmd{exec.CommandContext(t.Context(), cli, "--inline", "--inline-api")}
				baseline.Dir = root
				baseline.Stdin = strings.NewReader(`step('baseline', async () => {});`)
				platformCmdMutate(baseline)
				output, err := baseline.CombinedOutput()
				require.Error(t, err, "default cache must fail in the Agent permission fixture")
				require.Contains(t, string(output), "EACCES", "fixture must reproduce the filesystem collision")
			}
			// Run twice against the same inaccessible cache to exercise restart/reuse.
			for run := range 2 {
				t.Logf("%s run %d", mode, run+1)
				var cmd *SynthCmd
				var stdin *string
				if mode == "inline" || mode == "inline-api" {
					//nolint:gosec,noctx // The test explicitly selects the CLI; runCmd owns cancellation.
					cmd = &SynthCmd{exec.Command(cli, "--inline", "--screenshots", "on", "--workdir="+stale)}
					if mode == "inline-api" {
						cmd.Args = append(cmd.Args, "--inline-api")
					}
					cmd.Dir = root
					stdin = &script
				} else {
					factory, err := projectCommandFactory(root, "--screenshots", "on", "--workdir="+stale)
					require.NoError(t, err, "resolve project CLI without changing its cwd")
					cmd = factory()
				}
				ctx := context.WithValue(context.Background(), SynthexecTimeoutKey, time.Minute)
				mpx, err := runCmd(ctx, cmd, stdin, nil, FilterJourneyConfig{}, stdfields.StdMonitorFields{}, "")
				require.NoError(t, err, "launch actual synthetics CLI")
				events := drainWorkdirEvents(t, mpx)
				for _, event := range events {
					assert.Nil(t, event.Error, "synthetics event %s must succeed: %+v", event.Type, event)
					if event.Type == Stderr {
						t.Logf("synthetics stderr: %v", event.Payload)
					}
				}
				assert.Len(t, eventsWithType(JourneyEnd, events), 1, "real journey must finish")
				assert.Len(t, eventsWithType(CmdStatus, events), 1, "CLI must exit cleanly")
				if mode == "inline" || mode == "project" {
					assert.NotEmpty(t, eventsWithType(ScreenshotBlock, events), "screenshot data must survive cache relocation")
					assert.NotEmpty(t, eventsWithType(StepScreenshotRef, events), "screenshot references must survive cache relocation")
				}
				// Chromium may leave unrelated files in TMPDIR; only Heartbeat's
				// per-run directory is owned by this cleanup contract.
				scratch, err := filepath.Glob(filepath.Join(tempRoot, "heartbeat-synthetics-*"))
				require.NoError(t, err, "inspect Heartbeat scratch directories")
				assert.Empty(t, scratch, "real CLI run must remove its Heartbeat scratch directory")
				entries, err := os.ReadDir(stale)
				require.NoError(t, err, "inspect original cache")
				assert.Empty(t, entries, "CLI must not use Agent-managed cache")
			}
		})
	}
}

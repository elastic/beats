// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || synthetics

package synthexec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/security"
)

// TestWorkdirSubprocess exercises the actual child process boundary without Node.
func TestWorkdirSubprocess(t *testing.T) {
	marker := slices.Index(os.Args, "--heartbeat-workdir-helper")
	if marker < 0 {
		return
	}
	args := os.Args[marker+1:]
	idx := slices.Index(args, "--workdir")
	if idx < 0 || idx+1 == len(args) {
		os.Exit(10)
	}
	// Like commander, the final value wins when a flag is repeated.
	for i := idx + 1; i < len(args)-1; i++ {
		if args[i] == "--workdir" {
			idx = i
		}
	}
	dir := args[idx+1]
	cache := filepath.Join(dir, fmt.Sprint(os.Getpid()))
	//nolint:gosec // The parent test supplies its own temporary workdir.
	if err := os.Mkdir(cache, 0700); err != nil {
		os.Exit(11)
	}
	//nolint:gosec // The parent test supplies its own temporary workdir.
	if err := os.WriteFile(filepath.Join(cache, "screenshot"), []byte("image"), 0600); err != nil {
		os.Exit(12)
	}
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(13)
	}
	pipe := os.NewFile(3, "events")
	if err := json.NewEncoder(pipe).Encode(&SynthEvent{Type: "workdir/ready", Payload: map[string]any{"workdir": dir, "cache": cache, "cwd": cwd}}); err != nil {
		os.Exit(14)
	}
	for {
		//nolint:gosec // The parent test supplies its own release marker path.
		if _, err := os.Stat(filepath.Join(dir, "release")); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if args[0] == "failure" {
		os.Exit(23)
	}
	os.Exit(0)
}

func workdirCommand(t *testing.T, mode string) *SynthCmd {
	t.Helper()
	binary, err := os.Executable()
	require.NoError(t, err, "locate the subprocess test binary")
	//nolint:noctx // runCmd owns the subprocess timeout and cancellation.
	return &SynthCmd{exec.Command(binary, "-test.run=^TestWorkdirSubprocess$", "--", "--heartbeat-workdir-helper", mode)}
}

func nextWorkdirEvent(t *testing.T, mpx *ExecMultiplexer) *SynthEvent {
	t.Helper()
	select {
	case event := <-mpx.SynthEvents():
		return event
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for subprocess event")
		return nil
	}
}

func drainWorkdirEvents(t *testing.T, mpx *ExecMultiplexer) []*SynthEvent {
	t.Helper()
	var events []*SynthEvent
	for {
		event := nextWorkdirEvent(t, mpx)
		if event == nil {
			return events
		}
		events = append(events, event)
	}
}

func TestRunCmdWorkdirLifecycle(t *testing.T) {
	for _, mode := range []string{"success", "failure", "timeout", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			tempRoot := t.TempDir()
			t.Setenv("TMPDIR", tempRoot)
			agentDir := t.TempDir()
			staleCache := filepath.Join(agentDir, ".synthetics")
			require.NoError(t, os.Mkdir(staleCache, 0500), "simulate Agent-reset cache permissions")
			t.Cleanup(func() { _ = os.Chmod(staleCache, 0700) })
			cmd := workdirCommand(t, mode)
			cmd.Dir = agentDir
			// User-provided arguments must not redirect the runtime cache into Agent state.
			cmd.Args = append(cmd.Args, "--workdir", staleCache)
			timeout := time.Minute
			if mode == "timeout" {
				timeout = 2 * time.Second
			}
			ctx, cancel := context.WithCancel(context.WithValue(context.Background(), SynthexecTimeoutKey, timeout))
			defer cancel()
			mpx, err := runCmd(ctx, cmd, nil, nil, FilterJourneyConfig{}, stdfields.StdMonitorFields{}, "")
			require.NoError(t, err, "start helper with isolated scratch directory")
			ready := nextWorkdirEvent(t, mpx)
			require.NotNil(t, ready, "child must report its runtime cache")
			require.Equal(t, "workdir/ready", ready.Type, "child must initialize its cache successfully")
			dir, ok := ready.Payload["workdir"].(string)
			require.True(t, ok, "child must report a workdir path")
			assert.Equal(t, tempRoot, filepath.Dir(dir), "scratch space must be outside Agent state")
			assert.Equal(t, agentDir, ready.Payload["cwd"], "configuration discovery must retain the command directory")
			cache, ok := ready.Payload["cache"].(string)
			require.True(t, ok, "child must report a cache path")
			assert.FileExists(t, filepath.Join(cache, "screenshot"), "cache remains available while the child runs")
			info, err := os.Stat(dir)
			require.NoError(t, err, "inspect scratch directory permissions")
			assert.Equal(t, os.FileMode(0700), info.Mode().Perm(), "scratch artifacts must be private")
			switch mode {
			case "cancel":
				cancel()
			case "success", "failure":
				require.NoError(t, os.WriteFile(filepath.Join(dir, "release"), nil, 0600), "release child after checking its artifacts")
			}
			events := drainWorkdirEvents(t, mpx)
			statuses := eventsWithType(CmdStatus, events)
			require.Len(t, statuses, 1, "publish one completion status")
			if mode == "success" {
				assert.Nil(t, statuses[0].Error, "successful child must remain successful")
			} else {
				require.NotNil(t, statuses[0].Error, "failed or interrupted child must report an error")
				if mode == "timeout" {
					assert.Equal(t, "CMD_TIMEOUT", statuses[0].Error.Code, "preserve timeout classification")
				}
			}
			assert.NoDirExists(t, dir, "remove artifacts before closing the event stream")
			entries, err := os.ReadDir(tempRoot)
			require.NoError(t, err, "inspect scratch root after completion")
			assert.Empty(t, entries, "no scratch directory may leak")
			info, err = os.Stat(staleCache)
			require.NoError(t, err, "leave Agent-owned cache untouched")
			assert.Equal(t, os.FileMode(0500), info.Mode().Perm(), "do not repair permissions inside Agent state")
		})
	}
}

func TestRunCmdConcurrentWorkdirs(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), SynthexecTimeoutKey, time.Minute))
	defer cancel()
	var streams []*ExecMultiplexer
	var dirs []string
	for range 4 {
		mpx, err := runCmd(ctx, workdirCommand(t, "success"), nil, nil, FilterJourneyConfig{}, stdfields.StdMonitorFields{}, "")
		require.NoError(t, err, "start concurrent run")
		ready := nextWorkdirEvent(t, mpx)
		require.NotNil(t, ready, "concurrent child must initialize")
		require.Equal(t, "workdir/ready", ready.Type, "concurrent child must create artifacts")
		dir, ok := ready.Payload["workdir"].(string)
		require.True(t, ok, "child must report a workdir path")
		assert.NotContains(t, dirs, dir, "each run must have unique scratch space")
		dirs = append(dirs, dir)
		streams = append(streams, mpx)
	}
	for i, mpx := range streams {
		require.NoError(t, os.WriteFile(filepath.Join(dirs[i], "release"), nil, 0600), "finish one run")
		drainWorkdirEvents(t, mpx)
		assert.NoDirExists(t, dirs[i], "completed run must clean up")
		for _, dir := range dirs[i+1:] {
			assert.DirExists(t, dir, "cleanup must preserve other active runs")
		}
	}
}

func TestRunCmdWorkdirSetupFailures(t *testing.T) {
	for _, mode := range []string{"missing-temp-root", "stdout-pipe", "stderr-pipe", "start"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("TMPDIR", root)
			cmd := workdirCommand(t, "success")
			switch mode {
			case "missing-temp-root":
				t.Setenv("TMPDIR", filepath.Join(root, "missing"))
			case "stdout-pipe":
				cmd.Stdout = os.Stdout
			case "stderr-pipe":
				cmd.Stderr = os.Stderr
			case "start":
				cmd.Path = filepath.Join(root, "missing-command")
			}
			mpx, err := runCmd(context.Background(), cmd, nil, nil, FilterJourneyConfig{}, stdfields.StdMonitorFields{}, "")
			require.Error(t, err, "setup failure must reach caller")
			assert.Nil(t, mpx, "failed setup must not expose an event stream")
			if mode == "stderr-pipe" || mode == "start" {
				writer, ok := cmd.Stdout.(*os.File)
				require.True(t, ok, "stdout setup must have created a pipe writer")
				t.Cleanup(func() { _ = writer.Close() })
				_, err := writer.Stat()
				assert.ErrorIs(t, err, os.ErrClosed, "failed setup must close the child end of stdout")
			}
			for _, file := range cmd.ExtraFiles {
				_, err := file.Stat()
				assert.ErrorIs(t, err, os.ErrClosed, "failed setup must close JSON pipe descriptors")
			}
			entries, err := os.ReadDir(root)
			require.NoError(t, err, "inspect temporary root")
			assert.Empty(t, entries, "setup errors must not leak work directories")
		})
	}
}

func TestNewWorkdirRelativeTempRoot(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.Mkdir("scratch", 0700), "create relative temporary root")
	t.Setenv("TMPDIR", "scratch")
	cmd := workdirCommand(t, "success")
	dir, err := newWorkdir(cmd)
	require.NoError(t, err, "resolve relative TMPDIR")
	defer func() { _ = os.RemoveAll(dir) }()
	assert.True(t, filepath.IsAbs(dir), "workdir argument must not depend on project cwd")
	assert.Equal(t, filepath.Join(root, "scratch"), filepath.Dir(dir), "honor temporary root")
}

func TestNewWorkdirOwnershipFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires an unprivileged parent to reject chown")
	}
	root := t.TempDir()
	t.Setenv("TMPDIR", root)
	cmd := workdirCommand(t, "success")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 0, Gid: 0}}
	_, err := newWorkdir(cmd)
	require.ErrorContains(t, err, "could not assign synthetics work directory ownership", "do not launch with inaccessible scratch space")
	entries, err := os.ReadDir(root)
	require.NoError(t, err, "inspect temporary root")
	assert.Empty(t, entries, "ownership failures must clean up scratch space")
}

func TestRunCmdWorkdirRestrictedUser(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root to exercise the restricted subprocess user")
	}
	// Use a traversable parent: testing.T.TempDir normally creates private ancestors.
	root, err := os.MkdirTemp("/tmp", "heartbeat-workdir-credentials-")
	require.NoError(t, err, "create credential test root")
	defer func() { _ = os.RemoveAll(root) }()
	require.NoError(t, os.Chmod(root, 0755), "allow restricted child to traverse test root")
	t.Setenv("TMPDIR", root)
	agentDir := filepath.Join(root, "agent-state")
	require.NoError(t, os.Mkdir(agentDir, 0755), "create Agent component directory")
	stale := filepath.Join(agentDir, ".synthetics")
	require.NoError(t, os.Mkdir(stale, 0750), "simulate root-owned cache after Agent restart")
	original := security.NodeChildProcCred
	security.NodeChildProcCred = &syscall.Credential{Uid: 65534, Gid: 65534}
	defer func() { security.NodeChildProcCred = original }()
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), SynthexecTimeoutKey, time.Minute))
	defer cancel()
	for range 2 {
		cmd := workdirCommand(t, "success")
		cmd.Dir = agentDir
		mpx, err := runCmd(ctx, cmd, nil, nil, FilterJourneyConfig{}, stdfields.StdMonitorFields{}, "")
		require.NoError(t, err, "start synthetics as restricted user")
		ready := nextWorkdirEvent(t, mpx)
		require.NotNil(t, ready, "restricted user must report initialization")
		require.Equal(t, "workdir/ready", ready.Type, "restricted user must be able to write runtime artifacts")
		dir, ok := ready.Payload["workdir"].(string)
		require.True(t, ok, "child must report a workdir path")
		info, err := os.Stat(dir)
		require.NoError(t, err, "inspect restricted-user directory")
		stat, ok := info.Sys().(*syscall.Stat_t)
		require.True(t, ok, "inspect Linux directory ownership")
		assert.Equal(t, uint32(65534), stat.Uid, "scratch directory must belong to child UID")
		assert.Equal(t, uint32(65534), stat.Gid, "scratch directory must belong to child GID")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "release"), nil, 0644), "release restricted child")
		statuses := eventsWithType(CmdStatus, drainWorkdirEvents(t, mpx))
		require.Len(t, statuses, 1, "restricted child must complete")
		assert.Nil(t, statuses[0].Error, "restricted child must succeed after restart")
		assert.NoDirExists(t, dir, "parent must remove child-owned artifacts")
		info, err = os.Stat(stale)
		require.NoError(t, err, "Agent cache must survive unchanged")
		assert.Equal(t, os.FileMode(0750), info.Mode().Perm(), "Agent cache permissions must be untouched")
	}
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestOsqueryExit(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	otherExit := osqueryExitError(t, 1)

	t.Run("competing_extension_error", func(t *testing.T) { testOsqueryCompetingExtensionError(t, exit78) })
	t.Run("classification", func(t *testing.T) { testOsqueryExitClassification(t, exit78, otherExit) })
	t.Run("runtime_exit_78_then_startup_failure", func(t *testing.T) { testOsqueryRunnerRuntimeExit78ThenStartupFailure(t, exit78) })
	t.Run("readiness_during_exit_cleanup", func(t *testing.T) { testOsqueryReadinessDuringExitCleanup(t, exit78) })
}

func testOsqueryExitClassification(t *testing.T, exit78, otherExit error) {
	t.Helper()
	for _, tc := range []struct {
		name        string
		ready       bool
		err         error
		recoverable bool
	}{
		{
			name:        "startup_exit_78",
			ready:       false,
			err:         exit78,
			recoverable: false,
		},
		{
			name:        "runtime_exit_78",
			ready:       true,
			err:         exit78,
			recoverable: true,
		},
		{
			name:        "wrapped_runtime_exit_78",
			ready:       true,
			err:         fmt.Errorf("process: %w", exit78),
			recoverable: true,
		},
		{
			name:        "other_runtime_exit",
			ready:       true,
			err:         otherExit,
			recoverable: false,
		},
		{
			name:        "cancelled_runtime_exit",
			ready:       true,
			err:         errors.Join(context.Canceled, exit78),
			recoverable: false,
		},
		{
			name:        "deadline_runtime_exit",
			ready:       true,
			err:         errors.Join(context.DeadlineExceeded, exit78),
			recoverable: false,
		},
		{
			name:        "clean_exit",
			ready:       true,
			err:         nil,
			recoverable: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state osqueryRunState
			if tc.ready {
				state.ready()
			}
			err := state.exit(tc.err)
			assert.ErrorIs(t, err, tc.err, "preserve the original process error")
			assert.Equal(t, tc.recoverable, isRecoverableOsqueryError(err), "only runtime exit 78 adds recovery")
			state.ready()
			assert.Equal(t, osqueryExited, state.state.Load(), "late config callback must not revive an exited process")
		})
	}
}

func testOsqueryRunnerRuntimeExit78ThenStartupFailure(t *testing.T, exit78 error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var runs atomic.Int32
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, inputs <-chan []config.InputConfig) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-inputs:
			}
			var state osqueryRunState
			// More than five failures verifies that the existing unlimited retry
			// semantics remain intact. The replacement's startup failure is terminal.
			if runs.Add(1) <= 6 {
				state.ready()
			}
			return state.exit(exit78)
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{}}}), "start runner")
	select {
	case err := <-done:
		assert.ErrorIs(t, err, exit78, "replacement startup failure must remain terminal")
		assert.Equal(t, int32(7), runs.Load(), "runtime errors must restart without a retry cap")
	case <-time.After(10 * time.Second):
		t.Fatal("runner did not restart and return its startup failure")
	}
}

// delayedCleanupRunner reports child exit before allowing Run to return.
type delayedCleanupRunner struct {
	osqd.Runner
	err     error
	exited  chan struct{}
	cleanup chan struct{}
}

func (r *delayedCleanupRunner) Run(ctx context.Context, _ osqd.Flags, onExit func(error)) error {
	onExit(r.err)
	close(r.exited)
	select {
	case <-r.cleanup:
	case <-ctx.Done():
	}
	return r.err
}

func testOsqueryReadinessDuringExitCleanup(t *testing.T, exit78 error) {
	t.Helper()
	for _, initiallyReady := range []bool{false, true} {
		t.Run(fmt.Sprintf("ready=%v", initiallyReady), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			runner := &delayedCleanupRunner{err: exit78, exited: make(chan struct{}), cleanup: make(chan struct{})}
			var state osqueryRunState
			if initiallyReady {
				state.ready()
			}
			done := make(chan error, 1)
			go func() { done <- state.run(ctx, runner, nil) }()
			select {
			case <-runner.exited:
			case <-time.After(5 * time.Second):
				t.Fatal("child exit was not observed")
			}
			// Complete a config callback while Run is blocked in post-exit cleanup.
			state.ready()
			select {
			case <-done:
				t.Fatal("Run returned before cleanup was released")
			default:
			}
			close(runner.cleanup)
			select {
			case err := <-done:
				assert.ErrorIs(t, err, exit78, "preserve exit cause")
				assert.Equal(t, initiallyReady, isRecoverableOsqueryError(err), "only readiness before child exit permits recovery")
			case <-time.After(5 * time.Second):
				t.Fatal("Run did not finish cleanup")
			}
		})
	}
}

// TestOsqueryExitHelper runs only in a subprocess to produce a real exec.ExitError.
func TestOsqueryExitHelper(t *testing.T) {
	code := os.Getenv("OSQUERY_TEST_EXIT")
	if code == "" {
		t.Skip("exit code helper is not a test")
	}
	value, err := strconv.Atoi(code)
	if err != nil {
		os.Exit(2)
	}
	os.Exit(value)
}

func osqueryExitError(t *testing.T, code int) error {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err, "locate the test executable")
	command := exec.CommandContext(t.Context(), executable, "-test.run=^TestOsqueryExitHelper$")
	command.Env = append(os.Environ(), "OSQUERY_TEST_EXIT="+strconv.Itoa(code))
	err = command.Run()
	require.Error(t, err, "helper must exit unsuccessfully")
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError, "helper must produce a real process exit error")
	return err
}

func testOsqueryCompetingExtensionError(t *testing.T, exit78 error) {
	t.Helper()
	for _, tc := range []struct {
		name   string
		ready  bool
		cancel bool
	}{
		{name: "startup_exit"},
		{name: "runtime_exit", ready: true},
		{name: "parent_cancellation", cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, cancel := context.WithCancel(t.Context())
			defer cancel()
			var state osqueryRunState
			if tc.ready {
				state.ready()
			}
			runner := &delayedCleanupRunner{err: exit78, exited: make(chan struct{}), cleanup: make(chan struct{})}
			var cleanupOnce sync.Once
			cleanup := func() { cleanupOnce.Do(func() { close(runner.cleanup) }) }
			defer cleanup()
			extensionErr := errors.New("extension ping failed: timeout after 200ms")
			g, ctx := errgroup.WithContext(parent)
			g.Go(func() error {
				// Simulate cleanup that remains blocked despite sibling cancellation.
				return state.run(parent, runner, nil)
			})
			g.Go(func() error {
				select {
				case <-runner.exited:
					return extensionErr
				case <-ctx.Done():
					return ctx.Err()
				}
			})
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				t.Fatal("extension failure was not observed")
			}
			state.ready() // Late config callback while process cleanup is blocked.
			if tc.cancel {
				cancel()
			}
			cleanup()
			firstErr := g.Wait()
			require.ErrorIs(t, firstErr, extensionErr, "extension failure must win the errgroup race")
			err := state.result(parent, firstErr)
			switch {
			case tc.cancel:
				assert.ErrorIs(t, err, context.Canceled, "parent cancellation takes precedence")
			case tc.ready:
				assert.ErrorIs(t, err, extensionErr, "preserve the competing cleanup error")
				assert.ErrorIs(t, err, errOsqueryRuntimeExit78, "retain the confirmed runtime exit on this release")
				assert.True(t, isRecoverableOsqueryError(err), "runtime exit must recover even if the extension error wins")
			default:
				assert.ErrorIs(t, err, exit78, "startup exit must override the competing extension error")
				assert.False(t, isRecoverableOsqueryError(err), "startup failure must not restart")
			}
		})
	}
}

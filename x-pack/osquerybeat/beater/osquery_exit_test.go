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

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestOsqueryExit(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	otherExit := osqueryExitError(t, 1)

	t.Run("competing_extension_error_exit_78", func(t *testing.T) { testOsqueryCompetingExtensionError(t, exit78) })
	t.Run("competing_extension_error_other_exit", func(t *testing.T) { testOsqueryCompetingExtensionError(t, otherExit) })
	t.Run("classification", func(t *testing.T) { testOsqueryExitClassification(t, exit78, otherExit) })
	t.Run("runtime_exit_78_then_startup_failure", func(t *testing.T) { testOsqueryRunnerRuntimeExit78ThenStartupFailure(t, exit78) })
	t.Run("readiness_during_exit_cleanup", func(t *testing.T) { testOsqueryReadinessDuringExitCleanup(t, exit78) })
	t.Run("kill_after_cancellation", func(t *testing.T) { testOsqueryKillAfterCancellation(t, osqueryKilledError(t), exit78) })
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
			name: "other_startup_exit", err: otherExit,
		},
		{
			name:        "other_runtime_exit",
			ready:       true,
			err:         otherExit,
			recoverable: true,
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
			recoverable: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state osqueryRunState
			if tc.ready {
				state.ready()
			}
			err := state.exit(tc.err)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "preserve the original process error")
			}
			assert.Equal(t, tc.recoverable, isRecoverableOsqueryError(err), "only uncancelled runtime exits add recovery")
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
	runner.maxRestarts = 8
	runner.retryInitial = time.Millisecond
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-inputs:
			}
			var state osqueryRunState
			// With maxRestarts at 8, the first six runs fail after readiness and
			// all restart. The seventh fails before readiness, which is terminal.
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
		assert.Equal(t, int32(7), runs.Load(), "runtime exit 78 must restart while the budget lasts")
	case <-time.After(10 * time.Second):
		t.Fatal("runner did not restart and return its startup failure")
	}
}

// A clean osqueryd exit after readiness is an unexpected runtime exit, so it
// restarts with backoff; the same exit before readiness is a startup failure.
func TestOsqueryCleanExitAfterReadinessRestarts(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "clean_exit"))
	runner.retryInitial = time.Millisecond
	var states []status.Status
	runner.reportStatus = func(state status.Status, _ string) {
		states = append(states, state)
	}
	var attempts atomic.Int32
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start a policy")
	err := runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
		var state osqueryRunState
		if attempts.Add(1) == 1 {
			state.ready()
		}
		// osqd.Run reports a clean exit as a nil error.
		return state.run(ctx, &mockOsqueryd{}, nil)
	})
	assert.ErrorIs(t, err, ErrOsquerydExited, "a clean exit before readiness must fail startup")
	assert.Equal(t, int32(2), attempts.Load(), "a clean exit after readiness must restart once")
	assert.Contains(t, states, status.Degraded, "a clean runtime exit must report degraded while waiting to restart")
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
	if code == "hang" {
		time.Sleep(time.Minute)
		os.Exit(0)
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

// osqueryKilledError returns the exit error of a killed process, as osqd.Run
// reports it after its context is cancelled.
func osqueryKilledError(t *testing.T) error {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err, "locate the test executable")
	command := exec.CommandContext(t.Context(), executable, "-test.run=^TestOsqueryExitHelper$")
	command.Env = append(os.Environ(), "OSQUERY_TEST_EXIT=hang")
	require.NoError(t, command.Start(), "start the helper")
	require.NoError(t, command.Process.Kill(), "kill the helper")
	err = command.Wait()
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError, "a killed helper must produce a real process exit error")
	return err
}

// killedOnCancelRunner behaves like osqd.Run when its context is cancelled:
// the process is killed, onExit receives the kill, and Run returns nil.
type killedOnCancelRunner struct {
	mockOsqueryd
	err error
}

func (r *killedOnCancelRunner) Run(ctx context.Context, _ osqd.Flags, onExit func(error)) error {
	<-ctx.Done()
	onExit(r.err)
	return nil
}

func testOsqueryKillAfterCancellation(t *testing.T, killed, exit78 error) {
	t.Helper()
	extensionErr := errors.New("extension ping failed: timeout after 200ms")
	for _, tc := range []struct {
		name        string
		exit        error
		want        error
		recoverable bool
	}{
		{name: "sibling_failure_before_ready", exit: killed, want: extensionErr, recoverable: true},
		// osqueryd may exit 78 on its own just after a sibling notices it is gone.
		{name: "startup_exit_78_after_sibling_failure", exit: exit78, want: exit78, recoverable: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state osqueryRunState
			g, ctx := errgroup.WithContext(t.Context())
			g.Go(func() error { return state.run(ctx, &killedOnCancelRunner{err: tc.exit}, nil) })
			g.Go(func() error { return extensionErr })
			err := state.result(t.Context(), g.Wait())
			assert.ErrorIs(t, err, tc.want, "keep the cause of the shutdown")
			assert.Equal(t, tc.recoverable, isRecoverableOsqueryError(err), "keep the cause's recoverability")
		})
	}
	t.Run("intentional_restart_after_ready", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		var state osqueryRunState
		state.ready()
		err := state.run(ctx, &killedOnCancelRunner{err: killed}, nil)
		assert.NoError(t, err, "a kill caused by our cancellation is not an osqueryd failure")
	})
}

// After an intentional stop, runOsquery turns the killed process's nil exit
// into ErrOsquerydExited, which can win the errgroup race. The cancellation
// must still be the result, with or without a startup error.
func TestOsqueryRunResultPrefersCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, startupErr := range []bool{false, true} {
		var state osqueryRunState
		if startupErr {
			assert.ErrorIs(t, state.exit(nil), ErrOsquerydExited, "a clean exit before readiness is a startup failure")
		}
		err := state.result(ctx, ErrOsquerydExited)
		assert.ErrorIs(t, err, context.Canceled, "an intentional stop must not report an osqueryd exit (startup error: %v)", startupErr)
	}
}

func TestRunOsqueryLogsCancelledExitAtInfo(t *testing.T) {
	log, logs := logptest.NewTestingLoggerWithObserver(t, "")
	ob, b, _ := newStatusTestBeater(t, func(ob *osquerybeat) { ob.log = log })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	runner := &killedOnCancelRunner{err: osqueryKilledError(t)}
	rah := newResetableActionHandler(ob.liveResults, ob.pub, log)
	metrics := newOsquerydMetrics(monitoring.NewRegistry(), log)
	err := ob.runOsquery(ctx, b, runner, nil, config.ExtensionsConfig{}, make(chan []config.InputConfig), rah, metrics)
	assert.ErrorIs(t, err, context.Canceled, "shutdown must report cancellation")
	assert.Zero(t, logs.FilterMessageSnippet("Failed to run osqueryd").Len(), "an intentional stop must not log a failure")
	assert.Equal(t, 1, logs.FilterMessage("osqueryd process exited").Len(), "an intentional stop logs the exit at info")
}

func testOsqueryCompetingExtensionError(t *testing.T, exit error) {
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
			runner := &delayedCleanupRunner{err: exit, exited: make(chan struct{}), cleanup: make(chan struct{})}
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
				assert.ErrorIs(t, err, extensionErr, "runtime recovery keeps existing error selection")
			default:
				assert.ErrorIs(t, err, exit, "startup exit must override the competing extension error")
				assert.False(t, isRecoverableOsqueryError(err), "startup failure must not restart")
			}
		})
	}
}

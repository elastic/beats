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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestOsqueryExitHelper(t *testing.T) {
	if code := os.Getenv("OSQUERY_TEST_EXIT"); code != "" {
		value, err := strconv.Atoi(code)
		if err != nil {
			os.Exit(2)
		}
		os.Exit(value)
	}
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

func recoveryInput(name string, thriftTimeout int) []config.InputConfig {
	return []config.InputConfig{{Name: name, Osquery: &config.OsqueryConfig{Options: map[string]any{"thrift_timeout": thriftTimeout}}}}
}

func recoveryRunner(t *testing.T, run osqueryRunFunc, configure func(*osqueryRunner)) (*osqueryRunner, context.Context, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	runner.retryInitial, runner.retryMax = 10*time.Millisecond, 40*time.Millisecond
	if configure != nil {
		configure(runner)
	}
	done, finished := make(chan error, 1), make(chan struct{})
	go func() { defer close(finished); done <- runner.Run(ctx, run) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(2 * time.Second):
			t.Error("runner must stop during test cleanup")
		}
	})
	return runner, ctx, cancel, done
}

func waitRecoveryValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not deliver the expected transition")
	}
	var zero T
	return zero
}

func TestOsqueryRunnerExit78RequiresReadyConfiguration(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	tests := []struct {
		name     string
		ready    bool
		err      error
		wantRuns int
	}{
		{"runtime exit 78", true, fmt.Errorf("osqueryd: %w", exit78), 6},
		{"startup exit 78", false, fmt.Errorf("osqueryd: %w", exit78), 1},
		{"invalid policy after readiness", true, ErrECSMappingIsInvalid, 1},
		{"other process exit", true, osqueryExitError(t, 1), 1},
		{"unexpected clean exit", true, ErrOsquerydExited, 1},
		{"cancelled process exit", true, errors.Join(context.Canceled, exit78), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var runs atomic.Int32
			run := func(_ context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready func(), _ func()) error {
				runs.Add(1)
				if tc.ready {
					ready()
				}
				return tc.err
			}
			runner, ctx, cancel, done := recoveryRunner(t, run, nil)
			require.NoError(t, runner.Update(ctx, recoveryInput("policy", 3)), "start daemon with a real input")
			if errors.Is(tc.err, context.Canceled) {
				require.Eventually(t, func() bool { return runs.Load() == 1 }, time.Second, time.Millisecond, "cancelled run must execute once")
				cancel()
			}
			err := waitRecoveryValue(t, done)
			if errors.Is(tc.err, context.Canceled) {
				require.ErrorIs(t, err, context.Canceled, "intentional cancellation must be preserved")
			} else {
				require.ErrorIs(t, err, tc.err, "terminal failure must preserve its underlying cause")
			}
			assert.Equal(t, int32(tc.wantRuns), runs.Load(), "only ready runtime exit 78 may consume the five restart attempts")
			if tc.wantRuns > 1 {
				assert.ErrorContains(t, err, "restart limit exceeded after 5 retries", "repeated runtime crashes must become terminal")
			}
		})
	}
	assert.True(t, isOsqueryExit78(fmt.Errorf("wrapped: %w", exit78)), "wrapped exit 78 must be recognized")
	assert.False(t, isOsqueryExit78(errors.Join(context.Canceled, exit78)), "intentional cancellation must override process exit")
}

func TestOsqueryRunnerRecoveryKeepsLatestPolicyAndHealth(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	reports := make(chan status.Status, 10)
	nextPolicy := make(chan []config.InputConfig, 1)
	releaseReady := make(chan struct{})
	var runs atomic.Int32
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig, ready func(), _ func()) error {
		if runs.Add(1) == 1 {
			ready()
			return exit78
		}
		select {
		case input := <-inputs:
			nextPolicy <- input
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-releaseReady:
			ready()
		case <-ctx.Done():
			return ctx.Err()
		}
		<-ctx.Done()
		return ctx.Err()
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) {
		r.retryInitial = 100 * time.Millisecond
		r.reportStatus = func(s status.Status, _ string) { reports <- s }
	})
	require.NoError(t, runner.Update(ctx, recoveryInput("old", 3)), "start the initial policy")
	for waitRecoveryValue(t, reports) != status.Degraded {
	}
	require.NoError(t, runner.Update(ctx, recoveryInput("latest", 4)), "policy updates must remain responsive during backoff")
	policy := waitRecoveryValue(t, nextPolicy)
	assert.Equal(t, "latest", policy[0].Name, "restart must use the most recent input")
	assert.Equal(t, 4, policy[0].Osquery.Options["thrift_timeout"], "restart must use the latest flags")
	select {
	case s := <-reports:
		t.Errorf("recovery reported %v before current-run readiness", s)
	default:
	}
	close(releaseReady)
	assert.Equal(t, status.Running, waitRecoveryValue(t, reports), "current configuration readiness must restore Running")
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "shutdown must cancel the active replacement")
}

func TestOsqueryRunnerRecoveryBackoffAndCancellation(t *testing.T) {
	started := make(chan time.Time, 4)
	run := func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan []config.InputConfig, func(), func()) error {
		started <- time.Now()
		return errors.New("write: broken pipe")
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.retryInitial, r.retryMax = 30*time.Millisecond, 120*time.Millisecond })
	require.NoError(t, runner.Update(ctx, recoveryInput("policy", 3)), "start transient failure sequence")
	previous := waitRecoveryValue(t, started)
	for _, minimum := range []time.Duration{25 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond} {
		now := waitRecoveryValue(t, started)
		assert.GreaterOrEqual(t, now.Sub(previous), minimum, "consecutive failures must increase the retry delay")
		previous = now
	}
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "cancellation must interrupt backoff without waiting for its timer")
}

func TestOsqueryRunnerRecoveryBudgetResetsAfterStableReadiness(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	started := make(chan int32, 3)
	crash := make(chan struct{})
	var runs atomic.Int32
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready func(), _ func()) error {
		number := runs.Add(1)
		ready()
		started <- number
		if number == 1 {
			return exit78
		}
		if number == 2 {
			select {
			case <-crash:
				return exit78
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		<-ctx.Done()
		return ctx.Err()
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.maxRetries = 1; r.stablePeriod = 40 * time.Millisecond })
	require.NoError(t, runner.Update(ctx, recoveryInput("policy", 3)), "start budget-reset test")
	assert.Equal(t, int32(1), waitRecoveryValue(t, started), "initial daemon must start")
	assert.Equal(t, int32(2), waitRecoveryValue(t, started), "first failure consumes the only retry")
	time.Sleep(50 * time.Millisecond)
	close(crash)
	assert.Equal(t, int32(3), waitRecoveryValue(t, started), "stable ready operation must restore retry budget")
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "replacement must remain cancellable")
}

func TestOsqueryRunnerRecoveryIgnoresOldReadiness(t *testing.T) {
	callbacks := make(chan func(), 2)
	reports := make(chan status.Status, 10)
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig, ready func(), _ func()) error {
		callbacks <- ready
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-inputs:
			}
		}
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.reportStatus = func(s status.Status, _ string) { reports <- s } })
	require.NoError(t, runner.Update(ctx, recoveryInput("first", 3)), "start first run")
	stale := waitRecoveryValue(t, callbacks)
	require.NoError(t, runner.Update(ctx, recoveryInput("replacement", 4)), "changed flags must restart")
	current := waitRecoveryValue(t, callbacks)
	stale()
	for len(reports) > 0 {
		assert.NotEqual(t, status.Running, <-reports, "previous-run callback must not mark replacement ready")
	}
	current()
	assert.Equal(t, status.Running, waitRecoveryValue(t, reports), "current readiness must not be lost behind stale callbacks")
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "runner must stop")
}

func TestOsqueryRunnerRemovalSuspendsRecovery(t *testing.T) {
	degraded := make(chan struct{}, 1)
	started := make(chan struct{}, 2)
	var runs atomic.Int32
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, _, _ func()) error {
		started <- struct{}{}
		if runs.Add(1) == 1 {
			return errors.New("write: broken pipe")
		}
		<-ctx.Done()
		return ctx.Err()
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) {
		r.retryInitial = 100 * time.Millisecond
		r.reportStatus = func(s status.Status, _ string) {
			if s == status.Degraded {
				degraded <- struct{}{}
			}
		}
	})
	require.NoError(t, runner.Update(ctx, recoveryInput("first", 3)), "start real policy")
	waitRecoveryValue(t, started)
	waitRecoveryValue(t, degraded)
	require.NoError(t, runner.Update(ctx, nil), "remove inputs during backoff")
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(1), runs.Load(), "removed policy must not launch a replacement daemon")
	require.NoError(t, runner.Update(ctx, recoveryInput("readded", 4)), "new policy must resume daemon lifecycle")
	waitRecoveryValue(t, started)
	require.NoError(t, runner.Update(ctx, nil), "removal must also cancel an active daemon")
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "suspended runner must remain cancellable")
}

func TestOsqueryRunnerLatestPolicyDuringTeardown(t *testing.T) {
	stopping, finish := make(chan struct{}), make(chan struct{})
	policies := make(chan []config.InputConfig, 2)
	var runs atomic.Int32
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig, _, _ func()) error {
		select {
		case input := <-inputs:
			policies <- input
		case <-ctx.Done():
			return ctx.Err()
		}
		if runs.Add(1) == 1 {
			<-ctx.Done()
			close(stopping)
			<-finish
			return ctx.Err()
		}
		<-ctx.Done()
		return ctx.Err()
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, nil)
	require.NoError(t, runner.Update(ctx, recoveryInput("first", 3)), "start initial daemon")
	waitRecoveryValue(t, policies)
	require.NoError(t, runner.Update(ctx, recoveryInput("intermediate", 4)), "options change must start teardown")
	waitRecoveryValue(t, stopping)
	require.NoError(t, runner.Update(ctx, recoveryInput("latest", 5)), "runner must receive updates during slow teardown")
	close(finish)
	assert.Equal(t, "latest", waitRecoveryValue(t, policies)[0].Name, "replacement must discard intermediate teardown policy")
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "runner must join replacement cancellation")
}

func TestOsqueryRunnerTeardownDoesNotResetRecoveryBudget(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	var runs atomic.Int32
	run := func(_ context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready, exited func()) error {
		number := runs.Add(1)
		ready()
		exited()
		if number == 2 {
			time.Sleep(50 * time.Millisecond)
		}
		return exit78
	}
	runner, ctx, _, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.maxRetries = 1; r.stablePeriod = 30 * time.Millisecond })
	require.NoError(t, runner.Update(ctx, recoveryInput("policy", 3)), "start short-lived daemons with slow cleanup")
	err := waitRecoveryValue(t, done)
	assert.ErrorContains(t, err, "restart limit exceeded after 1 retries", "cleanup must not replenish the recovery budget")
	assert.Equal(t, int32(2), runs.Load(), "daemon lifetime, not cleanup time, must limit retries")
}

func TestOsqueryRunnerReadinessAfterExitIsIgnored(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	reports := make(chan status.Status, 5)
	run := func(_ context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready, exited func()) error {
		exited()
		ready()
		return exit78
	}
	runner, ctx, _, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.reportStatus = func(s status.Status, _ string) { reports <- s } })
	require.NoError(t, runner.Update(ctx, recoveryInput("policy", 3)), "start failed startup run")
	assert.ErrorIs(t, waitRecoveryValue(t, done), exit78, "post-exit configuration callback must not make startup failure recoverable")
	for len(reports) > 0 {
		assert.NotEqual(t, status.Running, <-reports, "exited daemon must not report running")
	}
}

func TestOsqueryRunnerStablePolicyRestartResetsRecoveryBudget(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	started := make(chan int32, 4)
	var runs atomic.Int32
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready, exited func()) error {
		number := runs.Add(1)
		ready()
		started <- number
		if number == 1 || number == 3 {
			exited()
			return exit78
		}
		<-ctx.Done()
		exited()
		return ctx.Err()
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.maxRetries = 1; r.stablePeriod = 30 * time.Millisecond })
	require.NoError(t, runner.Update(ctx, recoveryInput("first", 3)), "start initial policy")
	waitRecoveryValue(t, started)
	assert.Equal(t, int32(2), waitRecoveryValue(t, started), "first recovery consumes the retry")
	time.Sleep(40 * time.Millisecond)
	require.NoError(t, runner.Update(ctx, recoveryInput("replacement", 4)), "options restart must retain stable uptime evidence")
	assert.Equal(t, int32(3), waitRecoveryValue(t, started), "new options must restart the daemon")
	assert.Equal(t, int32(4), waitRecoveryValue(t, started), "stable intentional restart must have replenished the retry budget")
	cancel()
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "final replacement must stop")
}

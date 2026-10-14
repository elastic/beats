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

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestOsqueryRunnerUpdateWaitsForLifecycleDecision(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, _, _ func()) error {
		<-ctx.Done()
		return ctx.Err()
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) {
		r.reportStatus = func(s status.Status, _ string) {
			if s == status.Configuring {
				close(entered)
				<-release
			}
		}
	})
	t.Cleanup(unblock)
	updated := make(chan error, 1)
	go func() { updated <- runner.Update(ctx, recoveryInput("new", 3)) }()
	waitRecoveryValue(t, entered)
	select {
	case err := <-updated:
		t.Fatalf("Update returned before readiness was invalidated: %v", err)
	default:
	}
	unblock()
	require.NoError(t, waitRecoveryValue(t, updated), "Update must return after the lifecycle decision without waiting for daemon readiness")
	cancel()
	require.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "runner must remain cancellable")
}

func TestOsqueryRunnerUpdateWaitIsCancellable(t *testing.T) {
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	updated := make(chan error, 1)
	go func() { updated <- runner.Update(ctx, recoveryInput("queued", 3)) }()
	require.Eventually(t, func() bool { return len(runner.inputCh) == 1 }, time.Second, time.Millisecond, "update must be queued before cancelling its acknowledgement wait")
	cancel()
	require.ErrorIs(t, waitRecoveryValue(t, updated), context.Canceled, "waiting for lifecycle acknowledgement must honour cancellation")
}

func TestOsqueryRunnerUpdateAfterTermination(t *testing.T) {
	wantErr := errors.New("invalid startup configuration")
	run := func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan []config.InputConfig, func(), func()) error {
		return wantErr
	}
	runner, ctx, _, done := recoveryRunner(t, run, nil)
	require.NoError(t, runner.Update(ctx, recoveryInput("invalid", 3)), "an applied update must be acknowledged even if its run exits immediately")
	require.ErrorIs(t, waitRecoveryValue(t, done), wantErr, "runner must retain the terminal cause")
	require.ErrorIs(t, runner.Update(ctx, recoveryInput("late", 3)), wantErr, "a stopped runner must release update waiters without hiding the terminal cause")
}

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
			assert.Equal(t, tc.wantRuns, int(runs.Load()), "only ready runtime exit 78 may consume the five restart attempts")
			if tc.wantRuns > 1 {
				assert.ErrorContains(t, err, "restart limit exceeded after 5 retries", "repeated runtime crashes must become terminal")
			}
		})
	}
	assert.True(t, isOsqueryExit78(fmt.Errorf("wrapped: %w", exit78)), "wrapped exit 78 must be recognized")
	assert.False(t, isOsqueryExit78(errors.Join(context.Canceled, exit78)), "intentional cancellation must override process exit")
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

func TestOsqueryRunnerReportsExitBeforeTeardownCompletes(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	reports := make(chan status.Status, 10)
	release := make(chan struct{})
	run := func(_ context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready, exited func()) error {
		ready()
		exited()
		<-release
		return exit78
	}
	runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) { r.reportStatus = func(s status.Status, _ string) { reports <- s } })
	require.NoError(t, runner.Update(ctx, recoveryInput("policy", 3)), "start daemon whose cleanup is delayed")
	for waitRecoveryValue(t, reports) != status.Degraded {
	}
	select {
	case err := <-done:
		t.Errorf("runner returned before run cleanup completed: %v", err)
	default:
	}
	cancel()
	close(release)
	assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "shutdown must join the old run without launching another")
}

func TestOsqueryRunnerPolicyAfterExitPreservesRecovery(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	for _, tc := range []struct {
		name                string
		pendingExit         bool
		remove, stayRemoved bool
	}{
		{"replace", false, false, false},
		{"pending exit", true, false, false},
		{"remove and re-add", false, true, false},
		{"remain removed", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			running, unblockRunning := make(chan struct{}), make(chan struct{})
			crash, exitedSignal, cleanup := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var runningOnce, crashOnce, cleanupOnce sync.Once
			releaseRunning := func() { runningOnce.Do(func() { close(unblockRunning) }) }
			releaseCrash := func() { crashOnce.Do(func() { close(crash) }) }
			releaseCleanup := func() { cleanupOnce.Do(func() { close(cleanup) }) }
			type start struct {
				policy []config.InputConfig
				at     time.Time
			}
			started := make(chan start, 3)
			reports := make(chan status.Status, 20)
			var runs atomic.Int32
			run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig, ready, exited func()) error {
				var policy []config.InputConfig
				select {
				case policy = <-inputs:
				case <-ctx.Done():
					return ctx.Err()
				}
				number := runs.Add(1)
				started <- start{policy: policy, at: time.Now()}
				ready()
				if number == 1 {
					<-crash
					exited()
					close(exitedSignal)
					<-cleanup
				} else {
					exited()
				}
				return exit78
			}
			runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) {
				r.maxRetries = 1
				r.retryInitial = 80 * time.Millisecond
				r.reportStatus = func(s status.Status, _ string) {
					reports <- s
					if s == status.Running && runs.Load() == 1 {
						close(running)
						<-unblockRunning
					}
				}
			})
			t.Cleanup(func() { releaseRunning(); releaseCrash(); releaseCleanup() })
			require.NoError(t, runner.Update(ctx, recoveryInput("original", 3)), "start the original daemon")
			waitRecoveryValue(t, started)
			waitRecoveryValue(t, running)
			sendUpdate := func() <-chan error {
				updated := make(chan error, 1)
				go func() {
					if tc.remove {
						if err := runner.Update(ctx, nil); err != nil {
							updated <- err
							return
						}
					}
					if tc.stayRemoved {
						updated <- nil
						return
					}
					updated <- runner.Update(ctx, recoveryInput("replacement", 4))
				}()
				return updated
			}
			var updated <-chan error
			if tc.pendingExit {
				updated = sendUpdate()
				// Both events are queued before select resumes. Handling the update
				// first must still observe the exit and report degradation.
				require.Eventually(t, func() bool { return len(runner.inputCh) == 1 }, time.Second, time.Millisecond, "queue the policy beside the pending exit")
				releaseCrash()
				waitRecoveryValue(t, exitedSignal)
				releaseRunning()
			} else {
				releaseRunning()
				releaseCrash()
				waitRecoveryValue(t, exitedSignal)
				// The update is intentionally sent after the exit so this case
				// covers the ordinary observed-exit ordering.
				updated = sendUpdate()
				require.NoError(t, waitRecoveryValue(t, updated), "retain the new policy while cleanup is blocked")
			}
			if tc.pendingExit {
				require.NoError(t, waitRecoveryValue(t, updated), "retain the new policy while cleanup is blocked")
			}
			for waitRecoveryValue(t, reports) != status.Degraded {
			}
			assert.Equal(t, int32(1), runs.Load(), "policy update must not start a daemon during cleanup")
			releasedAt := time.Now()
			releaseCleanup()
			if tc.stayRemoved {
				for waitRecoveryValue(t, reports) != status.Configuring {
				}
				assert.Equal(t, int32(1), runs.Load(), "removing all inputs must suspend recovery after cleanup")
				cancel()
				assert.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "idle runner must stop without retrying")
				return
			}
			replacement := waitRecoveryValue(t, started)
			assert.GreaterOrEqual(t, replacement.at.Sub(releasedAt), 70*time.Millisecond, "an already-exited daemon must still wait for recovery backoff")
			assert.Equal(t, "replacement", replacement.policy[0].Name, "recovery must use the latest policy")
			assert.Equal(t, 4, replacement.policy[0].Osquery.Options["thrift_timeout"], "recovery must use the latest options")
			err := waitRecoveryValue(t, done)
			assert.ErrorContains(t, err, "restart limit exceeded after 1 retries", "the replacement must consume the retry budget")
			assert.Equal(t, int32(2), runs.Load(), "policy changes must not grant an uncounted restart")
		})
	}
}

func TestOsqueryRunnerReaddedInputsPreserveRetryBudget(t *testing.T) {
	started := make(chan time.Time, 8)
	failed := make(chan struct{}, 8)
	run := func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan []config.InputConfig, func(), func()) error {
		started <- time.Now()
		return errors.New("write: broken pipe")
	}
	runner, ctx, _, done := recoveryRunner(t, run, func(r *osqueryRunner) {
		r.maxRetries = 2
		r.retryInitial, r.retryMax = 80*time.Millisecond, 80*time.Millisecond
		r.reportStatus = func(s status.Status, _ string) {
			if s == status.Degraded {
				failed <- struct{}{}
			}
		}
	})
	require.NoError(t, runner.Update(ctx, recoveryInput("initial", 3)), "start the initial daemon")
	waitRecoveryValue(t, started)
	for range 2 {
		waitRecoveryValue(t, failed)
		require.NoError(t, runner.Update(ctx, nil), "suspend pending recovery")
		readded := time.Now()
		require.NoError(t, runner.Update(ctx, recoveryInput("replacement", 3)), "resume pending recovery")
		waitRecoveryValue(t, failed)
		next := waitRecoveryValue(t, started)
		assert.GreaterOrEqual(t, next.Sub(readded), 70*time.Millisecond, "re-added inputs must wait for a fresh backoff")
	}
	assert.ErrorContains(t, waitRecoveryValue(t, done), "restart limit exceeded after 2 retries", "policy toggles must not create uncounted replacement runs")
	assert.Empty(t, started, "the exhausted budget must prevent further replacements")
}

func TestOsqueryRunnerReaddedInputsDuringRetryPreserveRetryBudget(t *testing.T) {
	started := make(chan time.Time, 3)
	secondStarted, releaseSecond := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseSecond) }) }
	var runs atomic.Int32
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, _, _ func()) error {
		number := runs.Add(1)
		started <- time.Now()
		if number == 2 {
			close(secondStarted)
			<-ctx.Done()
			<-releaseSecond
			return ctx.Err()
		}
		return errors.New("write: broken pipe")
	}
	runner, ctx, _, done := recoveryRunner(t, run, func(r *osqueryRunner) {
		r.maxRetries = 2
		r.retryInitial, r.retryMax = 80*time.Millisecond, 80*time.Millisecond
	})
	t.Cleanup(release)
	require.NoError(t, runner.Update(ctx, recoveryInput("initial", 3)), "start the initial daemon")
	waitRecoveryValue(t, started)
	waitRecoveryValue(t, secondStarted)
	waitRecoveryValue(t, started)
	require.NoError(t, runner.Update(ctx, nil), "stop the active replacement")
	require.NoError(t, runner.Update(ctx, recoveryInput("replacement", 3)), "retain the latest policy while cleanup is pending")
	released := time.Now()
	release()
	third := waitRecoveryValue(t, started)
	assert.GreaterOrEqual(t, third.Sub(released), 70*time.Millisecond, "re-adding during an active retry must wait through recovery backoff")
	assert.ErrorContains(t, waitRecoveryValue(t, done), "restart limit exceeded after 2 retries", "the active retry must retain its budget after policy removal")
	assert.Equal(t, int32(3), runs.Load(), "policy changes must not grant an uncounted replacement")
}

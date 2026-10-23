// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"strings"
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

func TestOsqueryRecoveryBackoffExhaustsBudget(t *testing.T) {
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "recovery"))
	runner.retryInitial = time.Millisecond
	runner.retryMax = 3 * time.Millisecond
	runner.maxRestarts = 4
	var messages []string
	var states []status.Status
	runner.reportStatus = func(state status.Status, message string) {
		states = append(states, state)
		if state == status.Degraded {
			messages = append(messages, message)
		}
	}
	failure := errors.New("extension ping failed: unavailable")
	var attempts atomic.Int32
	require.NoError(t, runner.Update(t.Context(), []config.InputConfig{{}}), "supply an initial policy")
	err := runner.Run(t.Context(), func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan []config.InputConfig) error {
		attempts.Add(1)
		return failure
	})
	require.ErrorIs(t, err, failure, "exhaustion must preserve the underlying error")
	assert.ErrorContains(t, err, "exhausted 4 restart attempts", "bound repeated failures")
	assert.Equal(t, int32(5), attempts.Load(), "initial attempt plus four replacements")
	require.Len(t, messages, 4, "report each retry as degraded")
	for i, delay := range []string{"1ms", "2ms", "3ms", "3ms"} {
		assert.Contains(t, messages[i], "in "+delay+":", "grow backoff and cap it")
	}
	assert.Equal(t, status.Failed, states[len(states)-1], "exhaustion must reach status reporters")
}

func TestOsqueryRecoveryCancellationAndPolicyRemoval(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "remove inputs"}[remove], func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			runner := newOsqueryRunner(logptest.NewTestingLogger(t, "recovery"))
			runner.retryInitial = time.Hour
			degraded := make(chan struct{}, 1)
			idle := make(chan struct{}, 1)
			runner.reportStatus = func(state status.Status, message string) {
				if state == status.Degraded {
					degraded <- struct{}{}
				}
				if strings.Contains(message, "No osquery inputs") {
					idle <- struct{}{}
				}
			}
			var attempts atomic.Int32
			done := make(chan error, 1)
			go func() {
				done <- runner.Run(ctx, func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan []config.InputConfig) error {
					attempts.Add(1)
					return errOsqueryRuntimeExit78
				})
			}()
			require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start collection")
			select {
			case <-degraded:
			case <-time.After(time.Second):
				t.Fatal("runner did not enter backoff")
			}
			if remove {
				require.NoError(t, runner.Update(ctx, nil), "remove policy while waiting")
				select {
				case <-idle:
				case <-time.After(time.Second):
					t.Fatal("removal did not cancel backoff")
				}
			}
			cancel()
			select {
			case err := <-done:
				assert.ErrorIs(t, err, context.Canceled, "shutdown must cancel the timer")
			case <-time.After(time.Second):
				t.Fatal("shutdown waited for backoff")
			}
			assert.Equal(t, int32(1), attempts.Load(), "cancelled backoff must not start a replacement")
		})
	}
}

// Shutdown can race a queued policy, the end of a policy restart, or the end of
// a backoff delay. The policy restart case stops the beat from the runner's
// clock, which it reads after the old run ends. The idle and backoff cases rely
// on select picking the policy or the timer over the cancelled context, so they
// repeat.
func TestOsqueryRecoveryShutdownDoesNotStartReplacement(t *testing.T) {
	for _, name := range []string{"idle", "policy restart", "backoff"} {
		t.Run(name, func(t *testing.T) {
			for range 50 {
				ctx, cancel := context.WithCancel(t.Context())
				runner := newOsqueryRunner(logptest.NewTestingLogger(t, "shutdown"))
				runner.retryInitial = time.Nanosecond
				var stopped, restarting atomic.Bool
				var lateStarts atomic.Int32
				stop := func() {
					stopped.Store(true)
					cancel()
				}
				runner.now = func() time.Time {
					if restarting.Load() {
						stop()
					}
					return time.Now()
				}
				ready := make(chan struct{}, 1)
				runner.reportStatus = func(state status.Status, _ string) {
					if stopped.Load() && state == status.Configuring {
						lateStarts.Add(1)
					}
					switch state {
					case status.Running:
						ready <- struct{}{}
					case status.Degraded:
						if name == "backoff" {
							stop()
						}
					}
				}
				var attempts atomic.Int32
				want := int32(1)
				require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start the original policy")
				if name == "idle" {
					want = 0
					stop()
				}
				err := runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
					if attempts.Add(1) > 1 || name == "idle" {
						return ctx.Err()
					}
					if name == "backoff" {
						return errOsqueryRuntimeExit78
					}
					notifyOsqueryReady(ctx)
					<-ready
					restarting.Store(true)
					if err := runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{Options: map[string]any{"disable_watchdog": true}}}}); err != nil {
						return err
					}
					<-ctx.Done()
					return ctx.Err()
				})
				cancel()
				require.ErrorIs(t, err, context.Canceled, "shutdown must stop the runner")
				require.Equal(t, want, attempts.Load(), "shutdown must not start osqueryd")
				require.Zero(t, lateStarts.Load(), "shutdown must not report a replacement starting")
			}
		})
	}
}

func TestOsqueryRecoveryRemovalCancelsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "recovery"))
	runner.retryInitial = time.Hour
	degraded := make(chan struct{}, 1)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Degraded {
			degraded <- struct{}{}
		}
	}
	var attempts atomic.Int32
	started := make(chan struct{}, 2)
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
			started <- struct{}{}
			if attempts.Add(1) == 1 {
				return errOsqueryRuntimeExit78
			}
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start collection")
	<-started
	select {
	case <-degraded:
	case <-time.After(time.Second):
		t.Fatal("runner did not enter backoff")
	}
	require.NoError(t, runner.Update(ctx, nil), "remove policy while waiting")
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "add a policy again")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("a policy added after removal waited for the abandoned backoff")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled, "stop the new run")
}

func TestOsqueryRecoveryIgnoresReadinessWhileStopping(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "readiness"))
	healthy := make(chan struct{}, 1)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Running {
			select {
			case healthy <- struct{}{}:
			default:
			}
		}
	}
	var attempts atomic.Int32
	restarted := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
			if attempts.Add(1) == 2 {
				close(restarted)
				<-ctx.Done()
				return ctx.Err()
			}
			<-ctx.Done()
			// Readiness of a run being stopped races its done context, so repeat it
			// to make a send reach the runner.
			for range 20 {
				notifyOsqueryReady(ctx)
			}
			assert.Never(t, func() bool { return len(healthy) > 0 }, 100*time.Millisecond, 10*time.Millisecond, "a stopping run marked osqueryd healthy")
			return ctx.Err()
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start the original policy")
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{Options: map[string]any{"disable_watchdog": true}}}}), "change flags to restart osqueryd")
	select {
	case <-restarted:
	case <-time.After(5 * time.Second):
		t.Fatal("osqueryd did not restart for the new flags")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled, "stop the replacement")
}

func TestOsqueryRecoveryUsesLatestPolicyAfterBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "recovery"))
	// Long enough that the updates and the check below finish before the replacement starts.
	runner.retryInitial = time.Second
	degraded := make(chan struct{}, 1)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Degraded {
			degraded <- struct{}{}
		}
	}
	var attempts atomic.Int32
	seen := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig) error {
			n := attempts.Add(1)
			var policy []config.InputConfig
			select {
			case policy = <-inputs:
			case <-ctx.Done():
				return ctx.Err()
			}
			if n == 1 {
				return errOsqueryRuntimeExit78
			}
			seen <- policy[0].ID
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{ID: "original"}}), "start the original policy")
	select {
	case <-degraded:
	case <-time.After(time.Second):
		t.Fatal("no recovery delay")
	}
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{ID: "intermediate"}}), "update during backoff")
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{ID: "latest"}}), "replace pending policy")
	// The input channel holds one update, so this returns once the runner has taken "latest".
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{ID: "latest"}}), "wait for the runner to take the latest policy")
	assert.Never(t, func() bool { return attempts.Load() > 1 }, 200*time.Millisecond, 10*time.Millisecond, "a policy update during backoff started osqueryd early")
	select {
	case id := <-seen:
		assert.Equal(t, "latest", id, "replacement must not resurrect cached inputs")
	case <-time.After(time.Second):
		t.Fatal("replacement did not start")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled, "stop the replacement")
}

func TestOsqueryRecoveryResetsOnlyAfterStableReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "recovery"))
	runner.maxRestarts = 1
	runner.retryInitial = time.Millisecond
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	runner.now = func() time.Time { return time.Unix(0, clock.Load()) }
	ready := make(chan struct{}, 1)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Running {
			ready <- struct{}{}
		}
	}
	var attempts atomic.Int32
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start a policy")
	err := runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
		n := attempts.Add(1)
		if n == 2 {
			notifyOsqueryReady(ctx)
			select {
			case <-ready:
			case <-ctx.Done():
				return ctx.Err()
			}
			clock.Add(int64(runner.stablePeriod))
		}
		return errOsqueryRuntimeExit78
	})
	assert.ErrorContains(t, err, "exhausted 1 restart attempts", "a later unstable run must exhaust its fresh budget")
	assert.Equal(t, int32(3), attempts.Load(), "stable ready run must reset the budget exactly once")
}

func TestOsqueryRecoveryMeasuresStabilityFromFirstReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "recovery"))
	runner.maxRestarts = 1
	runner.retryInitial = time.Millisecond
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	runner.now = func() time.Time { return time.Unix(0, clock.Load()) }
	short := runner.stablePeriod - time.Nanosecond
	ready := make(chan struct{}, 1)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Running {
			ready <- struct{}{}
		}
	}
	readyFor := func(ctx context.Context, d time.Duration) error {
		notifyOsqueryReady(ctx)
		select {
		case <-ready:
		case <-ctx.Done():
			return ctx.Err()
		}
		clock.Add(int64(d))
		return nil
	}
	var attempts atomic.Int32
	terminal := errors.New("unexpected extra attempt")
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start a policy")
	err := runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
		var err error
		switch attempts.Add(1) {
		case 1, 3:
			// Ready, but not for the stable period: the restart counts.
			err = readyFor(ctx, short)
		case 2:
			// Ready again before the stable period ends: stable since the first readiness.
			err = readyFor(ctx, short)
			if err == nil {
				err = readyFor(ctx, runner.stablePeriod-short)
			}
		default:
			return terminal
		}
		if err != nil {
			return err
		}
		return errOsqueryRuntimeExit78
	})
	assert.ErrorContains(t, err, "exhausted 1 restart attempts", "a short ready run must not reset the budget")
	assert.Equal(t, int32(3), attempts.Load(), "only the run that stayed ready for the stable period resets the budget")
}

func TestOsqueryRecoveryIgnoresRetiredReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "readiness"))
	runner.retryInitial = time.Millisecond
	readyCallbacks := make(chan func(), 2)
	healthy := make(chan struct{}, 2)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Running {
			healthy <- struct{}{}
		}
	}
	var attempts atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
			readyCallbacks <- func() { notifyOsqueryReady(ctx) }
			if attempts.Add(1) == 1 {
				return errOsqueryRuntimeExit78
			}
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start the original run")
	var retired, replacement func()
	select {
	case retired = <-readyCallbacks:
	case <-time.After(time.Second):
		t.Fatal("original run did not start")
	}
	select {
	case replacement = <-readyCallbacks:
	case <-time.After(time.Second):
		t.Fatal("replacement did not start")
	}
	// A retired callback races its done context, so repeat it to make a stale
	// send reach the runner.
	for range 20 {
		retired()
	}
	assert.Never(t, func() bool { return len(healthy) > 0 }, 100*time.Millisecond, 10*time.Millisecond, "retired run marked the replacement healthy")
	replacement()
	select {
	case <-healthy:
	case <-time.After(time.Second):
		t.Fatal("replacement readiness did not restore health")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled, "cancel the replacement")
}

func TestOsqueryRecoveryStableBudgetSurvivesPolicyRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "policy_restart"))
	runner.maxRestarts = 1
	runner.retryInitial = time.Millisecond
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	runner.now = func() time.Time { return time.Unix(0, clock.Load()) }
	ready := make(chan struct{}, 1)
	runner.reportStatus = func(state status.Status, _ string) {
		if state == status.Running {
			ready <- struct{}{}
		}
	}
	var attempts atomic.Int32
	terminal := errors.New("invalid replacement configuration")
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{}}), "start the original policy")
	err := runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig) error {
		switch attempts.Add(1) {
		case 1, 3:
			return errOsqueryRuntimeExit78
		case 2:
			notifyOsqueryReady(ctx)
			select {
			case <-ready:
			case <-ctx.Done():
				return ctx.Err()
			}
			clock.Add(int64(runner.stablePeriod))
			if err := runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{Options: map[string]any{"disable_watchdog": true}}}}); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		default:
			return terminal
		}
	})
	assert.ErrorIs(t, err, terminal, "the post-policy-restart runtime failure must receive a fresh retry")
	assert.Equal(t, int32(4), attempts.Load(), "intentional restart must not lose the previous run's stable period")
}

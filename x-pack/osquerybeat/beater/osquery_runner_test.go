// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func waitGroupWithTimeout(ctx context.Context, g *errgroup.Group, to time.Duration) error {

	errCh := make(chan error, 1)

	go func() {
		err := g.Wait()
		errCh <- err
	}()

	ctx, cn := context.WithDeadline(ctx, time.Now().Add(to))
	defer cn()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func waitForStart(ctx context.Context, runCh <-chan struct{}, to time.Duration) error {
	ctx, cn := context.WithDeadline(ctx, time.Now().Add(to))
	defer cn()

	select {
	case <-runCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestOsqueryRunnerCancellable(t *testing.T) {
	to := 10 * time.Second

	parentCtx := context.Background()
	logger := logptest.NewTestingLogger(t, "osquery_runner")

	runCh := make(chan struct{}, 1)

	//nolint:unparam // false positive on returning nil error, need this signature
	runfn := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan runnerInput) error {
		runCh <- struct{}{}
		<-ctx.Done()
		return nil
	}

	ctx, cn := context.WithCancel(parentCtx)
	defer cn()

	g, ctx := errgroup.WithContext(ctx)

	// Start runner
	runner := newOsqueryRunner(logger)
	g.Go(func() error {
		return runner.Run(ctx, runfn)
	})

	// Sent input that will start the runner function
	err := runner.Update(ctx, nil)
	if err != nil {
		t.Fatal("failed runner update:", err)
	}

	// Wait for runner start
	err = waitForStart(ctx, runCh, to)
	if err != nil {
		t.Fatal("failed starting:", err)
	}

	// Cancel
	cn()

	// Wait for runner stop
	er := waitGroupWithTimeout(parentCtx, g, to)
	if er != nil && !errors.Is(er, context.Canceled) {
		t.Fatal("failed running:", er)
	}
}

func TestOsqueryRunnerRestart(t *testing.T) {
	to := 10 * time.Second

	parentCtx := context.Background()
	logger := logptest.NewTestingLogger(t, "osquery_runner")

	runCh := make(chan struct{}, 1)

	var runs int

	//nolint:unparam // false positive on returning nil error, need this signature
	runfn := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan runnerInput) error {
		runs++
		runCh <- struct{}{}
		<-ctx.Done()
		return nil
	}

	ctx, cn := context.WithCancel(parentCtx)
	defer cn()

	g, ctx := errgroup.WithContext(ctx)

	// Start runner
	runner := newOsqueryRunner(logger)
	g.Go(func() error {
		return runner.Run(ctx, runfn)
	})

	// Sent input that will start the runner function
	err := runner.Update(ctx, nil)
	if err != nil {
		t.Fatal("failed runner update:", err)
	}

	// Wait for runner start
	err = waitForStart(ctx, runCh, to)
	if err != nil {
		t.Fatal("failed starting:", err)
	}

	inputConfigs := []config.InputConfig{
		{
			Osquery: &config.OsqueryConfig{
				Options: map[string]any{
					"foo": "bar",
				},
			},
		},
	}

	// Update flags, this should restart the run function
	err = runner.Update(ctx, inputConfigs)
	if err != nil {
		t.Fatal("failed runner update:", err)
	}

	// Should get another run
	err = waitForStart(ctx, runCh, to)
	if err != nil {
		t.Fatal("failed starting after flags update:", err)
	}

	// Update with the same flags, should not restart the runner function
	err = runner.Update(ctx, inputConfigs)
	if err != nil {
		t.Fatal("failed runner update:", err)
	}

	// Should timeout on waiting for another run
	err = waitForStart(ctx, runCh, 300*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected error type after update with the same flags:", err)
	}

	// Cancel
	cn()

	// Wait for runner stop
	er := waitGroupWithTimeout(parentCtx, g, to)
	if er != nil && !errors.Is(er, context.Canceled) {
		t.Fatal("failed running:", er)
	}

	// Check that there were total of 2 executions of run function
	diff := cmp.Diff(2, runs)
	if diff != "" {
		t.Error(diff)
	}
}

func TestOsqueryRunnerRestartOnExtensionsChange(t *testing.T) {
	to := 10 * time.Second

	parentCtx := context.Background()
	logger := logptest.NewTestingLogger(t, "osquery_runner")

	runCh := make(chan struct{}, 1)

	var (
		mx       sync.Mutex
		runs     int
		lastExts config.ExtensionsConfig
	)

	// Drain inputCh like the real runOsquery does, so no-restart updates do not
	// block the runner loop.
	//nolint:unparam // false positive on returning nil error, need this signature
	runfn := func(ctx context.Context, _ osqd.Flags, exts config.ExtensionsConfig, inputCh <-chan runnerInput) error {
		mx.Lock()
		runs++
		lastExts = exts
		mx.Unlock()
		runCh <- struct{}{}
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-inputCh:
			}
		}
	}

	ctx, cn := context.WithCancel(parentCtx)
	defer cn()

	g, ctx := errgroup.WithContext(ctx)

	runner := newOsqueryRunner(logger)
	g.Go(func() error {
		return runner.Run(ctx, runfn)
	})

	withExt := func(ext config.ExtensionsConfig) []config.InputConfig {
		return []config.InputConfig{
			{
				Osquery: &config.OsqueryConfig{
					ElasticOptions: &config.ElasticOptions{
						Extensions: &ext,
					},
				},
			},
		}
	}

	// Initial start with one extension entry.
	if err := runner.Update(ctx, withExt(config.ExtensionsConfig{Paths: []string{"/opt/ext/a"}})); err != nil {
		t.Fatal("failed runner update:", err)
	}
	if err := waitForStart(ctx, runCh, to); err != nil {
		t.Fatal("failed starting:", err)
	}

	// Changing the entry set should restart the runner function.
	if err := runner.Update(ctx, withExt(config.ExtensionsConfig{Paths: []string{"/opt/ext/a", "/opt/ext/b"}})); err != nil {
		t.Fatal("failed runner update:", err)
	}
	if err := waitForStart(ctx, runCh, to); err != nil {
		t.Fatal("failed starting after extensions update:", err)
	}

	// Same entry set should not restart the runner function.
	if err := runner.Update(ctx, withExt(config.ExtensionsConfig{Paths: []string{"/opt/ext/a", "/opt/ext/b"}})); err != nil {
		t.Fatal("failed runner update:", err)
	}
	if err := waitForStart(ctx, runCh, 300*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected error type after update with the same extensions:", err)
	}

	// A timeout-only change should restart the runner function.
	if err := runner.Update(ctx, withExt(config.ExtensionsConfig{Paths: []string{"/opt/ext/a", "/opt/ext/b"}, Timeout: 30})); err != nil {
		t.Fatal("failed runner update:", err)
	}
	if err := waitForStart(ctx, runCh, to); err != nil {
		t.Fatal("failed starting after timeout update:", err)
	}

	// A require-only change should restart the runner function.
	if err := runner.Update(ctx, withExt(config.ExtensionsConfig{Paths: []string{"/opt/ext/a", "/opt/ext/b"}, Timeout: 30, Require: []string{"my_extension"}})); err != nil {
		t.Fatal("failed runner update:", err)
	}
	if err := waitForStart(ctx, runCh, to); err != nil {
		t.Fatal("failed starting after require update:", err)
	}

	cn()

	er := waitGroupWithTimeout(parentCtx, g, to)
	if er != nil && !errors.Is(er, context.Canceled) {
		t.Fatal("failed running:", er)
	}

	mx.Lock()
	defer mx.Unlock()
	if diff := cmp.Diff(4, runs); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff([]string{"/opt/ext/a", "/opt/ext/b"}, lastExts.Paths); diff != "" {
		t.Errorf("extensions not propagated to runfn: %s", diff)
	}
	if diff := cmp.Diff([]string{"my_extension"}, lastExts.Require); diff != "" {
		t.Errorf("extensions require not propagated to runfn: %s", diff)
	}
}

func TestOsqueryRunnerRecoversFromTransientErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "extension ping timeout",
			err:  errors.New("extension ping failed: timeout after 200ms"),
		},
		{
			name: "broken pipe",
			err:  errors.New("write: broken pipe"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			started := make(chan struct{}, 2)
			var runs atomic.Int32
			runfn := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan runnerInput) error {
				run := runs.Add(1)
				started <- struct{}{}
				if run == 1 {
					return tc.err
				}
				<-ctx.Done()
				return nil
			}

			g, ctx := errgroup.WithContext(ctx)
			runner := newOsqueryRunner(logptest.NewTestingLogger(t, "osquery_runner"))
			g.Go(func() error {
				return runner.Run(ctx, runfn)
			})

			inputs := []config.InputConfig{{Osquery: &config.OsqueryConfig{}}}
			require.NoError(t, runner.Update(ctx, inputs), "failed to start osquery runner")
			require.NoError(t, waitForStart(ctx, started, 10*time.Second), "first osquery run did not start")
			require.NoError(t, waitForStart(ctx, started, 10*time.Second), "osquery did not restart after %v", tc.err)

			cancel()
			err := waitGroupWithTimeout(t.Context(), g, 10*time.Second)
			require.ErrorIs(t, err, context.Canceled, "runner returned an unexpected error after recovery")
			assert.GreaterOrEqual(t, runs.Load(), int32(2), "recoverable error did not restart osquery")
		})
	}
}

func TestOsqueryRunnerReturnsUnrecoverableError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	wantErr := errors.New("invalid configuration")
	started := make(chan struct{}, 1)
	runfn := func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan runnerInput) error {
		started <- struct{}{}
		return wantErr
	}

	g, ctx := errgroup.WithContext(ctx)
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "osquery_runner"))
	g.Go(func() error {
		return runner.Run(ctx, runfn)
	})

	inputs := []config.InputConfig{{Osquery: &config.OsqueryConfig{}}}
	require.NoError(t, runner.Update(ctx, inputs), "failed to start osquery runner")
	require.NoError(t, waitForStart(ctx, started, 10*time.Second), "osquery run did not start")

	err := waitGroupWithTimeout(t.Context(), g, 10*time.Second)
	require.ErrorIs(t, err, wantErr, "runner did not return the unrecoverable error")
}

func TestOsqueryRunnerBackoffKeepsLatestConfigurationAndStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	runner.retryInitial = 100 * time.Millisecond
	runner.retryMax = 200 * time.Millisecond
	reports := make(chan status.Status, 10)
	runner.reportStatus = func(s status.Status, _ string) { reports <- s }
	second := make(chan runnerInput, 1)
	releaseReady := make(chan struct{})
	var runs atomic.Int32
	exitErr := osqueryExitError(t, 78)
	run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan runnerInput) error {
		if runs.Add(1) == 1 {
			return fmt.Errorf("osqueryd: %w", exitErr)
		}
		select {
		case input := <-inputs:
			second <- input
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-releaseReady:
			notifyOsqueryReady(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
		<-ctx.Done()
		return ctx.Err()
	}
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx, run) }()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{Name: "old"}}), "first run must start")
	require.Equal(t, status.Configuring, <-reports, "initial start must be configuring")
	require.Equal(t, status.Degraded, <-reports, "recoverable exit must report degradation")
	require.NoError(t, runner.updateGeneration(ctx, []config.InputConfig{{Name: "latest"}}, 42), "policy update must remain available during backoff")
	select {
	case input := <-second:
		assert.Equal(t, "latest", input.inputs[0].Name, "restart must use the latest policy")
		assert.Equal(t, uint64(42), input.generation, "restart must preserve the policy generation")
	case <-time.After(time.Second):
		t.Fatal("osquery was not restarted")
	}
	select {
	case s := <-reports:
		t.Errorf("status changed before readiness: %v", s)
	default:
	}
	close(releaseReady)
	select {
	case s := <-reports:
		assert.Equal(t, status.Running, s, "running must follow successful configuration application")
	case <-time.After(time.Second):
		t.Fatal("ready run did not restore running status")
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "shutdown must cancel the runner")
	case <-time.After(time.Second):
		t.Fatal("runner did not shut down")
	}
}

func TestOsqueryRunnerRepeatedCrashesBackOff(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	runner.retryInitial = 20 * time.Millisecond
	runner.retryMax = 80 * time.Millisecond
	started := make(chan time.Time, 4)
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(context.Context, osqd.Flags, config.ExtensionsConfig, <-chan runnerInput) error {
			started <- time.Now()
			return errors.New("write: broken pipe")
		})
	}()
	require.NoError(t, runner.Update(ctx, nil), "runner must start")
	var previous time.Time
	for i, minimum := range []time.Duration{0, 15 * time.Millisecond, 30 * time.Millisecond, 60 * time.Millisecond} {
		select {
		case now := <-started:
			if i > 0 {
				assert.GreaterOrEqual(t, now.Sub(previous), minimum, "repeated failures must increase restart delay")
			}
			previous = now
		case <-time.After(time.Second):
			t.Fatal("backoff run did not start")
		}
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "backoff must be cancellable")
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for the retry timer")
	}
}

func TestOsqueryRunnerIgnoresReadinessFromPreviousRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	reports := make(chan status.Status, 10)
	runner.reportStatus = func(s status.Status, _ string) { reports <- s }
	firstReady := make(chan func(), 1)
	second := make(chan struct{})
	secondReady := make(chan func(), 1)
	var runs atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan runnerInput) error {
			if runs.Add(1) == 1 {
				firstReady <- ctx.Value(osqueryReadyKey{}).(func())
			} else {
				secondReady <- ctx.Value(osqueryReadyKey{}).(func())
				close(second)
			}
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-inputs:
				}
			}
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{Options: map[string]any{"thrift_timeout": 3}}}}), "first run must start")
	staleReady := <-firstReady
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{Options: map[string]any{"thrift_timeout": 4}}}}), "changed options must restart osquery")
	select {
	case <-second:
	case <-time.After(time.Second):
		t.Fatal("replacement run did not start")
	}
	staleReady()
	time.Sleep(20 * time.Millisecond)
	for len(reports) > 0 {
		assert.NotEqual(t, status.Running, <-reports, "old readiness must not mark replacement healthy")
	}
	(<-secondReady)()
	select {
	case s := <-reports:
		assert.Equal(t, status.Running, s, "current readiness must not be lost behind stale notification")
	case <-time.After(time.Second):
		t.Fatal("current readiness notification was lost")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runner did not stop")
	}
	close(reports)
	for s := range reports {
		assert.NotEqual(t, status.Running, s, "old run readiness must not mark the replacement healthy")
	}
}

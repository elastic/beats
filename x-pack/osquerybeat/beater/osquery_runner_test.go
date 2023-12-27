// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp"
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
	logger := logp.NewLogger("osquery_runner")

	runCh := make(chan struct{}, 1)

	//nolint:unparam // false positive on returning nil error, need this signature
	runfn := func(ctx context.Context, _ osqd.Flags, _ <-chan []config.InputConfig) error {
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
	logger := logp.NewLogger("osquery_runner")

	runCh := make(chan struct{}, 1)

	var runs int

	//nolint:unparam // false positive on returning nil error, need this signature
	runfn := func(ctx context.Context, _ osqd.Flags, _ <-chan []config.InputConfig) error {
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
				Options: map[string]interface{}{
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

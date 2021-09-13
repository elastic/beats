// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package runner

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/sync/errgroup"
)

func simpleRun(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestRunnerAlreadyRunning(t *testing.T) {

	ctx := context.Background()
	runner := New()

	g, ctx := errgroup.WithContext(ctx)

	// Attempt to run again should result in error
	g.Go(func() error {
		return runner.Run(ctx, simpleRun)
	})

	g.Go(func() error {
		return runner.Run(ctx, simpleRun)
	})

	err := g.Wait()
	diff := cmp.Diff(ErrAlreadyRunning, err, cmpopts.EquateErrors())
	if diff != "" {
		t.Fatal(diff)
	}
}

func TestRunnerStop(t *testing.T) {
	ctx := context.Background()
	runner := New()

	g, ctx := errgroup.WithContext(ctx)

	// signal that it's running
	var isRunning sync.WaitGroup

	isRunning.Add(1)
	g.Go(func() error {
		return runner.Run(ctx, func(ctx context.Context) error {
			isRunning.Done()
			<-ctx.Done()
			return ctx.Err()
		})
	})

	isRunning.Wait()

	runner.Stop()

	err := g.Wait()

	diff := cmp.Diff(context.Canceled, err, cmpopts.EquateErrors())
	if diff != "" {
		t.Fatal(diff)
	}
}

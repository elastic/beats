// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"context"
	"fmt"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/functionbeat/function/telemetry"
)

// Runner is the interface that the coordinator will follow to manage a function goroutine.
type Runner interface {
	fmt.Stringer
	Run(context.Context, telemetry.T) error
}

// Coordinator takes care of managing the function goroutine, it receives the list of functions that
// need to be executed and manage the goroutine.  If an error happen and its not handled by the
// function, we assume its a fatal error and we will
// stop all the other goroutine and functionbeat will terminate.
type Coordinator struct {
	log     *logp.Logger
	runners []Runner
}

// NewCoordinator create a new coordinator objects receiving the clientFactory and the runner.
func NewCoordinator(log *logp.Logger,
	runners ...Runner,
) *Coordinator {
	if log == nil {
		log = logp.NewLogger("")
	}
	log = log.Named("Coordinator")
	return &Coordinator{log: log, runners: runners}
}

// Run starts each functions into an independent goroutine and wait until all the goroutine are
// stopped to exit.
func (r *Coordinator) Run(ctx context.Context, t telemetry.T) error {
	r.log.Debug("Coordinator is starting")
	defer r.log.Debug("Coordinator is stopped")

	// When an errors happen in a function and its not handled by the running function, we log an error
	// and we trigger a shutdown of all the others goroutine.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan error)
	defer close(results)

	r.log.Debugf("The coordinator is starting %d functions", len(r.runners))
	for _, rfn := range r.runners {
		go func(ctx context.Context, t telemetry.T, rfn Runner) {
			var err error
			defer func() { results <- err }()
			err = r.runFunc(ctx, t, rfn)
			if err != nil {
				cancel()
			}
		}(ctx, t, rfn)
	}

	// Wait for goroutine to complete and aggregate any errors from the goroutine and
	// raise them back to the main program.
	var errors multierror.Errors
	for range r.runners {
		err := <-results
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors.Err()
}

func (r *Coordinator) runFunc(
	ctx context.Context,
	t telemetry.T,
	rfn Runner,
) error {
	r.log.Infof("The function '%s' is starting", rfn.String())
	defer r.log.Infof("The function '%s' is stopped", rfn.String())

	err := rfn.Run(ctx, t)
	if err != nil {
		r.log.Errorf(
			"Nonrecoverable error when executing the function: '%s', error: '%+v', terminating all running functions",
			rfn,
			err,
		)
	}
	return err
}

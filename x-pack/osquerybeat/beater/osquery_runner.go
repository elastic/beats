// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"sync"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp"
)

type osqueryRunner struct {
	log     *logp.Logger
	inputCh chan []config.InputConfig
}

func newOsqueryRunner(log *logp.Logger) *osqueryRunner {
	r := &osqueryRunner{
		log:     log,
		inputCh: make(chan []config.InputConfig, 1),
	}
	return r
}

type osqueryRunFunc func(ctx context.Context, flags osqd.Flags, inputCh <-chan []config.InputConfig) error

// Run manages osqueryd lifecycle, processes inputs changes, restarts osquery if needed
func (r *osqueryRunner) Run(parentCtx context.Context, runfn osqueryRunFunc) error {
	var (
		flags osqd.Flags

		ctx context.Context
		cn  context.CancelFunc
		wg  sync.WaitGroup

		inputCh chan []config.InputConfig
	)

	var mx sync.Mutex
	cancel := func() {
		mx.Lock()
		if cn != nil {
			cn()
			cn = nil
		}
		defer mx.Unlock()
	}

	// Cleanup on exit
	defer cancel()

	errCh := make(chan error, 1)

	// lastKnownInputs is used for recovery after "broken pipe" error
	var lastKnownInputs []config.InputConfig

	process := func(inputs []config.InputConfig) {
		lastKnownInputs = inputs
		newFlags := config.GetOsqueryOptions(inputs)

		// If Osqueryd is running and flags are different: stop osquery
		if cn != nil && !osqd.FlagsAreSame(flags, newFlags) {
			r.log.Info("Osquery is running and options changed, stop osqueryd")

			// Cancel context
			cancel()

			// Wait until osquery runner exists
			wg.Wait()
		}

		// Set the flags to use
		flags = newFlags

		// Start osqueryd if not running
		if cn == nil {
			r.log.Info("Start osqueryd")
			inputCh = make(chan []config.InputConfig, 1)
			ctx, cn = context.WithCancel(parentCtx)

			wg.Add(1)
			go func() {
				defer wg.Done()
				err := runfn(ctx, flags, inputCh)

				// Reset cancellable
				cancel()

				// Forward error to main loop
				r.log.Debugf("Forward osquery run error to the main runner loop: %v", err)
				errCh <- err
			}()
		}

		select {
		case inputCh <- inputs:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case inputs := <-r.inputCh:
			r.log.Debug("Got configuration update")
			process(inputs)
		case err := <-errCh:
			if err == nil || errors.Is(err, context.Canceled) {
				r.log.Info("Osquery exited: ", err)
			} else {
				r.log.Error("Failed to run osquery:", err)
				if isBrokenPipeOrEOFError(err) {
					r.log.Infof("Recover osquery after broken pipe error")
					if lastKnownInputs != nil {
						select {
						case r.inputCh <- lastKnownInputs:
						case <-parentCtx.Done():
							return parentCtx.Err()
						}
					}
				} else {
					return err
				}
			}
		case <-parentCtx.Done():
			return parentCtx.Err()
		}
	}
}

func (r *osqueryRunner) Update(ctx context.Context, inputs []config.InputConfig) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.inputCh <- inputs:
	}
	return nil
}

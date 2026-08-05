// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.uber.org/zap/zapcore"

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

type osqueryRunFunc func(ctx context.Context, flags osqd.Flags, extensions config.ExtensionsConfig, inputCh <-chan []config.InputConfig) error

// Run manages osqueryd lifecycle, processes inputs changes, restarts osquery if needed
func (r *osqueryRunner) Run(parentCtx context.Context, runfn osqueryRunFunc) error {
	var (
		flags      osqd.Flags
		extensions config.ExtensionsConfig

		ctx context.Context
		cn  context.CancelFunc
		wg  sync.WaitGroup

		inputCh chan []config.InputConfig
	)

	var mx sync.Mutex
	cancel := func() {
		mx.Lock()
		defer mx.Unlock()
		if cn != nil {
			cn()
			cn = nil
		}
	}

	// Cleanup on exit: cancel child context first, then wait for the runfn
	// goroutine to exit. The order matters — cancel must run before wg.Wait
	// so the goroutine sees context cancellation and exits promptly.
	defer wg.Wait()
	defer cancel()

	errCh := make(chan error, 1)

	// lastKnownInputs is used for recovery after "broken pipe" error
	var lastKnownInputs []config.InputConfig

	logLevel := zapcore.LevelOf(r.log.Core())

	process := func(inputs []config.InputConfig) {
		lastKnownInputs = inputs
		newFlags := config.GetOsqueryOptions(inputs)
		newExtensions := config.GetOsqueryExtensions(inputs)
		newLogLevel := zapcore.LevelOf(r.log.Core())

		// cn is cleared by the spawned goroutine's cancel() on exit, so guard it with mx.
		mx.Lock()
		running := cn != nil
		mx.Unlock()

		// If Osqueryd is running and flags, log level, or the customer-managed extensions
		// changed: stop osquery so it is restarted with the new autoload file.
		if running && (!osqd.FlagsAreSame(flags, newFlags) || logLevel != newLogLevel || !extensionsAreSame(extensions, newExtensions)) {
			r.log.Info("Osquery is running and options changed, stop osqueryd")

			// Cancel context
			cancel()

			// Wait until osquery runner exits
			wg.Wait()
		}

		mx.Lock()
		// Start osqueryd if not running
		if cn == nil {
			r.log.Info("Start osqueryd")

			flags = newFlags
			extensions = newExtensions
			logLevel = newLogLevel
			inputCh = make(chan []config.InputConfig, 1)
			ctx, cn = context.WithCancel(parentCtx) //nolint:gosec // G118: cn is stored and invoked via the cancel() helper

			wg.Go(func() {
				err := runfn(ctx, flags, extensions, inputCh)

				// Reset cancellable
				cancel()

				// Forward error to main loop
				r.log.Debugf("Forward osquery run error to the main runner loop: %v", err)
				errCh <- err
			})
		}
		mx.Unlock()

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

// extensionsAreSame reports whether two customer-managed extension configurations
// are equivalent (same ordered paths, required names, and timeout), so the runner
// only restarts osqueryd when the extension configuration actually changes.
func extensionsAreSame(a, b config.ExtensionsConfig) bool {
	if a.Timeout != b.Timeout {
		return false
	}
	if !slices.Equal(a.Paths, b.Paths) {
		return false
	}
	return slices.Equal(a.Require, b.Require)
}

func (r *osqueryRunner) Update(ctx context.Context, inputs []config.InputConfig) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.inputCh <- inputs:
	}
	return nil
}

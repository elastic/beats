// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/elastic/beats/v7/libbeat/management/status"

	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp"
)

type osqueryRunner struct {
	log                                  *logp.Logger
	inputCh                              chan runnerInput
	retryInitial, retryMax, stablePeriod time.Duration
	reportStatus                         func(status.Status, string)
}

func newOsqueryRunner(log *logp.Logger) *osqueryRunner {
	r := &osqueryRunner{
		log:          log,
		inputCh:      make(chan runnerInput, 1),
		retryInitial: time.Second, retryMax: 30 * time.Second, stablePeriod: time.Minute,
	}
	return r
}

type osqueryRunFunc func(ctx context.Context, flags osqd.Flags, extensions config.ExtensionsConfig, inputCh <-chan runnerInput) error

// Run manages osqueryd lifecycle, processes inputs changes, restarts osquery if needed
func (r *osqueryRunner) Run(parentCtx context.Context, runfn osqueryRunFunc) error {
	var (
		latest     *runnerInput
		flags      osqd.Flags
		extensions config.ExtensionsConfig
		logLevel   zapcore.Level
		cancel     context.CancelFunc
		completed  <-chan error
		inputs     chan runnerInput
		pending    *runnerInput
		stopping   bool
		recovering bool
		runID      uint64
		readyAt    time.Time
		retry      *time.Timer
		retryCh    <-chan time.Time
	)
	ready := make(chan uint64, 1)
	backoff := r.retryInitial
	report := func(s status.Status, message string) {
		if r.reportStatus != nil {
			r.reportStatus(s, message)
		}
	}
	stopRetry := func() {
		if retry != nil {
			retry.Stop()
			retry = nil
			retryCh = nil
		}
	}
	defer stopRetry()
	defer func() {
		if cancel != nil {
			cancel()
			<-completed
		}
	}()
	start := func() {
		stopRetry()
		flags = config.GetOsqueryOptions(latest.inputs)
		extensions = config.GetOsqueryExtensions(latest.inputs)
		logLevel = zapcore.LevelOf(r.log.Core())
		runID++
		id := runID
		childCtx, childCancel := context.WithCancel(parentCtx)
		cancel = childCancel
		childCtx = context.WithValue(childCtx, osqueryReadyKey{}, func() {
			if childCtx.Err() != nil {
				return
			}
			select {
			case ready <- id:
			case <-childCtx.Done():
			}
		})
		inputs = make(chan runnerInput, 1)
		pending = latest
		done := make(chan error, 1)
		completed = done
		readyAt = time.Time{}
		stopping = false
		if !recovering {
			report(status.Configuring, "Starting osqueryd")
		}
		r.log.Info("Start osqueryd")
		// Capture per-run values; configuration can change while teardown completes.
		runFlags, runExtensions, runInputs := flags, extensions, inputs
		go func() { done <- runfn(childCtx, runFlags, runExtensions, runInputs) }()
	}
	for {
		var send chan runnerInput
		var next runnerInput
		if pending != nil && !stopping {
			send = inputs
			next = *pending
		}
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		case input := <-r.inputCh:
			latest = &input
			if completed == nil {
				if retryCh == nil {
					start()
				}
				continue
			}
			newFlags := config.GetOsqueryOptions(input.inputs)
			newExtensions := config.GetOsqueryExtensions(input.inputs)
			if !osqd.FlagsAreSame(flags, newFlags) || !extensionsAreSame(extensions, newExtensions) || logLevel != zapcore.LevelOf(r.log.Core()) {
				stopping = true
				pending = nil
				cancel()
			} else {
				pending = latest
			}
		case send <- next:
			pending = nil
		case id := <-ready:
			if id == runID && completed != nil && !stopping {
				if readyAt.IsZero() {
					readyAt = time.Now()
				}
				recovering = false
				report(status.Running, "Osqueryd configuration applied")
			}
		case err := <-completed:
			cancel()
			cancel = nil
			completed = nil
			inputs = nil
			pending = nil
			if stopping {
				start()
				continue
			}
			if parentCtx.Err() != nil {
				return parentCtx.Err()
			}
			if err == nil || errors.Is(err, context.Canceled) {
				r.log.Info("Osquery exited: ", err)
				continue
			}
			if !isRecoverableOsqueryError(err) {
				return err
			}
			if !readyAt.IsZero() && time.Since(readyAt) >= r.stablePeriod {
				backoff = r.retryInitial
			}
			recovering = true
			report(status.Degraded, "Osqueryd exited; restarting: "+err.Error())
			r.log.Warnf("Restart osqueryd after recoverable error in %s: %v", backoff, err)
			retry = time.NewTimer(backoff)
			retryCh = retry.C
			backoff = min(backoff*2, r.retryMax)
		case <-retryCh:
			// Updates received during backoff replace latest, without restarting early.
			start()
		}
	}
}

type osqueryReadyKey struct{}

func notifyOsqueryReady(ctx context.Context) {
	if ready, ok := ctx.Value(osqueryReadyKey{}).(func()); ok {
		ready()
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

type runnerInput struct {
	inputs     []config.InputConfig
	generation uint64
}

func (r *osqueryRunner) Update(ctx context.Context, inputs []config.InputConfig) error {
	return r.updateGeneration(ctx, inputs, 0)
}

func (r *osqueryRunner) updateGeneration(ctx context.Context, inputs []config.InputConfig, generation uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.inputCh <- runnerInput{inputs: inputs, generation: generation}:
	}
	return nil
}

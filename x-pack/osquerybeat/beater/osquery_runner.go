// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
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
	inputCh                              chan []config.InputConfig
	retryInitial, retryMax, stablePeriod time.Duration
	reportStatus                         func(status.Status, string)
	maxRestarts                          int
	now                                  func() time.Time
}

func newOsqueryRunner(log *logp.Logger) *osqueryRunner {
	r := &osqueryRunner{
		log:          log,
		now:          time.Now,
		inputCh:      make(chan []config.InputConfig, 1),
		retryInitial: time.Second,
		retryMax:     30 * time.Second,
		stablePeriod: 5 * time.Minute,
		maxRestarts:  5,
	}
	return r
}

type osqueryRunFunc func(ctx context.Context, flags osqd.Flags, extensions config.ExtensionsConfig, inputCh <-chan []config.InputConfig) error

// Run manages osqueryd lifecycle, processes inputs changes, restarts osquery if needed
func (r *osqueryRunner) Run(parentCtx context.Context, runfn osqueryRunFunc) error {
	var (
		latest     *[]config.InputConfig
		flags      osqd.Flags
		extensions config.ExtensionsConfig
		logLevel   zapcore.Level
		cancel     context.CancelFunc
		completed  <-chan error
		inputs     chan []config.InputConfig
		pending    *[]config.InputConfig
		stopping   bool
		recovering bool
		runID      uint64
		readyAt    time.Time
		retry      *time.Timer
		retryCh    <-chan time.Time
	)
	ready := make(chan uint64, 1)
	backoff := r.retryInitial
	restarts := 0
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
		// Shutdown can lose a select race to a queued policy, a finished run,
		// or the retry timer; the loop then returns on parentCtx.Done.
		if parentCtx.Err() != nil {
			return
		}
		stopRetry()
		if latest == nil || len(*latest) == 0 {
			recovering = false
			report(status.Running, "No osquery inputs configured")
			return
		}
		flags = config.GetOsqueryOptions(*latest)
		extensions = config.GetOsqueryExtensions(*latest)
		logLevel = zapcore.LevelOf(r.log.Core())
		runID++
		id := runID
		childCtx, childCancel := context.WithCancel(parentCtx)
		cancel = childCancel
		childCtx = context.WithValue(childCtx, osqueryReadyKey{}, func() {
			select {
			case ready <- id:
			case <-childCtx.Done():
			}
		})
		inputs = make(chan []config.InputConfig, 1)
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
		go func() {
			defer childCancel()
			done <- runfn(childCtx, runFlags, runExtensions, runInputs)
		}()
	}
	for {
		var send chan []config.InputConfig
		var next []config.InputConfig
		if pending != nil && !stopping {
			send = inputs
			next = *pending
		}
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		case input := <-r.inputCh:
			latest = &input
			if len(input) == 0 {
				recovering = false
				stopRetry()
				pending = nil
				if cancel != nil {
					stopping = true
					cancel()
				}
				report(status.Running, "No osquery inputs configured")
				continue
			}
			if completed == nil {
				if retryCh == nil {
					start()
				}
				continue
			}
			newFlags := config.GetOsqueryOptions(input)
			newExtensions := config.GetOsqueryExtensions(input)
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
					readyAt = r.now()
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
			if !readyAt.IsZero() && r.now().Sub(readyAt) >= r.stablePeriod {
				backoff = r.retryInitial
				restarts = 0
			}
			if stopping {
				if latest != nil && len(*latest) > 0 {
					start()
				}
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
				report(status.Failed, err.Error())
				return err
			}
			if restarts >= r.maxRestarts {
				failure := fmt.Errorf("osqueryd exhausted %d restart attempts: %w", r.maxRestarts, err)
				report(status.Failed, failure.Error())
				return failure
			}
			restarts++
			recovering = true
			report(status.Degraded, fmt.Sprintf("Osqueryd restart %d/%d in %s: %v", restarts, r.maxRestarts, backoff, err))
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

func (r *osqueryRunner) Update(ctx context.Context, inputs []config.InputConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.inputCh <- inputs:
	}
	return nil
}

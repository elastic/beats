// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp"
)

type osqueryRunner struct {
	log                                  *logp.Logger
	inputCh                              chan osqueryInputUpdate
	done                                 chan struct{}
	runErr                               error // Read only after done closes.
	retryInitial, retryMax, stablePeriod time.Duration
	maxRetries                           int
	reportStatus                         func(status.Status, string)
	reportWaiting                        func()
}

type osqueryInputUpdate struct {
	inputs  []config.InputConfig
	applied chan struct{}
}

func newOsqueryRunner(log *logp.Logger) *osqueryRunner {
	return &osqueryRunner{
		log:          log,
		inputCh:      make(chan osqueryInputUpdate, 1),
		done:         make(chan struct{}),
		retryInitial: time.Second,
		retryMax:     30 * time.Second,
		stablePeriod: time.Minute,
		maxRetries:   5,
	}
}

type osqueryRunFunc func(ctx context.Context, flags osqd.Flags, extensions config.ExtensionsConfig, inputCh <-chan []config.InputConfig, ready, exited func()) error

// Run manages osqueryd lifecycle and keeps configuration updates responsive during recovery.
func (r *osqueryRunner) Run(parentCtx context.Context, runfn osqueryRunFunc) (runErr error) {
	defer func() {
		r.runErr = runErr
		close(r.done)
	}()
	var (
		latest     []config.InputConfig
		flags      osqd.Flags
		extensions config.ExtensionsConfig
		logLevel   zapcore.Level
		cancel     context.CancelFunc
		completed  <-chan error
		inputs     chan []config.InputConfig
		pending    []config.InputConfig
		stopping   bool
		recovering bool
		ready      <-chan time.Time
		readyAt    time.Time
		exited     <-chan time.Time
		exitedAt   time.Time
		retry      *time.Timer
		retries    int
		retryErr   error // Non-nil while a recovery retry budget is active.
	)
	backoff := r.retryInitial
	report := func(s status.Status, message string) {
		if r.reportStatus != nil {
			r.reportStatus(s, message)
		}
	}
	waiting := func() {
		if r.reportWaiting != nil {
			r.reportWaiting()
		} else {
			report(status.Configuring, "Waiting for osquery input configuration")
		}
	}
	stopRetry := func() {
		if retry != nil {
			retry.Stop()
			retry = nil
		}
	}
	retryChannel := func() <-chan time.Time {
		if retry == nil {
			return nil
		}
		return retry.C
	}
	defer stopRetry()
	defer func() {
		if cancel != nil {
			cancel()
			<-completed
		}
	}()
	start := func() bool {
		stopRetry()
		// Prefer an update already accepted by Update even when the timer wins select.
		select {
		case update := <-r.inputCh:
			latest = update.inputs
			defer close(update.applied)
		default:
		}
		if len(latest) == 0 {
			recovering = false
			waiting()
			return false
		}
		flags = config.GetOsqueryOptions(latest)
		extensions = config.GetOsqueryExtensions(latest)
		logLevel = zapcore.LevelOf(r.log.Core())
		childCtx, childCancel := context.WithCancel(parentCtx)
		cancel = childCancel
		runReady := make(chan time.Time, 1)
		ready = runReady
		runExited := make(chan time.Time, 1)
		exited = runExited
		var readyOnce, exitedOnce sync.Once
		notifyExited := func() { exitedOnce.Do(func() { runExited <- time.Now() }) }
		notifyReady := func() {
			if childCtx.Err() == nil {
				readyOnce.Do(func() { runReady <- time.Now() })
			}
		}
		inputs = make(chan []config.InputConfig, 1)
		pending = latest
		done := make(chan error, 1)
		completed = done
		readyAt, exitedAt = time.Time{}, time.Time{}
		stopping = false
		if !recovering {
			report(status.Configuring, "Starting osqueryd")
		}
		r.log.Info("Start osqueryd")
		// Each callback and completion belongs to this run, including late notifications.
		runFlags, runExtensions, runInputs := flags, extensions, inputs
		go func() {
			defer childCancel()
			err := runfn(childCtx, runFlags, runExtensions, runInputs, notifyReady, notifyExited)
			notifyExited()
			done <- err
		}()
		return true
	}
	observeExit := func() {
		select {
		case exitedAt = <-exited:
			exited = nil
		default:
		}
	}
	observeReady := func() {
		if readyAt.IsZero() {
			select {
			case readyAt = <-ready:
				ready = nil
			default:
			}
		}
		if !exitedAt.IsZero() && readyAt.After(exitedAt) {
			readyAt = time.Time{}
		}
	}
	reportExit := func() {
		if !recovering && !readyAt.IsZero() && !exitedAt.IsZero() && !stopping && parentCtx.Err() == nil {
			recovering = true
			report(status.Degraded, "Osqueryd exited; recovering")
		}
	}
	scheduleRetry := func() error {
		if retries >= r.maxRetries {
			return fmt.Errorf("osqueryd restart limit exceeded after %d retries: %w", retries, retryErr)
		}
		if !recovering || readyAt.IsZero() {
			report(status.Degraded, "Osqueryd exited; restarting: "+retryErr.Error())
		}
		recovering = true
		r.log.Warnf("Restart osqueryd after recoverable error in %s (attempt %d/%d): %v", backoff, retries+1, r.maxRetries, retryErr)
		retry = time.NewTimer(backoff)
		return nil
	}
	for {
		var send chan []config.InputConfig
		if pending != nil && !stopping {
			send = inputs
		}
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		case update := <-r.inputCh:
			latest = update.inputs
			if completed != nil {
				// An exit can be buffered while its logger cleanup is still running.
				// A later policy must not turn that failure into an intentional restart.
				observeExit()
				if !exitedAt.IsZero() {
					observeReady()
					reportExit()
					pending = nil
					close(update.applied)
					continue
				}
			}
			if len(latest) == 0 {
				stopRetry()
				pending = nil
				recovering = false
				if completed != nil {
					stopping = true
					report(status.Configuring, "Stopping osqueryd without inputs")
					cancel()
				} else {
					waiting()
				}
				close(update.applied)
				continue
			}
			if completed == nil {
				if retry == nil {
					if retryErr != nil {
						if err := scheduleRetry(); err != nil {
							close(update.applied)
							return err
						}
					} else {
						start()
					}
				}
				close(update.applied)
				continue
			}
			newFlags := config.GetOsqueryOptions(latest)
			newExtensions := config.GetOsqueryExtensions(latest)
			if !osqd.FlagsAreSame(flags, newFlags) || !extensionsAreSame(extensions, newExtensions) || logLevel != zapcore.LevelOf(r.log.Core()) {
				stopping = true
				pending = nil
				if !recovering {
					report(status.Configuring, "Restarting osqueryd for updated configuration")
				}
				cancel()
			} else if !stopping {
				pending = latest
			}
			close(update.applied)
		case send <- pending:
			pending = nil
		case readyAt = <-ready:
			ready = nil
			observeExit()
			observeReady()
			reportExit()
			if !stopping && exitedAt.IsZero() && parentCtx.Err() == nil {
				recovering = false
				report(status.Running, "Osqueryd configuration applied")
			}
		case exitedAt = <-exited:
			exited = nil
			observeReady()
			reportExit()
		case err := <-completed:
			// Observe both buffered lifecycle notifications before classifying the error.
			observeExit()
			observeReady()
			reportExit()
			// Teardown time must not count as healthy operation, including on policy restarts.
			if !readyAt.IsZero() && exitedAt.Sub(readyAt) >= r.stablePeriod {
				retries, backoff = 0, r.retryInitial
				retryErr = nil
			}
			cancel()
			cancel, completed, inputs, ready, exited = nil, nil, nil, nil, nil
			pending = nil
			if parentCtx.Err() != nil {
				return parentCtx.Err()
			}
			if stopping {
				if len(latest) > 0 {
					if retryErr != nil {
						if err := scheduleRetry(); err != nil {
							return err
						}
					} else {
						start()
					}
				} else {
					waiting()
				}
				continue
			}
			if err == nil || errors.Is(err, context.Canceled) {
				retryErr = nil
				r.log.Info("Osquery exited: ", err)
				continue
			}
			// Exit 78 also means invalid startup configuration; only recover after readiness.
			if !isRecoverableOsqueryError(err) && (readyAt.IsZero() || !isOsqueryExit78(err)) {
				return err
			}
			// Keep the failure while inputs are absent so re-addition is a counted retry.
			retryErr = err
			if len(latest) == 0 {
				recovering = false
				waiting()
				continue
			}
			if err := scheduleRetry(); err != nil {
				return err
			}
		case <-retryChannel():
			// Policies received during backoff replace latest without bypassing the delay.
			if start() {
				retries++
				backoff = min(backoff*2, r.retryMax)
			}
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
	if err := ctx.Err(); err != nil {
		return err
	}
	update := osqueryInputUpdate{inputs: inputs, applied: make(chan struct{})}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return r.runErr
	case r.inputCh <- update:
	}
	// Reconciliation creates input status reporters immediately after Update.
	// Wait for the lifecycle decision so they cannot inherit an old run's readiness.
	// This does not wait for startup, backoff, or the previous run's cleanup.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		select {
		case <-update.applied:
			return nil
		default:
			return r.runErr
		}
	case <-update.applied:
		return nil
	}
}

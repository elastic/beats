// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/rcrowley/go-metrics"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

// SyncLoop schedules the full syncs and incremental updates of a provider
// that does not use minimal state.
type SyncLoop struct {
	Logger  *logp.Logger
	Metrics SyncMetrics

	// SyncInterval and UpdateInterval are the configured intervals between
	// full syncs and between incremental updates.
	SyncInterval, UpdateInterval time.Duration

	// LastSync and LastUpdate are the times of the last full sync and the
	// last incremental update before the loop starts, or zero.
	LastSync, LastUpdate time.Time

	FullSync          func() error
	IncrementalUpdate func() error
}

// SyncMetrics holds the metrics that a SyncLoop updates.
type SyncMetrics struct {
	SyncTotal            *monitoring.Uint
	SyncError            *monitoring.Uint
	SyncProcessingTime   metrics.Sample
	UpdateTotal          *monitoring.Uint
	UpdateError          *monitoring.Uint
	UpdateProcessingTime metrics.Sample
}

// Run runs syncs until inputCtx is cancelled, reporting the input's status.
// A failed full sync is retried after UpdateInterval, and the wait doubles
// after each further failure, up to SyncInterval. A panic in a sync is
// handled as a failure of that sync.
func (l *SyncLoop) Run(inputCtx v2.Context) error {
	syncTimer := time.NewTimer(time.Until(l.LastSync.Add(l.SyncInterval)))
	updateTimer := time.NewTimer(time.Until(l.LastUpdate.Add(l.UpdateInterval)))
	defer syncTimer.Stop()
	defer updateTimer.Stop()

	// The most recent full sync and incremental update errors. The input is
	// degraded while either is set. A successful full sync clears both, but
	// a successful update does not clear fullErr, since only a full sync
	// detects deletions.
	var fullErr, updateErr error
	retry := RetryBackoff{Initial: l.UpdateInterval, Max: l.SyncInterval}

	inputCtx.UpdateStatus(status.Running, "")
	for {
		select {
		case <-inputCtx.Cancelation.Done():
			if !errors.Is(inputCtx.Cancelation.Err(), context.Canceled) {
				err := inputCtx.Cancelation.Err()
				inputCtx.UpdateStatus(status.Stopping, err.Error())
				return err
			}
			inputCtx.UpdateStatus(status.Stopping, "Deadline passed")
			return nil

		case <-syncTimer.C:
			start := time.Now()
			wait := l.SyncInterval
			err := RecoverSync(inputCtx, l.Logger, l.FullSync)
			switch {
			case err == nil:
				fullErr, updateErr = nil, nil
				retry.Reset()
				updateSyncStatus(inputCtx, fullErr, updateErr, "Successful full sync")
			case SyncInterrupted(inputCtx, err):
				l.Logger.Infow(SyncInterruptedMsg, "error", err)
				l.Metrics.SyncError.Inc()
			default:
				l.Logger.Errorw("Error running full sync", "error", err)
				l.Metrics.SyncError.Inc()
				fullErr = err
				wait = retry.Failed()
				updateSyncStatus(inputCtx, fullErr, updateErr, "")
			}
			l.Metrics.SyncTotal.Inc()
			l.Metrics.SyncProcessingTime.Update(time.Since(start).Nanoseconds())

			syncTimer.Reset(wait)
			l.Logger.Debugf("Next sync expected at: %v", time.Now().Add(wait))

			// A full sync includes an update, so wait the configured interval
			// for the next one, even if the update timer has already fired.
			if !updateTimer.Stop() {
				select {
				case <-updateTimer.C:
				default:
				}
			}
			updateTimer.Reset(l.UpdateInterval)
			l.Logger.Debugf("Next update expected at: %v", time.Now().Add(l.UpdateInterval))

		case <-updateTimer.C:
			start := time.Now()
			err := RecoverSync(inputCtx, l.Logger, l.IncrementalUpdate)
			switch {
			case err == nil:
				updateErr = nil
				updateSyncStatus(inputCtx, fullErr, updateErr, "Successful incremental update")
			case SyncInterrupted(inputCtx, err):
				l.Logger.Infow(SyncInterruptedMsg, "error", err)
				l.Metrics.UpdateError.Inc()
			default:
				l.Logger.Errorw("Error running incremental update", "error", err)
				l.Metrics.UpdateError.Inc()
				updateErr = err
				updateSyncStatus(inputCtx, fullErr, updateErr, "")
			}
			l.Metrics.UpdateTotal.Inc()
			l.Metrics.UpdateProcessingTime.Update(time.Since(start).Nanoseconds())

			updateTimer.Reset(l.UpdateInterval)
			l.Logger.Debugf("Next update expected at: %v", time.Now().Add(l.UpdateInterval))
		}
	}
}

// updateSyncStatus reports the input as degraded if the most recent full
// sync or incremental update failed, and as running with msg otherwise.
func updateSyncStatus(inputCtx v2.Context, fullErr, updateErr error, msg string) {
	switch {
	case fullErr != nil:
		inputCtx.UpdateStatus(status.Degraded, fmt.Sprintf("Error running full sync: %v", fullErr))
	case updateErr != nil:
		inputCtx.UpdateStatus(status.Degraded, fmt.Sprintf("Error running incremental update: %v", updateErr))
	default:
		inputCtx.UpdateStatus(status.Running, msg)
	}
}

// RetryBackoff tracks the wait before retrying a failed full sync. It starts
// at Initial and doubles after each further failure, up to Max.
type RetryBackoff struct {
	Initial, Max time.Duration
	next         time.Duration // zero means Initial
}

// Failed returns the wait before the next retry, and doubles the wait
// for the retry after it.
func (b *RetryBackoff) Failed() time.Duration {
	if b.next == 0 {
		b.next = b.Initial
	}
	wait := min(b.next, b.Max)
	b.next = min(2*b.next, b.Max)
	return wait
}

// Reset restarts the backoff after a successful full sync.
func (b *RetryBackoff) Reset() {
	b.next = 0
}

// RecoverSync runs a single sync. A panic during the sync is logged with
// its stack and returned as an error, so that one failed sync does not stop
// the input.
func RecoverSync(inputCtx v2.Context, log *logp.Logger, sync func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Input %s sync panic: %+v\n%s", inputCtx.ID, r, debug.Stack())
			err = fmt.Errorf("sync panic: %+v", r)
		}
	}()
	return sync()
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rcrowley/go-metrics"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

// TestSyncLoop_FullSyncRetry checks that a failed or panicking full sync is
// retried with a wait that doubles up to the sync interval, and that the
// backoff starts again after a successful full sync.
func TestSyncLoop_FullSyncRetry(t *testing.T) {
	const update, full = 10 * time.Minute, time.Hour

	// Full syncs 1 to 5 fail, 6 succeeds, 7 fails and 8 succeeds.
	outcomes := []error{
		errPanic, errors.New("boom"), errors.New("boom"), errors.New("boom"), errors.New("boom"), nil,
		errors.New("boom"), nil,
	}
	want := []time.Duration{
		0,
		update, 2 * update, 4 * update, full, full, // Retries.
		full,   // Scheduled.
		update, // Retry after the backoff was reset.
	}

	var gaps []time.Duration
	synctest.Test(t, func(t *testing.T) {
		last := time.Now()
		l := newTestSyncLoop(t, full, update)
		l.FullSync = func() error {
			gaps = append(gaps, time.Since(last))
			last = time.Now()
			err := outcomes[len(gaps)-1]
			if errors.Is(err, errPanic) {
				panic("bad response")
			}
			return err
		}
		runSyncLoop(t, l, func([]statusUpdate) bool { return len(gaps) == len(outcomes) })

		if got := l.Metrics.SyncTotal.Get(); got != 8 {
			t.Errorf("sync_total = %d; want 8", got)
		}
		if got := l.Metrics.SyncError.Get(); got != 6 {
			t.Errorf("sync_error = %d; want 6", got)
		}
		if got := l.Metrics.SyncProcessingTime.Count(); got != 8 {
			t.Errorf("sync_processing_time count = %d; want 8", got)
		}
	})
	if !slices.Equal(gaps, want) {
		t.Errorf("waits before full syncs = %v; want %v", gaps, want)
	}
}

// TestSyncLoop_FullSyncStatus checks that the input is degraded from a failed
// full sync until a full sync succeeds, even if incremental updates succeed.
func TestSyncLoop_FullSyncStatus(t *testing.T) {
	var fulls, incrs int
	var updates []statusUpdate
	synctest.Test(t, func(t *testing.T) {
		l := newTestSyncLoop(t, time.Hour, time.Minute)
		l.LastUpdate = time.Now() // The first update runs after the first full sync.
		l.FullSync = func() error {
			fulls++
			switch fulls {
			case 1:
				panic("bad response")
			case 2, 3:
				return errors.New("boom")
			}
			return nil
		}
		l.IncrementalUpdate = func() error {
			incrs++
			return nil
		}
		updates = runSyncLoop(t, l, func(u []statusUpdate) bool {
			return slices.Contains(u, statusUpdate{status.Running, "Successful full sync"})
		})
	})

	if incrs == 0 {
		t.Error("no incremental update ran between the failed full syncs")
	}
	if !slices.Equal(updates[:1], []statusUpdate{{status.Running, ""}}) {
		t.Errorf("first update = %v; want running", updates[0])
	}
	success := slices.Index(updates, statusUpdate{status.Running, "Successful full sync"})
	for _, u := range updates[1:success] {
		if u.status != status.Degraded || !strings.HasPrefix(u.msg, "Error running full sync: ") {
			t.Errorf("status %v %q reported before a full sync succeeded; want degraded by the full sync", u.status, u.msg)
		}
	}
	for _, msg := range []string{"Error running full sync: sync panic: bad response", "Error running full sync: boom"} {
		if !slices.Contains(updates, statusUpdate{status.Degraded, msg}) {
			t.Errorf("no degraded status %q in %v", msg, updates)
		}
	}
}

// TestSyncLoop_IncrementalUpdateStatus checks that the input is degraded from
// a failed or panicking incremental update until an update or a full sync
// succeeds, and that a failed update does not change the full sync schedule.
func TestSyncLoop_IncrementalUpdateStatus(t *testing.T) {
	var incrs int
	var updates []statusUpdate
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		l := newTestSyncLoop(t, time.Hour, 25*time.Minute)
		l.LastSync, l.LastUpdate = start, start
		l.FullSync = func() error {
			if got := time.Since(start); got%time.Hour != 0 {
				t.Errorf("full sync after %v; want only every hour", got)
			}
			return nil
		}
		l.IncrementalUpdate = func() error {
			incrs++
			switch incrs {
			case 1:
				panic("bad response")
			case 3:
				return nil
			}
			return errors.New("boom")
		}
		updates = runSyncLoop(t, l, func(u []statusUpdate) bool {
			return len(u) == 7
		})

		if got := l.Metrics.UpdateTotal.Get(); got != 4 {
			t.Errorf("update_total = %d; want 4", got)
		}
		if got := l.Metrics.UpdateError.Get(); got != 3 {
			t.Errorf("update_error = %d; want 3", got)
		}
		if got := l.Metrics.UpdateProcessingTime.Count(); got != 4 {
			t.Errorf("update_processing_time count = %d; want 4", got)
		}
	})

	want := []statusUpdate{
		{status.Running, ""},
		{status.Degraded, "Error running incremental update: sync panic: bad response"},
		{status.Degraded, "Error running incremental update: boom"},
		{status.Running, "Successful full sync"},
		{status.Running, "Successful incremental update"},
		{status.Degraded, "Error running incremental update: boom"},
		{status.Running, "Successful full sync"},
	}
	if len(updates) < len(want) || !slices.Equal(updates[:len(want)], want) {
		t.Errorf("updates = %v; want %v", updates, want)
	}
}

// TestSyncLoop_Interrupted checks that a sync interrupted by the input's
// cancellation does not degrade the input.
func TestSyncLoop_Interrupted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		l := newTestSyncLoop(t, time.Hour, time.Minute)
		l.FullSync = func() error {
			cancel()
			return ctx.Err()
		}
		rec := &statusRecorder{}
		err := l.Run(v2.Context{Logger: l.Logger, ID: t.Name(), Cancelation: ctx}.WithStatusReporter(rec))
		if err != nil {
			t.Errorf("Run = %v; want nil", err)
		}
		for _, u := range rec.updates {
			if u.status == status.Degraded {
				t.Errorf("status %v %q reported for an interrupted sync", u.status, u.msg)
			}
		}
	})
}

// TestRetryBackoff checks the doubling, the cap and the reset of the full
// sync retry wait.
func TestRetryBackoff(t *testing.T) {
	t.Parallel()

	b := RetryBackoff{Initial: 10 * time.Second, Max: time.Minute}
	steps := []struct {
		reset bool
		want  time.Duration
	}{
		{want: 10 * time.Second},
		{want: 20 * time.Second},
		{want: 40 * time.Second},
		{want: time.Minute},
		{want: time.Minute},
		{reset: true, want: 10 * time.Second},
		{want: 20 * time.Second},
	}
	for i, step := range steps {
		if step.reset {
			b.Reset()
		}
		if got := b.Failed(); got != step.want {
			t.Errorf("step %d: Failed() = %v; want %v", i, got, step.want)
		}
	}
}

// errPanic is a placeholder for a sync that panics.
var errPanic = errors.New("panic")

type statusUpdate struct {
	status status.Status
	msg    string
}

// statusRecorder records status updates. It is only used from the goroutine
// of the loop, and after the loop has returned.
type statusRecorder struct {
	updates []statusUpdate
	changed func([]statusUpdate)
}

func (r *statusRecorder) UpdateStatus(s status.Status, msg string) {
	r.updates = append(r.updates, statusUpdate{s, msg})
	if r.changed != nil {
		r.changed(r.updates)
	}
}

// newTestSyncLoop returns a loop with syncs that succeed. No sync has run
// before it starts.
func newTestSyncLoop(t *testing.T, full, update time.Duration) *SyncLoop {
	reg := monitoring.NewRegistry()
	return &SyncLoop{
		Logger: logptest.NewTestingLogger(t, "test"),
		Metrics: SyncMetrics{
			SyncTotal:            monitoring.NewUint(reg, "sync_total"),
			SyncError:            monitoring.NewUint(reg, "sync_error"),
			SyncProcessingTime:   metrics.NewUniformSample(1024),
			UpdateTotal:          monitoring.NewUint(reg, "update_total"),
			UpdateError:          monitoring.NewUint(reg, "update_error"),
			UpdateProcessingTime: metrics.NewUniformSample(1024),
		},
		SyncInterval:      full,
		UpdateInterval:    update,
		FullSync:          func() error { return nil },
		IncrementalUpdate: func() error { return nil },
	}
}

// runSyncLoop runs l until the status updates it reports satisfy done, and
// returns the updates.
func runSyncLoop(t *testing.T, l *SyncLoop, done func([]statusUpdate) bool) []statusUpdate {
	t.Helper()

	// The timeout is in the fake time of the synctest bubble.
	ctx, cancel := context.WithTimeout(t.Context(), 48*time.Hour)
	defer cancel()
	stop, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	defer context.AfterFunc(ctx, cancelRun)()

	rec := &statusRecorder{}
	rec.changed = func(u []statusUpdate) {
		if done(u) {
			cancelRun()
		}
	}
	err := l.Run(v2.Context{Logger: l.Logger, ID: t.Name(), Cancelation: stop}.WithStatusReporter(rec))
	if err != nil {
		t.Errorf("Run = %v; want nil", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("timed out; updates: %v", rec.updates)
	}
	return rec.updates
}

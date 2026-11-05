// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/paths"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/entcollect"
)

// fakeProvider is an in-memory entcollect.Provider for testing.
type fakeProvider struct {
	mu       sync.Mutex
	fullFunc func(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error
	incrFunc func(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error
}

func (p *fakeProvider) FullSync(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
	p.mu.Lock()
	fn := p.fullFunc
	p.mu.Unlock()
	if fn != nil {
		return fn(ctx, store, pub, log)
	}
	return nil
}

func (p *fakeProvider) IncrementalSync(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
	p.mu.Lock()
	fn := p.incrFunc
	p.mu.Unlock()
	if fn != nil {
		return fn(ctx, store, pub, log)
	}
	return nil
}

type fakeClient struct {
	mu     sync.Mutex
	events []beat.Event
}

func (c *fakeClient) Publish(event beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

func (c *fakeClient) PublishAll(events []beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
}

func (c *fakeClient) Close() error { return nil }

func (c *fakeClient) Events() []beat.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	dst := make([]beat.Event, len(c.events))
	copy(dst, c.events)
	return dst
}

type fakeConnector struct {
	client beat.Client
}

func (f *fakeConnector) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return f.client, nil
}

func (f *fakeConnector) Connect() (beat.Client, error) {
	return f.client, nil
}

func (f *fakeConnector) Disconnect(ctx context.Context) error {
	return f.client.Close()
}
func TestMinimalInput_RunSync_FullCycle(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	log := logptest.NewTestingLogger(t, "test")
	store, err := kvstore.NewStore(log, dbFile, 0o600)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	client := &fakeClient{}
	bucketName := "entcollect.testprov"
	slogger := slogLogger(log)

	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if err := s.Set("cursor", "page-2"); err != nil {
				return err
			}
			return pub(ctx, entcollect.Document{
				ID:        "user-1",
				Kind:      entcollect.KindUser,
				Action:    entcollect.ActionDiscovered,
				Timestamp: time.Now(),
				Fields:    map[string]any{"user.name": "alice"},
			})
		},
	}

	ctx := t.Context()

	ackClient := &ackingClient{inner: client}
	s := &bboltSyncer{store: store, bucketName: bucketName}

	err = s.runSync(
		v2.Context{
			Logger:      log,
			ID:          "test-input-1",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		prov,
		ackClient,
		slogger,
		true,
	)
	if err != nil {
		t.Fatalf("runSync: %v", err)
	}

	events := client.Events()
	if got := len(events); got != 1 {
		t.Fatalf("len(events) = %d; want 1", got)
	}

	action, _ := events[0].Fields.GetValue("event.action")
	if action != "user-discovered" {
		t.Errorf("event.action = %v; want user-discovered", action)
	}

	var cursor string
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		return tx.Get([]byte(bucketName), []byte("cursor"), &cursor)
	})
	if err != nil {
		t.Fatalf("reading cursor: %v", err)
	}
	if cursor != "page-2" {
		t.Errorf("cursor = %q; want %q", cursor, "page-2")
	}
}

func TestMinimalInput_RunSync_ErrorDiscardsBuffer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test-err.db")

	log := logptest.NewTestingLogger(t, "test")
	store, err := kvstore.NewStore(log, dbFile, 0o600)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	client := &fakeClient{}
	ackClient := &ackingClient{inner: client}
	bucketName := "entcollect.testprov"
	slogger := slogLogger(log)

	syncErr := errors.New("provider failure")
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if err := s.Set("cursor", "bad"); err != nil {
				return err
			}
			_ = pub(ctx, entcollect.Document{
				ID:        "user-1",
				Kind:      entcollect.KindUser,
				Action:    entcollect.ActionDiscovered,
				Timestamp: time.Now(),
			})
			return syncErr
		},
	}

	ctx := context.Background()
	s := &bboltSyncer{store: store, bucketName: bucketName}

	err = s.runSync(
		v2.Context{
			Logger:      log,
			ID:          "test-input-err",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		prov,
		ackClient,
		slogger,
		true,
	)
	if !errors.Is(err, syncErr) {
		t.Fatalf("runSync = %v; want %v", err, syncErr)
	}

	// The cursor must NOT be committed (buffer was discarded, tx rolled back).
	var cursor string
	txErr := store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		return tx.Get([]byte(bucketName), []byte("cursor"), &cursor)
	})
	if txErr == nil {
		t.Fatal("cursor should not exist after failed sync, but Get succeeded")
	}
}

func TestMinimalInput_RunCancelation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	prov := &fakeProvider{}
	mi := &minimalStateInput{
		provider:         prov,
		providerName:     "testprov",
		fullSyncInterval: time.Hour,
		incrSyncInterval: time.Hour,
		logger:           logptest.NewTestingLogger(t, "test"),
		path:             &paths.Path{Data: tmpDir},
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &fakeClient{}
	connector := &fakeConnector{client: client}

	done := make(chan error, 1)
	go func() {
		done <- mi.Run(
			v2.Context{
				Logger:      logptest.NewTestingLogger(t, "test"),
				ID:          "test-cancel",
				Cancelation: v2.GoContextFromCanceler(ctx),
			},
			connector,
		)
	}()

	// Let the first sync fire (timer starts at 0).
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v; want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestMinimalInput_RunFullSyncRetry checks that a failed or panicking full
// sync is retried well before the full sync interval, and that the input
// is reported degraded until a full sync succeeds.
func TestMinimalInput_RunFullSyncRetry(t *testing.T) {
	t.Parallel()

	syncErr := errors.New("provider failure")
	var calls atomic.Int32
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			switch calls.Add(1) {
			case 1:
				panic("bad response")
			case 2:
				return syncErr
			}
			return nil
		},
	}
	mi := newTestMinimalInput(t, prov, time.Hour, 20*time.Millisecond)

	recovered := statusUpdate{status.Running, "Successful full sync"}
	updates := runUntil(t, mi, func(u []statusUpdate) bool {
		return slices.Contains(u, recovered)
	})

	start := []statusUpdate{{status.Starting, ""}, {status.Running, ""}}
	if len(updates) < 2 || !slices.Equal(updates[:2], start) {
		t.Errorf("first updates = %v; want %v", updates[:min(2, len(updates))], start)
	}
	var (
		sawPanic, sawErr bool
		failed           = -1
		success          = slices.Index(updates, recovered)
	)
	for i, u := range updates {
		switch {
		case u.status == status.Degraded && strings.Contains(u.msg, "sync panic: bad response"):
			sawPanic = true
		case u.status == status.Degraded && strings.Contains(u.msg, syncErr.Error()):
			sawErr = true
		}
		if u.status == status.Degraded && failed < 0 {
			failed = i
		}
	}
	if !sawPanic {
		t.Error("no degraded status for the panicking full sync")
	}
	if !sawErr {
		t.Error("no degraded status for the failed full sync")
	}
	if failed < 0 || success < failed {
		t.Fatalf("want degraded then running after a successful full sync; got %v", updates)
	}
	for _, u := range updates[failed:success] {
		if u.status != status.Degraded {
			t.Errorf("status %v %q reported before a full sync succeeded; want degraded", u.status, u.msg)
		}
	}
}

// TestMinimalInput_RunFullSyncBackoff checks that the wait between full
// sync retries grows, and that it starts again from the incremental sync
// interval after a successful full sync.
func TestMinimalInput_RunFullSyncBackoff(t *testing.T) {
	t.Parallel()

	// Full syncs 1 to 4 fail, 5 succeeds, 6 fails and 7 succeeds.
	var (
		mu    sync.Mutex
		times []time.Time
	)
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			mu.Lock()
			defer mu.Unlock()
			times = append(times, time.Now())
			switch len(times) {
			case 5, 7:
				return nil
			}
			return errors.New("provider failure")
		},
	}
	mi := newTestMinimalInput(t, prov, time.Second, 10*time.Millisecond)

	success := statusUpdate{status.Running, "Successful full sync"}
	runUntil(t, mi, func(u []statusUpdate) bool {
		n := 0
		for _, v := range u {
			if v == success {
				n++
			}
		}
		return n == 2
	})

	mu.Lock()
	defer mu.Unlock()
	if len(times) < 7 {
		t.Fatalf("got %d full syncs; want 7", len(times))
	}
	var gaps []time.Duration
	for i := 1; i < len(times); i++ {
		gaps = append(gaps, times[i].Sub(times[i-1]))
	}
	// Nominal gaps are 10, 20, 40, 80ms, then 1s after the success, then
	// 10ms again.
	if gaps[3] <= gaps[0] {
		t.Errorf("fourth retry wait %v not longer than first %v; want growing waits (gaps %v)", gaps[3], gaps[0], gaps)
	}
	if gaps[5] >= gaps[3] {
		t.Errorf("retry wait after a success %v not shorter than before it %v; want reset (gaps %v)", gaps[5], gaps[3], gaps)
	}
}

// TestMinimalInput_RunIncrementalSyncStatus checks that a failed
// incremental sync is reported as degraded, and that the next successful
// one reports running.
func TestMinimalInput_RunIncrementalSyncStatus(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	prov := &fakeProvider{
		incrFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if calls.Add(1) == 1 {
				return errors.New("incremental failure")
			}
			return nil
		},
	}
	mi := newTestMinimalInput(t, prov, time.Hour, 20*time.Millisecond)

	failed := statusUpdate{status.Degraded, "Error running incremental sync: incremental failure"}
	recovered := statusUpdate{status.Running, "Successful incremental sync"}
	updates := runUntil(t, mi, func(u []statusUpdate) bool {
		i := slices.Index(u, failed)
		return i >= 0 && slices.Contains(u[i:], recovered)
	})
	if i := slices.Index(updates, recovered); i < slices.Index(updates, failed) {
		t.Errorf("running reported before the failed incremental sync: %v", updates)
	}
}

// TestMinimalInput_RunFullSyncClearsIncrementalError checks that a
// successful full sync reports running even if the last incremental sync
// failed.
func TestMinimalInput_RunFullSyncClearsIncrementalError(t *testing.T) {
	t.Parallel()

	var fullCalls atomic.Int32
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			fullCalls.Add(1)
			return nil
		},
		incrFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			return errors.New("incremental failure")
		},
	}
	mi := newTestMinimalInput(t, prov, 100*time.Millisecond, 20*time.Millisecond)

	failed := statusUpdate{status.Degraded, "Error running incremental sync: incremental failure"}
	recovered := statusUpdate{status.Running, "Successful full sync"}
	runUntil(t, mi, func(u []statusUpdate) bool {
		i := slices.Index(u, failed)
		return i >= 0 && slices.Contains(u[i:], recovered)
	})
	if n := fullCalls.Load(); n < 2 {
		t.Errorf("got %d full syncs; want at least 2", n)
	}
}

func TestRetryBackoff(t *testing.T) {
	b := retryBackoff{initial: 10 * time.Second, max: time.Minute}
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
			b.reset()
		}
		if got := b.failed(); got != step.want {
			t.Errorf("step %d: failed() = %v; want %v", i, got, step.want)
		}
	}
}

// newTestMinimalInput returns a minimal-state input using prov with a
// bbolt store in a temporary directory.
func newTestMinimalInput(t *testing.T, prov entcollect.Provider, full, incr time.Duration) *minimalStateInput {
	return &minimalStateInput{
		provider:         prov,
		providerName:     "testprov",
		fullSyncInterval: full,
		incrSyncInterval: incr,
		logger:           logptest.NewTestingLogger(t, "test"),
		path:             &paths.Path{Data: t.TempDir()},
	}
}

// runUntil runs mi until the status updates it reports satisfy done, then
// stops it and returns the updates.
func runUntil(t *testing.T, mi *minimalStateInput, done func([]statusUpdate) bool) []statusUpdate {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reporter := &statusRecorder{done: done, seen: make(chan struct{})}
	runErr := make(chan error, 1)
	go func() {
		runErr <- mi.Run(
			v2.Context{
				Logger:      logptest.NewTestingLogger(t, "test"),
				ID:          "test-" + t.Name(),
				Cancelation: v2.GoContextFromCanceler(ctx),
			}.WithStatusReporter(reporter),
			&fakeConnector{client: &ackingClient{inner: &fakeClient{}}},
		)
	}()

	select {
	case <-reporter.seen:
	case err := <-runErr:
		t.Fatalf("Run returned %v early; updates: %v", err, reporter.updates())
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for status updates; got %v", reporter.updates())
	}
	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run = %v; want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	return reporter.updates()
}

// statusRecorder records status updates. It closes seen the first time
// the recorded updates satisfy done.
type statusRecorder struct {
	done func([]statusUpdate) bool
	seen chan struct{}

	mu     sync.Mutex
	upds   []statusUpdate
	closed bool
}

type statusUpdate struct {
	status status.Status
	msg    string
}

func (r *statusRecorder) UpdateStatus(s status.Status, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upds = append(r.upds, statusUpdate{s, msg})
	if !r.closed && r.done(r.upds) {
		close(r.seen)
		r.closed = true
	}
}

func (r *statusRecorder) updates() []statusUpdate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]statusUpdate(nil), r.upds...)
}

// ackingClient wraps a fakeClient and immediately ACKs each event by
// calling tracker.Ack() after delegating to the inner client.
type ackingClient struct {
	inner *fakeClient
}

func (c *ackingClient) Publish(event beat.Event) {
	c.inner.Publish(event)
	if t, ok := event.Private.(*kvstore.TxTracker); ok {
		t.Ack()
	}
}

func (c *ackingClient) PublishAll(events []beat.Event) {
	for _, e := range events {
		c.Publish(e)
	}
}

func (c *ackingClient) Close() error { return nil }

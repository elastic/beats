// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
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

func TestMinimalInput_RunSync_FullCycle(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	log := logptest.NewTestingLogger(t, "test")
	store, err := kvstore.NewStore(log, dbFile, 0600)
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

	mi := &minimalStateInput{
		provider:     prov,
		providerName: "testprov",
		logger:       log,
		path:         paths.New(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ackClient := &ackingClient{inner: client}

	err = mi.runSync(
		v2.Context{
			Logger:      log,
			ID:          "test-input-1",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		store,
		ackClient,
		slogger,
		bucketName,
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
	store, err := kvstore.NewStore(log, dbFile, 0600)
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

	mi := &minimalStateInput{
		provider:     prov,
		providerName: "testprov",
		logger:       log,
		path:         paths.New(),
	}

	ctx := context.Background()
	err = mi.runSync(
		v2.Context{
			Logger:      log,
			ID:          "test-input-err",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		store,
		ackClient,
		slogger,
		bucketName,
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

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package entityanalytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/entcollect"
)

func TestESIntegration_StateStoreAdapter_CRUD(t *testing.T) {
	store, conn, indexName := newESStore(t)
	a := kvstore.NewStateStoreAdapter(store)

	// Set
	if err := a.Set("cursor", "page-2"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := a.Set("users-count", 42); err != nil {
		t.Fatalf("Set users-count: %v", err)
	}

	// ES is near-real-time; force refresh.
	refreshIndex(t, conn, indexName)

	// Get
	var cursor string
	if err := a.Get("cursor", &cursor); err != nil {
		t.Fatalf("Get cursor: %v", err)
	}
	if cursor != "page-2" {
		t.Errorf("cursor = %q; want %q", cursor, "page-2")
	}

	var count int
	if err := a.Get("users-count", &count); err != nil {
		t.Fatalf("Get users-count: %v", err)
	}
	if count != 42 {
		t.Errorf("users-count = %d; want 42", count)
	}

	// Get missing key
	var missing string
	err := a.Get("nonexistent", &missing)
	if !errors.Is(err, entcollect.ErrKeyNotFound) {
		t.Fatalf("Get missing = %v; want ErrKeyNotFound", err)
	}

	// Delete present key
	if err := a.Delete("cursor"); err != nil {
		t.Fatalf("Delete cursor: %v", err)
	}
	refreshIndex(t, conn, indexName)
	err = a.Get("cursor", &cursor)
	if !errors.Is(err, entcollect.ErrKeyNotFound) {
		t.Fatalf("Get after Delete = %v; want ErrKeyNotFound", err)
	}

	// Delete absent key (the Has-before-Remove guard)
	if err := a.Delete("nonexistent"); err != nil {
		t.Fatalf("Delete absent key should succeed; got %v", err)
	}

	// Each
	seen := map[string]bool{}
	err = a.Each(func(key string, decode func(any) error) (bool, error) {
		seen[key] = true
		return true, nil
	})
	if err != nil {
		t.Fatalf("Each: %v", err)
	}
	if !seen["users-count"] {
		t.Errorf("Each did not visit 'users-count'; saw %v", seen)
	}
	if seen["cursor"] {
		t.Errorf("Each visited deleted key 'cursor'")
	}
}

func TestESIntegration_ESSyncer_RunSync(t *testing.T) {
	store, conn, indexName := newESStore(t)

	syncer := &esSyncer{store: store}
	client := &fakeClient{}
	ackClient := &ackingClient{inner: client}
	slogger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if err := s.Set("sync-cursor", "full-done"); err != nil {
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

	ctx := context.Background()
	err := syncer.runSync(
		v2.Context{
			Logger:      logptest.NewTestingLogger(t, "es-integ"),
			ID:          "es-integ-test",
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

	// Verify state was persisted.
	refreshIndex(t, conn, indexName)
	a := kvstore.NewStateStoreAdapter(store)
	var cursor string
	if err := a.Get("sync-cursor", &cursor); err != nil {
		t.Fatalf("Get sync-cursor after runSync: %v", err)
	}
	if cursor != "full-done" {
		t.Errorf("sync-cursor = %q; want %q", cursor, "full-done")
	}

	// Verify event was published.
	events := client.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d; want 1", len(events))
	}
	action, _ := events[0].Fields.GetValue("event.action")
	if action != "user-discovered" {
		t.Errorf("event.action = %v; want user-discovered", action)
	}

	// Second sync (incremental) can read state from first.
	prov.mu.Lock()
	prov.incrFunc = func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
		var c string
		if err := s.Get("sync-cursor", &c); err != nil {
			return fmt.Errorf("incremental could not read cursor: %w", err)
		}
		if c != "full-done" {
			return fmt.Errorf("incremental cursor = %q; want %q", c, "full-done")
		}
		return s.Set("sync-cursor", "incr-done")
	}
	prov.mu.Unlock()

	err = syncer.runSync(
		v2.Context{
			Logger:      logptest.NewTestingLogger(t, "es-integ"),
			ID:          "es-integ-test",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		prov,
		ackClient,
		slogger,
		false,
	)
	if err != nil {
		t.Fatalf("incremental runSync: %v", err)
	}

	refreshIndex(t, conn, indexName)
	if err := a.Get("sync-cursor", &cursor); err != nil {
		t.Fatalf("Get sync-cursor after incremental: %v", err)
	}
	if cursor != "incr-done" {
		t.Errorf("sync-cursor = %q; want %q", cursor, "incr-done")
	}
}

// newESStore creates a *statestore.Store backed by a real Elasticsearch
// instance. It uses the ES_HOST/ES_PORT environment variables (defaults
// to localhost:9200) with admin credentials. It also returns the
// underlying connection and index name for use in refreshIndex.
func newESStore(t *testing.T) (*statestore.Store, *eslegclient.Connection, string) {
	t.Helper()

	host := os.Getenv("ES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("ES_PORT")
	if port == "" {
		port = "9200"
	}
	user := os.Getenv("ES_SUPERUSER_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("ES_SUPERUSER_PASS")
	if pass == "" {
		pass = "testing"
	}

	esURL := fmt.Sprintf("http://%s:%s", host, port)
	log := logptest.NewTestingLogger(t, "es-backend")

	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:      esURL,
		Username: user,
		Password: pass,
	}, log)
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatalf("Connect to ES: %v", err)
	}

	inputID := fmt.Sprintf("integ-test-%d", time.Now().UnixNano())
	indexName := "agentless-state-" + inputID

	baseStore := es.NewStore(context.Background(), log, conn, inputID)
	reg := statestore.NewRegistry(&singleStoreRegistry{store: baseStore})
	t.Cleanup(func() { reg.Close() })

	s, err := reg.Get("entity-analytics")
	if err != nil {
		t.Fatalf("registry Get: %v", err)
	}
	s.SetID(inputID)
	t.Cleanup(func() {
		s.Close()
		// Clean up the test index.
		conn.Request("DELETE", "/"+indexName, "", nil, nil) //nolint:errcheck // best-effort cleanup
	})
	return s, conn, indexName
}

// refreshIndex forces an ES index refresh so writes become visible.
func refreshIndex(t *testing.T, conn *eslegclient.Connection, indexName string) {
	t.Helper()
	status, body, err := conn.Request("POST", "/"+indexName+"/_refresh", "", nil, nil)
	if err != nil {
		t.Fatalf("refresh index %q: %v", indexName, err)
	}
	if status >= 300 {
		t.Fatalf("refresh index %q: status=%d body=%s", indexName, status, body)
	}
}

// singleStoreRegistry wraps a single backend.Store for testing.
type singleStoreRegistry struct {
	store backend.Store
}

func (r *singleStoreRegistry) Access(string) (backend.Store, error) {
	return r.store, nil
}

func (r *singleStoreRegistry) Close() error {
	return r.store.Close()
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/entcollect"
	ecjamf "github.com/elastic/entcollect/provider/jamf"
)

// TestJamfIntegration_FullLifecycle exercises the minimal-state Jamf input
// end-to-end: real entcollect provider → minimalStateInput.Run → bbolt →
// fakeClient. No external services required (httptest only).
//
// Phase 1: first sync discovers 3 computers.
// Phase 2: one computer removed from the API; restart with same bbolt.
//
//	Expects 2 modified + 1 deleted.
func TestJamfIntegration_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	fixture := newAtomicFixture(makeComputersJSON(t,
		testComputer{Name: "dev-laptop", UDID: "AAA-111", Managed: true},
		testComputer{Name: "staging-server", UDID: "BBB-222", Managed: true},
		testComputer{Name: "to-remove", UDID: "CCC-333", Managed: true},
	))
	srv := startJamfIntegServer(t, fixture)
	host := hostFromURL(t, srv.URL)

	newProvider := func() *ecjamf.Provider {
		cfg := ecjamf.DefaultConfig()
		cfg.TenantID = host
		cfg.Username = "testuser"
		cfg.Password = "testpass"
		return ecjamf.NewWithClient(cfg, srv.Client())
	}

	// Phase 1: first sync discovers all computers.

	client1 := &fakeClient{}
	err1 := runInputOnce(t, tmpDir, "jamf", newProvider(), client1)
	if err1 != nil {
		t.Fatalf("first run failed: %v", err1)
	}

	events1 := client1.Events()
	if got := len(events1); got != 3 {
		t.Errorf("first sync: got %d events, want 3", got)
	}
	if got := countByAction(events1, "device-discovered"); got != 3 {
		t.Errorf("first sync: got %d discovered, want 3", got)
	}

	// Phase 2: remove one computer, restart with same bbolt.

	fixture.set(makeComputersJSON(t,
		testComputer{Name: "dev-laptop", UDID: "AAA-111", Managed: true},
		testComputer{Name: "staging-server", UDID: "BBB-222", Managed: true},
	))

	client2 := &fakeClient{}
	err2 := runInputOnce(t, tmpDir, "jamf", newProvider(), client2)
	if err2 != nil {
		t.Fatalf("second run failed: %v", err2)
	}

	events2 := client2.Events()
	if got := len(events2); got != 3 {
		t.Errorf("second sync: got %d events, want 3 (2 modified + 1 deleted)", got)
	}
	if got := countByAction(events2, "device-modified"); got != 2 {
		t.Errorf("second sync: got %d modified, want 2", got)
	}
	if got := countByAction(events2, "device-deleted"); got != 1 {
		t.Errorf("second sync: got %d deleted, want 1", got)
	}

	deletedID := ""
	for _, e := range events2 {
		action, _ := e.Fields.GetValue("event.action")
		if action == "device-deleted" {
			id, _ := e.Fields.GetValue("device.id")
			deletedID, _ = id.(string)
		}
	}
	if deletedID != "CCC-333" {
		t.Errorf("deleted device ID = %q, want %q", deletedID, "CCC-333")
	}
}

// TestJamfIntegration_CursorRoundTrip verifies that the entcollect cursor
// keys survive a restart via bbolt and are visible to the provider.
func TestJamfIntegration_CursorRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	fixture := newAtomicFixture(makeComputersJSON(t,
		testComputer{Name: "laptop", UDID: "CURSOR-TEST", Managed: true},
	))
	srv := startJamfIntegServer(t, fixture)
	host := hostFromURL(t, srv.URL)

	cfg := ecjamf.DefaultConfig()
	cfg.TenantID = host
	cfg.Username = "testuser"
	cfg.Password = "testpass"

	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "jamf", ecjamf.NewWithClient(cfg, srv.Client()), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if got := len(client1.Events()); got != 1 {
		t.Fatalf("first sync: got %d events, want 1", got)
	}

	// Second run: same fixture, same bbolt → cursor should be read back.
	// The device was already seen, so action should be "modified" not "discovered".
	client2 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "jamf", ecjamf.NewWithClient(cfg, srv.Client()), client2); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	events := client2.Events()
	if len(events) != 1 {
		t.Fatalf("second sync: got %d events, want 1", len(events))
	}
	action, _ := events[0].Fields.GetValue("event.action")
	if action != "device-modified" {
		t.Errorf("second sync action = %v, want %q (proves cursor roundtrip)", action, "device-modified")
	}
}

// runInputOnce starts a minimalStateInput, waits for the first full sync to
// complete (events arrive), and stops it. It returns the Run error.
func runInputOnce(t *testing.T, dataDir string, providerName string, p entcollect.Provider, client *fakeClient) error {
	t.Helper()
	log := logptest.NewTestingLogger(t, "integ")

	mi := &minimalStateInput{
		provider:         p,
		providerName:     providerName,
		fullSyncInterval: time.Hour,
		incrSyncInterval: time.Hour,
		logger:           log,
		path:             &paths.Path{Data: dataDir},
	}

	acking := &ackingClient{inner: client}
	connector := &fakeConnector{client: acking}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- mi.Run(
			v2.Context{
				Logger:      log,
				ID:          "integ-" + providerName,
				Cancelation: v2.GoContextFromCanceler(ctx),
			},
			connector,
		)
	}()

	// Wait for at least one event (the sync fires at timer 0).
	deadline := time.After(10 * time.Second)
	for len(client.Events()) == 0 {
		select {
		case <-deadline:
			cancel()
			t.Fatal("timed out waiting for events from first sync")
		case <-time.After(10 * time.Millisecond):
		}
	}

	// Events are published during FullSync, but the cursor and idset
	// state are committed to bbolt after FullSync returns (runSync
	// calls buf.Commit then tx.Commit). Canceling before that commit
	// causes runSync to discard the buffer via ctx.Err(), losing the
	// cursor. There is no external signal for commit completion, so
	// we wait long enough for the remaining events and the bbolt
	// write to finish.
	time.Sleep(500 * time.Millisecond)

	cancel()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
		return nil
	}
}

// runIncrementalOnce opens the bbolt store written by a previous
// runInputOnce and runs a single incremental sync via runSync. This
// bypasses the timer loop in Run(), which always fires a full sync
// first and cannot reliably trigger incremental-only execution.
func runIncrementalOnce(t *testing.T, dataDir string, providerName string, p entcollect.Provider, client *fakeClient) error {
	t.Helper()
	log := logptest.NewTestingLogger(t, "integ-incr")

	dbPath := filepath.Join(dataDir, "kvstore", "integ-"+providerName+".db")
	store, err := kvstore.NewStore(log, dbPath, 0600)
	if err != nil {
		t.Fatalf("open bbolt for incremental: %v", err)
	}
	defer store.Close()

	mi := &minimalStateInput{
		provider:         p,
		providerName:     providerName,
		fullSyncInterval: time.Hour,
		incrSyncInterval: time.Hour,
		logger:           log,
		path:             &paths.Path{Data: dataDir},
	}

	acking := &ackingClient{inner: client}
	ctx := context.Background()
	slogger := slog.New(slog.NewTextHandler(&testLogWriter{t}, nil))

	return mi.runSync(
		v2.Context{
			Logger:      log,
			ID:          "integ-" + providerName,
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		store,
		acking,
		slogger,
		"entcollect."+providerName,
		false,
	)
}

// assertBboltKey verifies that a key exists in the bbolt bucket used by
// runInputOnce. It opens the store read-only and reads the key.
func assertBboltKey(t *testing.T, dataDir, providerName, key string) {
	t.Helper()
	log := logptest.NewTestingLogger(t, "assert-bbolt")
	dbPath := filepath.Join(dataDir, "kvstore", "integ-"+providerName+".db")
	store, err := kvstore.NewStore(log, dbPath, 0600)
	if err != nil {
		t.Fatalf("open bbolt for assertion: %v", err)
	}
	defer store.Close()

	bucket := "entcollect." + providerName
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		_, err := tx.GetBytes([]byte(bucket), []byte(key))
		return err
	})
	if err != nil {
		t.Errorf("bbolt key %q not found in bucket %q: %v", key, bucket, err)
	}
}

// deleteBboltKey removes a key from the bbolt bucket. Used by
// corruption/recovery tests to simulate missing state.
func deleteBboltKey(t *testing.T, dataDir, providerName, key string) {
	t.Helper()
	log := logptest.NewTestingLogger(t, "delete-bbolt")
	dbPath := filepath.Join(dataDir, "kvstore", "integ-"+providerName+".db")
	store, err := kvstore.NewStore(log, dbPath, 0600)
	if err != nil {
		t.Fatalf("open bbolt for deletion: %v", err)
	}
	defer store.Close()

	bucket := "entcollect." + providerName
	err = store.RunTransaction(true, func(tx *kvstore.Transaction) error {
		return tx.Delete([]byte(bucket), []byte(key))
	})
	if err != nil {
		t.Fatalf("delete bbolt key %q in bucket %q: %v", key, bucket, err)
	}
}

type testLogWriter struct{ t *testing.T }

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// atomicFixture holds a mutable API response body that can be swapped
// between test phases while the httptest server is running.
type atomicFixture struct {
	mu   sync.Mutex
	data []byte
}

// newAtomicFixture returns an atomicFixture initialised with data.
func newAtomicFixture(data []byte) *atomicFixture {
	return &atomicFixture{data: data}
}

func (f *atomicFixture) get() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data
}

func (f *atomicFixture) set(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = data
}

// startJamfIntegServer returns a TLS httptest server that serves Jamf
// token and computer-list endpoints backed by the given fixture.
func startJamfIntegServer(t *testing.T, fixture *atomicFixture) *httptest.Server {
	t.Helper()

	var tokenMu sync.Mutex
	var currentToken string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		tokenMu.Lock()
		currentToken = uuid.Must(uuid.NewV4()).String()
		tok := currentToken
		tokenMu.Unlock()

		expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
		fmt.Fprintf(w, `{"token":%q,"expires":%q}`, tok, expires)
	})
	mux.HandleFunc("/api/preview/computers", func(w http.ResponseWriter, r *http.Request) {
		tokenMu.Lock()
		tok := currentToken
		tokenMu.Unlock()

		if tok == "" || r.Header.Get("Authorization") != "Bearer "+tok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture.get())
	})

	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// testComputer describes a minimal Jamf computer for fixture generation.
type testComputer struct {
	Name    string
	UDID    string
	Managed bool
}

// makeComputersJSON marshals computers into a Jamf API response body.
func makeComputersJSON(t *testing.T, computers ...testComputer) []byte {
	t.Helper()

	type result struct {
		Location         struct{} `json:"location"`
		Name             *string  `json:"name"`
		UDID             *string  `json:"udid"`
		IsManaged        *bool    `json:"isManaged"`
		LastContactDate  string   `json:"lastContactDate"`
		LastReportDate   string   `json:"lastReportDate"`
		LastEnrolledDate string   `json:"lastEnrolledDate"`
	}

	results := make([]result, len(computers))
	for i, c := range computers {
		name := c.Name
		udid := c.UDID
		managed := c.Managed
		results[i] = result{
			Name:             &name,
			UDID:             &udid,
			IsManaged:        &managed,
			LastContactDate:  "2024-01-01T00:00:00Z",
			LastReportDate:   "2024-01-01T00:00:00Z",
			LastEnrolledDate: "2024-01-01T00:00:00Z",
		}
	}

	body := struct {
		TotalCount int      `json:"totalCount"`
		Results    []result `json:"results"`
	}{
		TotalCount: len(results),
		Results:    results,
	}

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshalling fixture: %v", err)
	}
	return b
}

// hostFromURL extracts the host:port from a URL string.
func hostFromURL(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing server URL: %v", err)
	}
	return u.Host
}

// countByAction returns how many events have the given event.action value.
func countByAction(events []beat.Event, action string) int {
	n := 0
	for _, e := range events {
		a, _ := e.Fields.GetValue("event.action")
		if a == action {
			n++
		}
	}
	return n
}

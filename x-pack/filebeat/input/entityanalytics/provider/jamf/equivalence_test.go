// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/jamf/internal/jamf"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/entcollect"
	ecjamf "github.com/elastic/entcollect/provider/jamf"
)

// TestEquivalence_FullSync runs both the legacy and minimal-state Jamf
// providers against the same httptest mock and asserts that the jamf.*
// payloads are structurally equivalent.
//
// Intentional differences between the two providers (documented here):
//   - Legacy stores full Computer records in bbolt; minimal-state uses
//     idset + cursor keys only.
//   - Legacy IsManaged check has a bug (c.IsManaged != nil should be
//     == nil) that marks managed devices as Deleted on update;
//     minimal-state fixes this. First-sync output is unaffected because
//     all devices take the "not previously seen" path.
//   - Legacy emits start/end marker events; minimal-state does not.
//   - event.kind: "asset" only in minimal-state.
//   - event.action format differs: legacy uses "device-discovered" etc.;
//     minimal-state uses entcollect.ActionDiscovered.
//   - Minimal-state detects API absences as deletions via idset; legacy
//     has no equivalent detection on incremental.
func TestEquivalence_FullSync(t *testing.T) {
	tenant, username, password, client, cleanup := startEquivServer(t)
	defer cleanup()

	legacyPayloads := runLegacyFetch(t, tenant, username, password, client)
	minimalDocs := runMinimalFullSync(t, tenant, username, password, client)
	minimalDevices := filterDocsByKind(minimalDocs, entcollect.KindDevice)

	if len(legacyPayloads) == 0 {
		t.Fatal("legacy returned no devices; fixture may be empty")
	}
	if len(legacyPayloads) != len(minimalDevices) {
		t.Fatalf("device count: legacy=%d, minimal=%d", len(legacyPayloads), len(minimalDevices))
	}
	compareSortedEntries(t, legacyPayloads, minimalDevices)
}

// TestEquivalence_DeviceTypes unmarshals the shared fixture into both the
// legacy (internal/jamf.Computer) and entcollect (provider/jamf.Computer)
// types, then marshals each back to JSON and compares. This catches struct
// tag drift between the two type definitions.
func TestEquivalence_DeviceTypes(t *testing.T) {
	var legacy jamf.Computers
	if err := json.Unmarshal(computers, &legacy); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	var ec ecjamf.Computers
	if err := json.Unmarshal(computers, &ec); err != nil {
		t.Fatalf("unmarshal entcollect: %v", err)
	}

	if len(legacy.Results) != len(ec.Results) {
		t.Fatalf("count mismatch: legacy=%d, entcollect=%d",
			len(legacy.Results), len(ec.Results))
	}

	for i := range legacy.Results {
		legacyJSON, err := json.Marshal(legacy.Results[i])
		if err != nil {
			t.Fatal(err)
		}
		ecJSON, err := json.Marshal(ec.Results[i])
		if err != nil {
			t.Fatal(err)
		}
		if string(legacyJSON) != string(ecJSON) {
			t.Errorf("computer[%d] type mismatch:\n  legacy:     %s\n  entcollect: %s",
				i, legacyJSON, ecJSON)
		}
	}
}

// runLegacyFetch exercises the legacy doFetchComputers path and returns the
// raw JSON payload for each computer found.
func runLegacyFetch(t *testing.T, tenant, username, password string, client *http.Client) []json.RawMessage {
	t.Helper()

	dbFile := t.TempDir() + "/equiv-legacy.db"
	store := testSetupStore(t, dbFile)
	t.Cleanup(func() { testCleanupStore(store, dbFile) })

	a := jamfInput{
		cfg: conf{
			JamfTenant:   tenant,
			JamfUsername: username,
			JamfPassword: password,
		},
		client: client,
		logger: logptest.NewTestingLogger(t, "test-legacy"),
	}

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("newStateStore: %v", err)
	}
	defer ss.close(false)

	_, err = a.doFetchComputers(context.Background(), ss, true)
	if err != nil {
		t.Fatalf("legacy doFetchComputers: %v", err)
	}

	payloads := make([]json.RawMessage, 0, len(ss.computers))
	for _, c := range ss.computers {
		raw, err := json.Marshal(c.Computer)
		if err != nil {
			t.Fatal(err)
		}
		payloads = append(payloads, raw)
	}
	return payloads
}

// runMinimalFullSync exercises the entcollect FullSync path and returns the
// published documents.
func runMinimalFullSync(t *testing.T, tenant, username, password string, client *http.Client) []entcollect.Document {
	t.Helper()

	cfg := ecjamf.DefaultConfig()
	cfg.TenantID = tenant
	cfg.Username = username
	cfg.Password = password

	p := ecjamf.NewWithClient(cfg, client)

	store := newEquivMemStore()
	var docs []entcollect.Document
	pub := func(_ context.Context, doc entcollect.Document) error {
		docs = append(docs, doc)
		return nil
	}

	log := slog.New(slog.NewTextHandler(&testWriter{t}, nil))
	if err := p.FullSync(context.Background(), store, pub, log); err != nil {
		t.Fatalf("minimal FullSync: %v", err)
	}
	return docs
}

// compareSortedEntries sorts legacy and minimal payloads by UDID and asserts
// that the normalised JSON for each pair is identical.
func compareSortedEntries(t *testing.T, legacy []json.RawMessage, minimalDocs []entcollect.Document) {
	t.Helper()

	type idPayload struct {
		id      string
		payload json.RawMessage
	}

	legacyByID := make([]idPayload, 0, len(legacy))
	for _, raw := range legacy {
		var c struct {
			Udid *string `json:"udid"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		id := ""
		if c.Udid != nil {
			id = *c.Udid
		}
		legacyByID = append(legacyByID, idPayload{id: id, payload: raw})
	}
	sort.Slice(legacyByID, func(i, j int) bool { return legacyByID[i].id < legacyByID[j].id })

	minimalByID := make([]idPayload, 0, len(minimalDocs))
	for _, doc := range minimalDocs {
		raw, err := json.Marshal(doc.Fields["jamf"])
		if err != nil {
			t.Fatal(err)
		}
		minimalByID = append(minimalByID, idPayload{id: doc.ID, payload: raw})
	}
	sort.Slice(minimalByID, func(i, j int) bool { return minimalByID[i].id < minimalByID[j].id })

	for i := range legacyByID {
		if legacyByID[i].id != minimalByID[i].id {
			t.Errorf("device[%d] ID mismatch: legacy=%q, minimal=%q",
				i, legacyByID[i].id, minimalByID[i].id)
			continue
		}

		var legacyMap, minimalMap map[string]any
		if err := json.Unmarshal(legacyByID[i].payload, &legacyMap); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(minimalByID[i].payload, &minimalMap); err != nil {
			t.Fatal(err)
		}

		legacyNorm, _ := json.Marshal(legacyMap)
		minimalNorm, _ := json.Marshal(minimalMap)
		if string(legacyNorm) != string(minimalNorm) {
			t.Errorf("device %q payload mismatch:\n  legacy:  %s\n  minimal: %s",
				legacyByID[i].id, legacyNorm, minimalNorm)
		}
	}
}

// filterDocsByKind returns only the documents matching the given entity kind.
func filterDocsByKind(docs []entcollect.Document, kind entcollect.EntityKind) []entcollect.Document {
	var out []entcollect.Document
	for _, d := range docs {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	return out
}

// startEquivServer returns a TLS httptest server that serves Jamf token and
// computer-list endpoints using the package-level computers fixture.
func startEquivServer(t *testing.T) (tenant, username, password string, client *http.Client, cleanup func()) {
	t.Helper()

	username = "testuser"
	password = "testpassword"

	var tok jamf.Token
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		tok.Token = uuid.Must(uuid.NewV4()).String()
		tok.Expires = time.Now().UTC().Add(time.Hour)
		fmt.Fprintf(w, `{"token":%q,"expires":%q}`,
			tok.Token, tok.Expires.Format(time.RFC3339))
	})
	mux.HandleFunc("/api/preview/computers", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tok.Token || !tok.IsValidFor(0) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(computers)
	})

	srv := httptest.NewTLSServer(mux)
	u, err := url.Parse(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}
	return u.Host, username, password, srv.Client(), srv.Close
}

// testWriter adapts a *testing.T into an io.Writer for slog output.
type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// equivMemStore is an in-memory implementation of entcollect.Store for tests.
type equivMemStore struct {
	data map[string]json.RawMessage
}

// newEquivMemStore returns an empty equivMemStore.
func newEquivMemStore() *equivMemStore {
	return &equivMemStore{data: make(map[string]json.RawMessage)}
}

func (m *equivMemStore) Get(key string, dst any) error {
	raw, ok := m.data[key]
	if !ok {
		return entcollect.ErrKeyNotFound
	}
	return json.Unmarshal(raw, dst)
}

func (m *equivMemStore) Set(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.data[key] = raw
	return nil
}

func (m *equivMemStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *equivMemStore) Each(fn func(string, func(any) error) (bool, error)) error {
	for k, v := range m.data {
		v := v
		cont, err := fn(k, func(dst any) error { return json.Unmarshal(v, dst) })
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}
	return nil
}

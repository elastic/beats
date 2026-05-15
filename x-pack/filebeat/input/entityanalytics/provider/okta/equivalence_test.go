// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:errcheck // test file; error returns from w.Write are intentionally unchecked
package okta

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	legacyokta "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/internal/okta"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/entcollect"
	ecokta "github.com/elastic/entcollect/provider/okta"
)

// TestEquivalence_UserTypes verifies that the entcollect Okta User type
// and the legacy internal/okta User type produce identical JSON when
// given the same API response. This ensures the minimal-state provider
// emits the same user payloads as the legacy provider.
//
// Intentional differences between legacy and minimal-state (not tested
// here, documented for reference):
//   - Legacy emits start/end marker events; minimal-state does not.
//   - Legacy never emits deletion events (State.Deleted is never set);
//     minimal-state uses idset deletion detection.
//   - event.kind: asset is set only by the minimal-state consumer.
//   - Supervises on incremental sync is computed from the current batch
//     only (minimal-state), vs globally from all stored users (legacy).
func TestEquivalence_UserTypes(t *testing.T) {
	fixture := equivUserFixtureJSON()

	// Parse through legacy types.
	var legacyUsers []legacyokta.User
	if err := json.Unmarshal(fixture, &legacyUsers); err != nil {
		t.Fatalf("unmarshal legacy users: %v", err)
	}

	// Parse through entcollect types.
	var ecUsers []ecokta.User
	if err := json.Unmarshal(fixture, &ecUsers); err != nil {
		t.Fatalf("unmarshal entcollect users: %v", err)
	}

	if len(legacyUsers) != len(ecUsers) {
		t.Fatalf("user count: legacy=%d, entcollect=%d", len(legacyUsers), len(ecUsers))
	}

	for i := range legacyUsers {
		legacyJSON, _ := json.Marshal(legacyUsers[i])
		ecJSON, _ := json.Marshal(ecUsers[i])
		if string(legacyJSON) != string(ecJSON) {
			t.Errorf("user[%d] %q payload mismatch:\n  legacy:     %s\n  entcollect: %s",
				i, legacyUsers[i].ID, legacyJSON, ecJSON)
		}
	}
}

// TestEquivalence_DeviceTypes verifies the same JSON round-trip for devices.
func TestEquivalence_DeviceTypes(t *testing.T) {
	fixture := equivDeviceFixtureJSON()

	var legacyDevices []legacyokta.Device
	if err := json.Unmarshal(fixture, &legacyDevices); err != nil {
		t.Fatalf("unmarshal legacy devices: %v", err)
	}

	var ecDevices []ecokta.Device
	if err := json.Unmarshal(fixture, &ecDevices); err != nil {
		t.Fatalf("unmarshal entcollect devices: %v", err)
	}

	if len(legacyDevices) != len(ecDevices) {
		t.Fatalf("device count: legacy=%d, entcollect=%d", len(legacyDevices), len(ecDevices))
	}

	for i := range legacyDevices {
		legacyJSON, _ := json.Marshal(legacyDevices[i])
		ecJSON, _ := json.Marshal(ecDevices[i])
		if string(legacyJSON) != string(ecJSON) {
			t.Errorf("device[%d] %q payload mismatch:\n  legacy:     %s\n  entcollect: %s",
				i, legacyDevices[i].ID, legacyJSON, ecJSON)
		}
	}
}

// TestEquivalence_FullSync runs the entcollect provider's FullSync against
// an httptest mock and verifies it produces the expected user and device
// documents with correct group enrichment.
func TestEquivalence_FullSync(t *testing.T) {
	srv := startEquivOktaServer(t)

	cfg := ecokta.DefaultConfig()
	cfg.Domain = strings.TrimPrefix(srv.URL, "https://")
	cfg.Token = "test-token"
	cfg.EnrichWith = []string{"groups"}

	p := ecokta.NewWithClient(cfg, srv.Client())

	store := newEquivMemStore()
	var docs []entcollect.Document
	pub := func(_ context.Context, doc entcollect.Document) error {
		docs = append(docs, doc)
		return nil
	}

	log := slog.New(slog.NewTextHandler(&testWriter{t}, nil))
	if err := p.FullSync(t.Context(), store, pub, log); err != nil {
		t.Fatalf("FullSync: %v", err)
	}

	users := filterDocsByKind(docs, entcollect.KindUser)
	devices := filterDocsByKind(docs, entcollect.KindDevice)

	if len(users) != 3 {
		t.Errorf("user count: got %d, want 3", len(users))
	}
	if len(devices) != 1 {
		t.Errorf("device count: got %d, want 1", len(devices))
	}

	// Verify group enrichment: u1 is in both groups, u2 is in group g1 only.
	for _, doc := range users {
		groups, _ := doc.Fields["groups"].([]ecokta.Group)
		switch doc.ID {
		case "u1":
			if len(groups) != 2 {
				t.Errorf("u1 has %d groups, want 2", len(groups))
			}
		case "u2":
			if len(groups) != 1 {
				t.Errorf("u2 has %d groups, want 1", len(groups))
			}
		case "u3":
			if len(groups) != 0 {
				t.Errorf("u3 (DEPROVISIONED) has %d groups, want 0", len(groups))
			}
		}
	}

	// Verify the legacy API returns the same user set by calling
	// GetUserDetails directly against the same mock.
	log2 := logptest.NewTestingLogger(t, "equiv")
	lim := legacyokta.NewRateLimiter(time.Minute, nil)
	legacyUsers, _, err := legacyokta.GetUserDetails(t.Context(), srv.Client(), srv.Listener.Addr().String(), "test-token", "", nil, legacyokta.OmitCredentials|legacyokta.OmitCredentialsLinks|legacyokta.OmitTransitioningToStatus, lim, log2)
	if err != nil {
		t.Fatalf("legacy GetUserDetails: %v", err)
	}

	// The mock serves all users regardless of omit flags, so compare IDs.
	legacyIDs := make([]string, len(legacyUsers))
	for i, u := range legacyUsers {
		legacyIDs[i] = u.ID
	}
	sort.Strings(legacyIDs)

	ecIDs := make([]string, len(users))
	for i, d := range users {
		ecIDs[i] = d.ID
	}
	sort.Strings(ecIDs)

	if len(legacyIDs) != len(ecIDs) {
		t.Fatalf("user ID count: legacy=%d, entcollect=%d", len(legacyIDs), len(ecIDs))
	}
	for i := range legacyIDs {
		if legacyIDs[i] != ecIDs[i] {
			t.Errorf("user ID[%d]: legacy=%q, entcollect=%q", i, legacyIDs[i], ecIDs[i])
		}
	}
}

func startEquivOktaServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(equivUserFixtureJSON())
	})

	mux.HandleFunc("/api/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(equivGroupFixtureJSON())
	})

	mux.HandleFunc("/api/v1/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"u1"},{"id":"u2"}]`))
	})

	mux.HandleFunc("/api/v1/groups/g2/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"u1"}]`))
	})

	mux.HandleFunc("/api/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(equivDeviceFixtureJSON())
	})

	mux.HandleFunc("/api/v1/devices/d1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"created":"2026-01-01T12:00:00.000Z","managementStatus":"NOT_MANAGED","screenLockType":"NONE","user":{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}}}]`))
	})

	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func equivUserFixtureJSON() []byte {
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	return []byte(`[
		{
			"id": "u1",
			"status": "ACTIVE",
			"created": "2026-01-01T12:00:00.000Z",
			"activated": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "` + now.Format(ecokta.ISO8601) + `",
			"profile": {"login": "alice@example.com", "email": "alice@example.com", "firstName": "Alice", "lastName": "Smith", "managerId": "u2"}
		},
		{
			"id": "u2",
			"status": "ACTIVE",
			"created": "2026-01-01T12:00:00.000Z",
			"activated": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "` + now.Format(ecokta.ISO8601) + `",
			"profile": {"login": "bob@example.com", "email": "bob@example.com", "firstName": "Bob", "lastName": "Jones"}
		},
		{
			"id": "u3",
			"status": "DEPROVISIONED",
			"created": "2026-01-01T12:00:00.000Z",
			"activated": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "` + now.Format(ecokta.ISO8601) + `",
			"profile": {"login": "charlie@example.com", "email": "charlie@example.com", "firstName": "Charlie", "lastName": "Brown"}
		}
	]`)
}

func equivGroupFixtureJSON() []byte {
	return []byte(`[
		{"id": "g1", "profile": {"name": "Staff", "description": "All staff"}},
		{"id": "g2", "profile": {"name": "Engineering", "description": "Engineering team"}}
	]`)
}

func equivDeviceFixtureJSON() []byte {
	return []byte(`[
		{
			"id": "d1",
			"created": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "2026-01-02T12:00:00.000Z",
			"status": "ACTIVE",
			"profile": {"displayName": "Alice Laptop"},
			"resourceAlternateID": "",
			"resourceDisplayName": {"sensitive": false, "value": "Alice Laptop"},
			"resourceID": "d1",
			"resourceType": "UDDevice"
		}
	]`)
}

func filterDocsByKind(docs []entcollect.Document, kind entcollect.EntityKind) []entcollect.Document {
	var out []entcollect.Document
	for _, d := range docs {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	return out
}

type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

type equivMemStore struct {
	data map[string]json.RawMessage
}

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

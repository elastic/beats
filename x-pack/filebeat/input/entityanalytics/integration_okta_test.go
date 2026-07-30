// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	ecokta "github.com/elastic/entcollect/provider/okta"
)

func TestOktaIntegration_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	fix := newOktaFixture()
	fix.set(oktaData{
		users: `[
			{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}},
			{"id":"u2","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"bob@example.com","email":"bob@example.com","firstName":"Bob","lastName":"Jones"}},
			{"id":"u3","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"charlie@example.com","email":"charlie@example.com","firstName":"Charlie","lastName":"Brown"}}
		]`,
		groups: `[
			{"id":"g1","profile":{"name":"Staff","description":"All staff"}},
			{"id":"g2","profile":{"name":"Engineering","description":"Engineering team"}}
		]`,
		groupMembers: map[string]string{
			"g1": `[{"id":"u1"},{"id":"u2"},{"id":"u3"}]`,
			"g2": `[{"id":"u1"}]`,
		},
		devices: `[
			{"id":"d1","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","status":"ACTIVE","profile":{"displayName":"Alice Laptop"},"resourceID":"d1","resourceType":"UDDevice"}
		]`,
		deviceUsers: map[string]string{
			"d1": `[{"created":"2026-01-01T12:00:00.000Z","managementStatus":"NOT_MANAGED","screenLockType":"NONE","user":{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}}}]`,
		},
	})

	srv := startOktaIntegServer(t, fix)

	limitFixed := 1000
	newProvider := func() *ecokta.Provider {
		cfg := ecokta.DefaultConfig()
		cfg.Domain = strings.TrimPrefix(srv.URL, "https://")
		cfg.Token = "test-token"
		cfg.EnrichWith = []string{"groups"}
		cfg.LimitFixed = &limitFixed
		return ecokta.NewWithClient(cfg, srv.Client())
	}

	// Run 1: full sync discovers all entities.
	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "okta", newProvider(), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	events1 := client1.Events()
	if got := len(events1); got != 4 {
		t.Errorf("first sync: got %d events, want 4 (3 users + 1 device)", got)
	}
	if got := countByAction(events1, "user-discovered"); got != 3 {
		t.Errorf("first sync: got %d user-discovered, want 3", got)
	}
	if got := countByAction(events1, "device-discovered"); got != 1 {
		t.Errorf("first sync: got %d device-discovered, want 1", got)
	}

	// Assert bbolt state keys.
	assertBboltKey(t, tmpDir, "okta", "okta.cursor.user.last_sync")
	assertBboltKey(t, tmpDir, "okta", "okta.users.idset.meta")
	assertBboltKey(t, tmpDir, "okta", "okta.cursor.device.last_sync")
	assertBboltKey(t, tmpDir, "okta", "okta.devices.idset.meta")

	// Run 2: remove u3, restart with same bbolt.
	fix.set(oktaData{
		users: `[
			{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}},
			{"id":"u2","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"bob@example.com","email":"bob@example.com","firstName":"Bob","lastName":"Jones"}}
		]`,
		groups: `[
			{"id":"g1","profile":{"name":"Staff","description":"All staff"}},
			{"id":"g2","profile":{"name":"Engineering","description":"Engineering team"}}
		]`,
		groupMembers: map[string]string{
			"g1": `[{"id":"u1"},{"id":"u2"}]`,
			"g2": `[{"id":"u1"}]`,
		},
		devices: `[
			{"id":"d1","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","status":"ACTIVE","profile":{"displayName":"Alice Laptop"},"resourceID":"d1","resourceType":"UDDevice"}
		]`,
		deviceUsers: map[string]string{
			"d1": `[{"created":"2026-01-01T12:00:00.000Z","managementStatus":"NOT_MANAGED","screenLockType":"NONE","user":{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}}}]`,
		},
	})

	client2 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "okta", newProvider(), client2); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	events2 := client2.Events()
	if got := len(events2); got != 4 {
		t.Errorf("second sync: got %d events, want 4 (2 user-modified + 1 device-modified + 1 user-deleted)", got)
	}
	if got := countByAction(events2, "user-modified"); got != 2 {
		t.Errorf("second sync: got %d user-modified, want 2", got)
	}
	if got := countByAction(events2, "device-modified"); got != 1 {
		t.Errorf("second sync: got %d device-modified, want 1", got)
	}
	if got := countByAction(events2, "user-deleted"); got != 1 {
		t.Errorf("second sync: got %d user-deleted, want 1", got)
	}

	deletedID := ""
	for _, e := range events2 {
		action, _ := e.Fields.GetValue("event.action")
		if action == "user-deleted" {
			id, _ := e.Fields.GetValue("user.id")
			deletedID, _ = id.(string)
		}
	}
	if deletedID != "u3" {
		t.Errorf("deleted user ID = %q, want %q", deletedID, "u3")
	}
}

func TestOktaIntegration_CursorRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	fix := newOktaFixture()
	fix.set(oktaData{
		users: `[
			{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}}
		]`,
		groups:       `[]`,
		groupMembers: map[string]string{},
		devices:      `[]`,
		deviceUsers:  map[string]string{},
	})

	srv := startOktaIntegServer(t, fix)

	limitFixed := 1000
	cfg := ecokta.DefaultConfig()
	cfg.Domain = strings.TrimPrefix(srv.URL, "https://")
	cfg.Token = "test-token"
	cfg.EnrichWith = []string{"groups"}
	cfg.LimitFixed = &limitFixed

	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "okta", ecokta.NewWithClient(cfg, srv.Client()), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if got := len(client1.Events()); got != 1 {
		t.Fatalf("first sync: got %d events, want 1", got)
	}

	// Second run: same fixture, same bbolt -> cursor proves modified.
	client2 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "okta", ecokta.NewWithClient(cfg, srv.Client()), client2); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	events := client2.Events()
	if len(events) != 1 {
		t.Fatalf("second sync: got %d events, want 1", len(events))
	}
	action, _ := events[0].Fields.GetValue("event.action")
	if action != "user-modified" {
		t.Errorf("second sync action = %v, want %q (proves cursor roundtrip)", action, "user-modified")
	}
}

// Okta test infrastructure

type oktaData struct {
	users        string
	groups       string
	groupMembers map[string]string // groupID -> JSON members array
	devices      string
	deviceUsers  map[string]string // deviceID -> JSON device-users array
}

type oktaFixture struct {
	mu   sync.Mutex
	data oktaData
}

func newOktaFixture() *oktaFixture {
	return &oktaFixture{}
}

func (f *oktaFixture) set(d oktaData) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = d
}

func (f *oktaFixture) get() oktaData {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data
}

func startOktaIntegServer(t *testing.T, fix *oktaFixture) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fix.get().users))
	})

	mux.HandleFunc("GET /api/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fix.get().groups))
	})

	mux.HandleFunc("GET /api/v1/groups/{groupID}/users", func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("groupID")
		d := fix.get()
		members, ok := d.groupMembers[groupID]
		if !ok {
			members = "[]"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(members))
	})

	mux.HandleFunc("GET /api/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fix.get().devices))
	})

	mux.HandleFunc("GET /api/v1/devices/{deviceID}/users", func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.PathValue("deviceID")
		d := fix.get()
		users, ok := d.deviceUsers[deviceID]
		if !ok {
			users = "[]"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(users))
	})

	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

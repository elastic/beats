// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	ecentraid "github.com/elastic/entcollect/provider/entraid"
)

// TestEntraidIntegration_FullLifecycle exercises the EntraID adapter end-to-end.
//
// Run 1: full sync discovers users and devices.
// Run 2: full sync with one @removed user → remaining discovered + removed deleted.
func TestEntraidIntegration_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	fix := newEntraidFixture()
	fix.set(entraidData{
		users: []map[string]any{
			{"id": "u1", "userPrincipalName": "alice@example.com", "displayName": "Alice"},
			{"id": "u2", "userPrincipalName": "bob@example.com", "displayName": "Bob"},
		},
		devices: []map[string]any{
			{"id": "d1", "displayName": "Workstation-1", "operatingSystem": "Windows"},
		},
	})

	srv := startEntraidIntegServer(t, fix)

	newProvider := func() *ecentraid.Provider {
		cfg := ecentraid.DefaultConfig()
		cfg.TenantID = "test-tenant"
		cfg.ClientID = "test-client"
		cfg.ClientSecret = "test-secret"
		cfg.LoginEndpoint = srv.URL
		cfg.APIEndpoint = srv.URL + "/v1.0"
		cfg.Dataset = "all"
		return ecentraid.NewWithClient(cfg, srv.Client())
	}

	// Run 1: full sync discovers all entities.
	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "azure-ad", newProvider(), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	events1 := client1.Events()
	if got := len(events1); got != 3 {
		t.Errorf("first sync: got %d events, want 3 (2 users + 1 device)", got)
	}
	if got := countByAction(events1, "user-discovered"); got != 2 {
		t.Errorf("first sync: got %d user-discovered, want 2", got)
	}
	if got := countByAction(events1, "device-discovered"); got != 1 {
		t.Errorf("first sync: got %d device-discovered, want 1", got)
	}

	// Assert delta link keys are written.
	assertBboltKey(t, tmpDir, "azure-ad", "entraid.cursor.users_delta")
	assertBboltKey(t, tmpDir, "azure-ad", "entraid.cursor.devices_delta")

	// Run 2: @removed user, full sync → remaining discovered + removed deleted.
	fix.set(entraidData{
		users: []map[string]any{
			{"id": "u1", "userPrincipalName": "alice@example.com", "displayName": "Alice"},
			{"id": "u2", "@removed": map[string]any{"reason": "deleted"}},
		},
		devices: []map[string]any{
			{"id": "d1", "displayName": "Workstation-1", "operatingSystem": "Windows"},
		},
	})

	client2 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "azure-ad", newProvider(), client2); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	events2 := client2.Events()
	if got := len(events2); got != 3 {
		t.Errorf("second sync: got %d events, want 3 (1 user-discovered + 1 user-deleted + 1 device-discovered)", got)
	}
	if got := countByAction(events2, "user-discovered"); got != 1 {
		t.Errorf("second sync: got %d user-discovered, want 1", got)
	}
	if got := countByAction(events2, "user-deleted"); got != 1 {
		t.Errorf("second sync: got %d user-deleted, want 1", got)
	}
	if got := countByAction(events2, "device-discovered"); got != 1 {
		t.Errorf("second sync: got %d device-discovered, want 1", got)
	}

	deletedID := ""
	for _, e := range events2 {
		action, _ := e.Fields.GetValue("event.action")
		if action == "user-deleted" {
			id, _ := e.Fields.GetValue("user.id")
			deletedID, _ = id.(string)
		}
	}
	if deletedID != "u2" {
		t.Errorf("deleted user ID = %q, want %q", deletedID, "u2")
	}
}

// TestEntraidIntegration_IncrementalDeltaLink verifies that delta links
// stored in bbolt by full sync are read back and used by incremental
// sync. Entities arrive as *-modified (IncrementalSync), proving the
// delta link was round-tripped through bbolt.
func TestEntraidIntegration_IncrementalDeltaLink(t *testing.T) {
	tmpDir := t.TempDir()

	fix := newEntraidFixture()
	fix.set(entraidData{
		users: []map[string]any{
			{"id": "u1", "userPrincipalName": "alice@example.com", "displayName": "Alice"},
		},
		devices: []map[string]any{
			{"id": "d1", "displayName": "Workstation-1", "operatingSystem": "Windows"},
		},
		deltaUsers: []map[string]any{
			{"id": "u1", "displayName": "Alice Updated"},
		},
		deltaDevices: []map[string]any{
			{"id": "d1", "displayName": "Workstation-1 Updated"},
		},
	})

	srv := startEntraidIntegServer(t, fix)

	newProvider := func() *ecentraid.Provider {
		cfg := ecentraid.DefaultConfig()
		cfg.TenantID = "test-tenant"
		cfg.ClientID = "test-client"
		cfg.ClientSecret = "test-secret"
		cfg.LoginEndpoint = srv.URL
		cfg.APIEndpoint = srv.URL + "/v1.0"
		cfg.Dataset = "all"
		return ecentraid.NewWithClient(cfg, srv.Client())
	}

	// Seed: full sync stores delta links.
	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "azure-ad", newProvider(), client1); err != nil {
		t.Fatalf("seed run failed: %v", err)
	}
	assertBboltKey(t, tmpDir, "azure-ad", "entraid.cursor.users_delta")
	assertBboltKey(t, tmpDir, "azure-ad", "entraid.cursor.devices_delta")

	// Incremental sync: delta link from bbolt → delta data.
	client2 := &fakeClient{}
	if err := runIncrementalOnce(t, tmpDir, "azure-ad", newProvider(), client2); err != nil {
		t.Fatalf("incremental run failed: %v", err)
	}

	events := client2.Events()
	if got := len(events); got != 2 {
		t.Errorf("incremental sync: got %d events, want 2 (1 user + 1 device)", got)
	}
	if got := countByAction(events, "user-modified"); got != 1 {
		t.Errorf("incremental sync: got %d user-modified, want 1", got)
	}
	if got := countByAction(events, "device-modified"); got != 1 {
		t.Errorf("incremental sync: got %d device-modified, want 1", got)
	}
}

// TestEntraidIntegration_DeltaRecovery verifies that an expired delta
// link (HTTP 410) causes the provider to fall back to a full re-fetch
// within IncrementalSync. Entities still arrive as *-modified.
func TestEntraidIntegration_DeltaRecovery(t *testing.T) {
	tmpDir := t.TempDir()

	fix := newEntraidFixture()
	fix.set(entraidData{
		users: []map[string]any{
			{"id": "u1", "userPrincipalName": "alice@example.com", "displayName": "Alice"},
		},
		devices: []map[string]any{
			{"id": "d1", "displayName": "Workstation-1", "operatingSystem": "Windows"},
		},
	})

	srv := startEntraidIntegServer(t, fix)

	newProvider := func() *ecentraid.Provider {
		cfg := ecentraid.DefaultConfig()
		cfg.TenantID = "test-tenant"
		cfg.ClientID = "test-client"
		cfg.ClientSecret = "test-secret"
		cfg.LoginEndpoint = srv.URL
		cfg.APIEndpoint = srv.URL + "/v1.0"
		cfg.Dataset = "all"
		return ecentraid.NewWithClient(cfg, srv.Client())
	}

	// Seed: full sync stores delta links.
	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "azure-ad", newProvider(), client1); err != nil {
		t.Fatalf("seed run failed: %v", err)
	}

	// Expire the delta links: mock returns 410 for delta-token requests.
	fix.setDeltaStatus(http.StatusGone)

	// Incremental sync with expired delta → provider falls back to full fetch.
	client2 := &fakeClient{}
	if err := runIncrementalOnce(t, tmpDir, "azure-ad", newProvider(), client2); err != nil {
		t.Fatalf("delta recovery run failed: %v", err)
	}

	events := client2.Events()
	if len(events) == 0 {
		t.Fatal("delta recovery: no events produced; want entities from fallback fetch")
	}
	// IncrementalSync emits ActionModified even after recovery.
	if got := countByAction(events, "user-modified"); got != 1 {
		t.Errorf("delta recovery: got %d user-modified, want 1", got)
	}
	if got := countByAction(events, "device-modified"); got != 1 {
		t.Errorf("delta recovery: got %d device-modified, want 1", got)
	}
}

// EntraID test infrastructure

type entraidData struct {
	users        []map[string]any
	devices      []map[string]any
	deltaUsers   []map[string]any // served for delta-token requests; nil → use users
	deltaDevices []map[string]any // served for delta-token requests; nil → use devices
}

type entraidFixture struct {
	mu          sync.Mutex
	data        entraidData
	deltaStatus int // if non-zero, return this status for delta-token requests
}

func newEntraidFixture() *entraidFixture {
	return &entraidFixture{}
}

func (f *entraidFixture) set(d entraidData) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = d
	f.deltaStatus = 0
}

func (f *entraidFixture) setDeltaStatus(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deltaStatus = status
}

func (f *entraidFixture) get() (entraidData, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data, f.deltaStatus
}

func startEntraidIntegServer(t *testing.T, fix *entraidFixture) *httptest.Server {
	t.Helper()

	var srvURL string
	mux := http.NewServeMux()

	deltaLink := func(path string) string {
		return srvURL + path
	}

	// OAuth token endpoint.
	mux.HandleFunc("POST /test-tenant/oauth2/v2.0/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	// Users delta.
	mux.HandleFunc("GET /v1.0/users/delta", func(w http.ResponseWriter, r *http.Request) {
		d, status := fix.get()
		isDelta := r.URL.Query().Has("$deltatoken")

		if isDelta && status != 0 {
			w.WriteHeader(status)
			return
		}

		users := d.users
		if isDelta && d.deltaUsers != nil {
			users = d.deltaUsers
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"@odata.deltaLink": deltaLink("/v1.0/users/delta?$deltatoken=test"),
			"value":            users,
		})
	})

	// Devices delta.
	mux.HandleFunc("GET /v1.0/devices/delta", func(w http.ResponseWriter, r *http.Request) {
		d, status := fix.get()
		isDelta := r.URL.Query().Has("$deltatoken")

		if isDelta && status != 0 {
			w.WriteHeader(status)
			return
		}

		devices := d.devices
		if isDelta && d.deltaDevices != nil {
			devices = d.deltaDevices
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"@odata.deltaLink": deltaLink("/v1.0/devices/delta?$deltatoken=test"),
			"value":            devices,
		})
	})

	// Groups list — empty, not testing group enrichment.
	mux.HandleFunc("GET /v1.0/groups", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
	})

	// Device registered owners/users — empty.
	mux.HandleFunc("GET /v1.0/devices/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
	})

	srv := httptest.NewServer(mux)
	srvURL = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

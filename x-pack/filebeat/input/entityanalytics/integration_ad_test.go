// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/jimlambrt/gldap"

	ecad "github.com/elastic/entcollect/provider/ad"
)

func TestADIntegration_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	fix := &adFixture{
		users: []adEntry{
			{
				dn: "cn=alice,dc=example,dc=com",
				attrs: map[string][]string{
					"cn":                {"alice"},
					"distinguishedName": {"cn=alice,dc=example,dc=com"},
					"mail":              {"alice@example.com"},
					"whenChanged":       {"20260101120000.0Z"},
				},
			},
			{
				dn: "cn=bob,dc=example,dc=com",
				attrs: map[string][]string{
					"cn":                {"bob"},
					"distinguishedName": {"cn=bob,dc=example,dc=com"},
					"mail":              {"bob@example.com"},
					"whenChanged":       {"20260101130000.0Z"},
				},
			},
		},
		devices: []adEntry{
			{
				dn: "cn=workstation1,dc=example,dc=com",
				attrs: map[string][]string{
					"cn":                {"workstation1"},
					"distinguishedName": {"cn=workstation1,dc=example,dc=com"},
					"whenChanged":       {"20260101140000.0Z"},
				},
			},
		},
	}

	serverURL := startADIntegLDAPServer(t, fix)

	newProvider := func() *ecad.Provider {
		cfg := ecad.DefaultConfig()
		cfg.URL = serverURL
		cfg.BaseDN = "DC=example,DC=com"
		cfg.User = "cn=admin,dc=example,dc=com"
		cfg.Password = "pass"
		p, err := ecad.New(cfg)
		if err != nil {
			t.Fatalf("new AD provider: %v", err)
		}
		return p
	}

	// Run 1: full sync discovers all entities.
	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "activedirectory", newProvider(), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	events1 := client1.Events()
	if got := len(events1); got != 3 {
		t.Errorf("first sync: got %d events, want 3", got)
	}
	if got := countByAction(events1, "user-discovered"); got != 2 {
		t.Errorf("first sync: got %d user-discovered, want 2", got)
	}
	if got := countByAction(events1, "device-discovered"); got != 1 {
		t.Errorf("first sync: got %d device-discovered, want 1", got)
	}

	// Assert bbolt state keys after first run.
	assertBboltKey(t, tmpDir, "activedirectory", "ad.cursor.when_changed")
	assertBboltKey(t, tmpDir, "activedirectory", "ad.users.idset.meta")
	assertBboltKey(t, tmpDir, "activedirectory", "ad.devices.idset.meta")

	// Run 2: remove bob, restart with same bbolt.
	fix.users = []adEntry{
		{
			dn: "cn=alice,dc=example,dc=com",
			attrs: map[string][]string{
				"cn":                {"alice"},
				"distinguishedName": {"cn=alice,dc=example,dc=com"},
				"mail":              {"alice@example.com"},
				"whenChanged":       {"20260101120000.0Z"},
			},
		},
	}

	client2 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "activedirectory", newProvider(), client2); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	events2 := client2.Events()
	if got := len(events2); got != 3 {
		t.Errorf("second sync: got %d events, want 3 (1 user-modified + 1 device-modified + 1 user-deleted)", got)
	}
	if got := countByAction(events2, "user-modified"); got != 1 {
		t.Errorf("second sync: got %d user-modified, want 1", got)
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
	if deletedID != "cn=bob,dc=example,dc=com" {
		t.Errorf("deleted user ID = %q, want %q", deletedID, "cn=bob,dc=example,dc=com")
	}
}

func TestADIntegration_CursorRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	fix := &adFixture{
		users: []adEntry{
			{
				dn: "cn=alice,dc=example,dc=com",
				attrs: map[string][]string{
					"cn":                {"alice"},
					"distinguishedName": {"cn=alice,dc=example,dc=com"},
					"whenChanged":       {"20260101120000.0Z"},
				},
			},
		},
	}

	serverURL := startADIntegLDAPServer(t, fix)

	cfg := ecad.DefaultConfig()
	cfg.URL = serverURL
	cfg.BaseDN = "DC=example,DC=com"
	cfg.User = "cn=admin,dc=example,dc=com"
	cfg.Password = "pass"

	newProvider := func() *ecad.Provider {
		p, err := ecad.New(cfg)
		if err != nil {
			t.Fatalf("new AD provider: %v", err)
		}
		return p
	}

	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "activedirectory", newProvider(), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if got := len(client1.Events()); got != 1 {
		t.Fatalf("first sync: got %d events, want 1", got)
	}

	// Second run: same fixture, same bbolt -> cursor round-trip proves
	// "modified" instead of "discovered".
	client2 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "activedirectory", newProvider(), client2); err != nil {
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

// TestADIntegration_MissingCursor verifies that deleting the AD cursor
// from bbolt causes the incremental sync to treat it as a zero-time
// cursor and still emit entities. This tests the adapter-level recovery
// path — the provider falls back to fetching everything when the cursor
// is missing.
func TestADIntegration_MissingCursor(t *testing.T) {
	tmpDir := t.TempDir()

	fix := &adFixture{
		users: []adEntry{
			{
				dn: "cn=alice,dc=example,dc=com",
				attrs: map[string][]string{
					"cn":                {"alice"},
					"distinguishedName": {"cn=alice,dc=example,dc=com"},
					"whenChanged":       {"20260101120000.0Z"},
				},
			},
		},
	}

	serverURL := startADIntegLDAPServer(t, fix)

	cfg := ecad.DefaultConfig()
	cfg.URL = serverURL
	cfg.BaseDN = "DC=example,DC=com"
	cfg.User = "cn=admin,dc=example,dc=com"
	cfg.Password = "pass"

	newProvider := func() *ecad.Provider {
		p, err := ecad.New(cfg)
		if err != nil {
			t.Fatalf("new AD provider: %v", err)
		}
		return p
	}

	// Run 1: full sync to populate bbolt with cursor and idset.
	client1 := &fakeClient{}
	if err := runInputOnce(t, tmpDir, "activedirectory", newProvider(), client1); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if got := len(client1.Events()); got != 1 {
		t.Fatalf("first sync: got %d events, want 1", got)
	}
	assertBboltKey(t, tmpDir, "activedirectory", "ad.cursor.when_changed")

	// Delete the cursor from bbolt. The provider should treat the
	// missing cursor as zero time and fetch all entities.
	deleteBboltKey(t, tmpDir, "activedirectory", "ad.cursor.when_changed")

	// Run incremental sync with missing cursor.
	client2 := &fakeClient{}
	if err := runIncrementalOnce(t, tmpDir, "activedirectory", newProvider(), client2); err != nil {
		t.Fatalf("incremental with missing cursor failed: %v", err)
	}

	events := client2.Events()
	if len(events) == 0 {
		t.Error("incremental sync with missing cursor produced no events; want at least 1")
	}
}

// AD test infrastructure

type adFixture struct {
	users   []adEntry
	devices []adEntry
	groups  []adEntry
}

type adEntry struct {
	dn    string
	attrs map[string][]string
}

type adLDAPServer struct {
	fix *adFixture
}

// startADIntegLDAPServer starts a gldap server backed by fix and returns its
// URL. gldap does not expose the address of its listener, so the port is
// chosen by probing for a free one and then handing it to gldap. Another
// process can claim the port in between, and gldap reports Ready even when
// its listen fails, so each attempt is confirmed by binding to the server
// and the whole start is retried on failure.
func startADIntegLDAPServer(t *testing.T, fix *adFixture) string {
	t.Helper()

	const attempts = 5
	var lastErr error
	for range attempts {
		url, err := tryStartADIntegLDAPServer(t, fix)
		if err == nil {
			return url
		}
		lastErr = err
		t.Logf("starting gldap server: %v; retrying", err)
	}
	t.Fatalf("gldap server did not start after %d attempts: %v", attempts, lastErr)
	return ""
}

func tryStartADIntegLDAPServer(t *testing.T, fix *adFixture) (string, error) {
	t.Helper()

	s, err := gldap.NewServer()
	if err != nil {
		t.Fatalf("gldap new server: %v", err)
	}

	mux, err := gldap.NewMux()
	if err != nil {
		t.Fatalf("gldap new mux: %v", err)
	}

	// served is set when our own handler sees a bind, which distinguishes
	// this server from any other process that holds the port.
	var served atomic.Bool
	if err := mux.Bind(func(w *gldap.ResponseWriter, r *gldap.Request) {
		served.Store(true)
		resp := r.NewBindResponse()
		resp.SetResultCode(gldap.ResultSuccess)
		_ = w.Write(resp)
	}); err != nil {
		t.Fatalf("mux bind: %v", err)
	}

	srv := &adLDAPServer{fix: fix}
	if err := mux.Search(srv.searchHandler(t)); err != nil {
		t.Fatalf("mux search: %v", err)
	}

	if err := mux.Unbind(func(w *gldap.ResponseWriter, r *gldap.Request) {}); err != nil {
		t.Fatalf("mux unbind: %v", err)
	}

	if err := s.Router(mux); err != nil {
		t.Fatalf("gldap router: %v", err)
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	runErr := make(chan error, 1)
	go func() { runErr <- s.Run(addr) }()

	url := "ldap://" + addr
	err = probeADIntegLDAPServer(url, runErr, &served)
	if err != nil {
		_ = s.Stop()
		return "", err
	}
	t.Cleanup(func() { _ = s.Stop() })
	return url, nil
}

// probeADIntegLDAPServer binds to the server at url and reports an error
// unless the bind was handled by the server whose Run result is runErr.
func probeADIntegLDAPServer(url string, runErr <-chan error, served *atomic.Bool) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case err := <-runErr:
			if err == nil {
				return errors.New("server stopped before probe finished")
			}
			return fmt.Errorf("run: %w", err)
		default:
		}

		conn, err := ldap.DialURL(url, ldap.DialWithDialer(&net.Dialer{Timeout: time.Second}))
		if err == nil {
			conn.SetTimeout(time.Second)
			err = conn.Bind("cn=admin,dc=example,dc=com", "pass")
			conn.Close()
			if err == nil && served.Load() {
				return nil
			}
			if err == nil {
				return fmt.Errorf("port at %s is held by another process", url)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("probing %s: %w", url, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *adLDAPServer) searchHandler(t *testing.T) gldap.HandlerFunc {
	t.Helper()
	return func(w *gldap.ResponseWriter, r *gldap.Request) {
		msg, err := r.GetSearchMessage()
		if err != nil {
			t.Errorf("get search message: %v", err)
			return
		}

		filter := msg.Filter
		fix := s.fix

		var results []adEntry

		switch {
		case strings.Contains(filter, "objectClass=group"):
			results = fix.groups
		case strings.Contains(filter, "objectClass=computer"):
			results = fix.devices
		case strings.Contains(filter, "objectCategory=person"):
			results = fix.users
		default:
			t.Logf("unhandled filter: %s", filter)
		}

		for _, entry := range results {
			e := r.NewSearchResponseEntry(entry.dn)
			for name, vals := range entry.attrs {
				e.AddAttribute(name, vals)
			}
			_ = w.Write(e)
		}

		done := r.NewSearchDoneResponse()
		done.SetResultCode(gldap.ResultSuccess)
		_ = w.Write(done)
	}
}

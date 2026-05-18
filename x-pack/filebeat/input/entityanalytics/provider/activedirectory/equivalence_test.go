// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/jimlambrt/gldap"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/entcollect"
	ecad "github.com/elastic/entcollect/provider/ad"
)

// TestEquivalence_FullSync runs both the legacy and minimal-state AD providers
// against the same gldap mock and asserts that the activedirectory.* payloads
// are structurally equivalent.
//
// Intentional differences between the two providers (documented here):
//   - Legacy emits start/end marker events; minimal-state does not.
//   - Legacy uses event.action "user-discovered"; minimal-state uses
//     entcollect.ActionDiscovered (semantically equivalent, different format).
//   - Legacy does not set event.kind; minimal-state consumer may set "asset".
//   - Deletion detection differs: legacy uses kvstore state comparison;
//     minimal-state uses idset. First-sync output (no prior state) should
//     match for discovered entities.
func TestEquivalence_FullSync(t *testing.T) {
	url := startEquivLDAPServer(t)

	// Run the legacy provider's data extraction.
	legacyUsers, legacyDevices := runLegacyFetch(t, url)

	// Run the minimal-state provider.
	minimalDocs := runMinimalFullSync(t, url)

	// Compare user payloads.
	minimalUsers := filterDocsByKind(minimalDocs, entcollect.KindUser)
	if len(legacyUsers) != len(minimalUsers) {
		t.Fatalf("user count: legacy=%d, minimal=%d", len(legacyUsers), len(minimalUsers))
	}
	compareSortedEntries(t, "user", legacyUsers, minimalUsers)

	// Compare device payloads.
	minimalDevices := filterDocsByKind(minimalDocs, entcollect.KindDevice)
	if len(legacyDevices) != len(minimalDevices) {
		t.Fatalf("device count: legacy=%d, minimal=%d", len(legacyDevices), len(minimalDevices))
	}
	compareSortedEntries(t, "device", legacyDevices, minimalDevices)
}

func runLegacyFetch(t *testing.T, url string) (users, devices []json.RawMessage) {
	t.Helper()
	base, err := ldap.ParseDN("DC=example,DC=com")
	if err != nil {
		t.Fatal(err)
	}
	a := adInput{
		cfg: conf{
			BaseDN:   "DC=example,DC=com",
			URL:      url,
			User:     "cn=admin,dc=example,dc=com",
			Password: "pass",
		},
		baseDN: base,
		logger: logptest.NewTestingLogger(t, "test-legacy"),
	}
	ss := &stateStore{
		users:   make(map[string]*User),
		devices: make(map[string]*User),
		groups:  make(map[string]*User),
	}

	ctx := context.Background()
	fetchedUsers, err := a.doFetchUsers(ctx, ss, true)
	if err != nil {
		t.Fatalf("legacy doFetchUsers: %v", err)
	}
	fetchedDevices, err := a.doFetchDevices(ctx, ss, true)
	if err != nil {
		t.Fatalf("legacy doFetchDevices: %v", err)
	}

	for _, u := range fetchedUsers {
		b, err := json.Marshal(u.Entry)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, b)
	}
	for _, d := range fetchedDevices {
		b, err := json.Marshal(d.Entry)
		if err != nil {
			t.Fatal(err)
		}
		devices = append(devices, b)
	}
	return users, devices
}

// TestMinimalFullSync_ConfiguredAttrs verifies that when user_attributes
// is set, the minimal-state provider still produces entries with non-empty
// IDs and advances the whenChanged cursor. The gldap mock returns all
// attributes regardless of the request list, so this test exercises the
// entcollect withMandatory logic rather than the LDAP server filtering.
func TestMinimalFullSync_ConfiguredAttrs(t *testing.T) {
	url := startEquivLDAPServer(t)

	cfg := ecad.DefaultConfig()
	cfg.URL = url
	cfg.BaseDN = "DC=example,DC=com"
	cfg.User = "cn=admin,dc=example,dc=com"
	cfg.Password = "pass"
	cfg.Dataset = "users"
	cfg.UserAttrs = []string{"cn", "mail"}

	p, err := ecad.New(cfg)
	if err != nil {
		t.Fatalf("ecad.New: %v", err)
	}

	store := newEquivMemStore()
	var docs []entcollect.Document
	pub := func(_ context.Context, doc entcollect.Document) error {
		docs = append(docs, doc)
		return nil
	}

	log := slog.New(slog.NewTextHandler(&testWriter{t}, nil))
	if err := p.FullSync(context.Background(), store, pub, log); err != nil {
		t.Fatalf("FullSync: %v", err)
	}

	if len(docs) == 0 {
		t.Fatal("expected at least one document")
	}
	for _, doc := range docs {
		if doc.ID == "" {
			t.Errorf("document has empty ID; distinguishedName must be requested even with custom user_attributes")
		}
	}

	var cursor time.Time
	if err := store.Get("ad.cursor.when_changed", &cursor); err != nil {
		t.Fatalf("cursor not written: %v", err)
	}
	if cursor.IsZero() {
		t.Error("cursor is zero; whenChanged must be requested even with custom user_attributes")
	}
}

func runMinimalFullSync(t *testing.T, url string) []entcollect.Document {
	t.Helper()
	cfg := ecad.DefaultConfig()
	cfg.URL = url
	cfg.BaseDN = "DC=example,DC=com"
	cfg.User = "cn=admin,dc=example,dc=com"
	cfg.Password = "pass"

	p, err := ecad.New(cfg)
	if err != nil {
		t.Fatalf("ecad.New: %v", err)
	}

	store := newEquivMemStore()
	var docs []entcollect.Document
	pub := func(_ context.Context, doc entcollect.Document) error {
		docs = append(docs, doc)
		return nil
	}

	log := slog.New(slog.NewTextHandler(&testWriter{t}, nil))
	err = p.FullSync(context.Background(), store, pub, log)
	if err != nil {
		t.Fatalf("minimal FullSync: %v", err)
	}
	return docs
}

func compareSortedEntries(t *testing.T, kind string, legacy []json.RawMessage, minimalDocs []entcollect.Document) {
	t.Helper()

	type idPayload struct {
		id      string
		payload json.RawMessage
	}

	// Extract and sort legacy entries by ID.
	legacyByID := make([]idPayload, 0, len(legacy))
	for _, raw := range legacy {
		var entry struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Fatal(err)
		}
		legacyByID = append(legacyByID, idPayload{id: entry.ID, payload: raw})
	}
	sort.Slice(legacyByID, func(i, j int) bool { return legacyByID[i].id < legacyByID[j].id })

	// Extract and sort minimal entries by ID.
	minimalByID := make([]idPayload, 0, len(minimalDocs))
	for _, doc := range minimalDocs {
		adField := doc.Fields["activedirectory"]
		raw, err := json.Marshal(adField)
		if err != nil {
			t.Fatal(err)
		}
		minimalByID = append(minimalByID, idPayload{id: doc.ID, payload: raw})
	}
	sort.Slice(minimalByID, func(i, j int) bool { return minimalByID[i].id < minimalByID[j].id })

	// Compare IDs first.
	for i := range legacyByID {
		if legacyByID[i].id != minimalByID[i].id {
			t.Errorf("%s[%d] ID mismatch: legacy=%q, minimal=%q", kind, i, legacyByID[i].id, minimalByID[i].id)
			continue
		}

		// Normalize both payloads to map[string]any for comparison,
		// ignoring whenChanged format differences.
		var legacyMap, minimalMap map[string]any
		if err := json.Unmarshal(legacyByID[i].payload, &legacyMap); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(minimalByID[i].payload, &minimalMap); err != nil {
			t.Fatal(err)
		}

		// Remove whenChanged since time formatting may differ slightly.
		delete(legacyMap, "whenChanged")
		delete(minimalMap, "whenChanged")

		legacyNorm, _ := json.Marshal(legacyMap)
		minimalNorm, _ := json.Marshal(minimalMap)
		if string(legacyNorm) != string(minimalNorm) {
			t.Errorf("%s %q payload mismatch:\n  legacy:  %s\n  minimal: %s",
				kind, legacyByID[i].id, legacyNorm, minimalNorm)
		}
	}
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

// startEquivLDAPServer starts a gldap test server with deterministic fixtures.
func startEquivLDAPServer(t *testing.T) string {
	t.Helper()

	s, err := gldap.NewServer()
	if err != nil {
		t.Fatalf("gldap new server: %v", err)
	}
	t.Cleanup(func() { s.Stop() }) //nolint:errcheck // best-effort cleanup

	mux, err := gldap.NewMux()
	if err != nil {
		t.Fatal(err)
	}

	err = mux.Bind(func(w *gldap.ResponseWriter, r *gldap.Request) {
		resp := r.NewBindResponse()
		resp.SetResultCode(gldap.ResultSuccess)
		_ = w.Write(resp)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mux.Search(equivSearchHandler(t))
	if err != nil {
		t.Fatal(err)
	}

	err = mux.Unbind(func(w *gldap.ResponseWriter, r *gldap.Request) {})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Router(mux); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // only used to grab a free port
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	go func() { _ = s.Run(addr) }()
	for i := 0; i < 100; i++ {
		if s.Ready() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !s.Ready() {
		t.Fatal("gldap server not ready")
	}
	return "ldap://" + addr
}

func equivSearchHandler(t *testing.T) gldap.HandlerFunc {
	t.Helper()

	users := []struct {
		dn    string
		attrs map[string][]string
	}{
		{
			dn: "cn=alice,dc=example,dc=com",
			attrs: map[string][]string{
				"cn":                {"alice"},
				"distinguishedName": {"cn=alice,dc=example,dc=com"},
				"mail":              {"alice@example.com"},
				"memberOf":          {"cn=staff,dc=example,dc=com"},
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
	}

	devices := []struct {
		dn    string
		attrs map[string][]string
	}{
		{
			dn: "cn=workstation1,dc=example,dc=com",
			attrs: map[string][]string{
				"cn":                {"workstation1"},
				"distinguishedName": {"cn=workstation1,dc=example,dc=com"},
				"whenChanged":       {"20260101140000.0Z"},
			},
		},
	}

	return func(w *gldap.ResponseWriter, r *gldap.Request) {
		msg, err := r.GetSearchMessage()
		if err != nil {
			t.Errorf("get search message: %v", err)
			return
		}

		filter := msg.Filter

		type entry struct {
			dn    string
			attrs map[string][]string
		}
		var results []entry

		switch {
		case strings.Contains(filter, "objectClass=group"):
			// No groups in this fixture.
		case strings.Contains(filter, "objectClass=computer"):
			for _, d := range devices {
				results = append(results, entry{dn: d.dn, attrs: d.attrs})
			}
		case strings.Contains(filter, "objectCategory=person"):
			for _, u := range users {
				results = append(results, entry{dn: u.dn, attrs: u.attrs})
			}
		default:
			t.Logf("unhandled filter: %s", filter)
		}

		for _, e := range results {
			resp := r.NewSearchResponseEntry(e.dn)
			for name, vals := range e.attrs {
				resp.AddAttribute(name, vals)
			}
			_ = w.Write(resp)
		}

		done := r.NewSearchDoneResponse()
		done.SetResultCode(gldap.ResultSuccess)
		_ = w.Write(done)
	}
}

type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// equivMemStore is a minimal in-memory entcollect.Store for the equivalence test.
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

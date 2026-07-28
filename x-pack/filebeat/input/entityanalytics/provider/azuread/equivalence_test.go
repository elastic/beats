// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"cmp"
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/entcollect"
	ecentraid "github.com/elastic/entcollect/provider/entraid"

	mockauth "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator/mock"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	mockfetcher "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher/mock"
)

// TestEquivalence_FullSync runs both the legacy and minimal-state EntraID
// providers against the same fixture data and asserts that the azure_ad.*
// payloads are structurally equivalent.
//
// Scope: FullSync only. Incremental sync divergences — especially group
// membership invisibility (user delta endpoints do not report group membership
// changes) — are not covered.
//
// Input asymmetry: the legacy side reads Go structs from fetcher/mock (no HTTP,
// no JSON parsing); the entcollect side unmarshals JSON from an httptest Graph
// API mock. This test proves alignment on a shared logical dataset, not that
// both providers produce identical output from live Graph API responses. JSON
// parsing fidelity is tested separately in entcollect/provider/entraid.
//
// Intentional differences between the two providers (documented here):
//   - Legacy emits start/end marker events; minimal-state does not.
//   - Legacy stores entities, groups, and UUIDTree between syncs;
//     minimal-state stores only delta URLs.
//   - Legacy uses group delta queries for incremental group changes;
//     minimal-state re-fetches all groups each sync.
//   - Legacy puts MFA under azure_ad.mfa (fetcher.MFARegistrationDetails);
//     entcollect puts MFA under user.risk.mfa (ecentraid.MFADetails).
//     MFA user-ID coverage is compared; field values are not.
//   - Sign-in activity uses the same field path (azure_ad.signInActivity)
//     on both sides with compatible JSON tags, so values are compared.
//   - event.kind: asset present only in minimal-state (set by adapter).
//   - event.action format differs: legacy uses "user-discovered" etc.;
//     minimal-state uses entcollect.ActionDiscovered.
//   - Device ownership: legacy resolves from state store (cloned full user
//     objects with user.id); entcollect fetches via live API (raw
//     []map[string]any). Output shapes differ structurally, so ownership
//     is excluded from comparison. httptest serves empty ownership responses.
//     Users migrating configs will see different ownership document shapes
//     in production.
func TestEquivalence_FullSync(t *testing.T) {
	legacyUsers, legacyDevices, legacyUserGroups, legacyDeviceGroups, legacyMFAUserIDs, legacySignIn := runLegacyFetch(t)

	srv := startEquivGraphServer(t)
	minimalDocs := runMinimalFullSync(t, srv)

	minimalUsers := filterDocsByKind(minimalDocs, entcollect.KindUser)
	minimalDevices := filterDocsByKind(minimalDocs, entcollect.KindDevice)

	if len(legacyUsers) != len(minimalUsers) {
		t.Fatalf("user count: legacy=%d, minimal=%d", len(legacyUsers), len(minimalUsers))
	}
	if len(legacyDevices) != len(minimalDevices) {
		t.Fatalf("device count: legacy=%d, minimal=%d", len(legacyDevices), len(minimalDevices))
	}

	compareSortedEntries(t, "user", legacyUsers, minimalUsers)
	compareSortedEntries(t, "device", legacyDevices, minimalDevices)

	compareGroups(t, "user", legacyUserGroups, minimalUsers, "user.group")
	compareGroups(t, "device", legacyDeviceGroups, minimalDevices, "device.group")

	compareMFACoverage(t, legacyMFAUserIDs, minimalUsers)
	compareSignInActivity(t, legacySignIn, minimalUsers)

	// Sanity: verify we actually compared non-trivial data.
	if len(legacyUsers) == 0 {
		t.Error("sanity: no legacy users")
	}
	if len(legacyDevices) == 0 {
		t.Error("sanity: no legacy devices")
	}
	if len(legacyUserGroups) == 0 {
		t.Error("sanity: no legacy user groups")
	}
	if len(legacyDeviceGroups) == 0 {
		t.Error("sanity: no legacy device groups")
	}
	if len(legacyMFAUserIDs) == 0 {
		t.Error("sanity: no legacy MFA user IDs")
	}
	if len(legacySignIn) == 0 {
		t.Error("sanity: no legacy sign-in activity data")
	}
}

// runLegacyFetch runs the legacy doFetch path with the mock fetcher and
// extracts entity payloads, group memberships, MFA user-ID coverage,
// and sign-in activity data.
func runLegacyFetch(t *testing.T) (
	users, devices []idPayload,
	userGroups, deviceGroups map[string][]groupECS,
	mfaUserIDs []string,
	signInActivity map[string]json.RawMessage,
) {
	t.Helper()
	a := azure{
		conf:    conf{Dataset: "all", EnrichWith: []string{"mfa", "sign_in_activity"}},
		logger:  logptest.NewTestingLogger(t, "test-legacy"),
		auth:    mockauth.New(""),
		fetcher: mockfetcher.New(),
	}
	ss := &stateStore{
		users:   make(map[uuid.UUID]*fetcher.User),
		devices: make(map[uuid.UUID]*fetcher.Device),
		groups:  make(map[uuid.UUID]*fetcher.Group),
	}

	_, _, err := a.doFetch(context.Background(), ss, true)
	if err != nil {
		t.Fatalf("legacy doFetch: %v", err)
	}

	userGroups = make(map[string][]groupECS)
	signInActivity = make(map[string]json.RawMessage)
	for _, u := range ss.users {
		b, err := json.Marshal(u.Fields)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, idPayload{id: u.ID.String(), payload: b})

		var groups []groupECS
		u.TransitiveMemberOf.ForEach(func(gID uuid.UUID) {
			g, ok := ss.groups[gID]
			if !ok {
				return
			}
			groups = append(groups, groupECS{ID: g.ID.String(), Name: g.Name})
		})
		if len(groups) > 0 {
			userGroups[u.ID.String()] = groups
		}

		if u.MFA != nil {
			mfaUserIDs = append(mfaUserIDs, u.ID.String())
		}
		if u.SignInActivity != nil {
			raw, err := json.Marshal(u.SignInActivity)
			if err != nil {
				t.Fatal(err)
			}
			signInActivity[u.ID.String()] = raw
		}
	}

	deviceGroups = make(map[string][]groupECS)
	for _, d := range ss.devices {
		b, err := json.Marshal(d.Fields)
		if err != nil {
			t.Fatal(err)
		}
		devices = append(devices, idPayload{id: d.ID.String(), payload: b})

		var groups []groupECS
		d.TransitiveMemberOf.ForEach(func(gID uuid.UUID) {
			g, ok := ss.groups[gID]
			if !ok {
				return
			}
			groups = append(groups, groupECS{ID: g.ID.String(), Name: g.Name})
		})
		if len(groups) > 0 {
			deviceGroups[d.ID.String()] = groups
		}
	}

	return users, devices, userGroups, deviceGroups, mfaUserIDs, signInActivity
}

// runMinimalFullSync runs the entcollect FullSync path against the httptest
// server and collects the published documents.
func runMinimalFullSync(t *testing.T, srv *httptest.Server) []entcollect.Document {
	t.Helper()

	cfg := ecentraid.DefaultConfig()
	cfg.TenantID = "test-tenant"
	cfg.ClientID = "test-client"
	cfg.ClientSecret = "test-secret"
	cfg.LoginEndpoint = srv.URL
	cfg.APIEndpoint = srv.URL + "/v1.0"
	cfg.Dataset = "all"
	cfg.EnrichWith = []string{"mfa", "sign_in_activity"}

	p := ecentraid.NewWithClient(cfg, srv.Client())

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

// startEquivGraphServer starts an httptest server that serves Graph API
// responses derived from the fetcher/mock fixture data. Responses use
// proper delta envelopes with @odata.deltaLink and @odata.type annotations,
// following the patterns from entcollect's testGraphMux.
func startEquivGraphServer(t *testing.T) *httptest.Server {
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

	// Users delta — always full (first call in FullSync after clearing cursors).
	mux.HandleFunc("GET /v1.0/users/delta", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@odata.deltaLink": deltaLink("/v1.0/users/delta?$deltatoken=full"),
			"value":            mockUsersAsGraphJSON(),
		})
	})

	// Devices delta — always full.
	mux.HandleFunc("GET /v1.0/devices/delta", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@odata.deltaLink": deltaLink("/v1.0/devices/delta?$deltatoken=full"),
			"value":            mockDevicesAsGraphJSON(),
		})
	})

	// Groups list.
	groups := mockfetcher.GroupResponse
	mux.HandleFunc("GET /v1.0/groups", func(w http.ResponseWriter, _ *http.Request) {
		var groupList []map[string]any
		for _, g := range groups {
			groupList = append(groupList, map[string]any{
				"id":          g.ID.String(),
				"displayName": g.Name,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": groupList})
	})

	// Group members — route /v1.0/groups/{id}/members.
	groupMembers := buildGroupMembersMap()
	mux.HandleFunc("GET /v1.0/groups/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 || parts[4] != "members" {
			http.NotFound(w, r)
			return
		}
		groupID := parts[3]
		members := groupMembers[groupID]
		_ = json.NewEncoder(w).Encode(map[string]any{"value": members})
	})

	// Device registered owners/users — serve empty responses.
	mux.HandleFunc("GET /v1.0/devices/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		relation := parts[4]
		if relation != "registeredOwners" && relation != "registeredUsers" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
	})

	// MFA registration details.
	mux.HandleFunc("GET /v1.0/reports/authenticationMethods/userRegistrationDetails", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": mockMFAAsGraphJSON()})
	})

	// Sign-in activity.
	mux.HandleFunc("GET /v1.0/users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("$select") != "id,signInActivity" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": mockSignInActivityAsGraphJSON()})
	})

	srv := httptest.NewServer(mux)
	srvURL = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

// mockUsersAsGraphJSON converts mock fetcher users into Graph API delta
// response entries with "id" field injected back into the object.
func mockUsersAsGraphJSON() []map[string]any {
	var result []map[string]any
	for _, u := range mockfetcher.UserResponse {
		entry := make(map[string]any, len(u.Fields)+1)
		entry["id"] = u.ID.String()
		maps.Copy(entry, u.Fields)
		result = append(result, entry)
	}
	return result
}

// mockDevicesAsGraphJSON converts mock fetcher devices into Graph API delta
// response entries with "id" field injected back into the object.
func mockDevicesAsGraphJSON() []map[string]any {
	var result []map[string]any
	for _, d := range mockfetcher.DeviceResponse {
		entry := make(map[string]any, len(d.Fields)+1)
		entry["id"] = d.ID.String()
		maps.Copy(entry, d.Fields)
		result = append(result, entry)
	}
	return result
}

// buildGroupMembersMap builds a map of group ID → []member for the httptest
// server, using @odata.type annotations as the real Graph API does.
func buildGroupMembersMap() map[string][]map[string]any {
	result := make(map[string][]map[string]any)
	for _, g := range mockfetcher.GroupResponse {
		var members []map[string]any
		for _, m := range g.Members {
			var odataType string
			switch m.Type {
			case fetcher.MemberUser:
				odataType = "#microsoft.graph.user"
			case fetcher.MemberDevice:
				odataType = "#microsoft.graph.device"
			case fetcher.MemberGroup:
				odataType = "#microsoft.graph.group"
			}
			members = append(members, map[string]any{
				"id":          m.ID.String(),
				"@odata.type": odataType,
			})
		}
		result[g.ID.String()] = members
	}
	return result
}

// mockMFAAsGraphJSON converts mock fetcher MFA data into Graph API response
// entries with the user "id" field included.
func mockMFAAsGraphJSON() []map[string]any {
	var result []map[string]any
	for userID, mfa := range mockfetcher.MFAResponse {
		entry := map[string]any{
			"id":                    userID.String(),
			"isMfaCapable":          mfa.IsMFACapable,
			"isMfaRegistered":       mfa.IsMFARegistered,
			"isPasswordlessCapable": mfa.IsPasswordlessCapable,
			"isSsprCapable":         mfa.IsSsprCapable,
			"isSsprEnabled":         mfa.IsSsprEnabled,
			"isSsprRegistered":      mfa.IsSsprRegistered,
			"methodsRegistered":     mfa.MethodsRegistered,
			"userPreferredMethodForSecondaryAuthentication": mfa.UserPreferredMethodForSecondaryAuthentication,
			"userType": mfa.UserType,
		}
		result = append(result, entry)
	}
	return result
}

// mockSignInActivityAsGraphJSON converts mock fetcher sign-in activity data
// into Graph API response entries with nested signInActivity objects.
func mockSignInActivityAsGraphJSON() []map[string]any {
	var result []map[string]any
	for userID, sia := range mockfetcher.SignInActivityResponse {
		entry := map[string]any{
			"id": userID.String(),
			"signInActivity": map[string]any{
				"lastSignInDateTime":                sia.LastSignInDateTime,
				"lastSignInRequestId":               sia.LastSignInRequestId,
				"lastNonInteractiveSignInDateTime":  sia.LastNonInteractiveSignInDateTime,
				"lastNonInteractiveSignInRequestId": sia.LastNonInteractiveSignInRequestId,
				"lastSuccessfulSignInDateTime":      sia.LastSuccessfulSignInDateTime,
				"lastSuccessfulSignInRequestId":     sia.LastSuccessfulSignInRequestId,
			},
		}
		result = append(result, entry)
	}
	return result
}

// idPayload pairs an entity ID with its JSON-serialized azure_ad field blob.
// Used to carry payloads from both the legacy and minimal-state paths into
// compareSortedEntries.
type idPayload struct {
	id      string
	payload json.RawMessage
}

// groupECS mirrors the ECS group shape emitted by both providers
// (fetcher.GroupECS on the legacy side, ecentraid.GroupECS on the
// entcollect side). JSON tags match the wire format so we can
// unmarshal entcollect's group slices directly into this type.
type groupECS struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// compareSortedEntries compares azure_ad field blobs between legacy and
// minimal-state providers. Both sides are sorted by entity ID, then each
// payload is round-tripped through map[string]any to normalize key ordering
// before byte-level comparison. kind is "user" or "device" for error messages.
func compareSortedEntries(t *testing.T, kind string, legacy []idPayload, minimalDocs []entcollect.Document) {
	t.Helper()

	slices.SortFunc(legacy, func(a, b idPayload) int { return cmp.Compare(a.id, b.id) })

	minimalByID := make([]idPayload, 0, len(minimalDocs))
	for _, doc := range minimalDocs {
		adField := doc.Fields["azure_ad"]
		raw, err := json.Marshal(adField)
		if err != nil {
			t.Fatal(err)
		}
		minimalByID = append(minimalByID, idPayload{id: doc.ID, payload: raw})
	}
	slices.SortFunc(minimalByID, func(a, b idPayload) int { return cmp.Compare(a.id, b.id) })

	for i := range legacy {
		if legacy[i].id != minimalByID[i].id {
			t.Errorf("%s[%d] ID mismatch: legacy=%q, minimal=%q", kind, i, legacy[i].id, minimalByID[i].id)
			continue
		}

		var legacyMap, minimalMap map[string]any
		if err := json.Unmarshal(legacy[i].payload, &legacyMap); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(minimalByID[i].payload, &minimalMap); err != nil {
			t.Fatal(err)
		}

		legacyNorm, _ := json.Marshal(legacyMap)
		minimalNorm, _ := json.Marshal(minimalMap)
		if string(legacyNorm) != string(minimalNorm) {
			t.Errorf("%s %q payload mismatch:\n  legacy:  %s\n  minimal: %s",
				kind, legacy[i].id, legacyNorm, minimalNorm)
		}
	}
}

// compareGroups verifies that each entity's transitive group membership
// matches between legacy and minimal-state. legacyGroups is keyed by entity
// ID; minimalDocs are the entcollect documents. fieldKey is the document field
// to extract groups from ("user.group" or "device.group"). Both sides are
// sorted by group ID before element-wise comparison.
func compareGroups(t *testing.T, kind string, legacyGroups map[string][]groupECS, minimalDocs []entcollect.Document, fieldKey string) {
	t.Helper()

	for _, doc := range minimalDocs {
		legacyG := legacyGroups[doc.ID]
		slices.SortFunc(legacyG, func(a, b groupECS) int { return cmp.Compare(a.ID, b.ID) })

		var minimalG []groupECS
		if raw, ok := doc.Fields[fieldKey]; ok {
			b, err := json.Marshal(raw)
			if err != nil {
				t.Fatalf("%s %q: marshal minimal groups: %v", kind, doc.ID, err)
			}
			if err := json.Unmarshal(b, &minimalG); err != nil {
				t.Fatalf("%s %q: unmarshal minimal groups: %v", kind, doc.ID, err)
			}
		}
		slices.SortFunc(minimalG, func(a, b groupECS) int { return cmp.Compare(a.ID, b.ID) })

		if len(legacyG) != len(minimalG) {
			t.Errorf("%s %q group count mismatch: legacy=%d, minimal=%d", kind, doc.ID, len(legacyG), len(minimalG))
			continue
		}
		for j := range legacyG {
			if legacyG[j].ID != minimalG[j].ID || legacyG[j].Name != minimalG[j].Name {
				t.Errorf("%s %q group[%d] mismatch: legacy=%+v, minimal=%+v",
					kind, doc.ID, j, legacyG[j], minimalG[j])
			}
		}
	}
}

// compareMFACoverage checks that the same set of user IDs received MFA
// enrichment on both sides. It does not compare MFA field values because
// the two providers use different struct types (fetcher.MFARegistrationDetails
// vs ecentraid.MFADetails) at different field paths (azure_ad.mfa vs
// user.risk.mfa).
func compareMFACoverage(t *testing.T, legacyMFAUserIDs []string, minimalUsers []entcollect.Document) {
	t.Helper()

	slices.Sort(legacyMFAUserIDs)

	var minimalMFAUserIDs []string
	for _, doc := range minimalUsers {
		if _, ok := doc.Fields["user.risk.mfa"]; ok {
			minimalMFAUserIDs = append(minimalMFAUserIDs, doc.ID)
		}
	}
	slices.Sort(minimalMFAUserIDs)

	if len(legacyMFAUserIDs) != len(minimalMFAUserIDs) {
		t.Errorf("MFA user-ID count mismatch: legacy=%d, minimal=%d", len(legacyMFAUserIDs), len(minimalMFAUserIDs))
		return
	}
	for i := range legacyMFAUserIDs {
		if legacyMFAUserIDs[i] != minimalMFAUserIDs[i] {
			t.Errorf("MFA user-ID[%d] mismatch: legacy=%q, minimal=%q", i, legacyMFAUserIDs[i], minimalMFAUserIDs[i])
		}
	}
}

// compareSignInActivity checks that the same set of user IDs received
// sign-in activity enrichment and that the JSON-serialized values match.
// Unlike MFA (which uses different field paths), both providers publish
// sign-in activity at azure_ad.signInActivity with compatible JSON tags,
// so values can be compared directly.
func compareSignInActivity(t *testing.T, legacySignIn map[string]json.RawMessage, minimalUsers []entcollect.Document) {
	t.Helper()

	minimalSignIn := make(map[string]json.RawMessage)
	for _, doc := range minimalUsers {
		raw, ok := doc.Fields["azure_ad.signInActivity"]
		if !ok {
			continue
		}
		b, err := json.Marshal(raw)
		if err != nil {
			t.Fatalf("marshal minimal sign-in activity for %s: %v", doc.ID, err)
		}
		minimalSignIn[doc.ID] = b
	}

	if len(legacySignIn) != len(minimalSignIn) {
		t.Errorf("sign-in activity user count mismatch: legacy=%d, minimal=%d", len(legacySignIn), len(minimalSignIn))
	}

	for id, legacyRaw := range legacySignIn {
		minimalRaw, ok := minimalSignIn[id]
		if !ok {
			t.Errorf("sign-in activity for user %q: present in legacy, missing in minimal", id)
			continue
		}

		var legacyMap, minimalMap map[string]any
		if err := json.Unmarshal(legacyRaw, &legacyMap); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(minimalRaw, &minimalMap); err != nil {
			t.Fatal(err)
		}

		legacyNorm, _ := json.Marshal(legacyMap)
		minimalNorm, _ := json.Marshal(minimalMap)
		if string(legacyNorm) != string(minimalNorm) {
			t.Errorf("sign-in activity for user %q value mismatch:\n  legacy:  %s\n  minimal: %s",
				id, legacyNorm, minimalNorm)
		}
	}

	for id := range minimalSignIn {
		if _, ok := legacySignIn[id]; !ok {
			t.Errorf("sign-in activity for user %q: present in minimal, missing in legacy", id)
		}
	}
}

// filterDocsByKind returns only the documents matching the given EntityKind.
func filterDocsByKind(docs []entcollect.Document, kind entcollect.EntityKind) []entcollect.Document {
	var out []entcollect.Document
	for _, d := range docs {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	return out
}

// testWriter adapts testing.T for use as an io.Writer, routing all
// output through t.Log so it appears in verbose test output.
type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// equivMemStore is a minimal in-memory implementation of entcollect.Store
// for the equivalence test. It avoids the kvstore disk I/O that the legacy
// provider's stateStore normally requires.
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

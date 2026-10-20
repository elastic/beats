// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/redact"
)

const (
	testAppID    = "heartbeat-app"
	testPassword = "pa55w0rd-value"
	testToken    = "static-bearer-token"
)

// fakeCCP is a minimal CyberArk Central Credential Provider (AIMWebService)
// served by httptest.
type fakeCCP struct {
	t       *testing.T
	address string

	mu          sync.Mutex
	accounts    map[string]map[string]any
	denied      map[string]bool
	raw         map[string]string
	reads       int
	unavailable int // number of reads answered with 500 APPAP282E
	requests    []*http.Request
}

func newFakeCCP(t *testing.T) *fakeCCP {
	t.Helper()
	fc := &fakeCCP{
		t: t,
		accounts: map[string]map[string]any{
			"MySafe/MyAccount": {
				"Content":                 testPassword,
				"UserName":                "bob",
				"Address":                 "db.internal",
				"Database":                "prod",
				"Port":                    json.Number("5432"),
				"PasswordChangeInProcess": false,
			},
			"MySafe/SubFolder/MyAccount": {
				"Content": "folder-secret",
			},
		},
		denied: map[string]bool{},
		raw:    map[string]string{},
	}
	srv := httptest.NewServer(http.HandlerFunc(fc.handle))
	t.Cleanup(srv.Close)
	fc.address = srv.URL
	return fc
}

func unreachableAddress(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	return srv.URL
}

func (fc *fakeCCP) handle(w http.ResponseWriter, r *http.Request) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.requests = append(fc.requests, r.Clone(context.Background()))

	writeJSON := func(status int, body any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}

	if r.Method != http.MethodGet || r.URL.Path != defaultServicePath {
		writeJSON(http.StatusNotFound, ccpError{ErrorCode: "AIMWS001E", ErrorMsg: "route not found"})
		return
	}

	fc.reads++
	if fc.unavailable > 0 {
		fc.unavailable--
		writeJSON(http.StatusInternalServerError, ccpError{
			ErrorCode: "APPAP282E",
			ErrorMsg:  "Password is currently being changed by the CPM",
		})
		return
	}

	q := r.URL.Query()
	if q.Get("AppID") != testAppID {
		writeJSON(http.StatusForbidden, ccpError{
			ErrorCode: "APPAP306E",
			ErrorMsg:  "Authentication error for App ID",
		})
		return
	}

	key := q.Get("Safe") + "/" + q.Get("Object")
	if folder := q.Get("Folder"); folder != "" {
		key = q.Get("Safe") + "/" + folder + "/" + q.Get("Object")
	}

	if fc.denied[key] {
		writeJSON(http.StatusForbidden, ccpError{
			ErrorCode: "APPAP008E",
			ErrorMsg:  "Safe access denied",
		})
		return
	}
	if body, ok := fc.raw[key]; ok {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
		return
	}
	data, ok := fc.accounts[key]
	if !ok {
		writeJSON(http.StatusNotFound, ccpError{
			ErrorCode: "APPAP004E",
			ErrorMsg:  "Password object was not found",
		})
		return
	}
	writeJSON(http.StatusOK, data)
}

func (fc *fakeCCP) readCount() int {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.reads
}

func (fc *fakeCCP) setUnavailable(n int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.unavailable = n
}

func (fc *fakeCCP) deny(key string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.denied[key] = true
}

func (fc *fakeCCP) lastRequest() *http.Request {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.requests[len(fc.requests)-1]
}

func ccpEntry(address string) map[string]any {
	return map[string]any{"address": address, "app_id": testAppID}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestStore(t *testing.T, entry map[string]any, logger *logp.Logger) (*store, *testClock, *[]time.Duration) {
	t.Helper()
	cfg, err := conf.NewConfigFrom(entry)
	require.NoError(t, err, "test entry must be valid")
	if logger == nil {
		logger = logptest.NewTestingLogger(t, "")
	}
	s, err := newStore(cfg, logger)
	require.NoError(t, err, "store must be created")
	st, ok := s.(*store)
	require.True(t, ok, "newStore must return a *store")

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	var sleeps []time.Duration
	st.reader.Now = clock.Now
	st.reader.Sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	return st, clock, &sleeps
}

func TestRegistered(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":                 "http",
		"header":               "cyberark://MySafe/MyAccount",
		secretstores.ConfigKey: []any{map[string]any{"type": storeType, "address": "https://ccp:8443", "app_id": testAppID}},
	})
	require.NoError(t, err, "test config must be valid")
	err = secretstores.NewResolver(logptest.NewTestingLogger(t, "")).Validate(cfg)
	assert.ErrorContains(t, err, "must have the form cyberark://<safe>/<object>#<field>", "the cyberark store must be registered and validate references")
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   map[string]any
		wantErr string
	}{
		{name: "minimal app_id", entry: ccpEntry("https://ccp:8443")},
		{name: "token auth", entry: map[string]any{"address": "https://ccp:8443", "app_id": testAppID, "auth": map[string]any{"token": testToken}}},
		{name: "basic auth", entry: map[string]any{"address": "https://ccp:8443", "app_id": testAppID, "auth": map[string]any{"basic": map[string]any{"username": "u", "password": "p"}}}},
		{name: "missing address", entry: map[string]any{"app_id": testAppID}, wantErr: "'address' is required"},
		{name: "bad address", entry: map[string]any{"address": "ccp:8443", "app_id": testAppID}, wantErr: "must be an http or https URL"},
		{name: "missing app_id", entry: map[string]any{"address": "https://ccp:8443"}, wantErr: "'app_id' is required"},
		{name: "empty service_path", entry: map[string]any{"address": "https://ccp:8443", "app_id": testAppID, "service_path": " "}, wantErr: "'service_path' must not be empty"},
		{
			name: "two auth methods",
			entry: map[string]any{"address": "https://ccp:8443", "app_id": testAppID, "auth": map[string]any{
				"token": testToken,
				"basic": map[string]any{"username": "u", "password": "p"},
			}},
			wantErr: "at most one method",
		},
		{
			name:    "basic without password",
			entry:   map[string]any{"address": "https://ccp:8443", "app_id": testAppID, "auth": map[string]any{"basic": map[string]any{"username": "u"}}},
			wantErr: "'auth.basic.username' and 'auth.basic.password' are required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(tc.entry)
			require.NoError(t, err, "test entry must be valid")
			_, err = newStore(cfg, logptest.NewTestingLogger(t, ""))
			if tc.wantErr == "" {
				assert.NoError(t, err, "config must be valid")
				return
			}
			assert.ErrorContains(t, err, tc.wantErr, "config must be rejected")
		})
	}
}

func TestCredentialsAreRedacted(t *testing.T) {
	entries := map[string]map[string]any{
		"token": {"address": "https://ccp:8443", "app_id": testAppID, "auth": map[string]any{"token": testToken}},
		"basic": {"address": "https://ccp:8443", "app_id": testAppID, "auth": map[string]any{"basic": map[string]any{"username": "u", "password": testPassword}}},
	}
	for name, entry := range entries {
		t.Run(name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(entry)
			require.NoError(t, err, "test entry must be valid")
			_, err = newStore(cfg, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err, "the entry must be a valid config")

			raw, err := yaml.Marshal(map[string]any{"secret_stores": []any{entry}})
			require.NoError(t, err, "entry must marshal")
			var content map[string]any
			require.NoError(t, yaml.Unmarshal(raw, &content), "entry must unmarshal")
			redact.Redact(content)
			redacted, err := yaml.Marshal(content)
			require.NoError(t, err, "redacted entry must marshal")

			for _, credential := range []string{testToken, testPassword} {
				assert.NotContains(t, string(redacted), credential, "credentials must be redacted")
			}
			assert.Contains(t, string(redacted), redact.REDACTED, "the entry must have redacted values")
		})
	}
}

func TestReadAndCache(t *testing.T) {
	fc := newFakeCCP(t)
	s, clock, _ := newTestStore(t, ccpEntry(fc.address), nil)
	ctx := context.Background()

	// Both canonical CyberArk names and lowercase aliases resolve from a single cached read.
	wants := map[string]string{
		"Content":                 testPassword,
		"password":                testPassword,
		"UserName":                "bob",
		"username":                "bob",
		"Address":                 "db.internal",
		"address":                 "db.internal",
		"Database":                "prod",
		"database":                "prod",
		"Port":                    "5432",
		"PasswordChangeInProcess": "false",
	}
	for field, want := range wants {
		got, err := s.resolve(ctx, "MySafe/MyAccount#"+field)
		require.NoError(t, err, "field %q must resolve", field)
		assert.Equal(t, want, got, "unexpected value for field %q", field)
	}
	assert.Equal(t, 1, fc.readCount(), "one read must serve every field while cached")

	// Folder-qualified reference.
	got, err := s.resolve(ctx, "MySafe/SubFolder/MyAccount#password")
	require.NoError(t, err, "folder reference must resolve")
	assert.Equal(t, "folder-secret", got, "unexpected folder secret value")
	assert.Equal(t, "SubFolder", fc.lastRequest().URL.Query().Get("Folder"), "Folder query param must be sent")

	clock.Advance(secretstores.DefaultCacheTTL + time.Second)
	_, err = s.resolve(ctx, "MySafe/MyAccount#password")
	require.NoError(t, err, "account must be read again after cache expiry")
	assert.Equal(t, 3, fc.readCount(), "expired cache entry must be read again")
}

func TestAuthHeaders(t *testing.T) {
	fc := newFakeCCP(t)

	t.Run("token", func(t *testing.T) {
		entry := ccpEntry(fc.address)
		entry["auth"] = map[string]any{"token": testToken}
		s, _, _ := newTestStore(t, entry, nil)
		_, err := s.resolve(context.Background(), "MySafe/MyAccount#password")
		require.NoError(t, err, "read must succeed")
		assert.Equal(t, "Bearer "+testToken, fc.lastRequest().Header.Get("Authorization"), "Bearer token must be sent")
	})

	t.Run("basic", func(t *testing.T) {
		entry := ccpEntry(fc.address)
		entry["auth"] = map[string]any{"basic": map[string]any{"username": "svc", "password": testPassword}}
		s, _, _ := newTestStore(t, entry, nil)
		_, err := s.resolve(context.Background(), "MySafe/MyAccount#password")
		require.NoError(t, err, "read must succeed")
		u, p, ok := fc.lastRequest().BasicAuth()
		require.True(t, ok, "Basic auth header must be sent")
		assert.Equal(t, "svc", u, "unexpected Basic username")
		assert.Equal(t, testPassword, p, "unexpected Basic password")
	})
}

func TestCPMRotationInProgressIsRetried(t *testing.T) {
	fc := newFakeCCP(t)
	s, _, sleeps := newTestStore(t, ccpEntry(fc.address), nil)
	fc.setUnavailable(2)

	v, err := s.resolve(context.Background(), "MySafe/MyAccount#password")
	require.NoError(t, err, "read must succeed after retries")
	assert.Equal(t, testPassword, v, "unexpected secret value")
	assert.Equal(t, secretstores.DefaultBackoff, *sleeps, "configured backoff must be used")
}

func TestFailureCache(t *testing.T) {
	fc := newFakeCCP(t)
	s, clock, sleeps := newTestStore(t, ccpEntry(fc.address), nil)
	ctx := context.Background()
	fc.setUnavailable(100)

	_, err := s.resolve(ctx, "MySafe/MyAccount#password")
	assert.ErrorContains(t, err, "APPAP282E", "the CCP error code must be reported")
	reads := fc.readCount()
	assert.Equal(t, len(secretstores.DefaultBackoff)+1, reads, "every attempt must reach CyberArk")
	assert.Len(t, *sleeps, len(secretstores.DefaultBackoff), "a backoff is expected between attempts")

	_, err = s.resolve(ctx, "OtherSafe/OtherAccount#password")
	assert.ErrorContains(t, err, "cyberark: "+fc.address+" is unavailable", "the failure must be returned for the whole connection")
	assert.NotContains(t, err.Error(), "MySafe/MyAccount", "the failure must not name the account of another monitor")
	assert.Equal(t, reads, fc.readCount(), "CyberArk must not be contacted while the failure is cached")

	fc.setUnavailable(0)
	clock.Advance(secretstores.DefaultFailureTTL + time.Second)
	_, err = s.resolve(ctx, "MySafe/MyAccount#password")
	assert.NoError(t, err, "CyberArk must be contacted again after the failure expires")
}

func TestPathFailureCache(t *testing.T) {
	fc := newFakeCCP(t)
	fc.deny("MySafe/Denied")
	fc.raw["MySafe/BadJSON"] = "<html>proxy error</html>"
	s, clock, sleeps := newTestStore(t, ccpEntry(fc.address), nil)
	ctx := context.Background()

	for _, ref := range []string{"MySafe/Missing#password", "MySafe/Denied#password", "MySafe/BadJSON#password"} {
		_, err := s.resolve(ctx, ref)
		require.Error(t, err, "reading %s must fail", ref)
	}
	assert.Empty(t, *sleeps, "path errors must not be retried")
	reads := fc.readCount()

	for _, ref := range []string{"MySafe/Missing#password", "MySafe/Denied#password", "MySafe/BadJSON#password"} {
		_, err := s.resolve(ctx, ref)
		assert.Error(t, err, "the cached failure for %s must be returned", ref)
	}
	assert.Equal(t, reads, fc.readCount(), "CyberArk must not be contacted while path failures are cached")

	v, err := s.resolve(ctx, "MySafe/MyAccount#password")
	require.NoError(t, err, "path failures must not affect other accounts")
	assert.Equal(t, testPassword, v, "unexpected secret value")

	clock.Advance(secretstores.DefaultFailureTTL + time.Second)
	_, err = s.resolve(ctx, "MySafe/Missing#password")
	assert.ErrorContains(t, err, "cyberark: no account at MySafe/Missing", "missing account must be reported again after expiry")
	assert.Equal(t, reads+2, fc.readCount(), "CyberArk must be read again after the path failure expires")
}

func TestConnectionErrorIsRetried(t *testing.T) {
	s, _, sleeps := newTestStore(t, ccpEntry(unreachableAddress(t)), nil)
	_, err := s.resolve(context.Background(), "MySafe/MyAccount#password")
	require.Error(t, err, "an unreachable CyberArk must be reported")
	assert.True(t, secretstores.IsConnectionError(err), "a refused connection must be a connection error")
	assert.Len(t, *sleeps, len(secretstores.DefaultBackoff), "connection errors must be retried")
	assert.NotContains(t, err.Error(), defaultServicePath, "the request URL must not be in errors")
}

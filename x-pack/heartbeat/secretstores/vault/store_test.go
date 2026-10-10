// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

const (
	testRoleID   = "test-role-id"
	testSecretID = "test-secret-id"
	testToken    = "static-test-token"
	testPassword = "pa55w0rd-value"
)

const testAddress = "http://vault.test:8200"

// fakeVault is a minimal in-memory Vault: AppRole login and KV v2 reads.
// It is used as the client transport, so tests need no network.
type fakeVault struct {
	t *testing.T

	mu          sync.Mutex
	secrets     map[string]map[string]any
	validTokens map[string]bool
	logins      int
	reads       int
	unavailable int // number of reads answered with 503
	requests    []*http.Request
	nextToken   int
}

func newFakeVault(t *testing.T) *fakeVault {
	t.Helper()
	fv := &fakeVault{
		t: t,
		secrets: map[string]map[string]any{
			"myapp/creds": {"password": testPassword, "user": "bob", "port": json.Number("5432"), "enabled": true, "obj": map[string]any{"a": "b"}},
		},
		validTokens: map[string]bool{testToken: true},
	}
	return fv
}

// RoundTrip serves the request in memory.
func (fv *fakeVault) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	fv.handle(rec, r)
	resp := rec.Result()
	resp.Request = r
	return resp, nil
}

// failingTransport simulates an unreachable Vault.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp: connection refused")
}

func (fv *fakeVault) handle(w http.ResponseWriter, r *http.Request) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.requests = append(fv.requests, r.Clone(context.Background()))

	writeJSON := func(status int, body any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}

	switch {
	case r.Method == http.MethodPut || r.Method == http.MethodPost:
		if r.URL.Path != "/v1/auth/approle/login" {
			writeJSON(http.StatusNotFound, map[string]any{"errors": []string{"no handler for route"}})
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["role_id"] != testRoleID || body["secret_id"] != testSecretID {
			writeJSON(http.StatusBadRequest, map[string]any{"errors": []string{"invalid role or secret ID"}})
			return
		}
		fv.logins++
		fv.nextToken++
		token := fmt.Sprintf("approle-token-%d", fv.nextToken)
		fv.validTokens[token] = true
		writeJSON(http.StatusOK, map[string]any{"auth": map[string]any{"client_token": token, "lease_duration": 60, "renewable": true}})
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/secret/data/"):
		fv.reads++
		if fv.unavailable > 0 {
			fv.unavailable--
			writeJSON(http.StatusServiceUnavailable, map[string]any{"errors": []string{"Vault is sealed"}})
			return
		}
		if !fv.validTokens[r.Header.Get("X-Vault-Token")] {
			writeJSON(http.StatusForbidden, map[string]any{"errors": []string{"permission denied"}})
			return
		}
		data, ok := fv.secrets[strings.TrimPrefix(r.URL.Path, "/v1/secret/data/")]
		if !ok {
			writeJSON(http.StatusNotFound, map[string]any{"errors": []string{}})
			return
		}
		writeJSON(http.StatusOK, map[string]any{"data": map[string]any{
			"data":     data,
			"metadata": map[string]any{"version": 1, "created_time": "2026-01-01T00:00:00Z", "deletion_time": "", "destroyed": false},
		}})
	default:
		writeJSON(http.StatusNotFound, map[string]any{"errors": []string{"no handler for route"}})
	}
}

// expireTokens makes Vault reject every token issued so far.
func (fv *fakeVault) expireTokens() {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.validTokens = map[string]bool{}
}

func (fv *fakeVault) counts() (logins, reads int) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	return fv.logins, fv.reads
}

func (fv *fakeVault) setUnavailable(n int) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.unavailable = n
}

func (fv *fakeVault) lastRequest() *http.Request {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	return fv.requests[len(fv.requests)-1]
}

func tokenEntry(address string) map[string]any {
	return map[string]any{"address": address, "auth": map[string]any{"token": map[string]any{"value": testToken}}}
}

func appRoleEntry(address string) map[string]any {
	return map[string]any{"address": address, "auth": map[string]any{"approle": map[string]any{"role_id": testRoleID, "secret_id": testSecretID}}}
}

// testClock is a manually advanced clock.
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

// newTestStore builds a store with a manual clock and recorded sleeps.
func newTestStore(t *testing.T, entry map[string]any, transport http.RoundTripper, logger *logp.Logger) (*store, *testClock, *[]time.Duration) {
	t.Helper()
	cfg, err := conf.NewConfigFrom(entry)
	require.NoError(t, err, "test entry must be valid")
	if logger == nil {
		logger = logptest.NewTestingLogger(t, "")
	}
	st, err := newStoreWithTransport(cfg, logger, transport)
	require.NoError(t, err, "store must be created")

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	var sleeps []time.Duration
	st.now = clock.Now
	st.sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	return st, clock, &sleeps
}

func TestRegistered(t *testing.T) {
	// References to unregistered types are plain text and pass validation.
	cfg, err := conf.NewConfigFrom(map[string]any{"type": "http", "header": "$vault{myapp/creds#token}"})
	require.NoError(t, err, "test config must be valid")
	err = secretstores.NewResolver(logptest.NewTestingLogger(t, "")).Validate(cfg)
	assert.ErrorContains(t, err, `requires a "vault" entry`, "the vault store must be registered")
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   map[string]any
		wantErr string
	}{
		{name: "token", entry: tokenEntry("https://vault:8200")},
		{name: "approle", entry: appRoleEntry("http://vault:8200")},
		{name: "missing address", entry: map[string]any{"auth": map[string]any{"token": map[string]any{"value": "x"}}}, wantErr: "'address' is required"},
		{name: "bad address", entry: map[string]any{"address": "vault:8200", "auth": map[string]any{"token": map[string]any{"value": "x"}}}, wantErr: "must be an http or https URL"},
		{name: "no auth", entry: map[string]any{"address": "https://vault"}, wantErr: "'auth' must have one method"},
		{
			name: "two auth methods",
			entry: map[string]any{"address": "https://vault", "auth": map[string]any{
				"token":   map[string]any{"value": "x"},
				"approle": map[string]any{"role_id": "r", "secret_id": "s"},
			}},
			wantErr: "exactly one method",
		},
		{name: "empty token", entry: map[string]any{"address": "https://vault", "auth": map[string]any{"token": map[string]any{"value": ""}}}, wantErr: "'auth.token.value' is required"},
		{name: "approle without secret_id", entry: map[string]any{"address": "https://vault", "auth": map[string]any{"approle": map[string]any{"role_id": "r"}}}, wantErr: "are required"},
		{name: "empty kv_mount", entry: map[string]any{"address": "https://vault", "kv_mount": "", "auth": map[string]any{"token": map[string]any{"value": "x"}}}, wantErr: "'kv_mount' must not be empty"},
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

func TestConfigDefaults(t *testing.T) {
	s, _, _ := newTestStore(t, appRoleEntry("https://vault:8200"), newFakeVault(t), nil)
	assert.Equal(t, "secret", s.cfg.KVMount, "kv_mount must default to secret")
	assert.Equal(t, "approle", s.cfg.Auth.AppRole.MountPath, "approle mount_path must default to approle")
}

func TestTokenAuth(t *testing.T) {
	fv := newFakeVault(t)
	entry := tokenEntry(testAddress)
	entry["namespace"] = "admin/team-a"
	s, _, _ := newTestStore(t, entry, fv, nil)

	v, err := s.Resolve(context.Background(), "myapp/creds", "password")
	require.NoError(t, err, "secret must be read")
	assert.Equal(t, testPassword, v, "unexpected secret value")

	req := fv.lastRequest()
	assert.Equal(t, testToken, req.Header.Get("X-Vault-Token"), "the configured token must be sent")
	assert.Equal(t, "admin/team-a", req.Header.Get("X-Vault-Namespace"), "the configured namespace must be sent")
	assert.Equal(t, "true", req.Header.Get("X-Vault-Request"), "the Vault request header must be sent")
}

func TestTokenAuthRejectedIsNotRetried(t *testing.T) {
	fv := newFakeVault(t)
	fv.expireTokens()
	s, _, sleeps := newTestStore(t, tokenEntry(testAddress), fv, nil)

	_, err := s.Resolve(context.Background(), "myapp/creds", "password")
	assert.ErrorContains(t, err, "status 403: permission denied", "a rejected token must be reported")
	logins, reads := fv.counts()
	assert.Zero(t, logins, "token auth must never log in")
	assert.Equal(t, 1, reads, "a rejected token must not be retried")
	assert.Empty(t, *sleeps, "no backoff is expected")
}

func TestAppRoleLoginAndCache(t *testing.T) {
	fv := newFakeVault(t)
	s, clock, _ := newTestStore(t, appRoleEntry(testAddress), fv, nil)
	ctx := context.Background()

	for _, field := range []string{"password", "user", "password"} {
		_, err := s.Resolve(ctx, "myapp/creds", field)
		require.NoError(t, err, "field %q must be read", field)
	}
	logins, reads := fv.counts()
	assert.Equal(t, 1, logins, "one login is expected")
	assert.Equal(t, 1, reads, "one read must serve every field while cached")

	clock.Advance(cacheTTL + time.Second)
	_, err := s.Resolve(ctx, "myapp/creds", "password")
	require.NoError(t, err, "secret must be read again")
	logins, reads = fv.counts()
	assert.Equal(t, 1, logins, "the token must be reused")
	assert.Equal(t, 2, reads, "an expired cache entry must be read again")
}

func TestAppRoleLogsInAgainOnce(t *testing.T) {
	fv := newFakeVault(t)
	s, _, _ := newTestStore(t, appRoleEntry(testAddress), fv, nil)
	ctx := context.Background()

	_, err := s.Resolve(ctx, "myapp/creds", "password")
	require.NoError(t, err, "first read must work")

	fv.expireTokens()
	s.mu.Lock()
	s.cache = map[string]cacheEntry{}
	s.mu.Unlock()

	_, err = s.Resolve(ctx, "myapp/creds", "password")
	require.NoError(t, err, "an expired token must be replaced")
	logins, reads := fv.counts()
	assert.Equal(t, 2, logins, "one more login is expected")
	assert.Equal(t, 3, reads, "the read must be retried once after the login")
}

func TestAppRoleConcurrentExpiryLogsInOnce(t *testing.T) {
	fv := newFakeVault(t)
	for i := range 10 {
		fv.secrets[fmt.Sprintf("app%d", i)] = map[string]any{"password": "x"}
	}
	s, _, _ := newTestStore(t, appRoleEntry(testAddress), fv, nil)
	ctx := context.Background()

	_, err := s.Resolve(ctx, "myapp/creds", "password")
	require.NoError(t, err, "first read must work")
	fv.expireTokens()

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := range 10 {
		wg.Go(func() {
			_, err := s.Resolve(ctx, fmt.Sprintf("app%d", i), "password")
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err, "every read must work after the login")
	}

	logins, _ := fv.counts()
	assert.Equal(t, 2, logins, "concurrent auth failures must cause a single login")
}

func TestAppRoleInvalidCredentials(t *testing.T) {
	fv := newFakeVault(t)
	entry := map[string]any{"address": testAddress, "auth": map[string]any{"approle": map[string]any{
		"role_id": testRoleID, "secret_id": "wrong-secret-id-value",
	}}}
	s, _, sleeps := newTestStore(t, entry, fv, nil)

	_, err := s.Resolve(context.Background(), "myapp/creds", "password")
	require.Error(t, err, "a failed login must be reported")
	assert.Contains(t, err.Error(), "invalid role or secret ID", "the Vault error must be reported")
	assert.NotContains(t, err.Error(), "wrong-secret-id-value", "credentials must not be in errors")
	assert.NotContains(t, err.Error(), testRoleID, "credentials must not be in errors")
	assert.Empty(t, *sleeps, "a rejected login must not be retried")
}

func TestTransientErrorsAreRetried(t *testing.T) {
	fv := newFakeVault(t)
	s, _, sleeps := newTestStore(t, tokenEntry(testAddress), fv, nil)
	fv.setUnavailable(2)

	v, err := s.Resolve(context.Background(), "myapp/creds", "password")
	require.NoError(t, err, "the read must work after retries")
	assert.Equal(t, testPassword, v, "unexpected secret value")
	assert.Equal(t, retryBackoff, *sleeps, "the configured backoff must be used")
}

func TestFailureCache(t *testing.T) {
	fv := newFakeVault(t)
	s, clock, sleeps := newTestStore(t, tokenEntry(testAddress), fv, nil)
	ctx := context.Background()
	fv.setUnavailable(100)

	_, err := s.Resolve(ctx, "myapp/creds", "password")
	assert.ErrorContains(t, err, "status 503", "an unavailable Vault must be reported")
	_, reads := fv.counts()
	assert.Equal(t, len(retryBackoff)+1, reads, "every attempt must reach Vault")
	assert.Len(t, *sleeps, len(retryBackoff), "a backoff is expected between attempts")

	_, err = s.Resolve(ctx, "myapp/other", "password")
	assert.ErrorContains(t, err, "status 503", "the failure must be returned for the whole connection")
	_, readsAfter := fv.counts()
	assert.Equal(t, reads, readsAfter, "Vault must not be contacted while the failure is cached")

	fv.setUnavailable(0)
	clock.Advance(failureTTL + time.Second)
	_, err = s.Resolve(ctx, "myapp/creds", "password")
	assert.NoError(t, err, "Vault must be contacted again after the failure expires")
}

func TestConnectionErrorIsTransient(t *testing.T) {
	s, _, sleeps := newTestStore(t, tokenEntry(testAddress), failingTransport{}, nil)
	_, err := s.Resolve(context.Background(), "myapp/creds", "password")
	require.Error(t, err, "an unreachable Vault must be reported")
	assert.True(t, isTransient(err), "connection errors must be transient")
	assert.Len(t, *sleeps, len(retryBackoff), "connection errors must be retried")
	assert.NotContains(t, err.Error(), testToken, "the token must not be in errors")
}

func TestNotFoundErrors(t *testing.T) {
	fv := newFakeVault(t)
	s, _, sleeps := newTestStore(t, tokenEntry(testAddress), fv, nil)
	ctx := context.Background()

	_, err := s.Resolve(ctx, "missing", "password")
	assert.EqualError(t, err, "vault: no secret at secret/data/missing", "a missing secret must be reported")

	_, err = s.Resolve(ctx, "myapp/creds", "nope")
	assert.EqualError(t, err, `vault: field "nope" not found at secret/data/myapp/creds`, "a missing field must be reported")
	assert.Empty(t, *sleeps, "not found errors must not be retried")
}

func TestFieldValueConversion(t *testing.T) {
	fv := newFakeVault(t)
	s, _, _ := newTestStore(t, tokenEntry(testAddress), fv, nil)

	for field, want := range map[string]string{"port": "5432", "enabled": "true", "obj": `{"a":"b"}`} {
		v, err := s.Resolve(context.Background(), "myapp/creds", field)
		require.NoError(t, err, "field %q must be read", field)
		assert.Equal(t, want, v, "field %q must be converted to text", field)
	}
}

func TestEnvironmentIsIgnored(t *testing.T) {
	fv := newFakeVault(t)
	t.Setenv("VAULT_ADDR", "http://127.0.0.1:1")
	t.Setenv("VAULT_TOKEN", "token-from-env")
	t.Setenv("VAULT_NAMESPACE", "namespace-from-env")
	t.Setenv("VAULT_HEADERS", `{"X-From-Env":"1"}`)
	t.Setenv("VAULT_SKIP_VERIFY", "true")
	t.Setenv("VAULT_MAX_RETRIES", "5")

	s, _, _ := newTestStore(t, appRoleEntry(testAddress), fv, nil)
	_, err := s.Resolve(context.Background(), "myapp/creds", "password")
	require.NoError(t, err, "the configured address must be used")

	logins, _ := fv.counts()
	assert.Equal(t, 1, logins, "VAULT_TOKEN must not replace the AppRole login")
	req := fv.lastRequest()
	assert.Equal(t, "vault.test:8200", req.URL.Host, "VAULT_ADDR must be ignored")
	assert.NotEqual(t, "token-from-env", req.Header.Get("X-Vault-Token"), "VAULT_TOKEN must be ignored")
	assert.Empty(t, req.Header.Get("X-Vault-Namespace"), "VAULT_NAMESPACE must be ignored")
	assert.Empty(t, req.Header.Get("X-From-Env"), "VAULT_HEADERS must be ignored")
}

func TestSecretsAreNotLogged(t *testing.T) {
	fv := newFakeVault(t)
	logger := logptest.NewFileLogger(t, "")
	s, _, _ := newTestStore(t, appRoleEntry(testAddress), fv, logger.Logger)
	ctx := context.Background()

	_, err := s.Resolve(ctx, "myapp/creds", "password")
	require.NoError(t, err, "secret must be read")
	fv.expireTokens()
	s.mu.Lock()
	s.cache = map[string]cacheEntry{}
	s.mu.Unlock()
	_, err = s.Resolve(ctx, "myapp/creds", "password")
	require.NoError(t, err, "secret must be read again")

	logger.WaitLogsContains(t, "Logged in with AppRole", time.Second, "debug logs are expected")
	for _, secret := range []string{testPassword, testRoleID, testSecretID, "approle-token-"} {
		found, err := logger.FindInLogs(secret)
		require.NoError(t, err, "logs must be readable")
		assert.False(t, found, "%q must not be logged", secret)
		logger.ResetOffset()
	}
}

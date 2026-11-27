// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/redact"
)

const (
	testRoleID   = "test-role-id"
	testSecretID = "test-secret-id"
	testToken    = "static-test-token"
	testPassword = "pa55w0rd-value"
)

// fakeVault is a minimal Vault, served by httptest: AppRole login and KV v2
// reads.
type fakeVault struct {
	t *testing.T
	// address is the URL of the server.
	address string

	mu          sync.Mutex
	secrets     map[string]map[string]any
	validTokens map[string]bool
	// denied holds the paths valid tokens can't read.
	denied map[string]bool
	// noLookup denies tokens their own lookup.
	noLookup bool
	// raw holds the paths answered with a 200 and a raw body, which isn't a
	// KV v2 secret.
	raw         map[string]string
	logins      int
	lookups     int
	reads       int
	revokes     int
	revokedToks []string
	revokeErr   int // HTTP status returned on revoke-self when > 0
	revokeBlock chan struct{}
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
		denied:      map[string]bool{},
		raw:         map[string]string{},
	}
	srv := httptest.NewServer(http.HandlerFunc(fv.handle))
	t.Cleanup(srv.Close)
	fv.address = srv.URL
	return fv
}

// unreachableAddress returns the address of a closed server, so connections
// are refused.
func unreachableAddress(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	return srv.URL
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
	case (r.Method == http.MethodPut || r.Method == http.MethodPost) && r.URL.Path == "/v1/auth/token/revoke-self":
		block := fv.revokeBlock
		if block != nil {
			fv.mu.Unlock()
			select {
			case <-block:
			case <-r.Context().Done():
				fv.mu.Lock()
				return
			}
			fv.mu.Lock()
		}
		fv.revokes++
		tok := r.Header.Get("X-Vault-Token")
		fv.revokedToks = append(fv.revokedToks, tok)
		if fv.revokeErr > 0 {
			writeJSON(fv.revokeErr, map[string]any{"errors": []string{"revoke failed"}})
			return
		}
		delete(fv.validTokens, tok)
		w.WriteHeader(http.StatusNoContent)
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
	case r.Method == http.MethodGet && r.URL.Path == "/v1/auth/token/lookup-self":
		fv.lookups++
		if fv.noLookup || !fv.validTokens[r.Header.Get("X-Vault-Token")] {
			writeJSON(http.StatusForbidden, map[string]any{"errors": []string{"permission denied"}})
			return
		}
		writeJSON(http.StatusOK, map[string]any{"data": map[string]any{"ttl": 60}})
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/secret/data/"):
		fv.reads++
		if fv.unavailable > 0 {
			fv.unavailable--
			writeJSON(http.StatusServiceUnavailable, map[string]any{"errors": []string{"Vault is sealed"}})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/v1/secret/data/")
		if !fv.validTokens[r.Header.Get("X-Vault-Token")] || fv.denied[path] {
			writeJSON(http.StatusForbidden, map[string]any{"errors": []string{"permission denied"}})
			return
		}
		if body, ok := fv.raw[path]; ok {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
			return
		}
		data, ok := fv.secrets[path]
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

func (fv *fakeVault) lookupCount() int {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	return fv.lookups
}

func (fv *fakeVault) revokeState() (count int, tokens []string) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	return fv.revokes, append([]string(nil), fv.revokedToks...)
}

func (fv *fakeVault) isTokenValid(token string) bool {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	return fv.validTokens[token]
}

func (fv *fakeVault) setRevokeError(status int) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.revokeErr = status
}

func (fv *fakeVault) setRevokeBlock(ch chan struct{}) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.revokeBlock = ch
}

func (fv *fakeVault) deny(path string) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.denied[path] = true
}

func (fv *fakeVault) setNoLookup() {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.noLookup = true
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
	return map[string]any{"address": address, "auth": map[string]any{"token": testToken}}
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
	st.now = clock.Now
	st.sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	return st, clock, &sleeps
}

func TestRegistered(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":                 "http",
		"header":               "vault://main@myapp/creds",
		secretstores.ConfigKey: []any{map[string]any{"type": storeType, "name": "main", "address": "https://vault:8200", "auth": map[string]any{"token": testToken}}},
	})
	require.NoError(t, err, "test config must be valid")
	err = secretstores.NewResolver(logptest.NewTestingLogger(t, "")).Validate(cfg)
	assert.ErrorContains(t, err, "must have the form vault://<name>@<path>#<field>", "the vault store must be registered and validate references")
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   map[string]any
		wantErr string
	}{
		{name: "token over verified https", entry: tokenEntry("https://vault:8200")},
		{name: "approle over loopback http", entry: appRoleEntry("http://localhost:8200")},
		{name: "remote http hostname", entry: tokenEntry("http://vault:8200"), wantErr: "must use https unless its host is loopback"},
		{name: "remote http with loopback userinfo", entry: tokenEntry("http://localhost@vault.example:8200"), wantErr: "must use https unless its host is loopback"},
		{name: "remote http hostname resolving to loopback", entry: tokenEntry("http://127.0.0.1.nip.io:8200"), wantErr: "must use https unless its host is loopback"},
		{name: "remote http private IPv4", entry: tokenEntry("http://10.0.0.5:8200"), wantErr: "must use https unless its host is loopback"},
		{name: "remote http IPv6", entry: tokenEntry("http://[2001:db8::1]:8200"), wantErr: "must use https unless its host is loopback"},
		{name: "localhost http", entry: tokenEntry("http://localhost:8200")},
		{name: "mixed-case localhost http", entry: tokenEntry("http://LOCALHOST:8200")},
		{name: "IPv4 loopback http", entry: tokenEntry("http://127.42.0.1:8200")},
		{name: "IPv6 loopback http", entry: tokenEntry("http://[::1]:8200")},
		{name: "expanded IPv6 loopback http", entry: tokenEntry("http://[0:0:0:0:0:0:0:1]:8200")},
		{
			name: "remote https without verification",
			entry: map[string]any{
				"address": "https://vault:8200",
				"auth":    map[string]any{"token": "x"},
				"ssl":     map[string]any{"verification_mode": "none"},
			},
			wantErr: "'ssl.verification_mode' must be 'full' or 'strict' unless the address host is loopback",
		},
		{
			name: "remote https without hostname verification",
			entry: map[string]any{
				"address": "https://vault:8200",
				"auth":    map[string]any{"token": "x"},
				"ssl":     map[string]any{"verification_mode": "certificate"},
			},
			wantErr: "'ssl.verification_mode' must be 'full' or 'strict' unless the address host is loopback",
		},
		{
			name: "remote https with strict verification",
			entry: map[string]any{
				"address": "https://vault:8200",
				"auth":    map[string]any{"token": "x"},
				"ssl":     map[string]any{"verification_mode": "strict"},
			},
		},
		{
			name: "remote https with disabled ssl settings and no verification",
			entry: map[string]any{
				"address": "https://vault:8200",
				"auth":    map[string]any{"token": "x"},
				"ssl":     map[string]any{"enabled": false, "verification_mode": "none"},
			},
			wantErr: "'ssl.verification_mode' must be 'full' or 'strict' unless the address host is loopback",
		},
		{
			name: "loopback https without verification",
			entry: map[string]any{
				"address": "https://[::1]:8200",
				"auth":    map[string]any{"token": "x"},
				"ssl":     map[string]any{"verification_mode": "none"},
			},
		},
		{
			name: "loopback https without hostname verification",
			entry: map[string]any{
				"address": "https://localhost:8200",
				"auth":    map[string]any{"token": "x"},
				"ssl":     map[string]any{"verification_mode": "certificate"},
			},
		},
		{name: "missing address", entry: map[string]any{"auth": map[string]any{"token": "x"}}, wantErr: "'address' is required"},
		{name: "bad address", entry: map[string]any{"address": "vault:8200", "auth": map[string]any{"token": "x"}}, wantErr: "must be an http or https URL"},
		{name: "no auth", entry: map[string]any{"address": "https://vault"}, wantErr: "'auth' must have one method"},
		{
			name: "two auth methods",
			entry: map[string]any{"address": "https://vault", "auth": map[string]any{
				"token":   "x",
				"approle": map[string]any{"role_id": "r", "secret_id": "s"},
			}},
			wantErr: "exactly one method",
		},
		{name: "empty token", entry: map[string]any{"address": "https://vault", "auth": map[string]any{"token": ""}}, wantErr: "'auth' must have one method"},
		{name: "approle without secret_id", entry: map[string]any{"address": "https://vault", "auth": map[string]any{"approle": map[string]any{"role_id": "r"}}}, wantErr: "are required"},
		{name: "empty kv_mount", entry: map[string]any{"address": "https://vault", "kv_mount": "", "auth": map[string]any{"token": "x"}}, wantErr: "'kv_mount' must not be empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(tc.entry)
			require.NoError(t, err, "test entry must be valid")
			store, err := newStore(cfg, logptest.NewTestingLogger(t, ""))
			if tc.wantErr == "" {
				assert.NoError(t, err, "config must be valid")
				return
			}
			assert.ErrorContains(t, err, tc.wantErr, "config must be rejected")
			assert.Nil(t, store, "invalid config must not create a store that can send credentials")
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	s, _, _ := newTestStore(t, appRoleEntry("https://vault:8200"), nil)
	assert.Equal(t, "secret", s.cfg.KVMount, "kv_mount must default to secret")
	assert.Equal(t, "approle", s.cfg.Auth.AppRole.MountPath, "approle mount_path must default to approle")
}

// TestCredentialsAreRedacted checks that the Elastic Agent diagnostics
// redact the credentials of a secret_stores entry: they only redact strings
// under keys such as token or secret_id.
func TestCredentialsAreRedacted(t *testing.T) {
	entries := map[string]map[string]any{
		"token":   tokenEntry("https://vault:8200"),
		"approle": appRoleEntry("https://vault:8200"),
	}
	for name, entry := range entries {
		t.Run(name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(entry)
			require.NoError(t, err, "test entry must be valid")
			_, err = newStore(cfg, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err, "the entry must be a valid config")

			// The diagnostics redact YAML content, as the agent does.
			raw, err := yaml.Marshal(map[string]any{"secret_stores": []any{entry}})
			require.NoError(t, err, "entry must marshal")
			var content map[string]any
			require.NoError(t, yaml.Unmarshal(raw, &content), "entry must unmarshal")
			redact.Redact(content)
			redacted, err := yaml.Marshal(content)
			require.NoError(t, err, "redacted entry must marshal")

			for _, credential := range []string{testToken, testSecretID} {
				assert.NotContains(t, string(redacted), credential, "credentials must be redacted")
			}
			assert.Contains(t, string(redacted), redact.REDACTED, "the entry must have redacted values")
		})
	}
}

func TestTokenAuth(t *testing.T) {
	fv := newFakeVault(t)
	entry := tokenEntry(fv.address)
	entry["namespace"] = "admin/team-a"
	s, _, _ := newTestStore(t, entry, nil)

	v, err := s.resolve(context.Background(), "myapp/creds#password")
	require.NoError(t, err, "secret must be read")
	assert.Equal(t, testPassword, v, "unexpected secret value")

	req := fv.lastRequest()
	assert.Equal(t, testToken, req.Header.Get("X-Vault-Token"), "the configured token must be sent")
	assert.Equal(t, "admin/team-a", req.Header.Get("X-Vault-Namespace"), "the configured namespace must be sent")
	assert.Equal(t, "true", req.Header.Get("X-Vault-Request"), "the Vault request header must be sent")
}

func TestResolvedSecretsAreScrubbed(t *testing.T) {
	fv := newFakeVault(t)
	entry := tokenEntry(fv.address)
	entry["type"] = storeType
	entry["name"] = "main"
	cfg, err := conf.NewConfigFrom(map[string]any{
		"url":                  "https://host/?token=vault://main@myapp/creds#password",
		secretstores.ConfigKey: []any{entry},
	})
	require.NoError(t, err, "monitor config must be valid")

	resolved, scrubber, err := secretstores.NewResolver(logptest.NewTestingLogger(t, "")).Resolve(cfg, nil)
	require.NoError(t, err, "references must resolve")
	target, err := resolved.String("url", -1)
	require.NoError(t, err, "resolved config must have the url")
	require.Contains(t, target, testPassword, "the reference must be replaced")

	require.NotNil(t, scrubber, "resolved secrets must be scrubbed")
	assert.Equal(t, "https://host/?token=<hidden>", scrubber.String(target), "the secret must be removed")
}

func TestTokenAuthRejectedIsNotRetried(t *testing.T) {
	fv := newFakeVault(t)
	fv.expireTokens()
	s, _, sleeps := newTestStore(t, tokenEntry(fv.address), nil)

	_, err := s.resolve(context.Background(), "myapp/creds#password")
	assert.ErrorContains(t, err, "status 403: permission denied", "a rejected token must be reported")
	logins, reads := fv.counts()
	assert.Zero(t, logins, "token auth must never log in")
	assert.Equal(t, 1, reads, "a rejected token must not be retried")
	assert.Empty(t, *sleeps, "no backoff is expected")
}

func TestAppRoleLoginAndCache(t *testing.T) {
	fv := newFakeVault(t)
	s, clock, _ := newTestStore(t, appRoleEntry(fv.address), nil)
	ctx := context.Background()

	for _, field := range []string{"password", "user", "password"} {
		_, err := s.resolve(ctx, "myapp/creds#"+field)
		require.NoError(t, err, "field %q must be read", field)
	}
	logins, reads := fv.counts()
	assert.Equal(t, 1, logins, "one login is expected")
	assert.Equal(t, 1, reads, "one read must serve every field while cached")

	clock.Advance(cacheTTL + time.Second)
	_, err := s.resolve(ctx, "myapp/creds#password")
	require.NoError(t, err, "secret must be read again")
	logins, reads = fv.counts()
	assert.Equal(t, 1, logins, "the token must be reused")
	assert.Equal(t, 2, reads, "an expired cache entry must be read again")
}

func TestAppRoleLogsInAgainOnce(t *testing.T) {
	fv := newFakeVault(t)
	s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)
	ctx := context.Background()

	_, err := s.resolve(ctx, "myapp/creds#password")
	require.NoError(t, err, "first read must work")

	fv.expireTokens()
	s.mu.Lock()
	s.cache = map[string]cacheEntry{}
	s.mu.Unlock()

	_, err = s.resolve(ctx, "myapp/creds#password")
	require.NoError(t, err, "an expired token must be replaced")
	logins, reads := fv.counts()
	assert.Equal(t, 2, logins, "one more login is expected")
	assert.Equal(t, 3, reads, "the read must be retried once after the login")
}

func TestAppRoleDeniedPath(t *testing.T) {
	tests := map[string]struct {
		noLookup   bool
		wantLogins int
	}{
		// A valid token can look itself up: the path is denied, not the token.
		"token can look itself up": {wantLogins: 1},
		// Without the lookup, the token is considered rejected.
		"token can't look itself up": {noLookup: true, wantLogins: 2},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fv := newFakeVault(t)
			fv.deny("myapp/creds")
			if tc.noLookup {
				fv.setNoLookup()
			}
			s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)

			_, err := s.resolve(context.Background(), "myapp/creds#password")
			assert.ErrorContains(t, err, "status 403: permission denied", "the denied path must be reported")
			logins, _ := fv.counts()
			assert.Equal(t, tc.wantLogins, logins, "unexpected number of logins")
			assert.Equal(t, 1, fv.lookupCount(), "the token must be looked up once")
		})
	}
}

func TestAppRoleConcurrentExpiryLogsInOnce(t *testing.T) {
	fv := newFakeVault(t)
	for i := range 10 {
		fv.secrets[fmt.Sprintf("app%d", i)] = map[string]any{"password": "x"}
	}
	s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)
	ctx := context.Background()

	_, err := s.resolve(ctx, "myapp/creds#password")
	require.NoError(t, err, "first read must work")
	fv.expireTokens()

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := range 10 {
		wg.Go(func() {
			_, err := s.resolve(ctx, fmt.Sprintf("app%d#password", i))
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
	entry := map[string]any{"address": fv.address, "auth": map[string]any{"approle": map[string]any{
		"role_id": testRoleID, "secret_id": "wrong-secret-id-value",
	}}}
	s, _, sleeps := newTestStore(t, entry, nil)

	_, err := s.resolve(context.Background(), "myapp/creds#password")
	require.Error(t, err, "a failed login must be reported")
	assert.Contains(t, err.Error(), "invalid role or secret ID", "the Vault error must be reported")
	assert.NotContains(t, err.Error(), "wrong-secret-id-value", "credentials must not be in errors")
	assert.NotContains(t, err.Error(), testRoleID, "credentials must not be in errors")
	assert.Empty(t, *sleeps, "a rejected login must not be retried")
}

func TestTransientErrorsAreRetried(t *testing.T) {
	fv := newFakeVault(t)
	s, _, sleeps := newTestStore(t, tokenEntry(fv.address), nil)
	fv.setUnavailable(2)

	v, err := s.resolve(context.Background(), "myapp/creds#password")
	require.NoError(t, err, "the read must work after retries")
	assert.Equal(t, testPassword, v, "unexpected secret value")
	assert.Equal(t, retryBackoff, *sleeps, "the configured backoff must be used")
}

func TestFailureCache(t *testing.T) {
	fv := newFakeVault(t)
	s, clock, sleeps := newTestStore(t, tokenEntry(fv.address), nil)
	ctx := context.Background()
	fv.setUnavailable(100)

	_, err := s.resolve(ctx, "myapp/creds#password")
	assert.ErrorContains(t, err, "status 503", "an unavailable Vault must be reported")
	_, reads := fv.counts()
	assert.Equal(t, len(retryBackoff)+1, reads, "every attempt must reach Vault")
	assert.Len(t, *sleeps, len(retryBackoff), "a backoff is expected between attempts")

	_, err = s.resolve(ctx, "myapp/other#password")
	assert.ErrorContains(t, err, "vault: "+fv.address+" is unavailable", "the failure must be returned for the whole connection")
	assert.ErrorContains(t, err, "status 503: Vault is sealed", "the failure must explain the cause")
	assert.NotContains(t, err.Error(), "myapp/creds", "the failure must not name the path of another monitor")
	_, readsAfter := fv.counts()
	assert.Equal(t, reads, readsAfter, "Vault must not be contacted while the failure is cached")

	fv.setUnavailable(0)
	clock.Advance(failureTTL + time.Second)
	_, err = s.resolve(ctx, "myapp/creds#password")
	assert.NoError(t, err, "Vault must be contacted again after the failure expires")
}

func TestPathFailureCache(t *testing.T) {
	fv := newFakeVault(t)
	fv.secrets["other"] = map[string]any{"password": "x"}
	fv.deny("denied")
	s, clock, _ := newTestStore(t, appRoleEntry(fv.address), nil)
	ctx := context.Background()

	for _, path := range []string{"missing", "denied"} {
		_, err := s.resolve(ctx, path+"#password")
		require.Error(t, err, "reading %s must fail", path)
	}
	logins, reads := fv.counts()
	lookups := fv.lookupCount()

	for _, path := range []string{"missing", "denied"} {
		_, err := s.resolve(ctx, path+"#password")
		assert.Error(t, err, "the cached failure for %s must be returned", path)
	}
	loginsAfter, readsAfter := fv.counts()
	assert.Equal(t, reads, readsAfter, "Vault must not be read while the failures are cached")
	assert.Equal(t, logins, loginsAfter, "Vault must not be logged in while the failures are cached")
	assert.Equal(t, lookups, fv.lookupCount(), "the token must not be looked up while the failures are cached")

	_, err := s.resolve(ctx, "other#password")
	assert.NoError(t, err, "a failure must not affect other paths")

	clock.Advance(failureTTL + time.Second)
	_, err = s.resolve(ctx, "missing#password")
	assert.Error(t, err, "the path must fail again")
	_, readsAfter = fv.counts()
	assert.Equal(t, reads+2, readsAfter, "Vault must be read again after the failure expires")
}

func TestConnectionErrorIsRetried(t *testing.T) {
	s, _, sleeps := newTestStore(t, tokenEntry(unreachableAddress(t)), nil)
	_, err := s.resolve(context.Background(), "myapp/creds#password")
	require.Error(t, err, "an unreachable Vault must be reported")
	assert.True(t, isConnectionError(err), "a refused connection must be a connection error")
	assert.Len(t, *sleeps, len(retryBackoff), "connection errors must be retried")
	assert.NotContains(t, err.Error(), testToken, "the token must not be in errors")
	assert.NotContains(t, err.Error(), "/v1/", "the request URL must not be in errors")
}

func TestCertificateErrorIsNotRetried(t *testing.T) {
	// The server certificate is not trusted by the store.
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	s, _, sleeps := newTestStore(t, tokenEntry(srv.URL), nil)
	ctx := context.Background()

	_, err := s.resolve(ctx, "myapp/creds#password")
	require.Error(t, err, "an untrusted certificate must be reported")
	assert.True(t, isConnectionError(err), "an untrusted certificate must be a connection error")
	assert.Empty(t, *sleeps, "an untrusted certificate must not be retried")

	_, err = s.resolve(ctx, "myapp/other#password")
	assert.ErrorContains(t, err, "is unavailable", "the failure must be returned for the whole connection")
}

func TestRequestErrorClassification(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
	// Wrapped as the Vault client does.
	viaClient := func(err error) error {
		return fmt.Errorf("error encountered while reading secret at secret/data/app: %w", err)
	}
	tests := map[string]struct {
		err            error
		wantConnection bool
		wantRetry      bool
	}{
		"server error":         {err: &vaultapi.ResponseError{StatusCode: http.StatusServiceUnavailable}, wantConnection: true, wantRetry: true},
		"rate limited":         {err: &vaultapi.ResponseError{StatusCode: http.StatusTooManyRequests}, wantConnection: true, wantRetry: true},
		"denied":               {err: &vaultapi.ResponseError{StatusCode: http.StatusForbidden}},
		"connection refused":   {err: viaClient(&url.Error{Op: "Get", URL: "https://vault/v1/secret/data/app", Err: refused}), wantConnection: true, wantRetry: true},
		"untrusted authority":  {err: viaClient(&url.Error{Op: "Get", URL: "https://vault/v1/secret/data/app", Err: x509.UnknownAuthorityError{}}), wantConnection: true},
		"verification failure": {err: viaClient(&url.Error{Op: "Get", URL: "https://vault/v1/secret/data/app", Err: &tls.CertificateVerificationError{Err: x509.HostnameError{Host: "vault"}}}), wantConnection: true},
		"unexpected response":  {err: viaClient(errors.New("missing expected 'data' element"))},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := requestError(tc.err, "reading secret/data/app")
			assert.Equal(t, tc.wantConnection, isConnectionError(err), "unexpected connection error classification")
			assert.Equal(t, tc.wantRetry, isRetryable(err), "unexpected retry classification")
			if tc.wantConnection {
				assert.NotContains(t, connectionCause(err), "secret/data/app", "the cause must not name the path")
			}
		})
	}
}

func TestUnexpectedResponseIsPathError(t *testing.T) {
	tests := map[string]struct {
		body    string
		wantErr string
	}{
		"KV v1 secret": {body: `{"data": {"password": "x"}}`, wantErr: "missing expected 'data' element"},
		"not JSON":     {body: "<html>proxy login</html>", wantErr: "invalid character"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fv := newFakeVault(t)
			fv.raw["unexpected"] = tc.body
			s, _, sleeps := newTestStore(t, tokenEntry(fv.address), nil)
			ctx := context.Background()

			_, err := s.resolve(ctx, "unexpected#password")
			assert.ErrorContains(t, err, tc.wantErr, "the unexpected response must be reported")
			assert.False(t, isConnectionError(err), "an unexpected response must not be a connection error")
			assert.Empty(t, *sleeps, "an unexpected response must not be retried")

			v, err := s.resolve(ctx, "myapp/creds#password")
			require.NoError(t, err, "other paths must not be affected")
			assert.Equal(t, testPassword, v, "unexpected secret value")
		})
	}
}

func TestNotFoundErrors(t *testing.T) {
	fv := newFakeVault(t)
	s, _, sleeps := newTestStore(t, tokenEntry(fv.address), nil)
	ctx := context.Background()

	_, err := s.resolve(ctx, "missing#password")
	assert.EqualError(t, err, "vault: no secret at secret/data/missing", "a missing secret must be reported")

	_, err = s.resolve(ctx, "myapp/creds#nope")
	assert.EqualError(t, err, `vault: field "nope" not found at secret/data/myapp/creds`, "a missing field must be reported")
	assert.Empty(t, *sleeps, "not found errors must not be retried")
}

func TestFieldValueConversion(t *testing.T) {
	fv := newFakeVault(t)
	s, _, _ := newTestStore(t, tokenEntry(fv.address), nil)

	for field, want := range map[string]string{"port": "5432", "enabled": "true", "obj": `{"a":"b"}`} {
		v, err := s.resolve(context.Background(), "myapp/creds#"+field)
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

	s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)
	_, err := s.resolve(context.Background(), "myapp/creds#password")
	require.NoError(t, err, "the configured address must be used")

	logins, _ := fv.counts()
	assert.Equal(t, 1, logins, "VAULT_TOKEN must not replace the AppRole login")
	req := fv.lastRequest()
	assert.Equal(t, strings.TrimPrefix(fv.address, "http://"), req.Host, "VAULT_ADDR must be ignored")
	assert.NotEqual(t, "token-from-env", req.Header.Get("X-Vault-Token"), "VAULT_TOKEN must be ignored")
	assert.Empty(t, req.Header.Get("X-Vault-Namespace"), "VAULT_NAMESPACE must be ignored")
	assert.Empty(t, req.Header.Get("X-From-Env"), "VAULT_HEADERS must be ignored")
}

func TestSecretsAreNotLogged(t *testing.T) {
	fv := newFakeVault(t)
	logger := logptest.NewFileLogger(t, "")
	s, _, _ := newTestStore(t, appRoleEntry(fv.address), logger.Logger)
	ctx := context.Background()

	_, err := s.resolve(ctx, "myapp/creds#password")
	require.NoError(t, err, "secret must be read")
	fv.expireTokens()
	s.mu.Lock()
	s.cache = map[string]cacheEntry{}
	s.mu.Unlock()
	_, err = s.resolve(ctx, "myapp/creds#password")
	require.NoError(t, err, "secret must be read again")

	logger.WaitLogsContains(t, "Logged in with AppRole", time.Second, "debug logs are expected")
	for _, secret := range []string{testPassword, testRoleID, testSecretID, "approle-token-"} {
		found, err := logger.FindInLogs(secret)
		require.NoError(t, err, "logs must be readable")
		assert.False(t, found, "%q must not be logged", secret)
		logger.ResetOffset()
	}
}

func TestClose(t *testing.T) {
	t.Run("approle revokes token and clears state", func(t *testing.T) {
		fv := newFakeVault(t)
		s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)

		_, err := s.resolve(context.Background(), "myapp/creds#password")
		require.NoError(t, err, "secret must be read")
		token := s.client.Token()

		require.NoError(t, s.Close(), "closing AppRole store must revoke token")
		require.NoError(t, s.Close(), "closing twice must be a no-op")

		revokes, revokedToks := fv.revokeState()
		assert.Equal(t, 1, revokes, "revoke-self must be called once")
		assert.Equal(t, []string{token}, revokedToks, "revoke-self must revoke the acquired AppRole token")
		assert.False(t, fv.isTokenValid(token), "token must no longer be valid in Vault")
		assert.Empty(t, s.client.Token(), "client token must be cleared")
		assert.Empty(t, s.cfg.Auth.AppRole.RoleID, "role_id must be cleared")
		assert.Empty(t, s.cfg.Auth.AppRole.SecretID, "secret_id must be cleared")
		assert.Empty(t, s.cache, "cached secrets must be cleared")

		_, err = s.resolve(context.Background(), "myapp/creds#password")
		assert.ErrorIs(t, err, errStoreClosed, "resolve after close must fail")
		logins, _ := fv.counts()
		assert.Equal(t, 1, logins, "closed store must not log in again")
	})

	t.Run("approle without login does not revoke", func(t *testing.T) {
		fv := newFakeVault(t)
		s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)

		require.NoError(t, s.Close(), "closing unused AppRole store must succeed")
		revokes, _ := fv.revokeState()
		assert.Zero(t, revokes, "unused store must not call revoke-self")
		assert.Empty(t, s.cfg.Auth.AppRole.SecretID, "secret_id must be cleared")
	})

	t.Run("static token is not revoked", func(t *testing.T) {
		fv := newFakeVault(t)
		s, _, _ := newTestStore(t, tokenEntry(fv.address), nil)

		_, err := s.resolve(context.Background(), "myapp/creds#password")
		require.NoError(t, err, "secret must be read")
		require.NoError(t, s.Close(), "closing static-token store must succeed")

		revokes, _ := fv.revokeState()
		assert.Zero(t, revokes, "static token must not be revoked")
		assert.True(t, fv.isTokenValid(testToken), "static token must stay valid in Vault")
		assert.Empty(t, s.client.Token(), "client token must be cleared")
		assert.Empty(t, s.cfg.Auth.Token, "configured token must be cleared")
		assert.Empty(t, s.cache, "cached secrets must be cleared")
	})

	t.Run("revoke error and timeout still clear state", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			setup func(*fakeVault)
		}{
			{name: "vault 500", setup: func(fv *fakeVault) { fv.setRevokeError(http.StatusInternalServerError) }},
			{name: "timeout", setup: func(fv *fakeVault) {
				prev := revokeTimeout
				revokeTimeout = 50 * time.Millisecond
				t.Cleanup(func() { revokeTimeout = prev })
				block := make(chan struct{})
				t.Cleanup(func() { close(block) })
				fv.setRevokeBlock(block)
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				fv := newFakeVault(t)
				s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)
				_, err := s.resolve(context.Background(), "myapp/creds#password")
				require.NoError(t, err, "secret must be read")
				token := s.client.Token()

				tc.setup(fv)
				err = s.Close()
				assert.ErrorContains(t, err, "revoking AppRole token", "revoke failure must be reported")
				assert.NotContains(t, err.Error(), token, "error must not leak the token")
				assert.Empty(t, s.client.Token(), "client token must still be cleared")
				assert.Empty(t, s.cache, "cached secrets must still be cleared")
			})
		}
	})
}

func TestConcurrentResolveAndClose(t *testing.T) {
	fv := newFakeVault(t)
	s, _, _ := newTestStore(t, appRoleEntry(fv.address), nil)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 5 {
				_, _ = s.resolve(context.Background(), "myapp/creds#password")
			}
		})
	}
	assert.NoError(t, s.Close(), "concurrent close must succeed")
	wg.Wait()

	assert.Empty(t, s.client.Token(), "client token must stay cleared")
	assert.Empty(t, s.cache, "cache must stay empty")
}

func TestPluginRecoversAfterCAFileAppears(t *testing.T) {
	fv := newFakeVault(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(fv.handle))
	t.Cleanup(srv.Close)
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	entry := tokenEntry(srv.URL)
	entry["type"], entry["name"] = storeType, "main"
	entry["ssl"] = map[string]any{"certificate_authorities": []string{caPath}}
	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":                 "http",
		"password":             "vault://main@myapp/creds#password",
		secretstores.ConfigKey: []any{entry},
	})
	require.NoError(t, err, "monitor config must be valid")

	logger := logptest.NewTestingLogger(t, "")
	inner := plugin.PluginFactory{
		Name: "test",
		Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
			password, err := cfg.String("password", -1)
			require.NoError(t, err, "resolved password must be readable")
			assert.Equal(t, testPassword, password, "plugin must receive the resolved secret")
			return plugin.Plugin{
				Endpoints: 1,
				Logger:    info.Logger,
				Jobs:      []jobs.Job{func(*beat.Event) ([]jobs.Job, error) { return nil, nil }},
			}, nil
		},
	}
	wrapped := secretstores.NewResolver(logger).WrapFactory(inner, secretstores.NewRedactor())
	p, err := wrapped.Make("test", cfg, beat.Info{Logger: logger})
	require.NoError(t, err, "store creation failure must be deferred to scheduled runs")
	t.Cleanup(func() { assert.NoError(t, p.Close(), "plugin must close successfully") })
	_, err = p.Jobs[0](&beat.Event{})
	require.ErrorContains(t, err, "initial CA load failed", "missing CA must prevent store creation")

	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(caPath, cert, 0600), "CA file must be provisioned")
	_, err = p.Jobs[0](&beat.Event{})
	require.NoError(t, err, "next run must recover without updating the config")
	_, reads := fv.counts()
	assert.Equal(t, 1, reads, "recovered store must read the secret")
}

func TestVaultStoreEvictedAndRevokedOnPluginClose(t *testing.T) {
	fv := newFakeVault(t)
	logger := logptest.NewTestingLogger(t, "")
	inner := plugin.PluginFactory{
		Name: "test",
		Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
			return plugin.Plugin{
				Endpoints: 1,
				Logger:    info.Logger,
				Jobs:      []jobs.Job{func(*beat.Event) ([]jobs.Job, error) { return nil, nil }},
			}, nil
		},
	}

	wrapped := secretstores.NewResolver(logger).WrapFactory(inner, secretstores.NewRedactor())
	entry := appRoleEntry(fv.address)
	entry["type"] = storeType
	entry["name"] = "main"
	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":                 "http",
		"password":             "vault://main@myapp/creds#password",
		secretstores.ConfigKey: []any{entry},
	})
	require.NoError(t, err, "monitor config must be valid")

	p1, err := wrapped.Make("test", cfg, beat.Info{Logger: logger})
	require.NoError(t, err, "first plugin must be created")
	require.NoError(t, p1.Close(), "closing plugin must succeed")

	revokes, revokedToks := fv.revokeState()
	assert.Equal(t, 1, revokes, "evicting the Vault store must revoke its AppRole token")
	require.Len(t, revokedToks, 1, "one token must be revoked")
	assert.False(t, fv.isTokenValid(revokedToks[0]), "revoked token must be invalid in Vault")

	p2, err := wrapped.Make("test", cfg, beat.Info{Logger: logger})
	require.NoError(t, err, "second plugin must be created")
	defer func() { _ = p2.Close() }()

	logins, reads := fv.counts()
	assert.Equal(t, 2, logins, "recreated store must log in again after eviction")
	assert.Equal(t, 2, reads, "recreated store must re-read after cache eviction")
}

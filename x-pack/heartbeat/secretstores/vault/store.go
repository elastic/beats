// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package vault implements the "vault" secret store for Heartbeat monitors:
// HashiCorp Vault KV v2 secrets, read with token or AppRole authentication.
//
// References have the form $vault{<path>#<field>}, where path is relative to
// the configured KV mount.
package vault

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// storeType is the secret_stores type and the reference prefix.
const storeType = "vault"

const (
	requestTimeout = 10 * time.Second
	// cacheTTL is how long a secret read is reused by every monitor.
	cacheTTL = 5 * time.Minute
	// failureTTL is how long a connection failure is returned without
	// contacting Vault again, so an outage does not slow down every monitor.
	failureTTL = 30 * time.Second
)

// retryBackoff holds the waits between attempts for transient errors.
var retryBackoff = []time.Duration{time.Second, 2 * time.Second}

func init() {
	secretstores.Register(storeType, newStore)
}

type store struct {
	cfg    config
	client *vaultapi.Client
	logger *logp.Logger

	// loginMu serializes AppRole logins, so concurrent auth failures result
	// in a single login.
	loginMu sync.Mutex

	mu          sync.Mutex
	cache       map[string]cacheEntry
	failure     error
	failedUntil time.Time

	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

type cacheEntry struct {
	data    map[string]any
	expires time.Time
}

func newStore(cfg *conf.C, logger *logp.Logger) (secretstores.Store, error) {
	return newStoreWithTransport(cfg, logger, nil)
}

// newStoreWithTransport creates a store; a nil transport means the default
// HTTP transport.
func newStoreWithTransport(cfg *conf.C, logger *logp.Logger, transport http.RoundTripper) (*store, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}

	client, err := newClient(c, transport)
	if err != nil {
		return nil, err
	}

	return &store{
		cfg:    c,
		client: client,
		logger: logger.Named(storeType),
		cache:  map[string]cacheEntry{},
		now:    time.Now,
		sleep:  sleepContext,
	}, nil
}

// newClient creates a Vault client that only uses the given config. The
// Vault library reads VAULT_* environment variables by default; none of them
// must change how Heartbeat connects.
func newClient(c config, transport http.RoundTripper) (*vaultapi.Client, error) {
	if transport == nil {
		// Same defaults as the Vault library, honoring the proxy environment.
		t := cleanhttp.DefaultPooledTransport()
		t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		transport = t
	}

	// Passing a config, instead of vaultapi.DefaultConfig(), keeps the values
	// read from the environment out of the client.
	client, err := vaultapi.NewClient(&vaultapi.Config{
		Address: c.Address,
		HttpClient: &http.Client{
			Transport: transport,
			Timeout:   requestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Timeout:    requestTimeout,
		MaxRetries: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	// NewClient always reads VAULT_TOKEN, VAULT_NAMESPACE and VAULT_HEADERS.
	client.SetHeaders(http.Header{vaultapi.RequestHeaderName: []string{"true"}})
	client.ClearToken()
	if c.Namespace != "" {
		client.SetNamespace(c.Namespace)
	}
	if c.Auth.Token != nil {
		client.SetToken(c.Auth.Token.Value)
	}
	return client, nil
}

// Resolve returns the value of field in the KV v2 secret at path.
func (s *store) Resolve(ctx context.Context, path, field string) (string, error) {
	data, err := s.secret(ctx, path)
	if err != nil {
		return "", err
	}
	v, ok := data[field]
	if !ok {
		return "", fmt.Errorf("vault: field %q not found at %s", field, s.location(path))
	}
	return stringValue(v)
}

// secret returns the data of the secret at path, from the cache when
// possible.
func (s *store) secret(ctx context.Context, path string) (map[string]any, error) {
	s.mu.Lock()
	if e, ok := s.cache[path]; ok && s.now().Before(e.expires) {
		s.mu.Unlock()
		s.logger.Debugf("Using cached secret %s", s.location(path))
		return e.data, nil
	}
	if s.failure != nil && s.now().Before(s.failedUntil) {
		err := s.failure
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()

	data, err := s.fetchWithRetry(ctx, path)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		if isTransient(err) {
			s.failure = err
			s.failedUntil = s.now().Add(failureTTL)
		}
		return nil, err
	}
	s.failure = nil
	s.purgeExpired()
	s.cache[path] = cacheEntry{data: data, expires: s.now().Add(cacheTTL)}
	return data, nil
}

// purgeExpired removes expired cache entries. s.mu must be held.
func (s *store) purgeExpired() {
	now := s.now()
	for k, e := range s.cache {
		if !now.Before(e.expires) {
			delete(s.cache, k)
		}
	}
}

func (s *store) fetchWithRetry(ctx context.Context, path string) (map[string]any, error) {
	for attempt := 0; ; attempt++ {
		data, err := s.fetch(ctx, path)
		if err == nil || !isTransient(err) || attempt >= len(retryBackoff) {
			return data, err
		}
		s.logger.Debugf("Reading secret %s failed, retrying: %v", s.location(path), err)
		if sleepErr := s.sleep(ctx, retryBackoff[attempt]); sleepErr != nil {
			return nil, err
		}
	}
}

// fetch reads the secret at path. With AppRole it logs in when needed, and
// logs in again once if Vault rejects the token.
func (s *store) fetch(ctx context.Context, path string) (map[string]any, error) {
	appRole := s.cfg.Auth.AppRole != nil
	if appRole && s.client.Token() == "" {
		if err := s.login(ctx, ""); err != nil {
			return nil, err
		}
	}

	token := s.client.Token()
	secret, err := s.client.KVv2(s.cfg.KVMount).Get(ctx, path)
	if appRole && isAuthError(err) {
		s.logger.Debug("Vault rejected the token, logging in again")
		if err := s.login(ctx, token); err != nil {
			return nil, err
		}
		secret, err = s.client.KVv2(s.cfg.KVMount).Get(ctx, path)
	}
	if errors.Is(err, vaultapi.ErrSecretNotFound) {
		return nil, fmt.Errorf("vault: no secret at %s", s.location(path))
	}
	if err != nil {
		return nil, requestError(err, "reading "+s.location(path))
	}
	// Data is nil when the latest version was deleted.
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("vault: no secret at %s", s.location(path))
	}
	return secret.Data, nil
}

// login performs an AppRole login, unless the token was already replaced
// since staleToken was used.
func (s *store) login(ctx context.Context, staleToken string) error {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()

	if s.client.Token() != staleToken {
		return nil
	}

	approle := s.cfg.Auth.AppRole
	resp, err := s.client.Logical().WriteWithContext(ctx, "auth/"+approle.MountPath+"/login", map[string]any{
		"role_id":   approle.RoleID,
		"secret_id": approle.SecretID,
	})
	if err != nil {
		return requestError(err, "logging in with AppRole at auth/"+approle.MountPath)
	}
	if resp == nil || resp.Auth == nil || resp.Auth.ClientToken == "" {
		return fmt.Errorf("vault: AppRole login at auth/%s returned no token", approle.MountPath)
	}
	s.client.SetToken(resp.Auth.ClientToken)
	s.logger.Debugf("Logged in with AppRole at auth/%s", approle.MountPath)
	return nil
}

// requestError converts a Vault client error into an error without secrets,
// marking it transient when retrying may help.
func requestError(err error, msg string) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("vault: %s: %w", msg, err)
	}

	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		text := fmt.Sprintf("vault: %s failed with status %d", msg, respErr.StatusCode)
		// Raw bodies are not Vault errors and could contain anything.
		if !respErr.RawError && len(respErr.Errors) > 0 {
			text += ": " + strings.Join(respErr.Errors, "; ")
		}
		if respErr.StatusCode == http.StatusTooManyRequests || respErr.StatusCode >= 500 {
			return &transientError{errors.New(text)}
		}
		return errors.New(text)
	}

	// Connection errors and timeouts.
	return &transientError{fmt.Errorf("vault: %s failed: %w", msg, err)}
}

func (s *store) location(path string) string {
	return s.cfg.KVMount + "/data/" + path
}

func isAuthError(err error) bool {
	var respErr *vaultapi.ResponseError
	return errors.As(err, &respErr) &&
		(respErr.StatusCode == http.StatusUnauthorized || respErr.StatusCode == http.StatusForbidden)
}

// transientError marks errors that retrying may fix.
type transientError struct {
	err error
}

func (e *transientError) Error() string { return e.err.Error() }
func (e *transientError) Unwrap() error { return e.err }

func isTransient(err error) bool {
	var t *transientError
	return errors.As(err, &t)
}

// stringValue converts a KV field value into the string put in the config.
func stringValue(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case json.Number:
		return val.String(), nil
	case nil:
		return "", nil
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("vault: cannot convert field value of type %T", val)
		}
		return string(b), nil
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

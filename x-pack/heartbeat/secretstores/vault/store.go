// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package vault implements the "vault" secret store for Heartbeat monitors:
// HashiCorp Vault KV v2 secrets, read with token or AppRole authentication.
//
// References have the form vault://<path>#<field>, where path is relative to
// the configured KV mount. Path and field are percent-decoded.
package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// storeType is the secret_stores type and the reference prefix.
const storeType = "vault"

const (
	requestTimeout = 10 * time.Second
	// cacheTTL is how long a secret read is reused by every monitor.
	cacheTTL = 5 * time.Minute
	// failureTTL is how long a failure is returned without contacting Vault
	// again: a connection failure for every path, so an outage does not slow
	// down every monitor, and other failures for their path, so a
	// misconfigured monitor does not contact Vault on every run.
	failureTTL = 30 * time.Second
)

// retryBackoff holds the waits between attempts for transient errors.
var retryBackoff = []time.Duration{time.Second, 2 * time.Second}

func init() {
	// Registration only fails on programming errors, such as a duplicate type.
	if err := secretstores.Register(storeType, newStore); err != nil {
		panic(err)
	}
}

type store struct {
	cfg    config
	client *vaultapi.Client
	logger *logp.Logger

	// loginMu is used to avoid concurrent auth failures
	loginMu sync.Mutex

	mu    sync.Mutex
	cache map[string]cacheEntry
	// failure is returned for every path until failedUntil after a
	// connection error. It doesn't name the path that failed, which
	// belongs to another monitor.
	failure     error
	failedUntil time.Time

	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

// cacheEntry is the data read at a path, or the error reading it.
type cacheEntry struct {
	data    map[string]any
	err     error
	expires time.Time
}

func newStore(cfg *conf.C, logger *logp.Logger) (secretstores.Store, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}

	client, err := newClient(c, logger)
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
func newClient(c config, logger *logp.Logger) (*vaultapi.Client, error) {
	tlsConfig, err := clientTLSConfig(c, logger)
	if err != nil {
		return nil, err
	}
	// Same defaults as the Vault library, honoring the proxy environment.
	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = tlsConfig

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
	if c.Auth.Token != "" {
		client.SetToken(c.Auth.Token)
	}
	return client, nil
}

// clientTLSConfig returns the TLS config of the connection to Vault. The
// libbeat defaults, used for the unset ssl settings, require TLS 1.2.
func clientTLSConfig(c config, logger *logp.Logger) (*tls.Config, error) {
	if !c.SSL.IsEnabled() {
		return &tls.Config{MinVersion: tls.VersionTLS12}, nil
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(c.SSL, logger)
	if err != nil {
		return nil, fmt.Errorf("invalid 'ssl': %w", err)
	}
	// The address was validated, it parses.
	u, err := url.Parse(c.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid 'address': %w", err)
	}
	return tlsConfig.BuildModuleClientConfig(u.Hostname(), tlscommon.WithLogger(logger)), nil
}

// resolve returns the value of the field in the KV v2 secret that content,
// <path>#<field>, refers to.
func (s *store) resolve(ctx context.Context, content string) (string, error) {
	path, field, err := parseReference(content)
	if err != nil {
		return "", fmt.Errorf("vault: invalid reference: %w", err)
	}
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
		if e.err != nil {
			return nil, e.err
		}
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
	s.purgeExpired()
	switch {
	case err == nil:
		s.failure = nil
		s.cache[path] = cacheEntry{data: data, expires: s.now().Add(cacheTTL)}
		return data, nil
	case isConnectionError(err):
		s.failedUntil = s.now().Add(failureTTL)
		s.failure = fmt.Errorf("vault: %s is unavailable, retrying after %s: %s",
			redactedAddress(s.cfg.Address), s.failedUntil.Format(time.RFC3339), connectionCause(err))
	case !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded):
		// Such as a denied or missing path: reading it again fails the same way.
		s.cache[path] = cacheEntry{err: err, expires: s.now().Add(failureTTL)}
	}
	return nil, err
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
		if err == nil || !isRetryable(err) || attempt >= len(retryBackoff) {
			return data, err
		}
		s.logger.Debugf("Reading secret %s failed, retrying: %v", s.location(path), err)
		if sleepErr := s.sleep(ctx, retryBackoff[attempt]); sleepErr != nil {
			return nil, err
		}
	}
}

// fetch reads the secret at path. With AppRole it logs in when needed, and
// logs in again once if Vault rejects the token, not the path.
func (s *store) fetch(ctx context.Context, path string) (map[string]any, error) {
	appRole := s.cfg.Auth.AppRole != nil
	if appRole && s.client.Token() == "" {
		if err := s.login(ctx, ""); err != nil {
			return nil, err
		}
	}

	token := s.client.Token()
	secret, err := s.client.KVv2(s.cfg.KVMount).Get(ctx, path)
	// Logging in again doesn't fix a token without access to the path.
	if appRole && isAuthError(err) && s.tokenRejected(ctx, token) {
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

// tokenRejected reports whether token, used for a read Vault denied, must
// be replaced: it was replaced already, or Vault rejects it. Vault answers
// 403 both to an invalid token and to a valid token without access to a
// path; only the first one can look itself up. A token without access to its
// own lookup is considered rejected.
func (s *store) tokenRejected(ctx context.Context, token string) bool {
	if s.client.Token() != token {
		return true
	}
	if _, err := s.client.Auth().Token().LookupSelfWithContext(ctx); err != nil {
		s.logger.Debugf("Token lookup failed: %v", requestError(err, "looking up the token"))
		return true
	}
	// The lookup used the current token, which must still be token.
	return s.client.Token() != token
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

// requestError converts a Vault client error into an error without secrets.
// Errors of the connection to Vault, rather than of the request, are
// connectionErrors.
func requestError(err error, msg string) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("vault: %s: %w", msg, err)
	}

	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		cause := fmt.Sprintf("status %d", respErr.StatusCode)
		// Raw bodies are not Vault errors and could contain anything.
		if !respErr.RawError && len(respErr.Errors) > 0 {
			cause += ": " + strings.Join(respErr.Errors, "; ")
		}
		text := fmt.Sprintf("vault: %s failed with %s", msg, cause)
		if respErr.StatusCode == http.StatusTooManyRequests || respErr.StatusCode >= 500 {
			return &connectionError{err: errors.New(text), cause: cause, retry: true}
		}
		return errors.New(text)
	}

	// The cause of a url.Error, without the request URL: the path is already
	// in msg, and other monitors must not see it.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &connectionError{
			err:   fmt.Errorf("vault: %s failed: %w", msg, urlErr.Err),
			cause: urlErr.Err.Error(),
			// Retrying doesn't fix an untrusted certificate.
			retry: !isCertificateError(urlErr.Err),
		}
	}

	// Such as a response that isn't a KV v2 secret.
	return fmt.Errorf("vault: %s failed: %w", msg, err)
}

func isCertificateError(err error) bool {
	var (
		unknownAuthority x509.UnknownAuthorityError
		hostname         x509.HostnameError
		invalid          x509.CertificateInvalidError
		verification     *tls.CertificateVerificationError
	)
	return errors.As(err, &unknownAuthority) ||
		errors.As(err, &hostname) ||
		errors.As(err, &invalid) ||
		errors.As(err, &verification)
}

// redactedAddress returns the Vault address without a password.
func redactedAddress(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return "<invalid address>"
	}
	return u.Redacted()
}

func (s *store) location(path string) string {
	return s.cfg.KVMount + "/data/" + path
}

func isAuthError(err error) bool {
	var respErr *vaultapi.ResponseError
	return errors.As(err, &respErr) &&
		(respErr.StatusCode == http.StatusUnauthorized || respErr.StatusCode == http.StatusForbidden)
}

// connectionError is a failure of the connection to Vault, such as a
// refused connection or an unavailable server, as opposed to a failure of
// one request.
type connectionError struct {
	err error
	// cause describes the failure without the request.
	cause string
	// retry is true if retrying may fix the failure.
	retry bool
}

func (e *connectionError) Error() string { return e.err.Error() }
func (e *connectionError) Unwrap() error { return e.err }

func isConnectionError(err error) bool {
	var c *connectionError
	return errors.As(err, &c)
}

func isRetryable(err error) bool {
	var c *connectionError
	return errors.As(err, &c) && c.retry
}

func connectionCause(err error) string {
	var c *connectionError
	if errors.As(err, &c) {
		return c.cause
	}
	return ""
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

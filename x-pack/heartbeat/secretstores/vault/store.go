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

const requestTimeout = 10 * time.Second

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
	reader *secretstores.Reader[string]

	// loginMu is used to avoid concurrent auth failures
	loginMu sync.Mutex
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

	st := &store{cfg: c, client: client, logger: logger.Named(storeType)}
	st.reader = secretstores.NewReader[string](st.logger)
	st.reader.Name = storeType
	st.reader.Address = c.Address
	st.reader.Noun = "secret"
	st.reader.Describe = st.location
	st.reader.Fetch = st.fetch
	return st, nil
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

func clientTLSConfig(c config, logger *logp.Logger) (*tls.Config, error) {
	return secretstores.ClientTLSConfig(c.SSL, c.Address, logger)
}

// resolve returns the value of the field in the KV v2 secret that content,
// <path>#<field>, refers to.
func (s *store) resolve(ctx context.Context, content string) (string, error) {
	path, field, err := parseReference(content)
	if err != nil {
		return "", fmt.Errorf("vault: invalid reference: %w", err)
	}
	data, err := s.reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	v, ok := data[field]
	if !ok {
		return "", fmt.Errorf("vault: field %q not found at %s", field, s.location(path))
	}
	return secretstores.StringValue(storeType, v)
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
			return &secretstores.ConnectionError{Err: errors.New(text), Cause: cause, Retry: true}
		}
		return errors.New(text)
	}

	if connErr, ok := secretstores.URLConnectionError(storeType, msg, err); ok {
		return connErr
	}

	// Such as a response that isn't a KV v2 secret.
	return fmt.Errorf("vault: %s failed: %w", msg, err)
}

func (s *store) location(path string) string {
	return s.cfg.KVMount + "/data/" + path
}

func isAuthError(err error) bool {
	var respErr *vaultapi.ResponseError
	return errors.As(err, &respErr) &&
		(respErr.StatusCode == http.StatusUnauthorized || respErr.StatusCode == http.StatusForbidden)
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package cyberark implements the "cyberark" secret store for Heartbeat
// monitors: CyberArk Central Credential Provider (CCP / AIMWebService)
// accounts, read over HTTP/HTTPS with mTLS client certificates, token, or
// Basic authentication.
//
// References have the form cyberark://<safe>/<object>#<field> or
// cyberark://<safe>/<folder>/<object>#<field>. Segments and field are
// percent-decoded.
package cyberark

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// storeType is the secret_stores type and the reference prefix.
const storeType = "cyberark"

const (
	requestTimeout  = 10 * time.Second
	maxResponseSize = 1 << 20
	// cacheTTL is how long an account read is reused by every monitor.
	cacheTTL = 5 * time.Minute
	// failureTTL is how long a failure is returned without contacting
	// CyberArk again.
	failureTTL = 30 * time.Second
)

// retryBackoff holds the waits between attempts for retryable connection errors.
var retryBackoff = []time.Duration{time.Second, 2 * time.Second}

// fieldAliases maps common lowercase field names to their CyberArk CCP
// response property names when the lowercase name is absent from the response.
var fieldAliases = map[string]string{
	"password": "Content",
	"username": "UserName",
	"address":  "Address",
	"database": "Database",
}

func init() {
	if err := secretstores.Register(storeType, newStore); err != nil {
		panic(err)
	}
}

type store struct {
	cfg    config
	client *http.Client
	logger *logp.Logger

	mu          sync.Mutex
	cache       map[string]cacheEntry
	failure     error
	failedUntil time.Time

	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

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

func newClient(c config, logger *logp.Logger) (*http.Client, error) {
	tlsConfig, err := clientTLSConfig(c, logger)
	if err != nil {
		return nil, err
	}
	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func clientTLSConfig(c config, logger *logp.Logger) (*tls.Config, error) {
	if !c.SSL.IsEnabled() {
		return &tls.Config{MinVersion: tls.VersionTLS12}, nil
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(c.SSL, logger)
	if err != nil {
		return nil, fmt.Errorf("invalid 'ssl': %w", err)
	}
	u, err := url.Parse(c.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid 'address': %w", err)
	}
	return tlsConfig.BuildModuleClientConfig(u.Hostname(), tlscommon.WithLogger(logger)), nil
}

func (s *store) resolve(ctx context.Context, content string) (string, error) {
	ref, field, err := parseReference(content)
	if err != nil {
		return "", fmt.Errorf("cyberark: invalid reference: %w", err)
	}
	data, err := s.account(ctx, ref)
	if err != nil {
		return "", err
	}
	v, ok := lookupField(data, field)
	if !ok {
		return "", fmt.Errorf("cyberark: field %q not found in account %s", field, ref)
	}
	return stringValue(v)
}

func lookupField(data map[string]any, field string) (any, bool) {
	if v, ok := data[field]; ok {
		return v, true
	}
	if canonical, ok := fieldAliases[field]; ok {
		v, found := data[canonical]
		return v, found
	}
	return nil, false
}

func (s *store) account(ctx context.Context, ref accountRef) (map[string]any, error) {
	key := ref.String()
	s.mu.Lock()
	if e, ok := s.cache[key]; ok && s.now().Before(e.expires) {
		s.mu.Unlock()
		if e.err != nil {
			return nil, e.err
		}
		s.logger.Debugf("Using cached account %s", key)
		return e.data, nil
	}
	if s.failure != nil && s.now().Before(s.failedUntil) {
		err := s.failure
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()

	data, err := s.fetchWithRetry(ctx, ref)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpired()
	switch {
	case err == nil:
		s.failure = nil
		s.cache[key] = cacheEntry{data: data, expires: s.now().Add(cacheTTL)}
		return data, nil
	case isConnectionError(err):
		s.failedUntil = s.now().Add(failureTTL)
		s.failure = fmt.Errorf("cyberark: %s is unavailable, retrying after %s: %s",
			redactedAddress(s.cfg.Address), s.failedUntil.Format(time.RFC3339), connectionCause(err))
	case !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded):
		s.cache[key] = cacheEntry{err: err, expires: s.now().Add(failureTTL)}
	}
	return nil, err
}

func (s *store) purgeExpired() {
	now := s.now()
	for k, e := range s.cache {
		if !now.Before(e.expires) {
			delete(s.cache, k)
		}
	}
}

func (s *store) fetchWithRetry(ctx context.Context, ref accountRef) (map[string]any, error) {
	for attempt := 0; ; attempt++ {
		data, err := s.fetch(ctx, ref)
		if err == nil || !isRetryable(err) || attempt >= len(retryBackoff) {
			return data, err
		}
		s.logger.Debugf("Reading account %s failed, retrying: %v", ref, err)
		if sleepErr := s.sleep(ctx, retryBackoff[attempt]); sleepErr != nil {
			return nil, err
		}
	}
}

func (s *store) fetch(ctx context.Context, ref accountRef) (map[string]any, error) {
	reqURL, err := s.accountURL(ref)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cyberark: could not build request for %s: %w", ref, err)
	}
	req.Header.Set("Accept", "application/json")
	if s.cfg.Auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Auth.Token)
	} else if s.cfg.Auth.Basic != nil {
		req.SetBasicAuth(s.cfg.Auth.Basic.Username, s.cfg.Auth.Basic.Password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, requestError(err, "reading "+ref.String())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, requestError(err, "reading "+ref.String())
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cyberark: no account at %s", ref)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp.StatusCode, body, "reading "+ref.String())
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("cyberark: reading %s failed: invalid JSON response: %w", ref, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("cyberark: no account at %s", ref)
	}
	return data, nil
}

func (s *store) accountURL(ref accountRef) (string, error) {
	u, err := url.Parse(strings.TrimRight(s.cfg.Address, "/") + s.cfg.ServicePath)
	if err != nil {
		return "", fmt.Errorf("cyberark: invalid address: %w", err)
	}
	q := u.Query()
	q.Set("AppID", s.cfg.AppID)
	q.Set("Safe", ref.safe)
	if ref.folder != "" {
		q.Set("Folder", ref.folder)
	}
	q.Set("Object", ref.object)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ccpError is the JSON error payload returned by CyberArk CCP AIMWebService.
type ccpError struct {
	ErrorCode string `json:"ErrorCode"`
	ErrorMsg  string `json:"ErrorMsg"`
}

func responseError(statusCode int, body []byte, msg string) error {
	cause := fmt.Sprintf("status %d", statusCode)
	var apiErr ccpError
	if err := json.Unmarshal(body, &apiErr); err == nil {
		switch {
		case apiErr.ErrorCode != "" && apiErr.ErrorMsg != "":
			cause += fmt.Sprintf(": %s: %s", apiErr.ErrorCode, strings.TrimSpace(apiErr.ErrorMsg))
		case apiErr.ErrorMsg != "":
			cause += ": " + strings.TrimSpace(apiErr.ErrorMsg)
		case apiErr.ErrorCode != "":
			cause += ": " + apiErr.ErrorCode
		}
	}
	text := fmt.Sprintf("cyberark: %s failed with %s", msg, cause)
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return &connectionError{err: errors.New(text), cause: cause, retry: true}
	}
	return errors.New(text)
}

func requestError(err error, msg string) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("cyberark: %s: %w", msg, err)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &connectionError{
			err:   fmt.Errorf("cyberark: %s failed: %w", msg, urlErr.Err),
			cause: urlErr.Err.Error(),
			retry: !isCertificateError(urlErr.Err),
		}
	}
	return fmt.Errorf("cyberark: %s failed: %w", msg, err)
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

func redactedAddress(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return "<invalid address>"
	}
	return u.Redacted()
}

type connectionError struct {
	err   error
	cause string
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
			return "", fmt.Errorf("cyberark: cannot convert field value of type %T", val)
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

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package secretstores

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	// DefaultCacheTTL is how long a read is reused by every monitor.
	DefaultCacheTTL = 5 * time.Minute
	// DefaultFailureTTL is how long a failure is returned without contacting
	// the secret store again: a connection failure for every key, so an outage
	// does not slow down every monitor, and other failures for their key, so a
	// misconfigured monitor does not contact the store on every run.
	DefaultFailureTTL = 30 * time.Second
)

// DefaultBackoff holds the waits between attempts for transient errors.
var DefaultBackoff = []time.Duration{time.Second, 2 * time.Second}

// Reader reads the secret data of keys of type K with a shared cache, retries
// of transient errors, and caching of failures. It is safe for concurrent
// use. The fields must not change after the first Read, except in tests.
type Reader[K comparable] struct {
	// Name prefixes the errors, such as "vault".
	Name string
	// Address is the secret store address, shown without credentials when the
	// connection fails.
	Address string
	// Noun names what is read in logs, such as "secret".
	Noun string
	// Describe returns the location of key for logs. It must not include
	// secrets.
	Describe func(key K) string
	// Fetch reads key from the secret store. Errors of the connection to the
	// store must be ConnectionErrors.
	Fetch func(ctx context.Context, key K) (map[string]any, error)

	CacheTTL   time.Duration
	FailureTTL time.Duration
	Backoff    []time.Duration
	Now        func() time.Time
	Sleep      func(ctx context.Context, d time.Duration) error

	logger *logp.Logger

	mu    sync.Mutex
	cache map[K]cacheEntry
	// failure is returned for every key until failedUntil after a connection
	// error. It doesn't name the key that failed, which belongs to another
	// monitor.
	failure     error
	failedUntil time.Time
}

// cacheEntry is the data read for a key, or the error reading it.
type cacheEntry struct {
	data    map[string]any
	err     error
	expires time.Time
}

// NewReader returns a Reader with the default TTLs and backoff. The caller
// sets Name, Address, Noun, Describe and Fetch before the first Read.
func NewReader[K comparable](logger *logp.Logger) *Reader[K] {
	return &Reader[K]{
		CacheTTL:   DefaultCacheTTL,
		FailureTTL: DefaultFailureTTL,
		Backoff:    DefaultBackoff,
		Now:        time.Now,
		Sleep:      SleepContext,
		logger:     logger,
		cache:      map[K]cacheEntry{},
	}
}

// Reset drops the cached reads and failures.
func (r *Reader[K]) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = map[K]cacheEntry{}
	r.failure = nil
}

// Read returns the data of key, from the cache when possible.
func (r *Reader[K]) Read(ctx context.Context, key K) (map[string]any, error) {
	r.mu.Lock()
	if e, ok := r.cache[key]; ok && r.Now().Before(e.expires) {
		r.mu.Unlock()
		if e.err != nil {
			return nil, e.err
		}
		r.logger.Debugf("Using cached %s %s", r.Noun, r.Describe(key))
		return e.data, nil
	}
	if r.failure != nil && r.Now().Before(r.failedUntil) {
		err := r.failure
		r.mu.Unlock()
		return nil, err
	}
	r.mu.Unlock()

	data, err := r.fetchWithRetry(ctx, key)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.purgeExpired()
	switch {
	case err == nil:
		r.failure = nil
		r.cache[key] = cacheEntry{data: data, expires: r.Now().Add(r.CacheTTL)}
		return data, nil
	case IsConnectionError(err):
		r.failedUntil = r.Now().Add(r.FailureTTL)
		r.failure = fmt.Errorf("%s: %s is unavailable, retrying after %s: %s",
			r.Name, RedactedAddress(r.Address), r.failedUntil.Format(time.RFC3339), ConnectionCause(err))
	case !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded):
		// Such as a denied or missing key: reading it again fails the same way.
		r.cache[key] = cacheEntry{err: err, expires: r.Now().Add(r.FailureTTL)}
	}
	return nil, err
}

// purgeExpired removes expired cache entries. r.mu must be held.
func (r *Reader[K]) purgeExpired() {
	now := r.Now()
	for k, e := range r.cache {
		if !now.Before(e.expires) {
			delete(r.cache, k)
		}
	}
}

func (r *Reader[K]) fetchWithRetry(ctx context.Context, key K) (map[string]any, error) {
	for attempt := 0; ; attempt++ {
		data, err := r.Fetch(ctx, key)
		if err == nil || !IsRetryable(err) || attempt >= len(r.Backoff) {
			return data, err
		}
		r.logger.Debugf("Reading %s %s failed, retrying: %v", r.Noun, r.Describe(key), err)
		if sleepErr := r.Sleep(ctx, r.Backoff[attempt]); sleepErr != nil {
			return nil, err
		}
	}
}

// ConnectionError is a failure of the connection to the secret store, such as
// a refused connection or an unavailable server, as opposed to a failure of
// one request.
type ConnectionError struct {
	Err error
	// Cause describes the failure without the request.
	Cause string
	// Retry is true if retrying may fix the failure.
	Retry bool
}

func (e *ConnectionError) Error() string { return e.Err.Error() }
func (e *ConnectionError) Unwrap() error { return e.Err }

// IsConnectionError reports whether err is a ConnectionError.
func IsConnectionError(err error) bool {
	var c *ConnectionError
	return errors.As(err, &c)
}

// IsRetryable reports whether err is a ConnectionError worth retrying.
func IsRetryable(err error) bool {
	var c *ConnectionError
	return errors.As(err, &c) && c.Retry
}

// ConnectionCause returns the cause of a ConnectionError, or "".
func ConnectionCause(err error) string {
	var c *ConnectionError
	if errors.As(err, &c) {
		return c.Cause
	}
	return ""
}

// URLConnectionError converts a transport error of a request into a
// ConnectionError, without the request URL: the location is already in msg,
// and other monitors must not see it. ok is false if err is not a *url.Error.
func URLConnectionError(name, msg string, err error) (connErr error, ok bool) {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return nil, false
	}
	return &ConnectionError{
		Err:   fmt.Errorf("%s: %s failed: %w", name, msg, urlErr.Err),
		Cause: urlErr.Err.Error(),
		// Retrying doesn't fix an untrusted certificate.
		Retry: !IsCertificateError(urlErr.Err),
	}, true
}

// IsCertificateError reports whether err is a TLS certificate verification
// failure.
func IsCertificateError(err error) bool {
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

// RedactedAddress returns address without a password.
func RedactedAddress(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return "<invalid address>"
	}
	return u.Redacted()
}

// StringValue converts a secret field value into the string put in the config.
func StringValue(name string, v any) (string, error) {
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
			return "", fmt.Errorf("%s: cannot convert field value of type %T", name, val)
		}
		return string(b), nil
	}
}

// ClientTLSConfig returns the TLS config of the connection to the secret
// store at address. The libbeat defaults, used for the unset ssl settings,
// require TLS 1.2.
func ClientTLSConfig(ssl *tlscommon.Config, address string, logger *logp.Logger) (*tls.Config, error) {
	if !ssl.IsEnabled() {
		return &tls.Config{MinVersion: tls.VersionTLS12}, nil
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(ssl, logger)
	if err != nil {
		return nil, fmt.Errorf("invalid 'ssl': %w", err)
	}
	// The address was validated, it parses.
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("invalid 'address': %w", err)
	}
	return tlsConfig.BuildModuleClientConfig(u.Hostname(), tlscommon.WithLogger(logger)), nil
}

// SleepContext waits for d, or until ctx is done.
func SleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

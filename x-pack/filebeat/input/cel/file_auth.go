// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// fileAuthTransport is an http.RoundTripper that injects authentication
// credentials from a file into HTTP request headers. It periodically reloads
// the file to support credential rotation without service restart.
type fileAuthTransport struct {
	header  string            // The HTTP header name to set (e.g., "Authorization")
	prefix  string            // Optional prefix to prepend to the file content (e.g., "Bearer ")
	path    string            // Path to the file containing the authentication value
	refresh time.Duration     // How often to reload the file
	base    http.RoundTripper // The underlying transport to use for requests
	clock   func() time.Time  // Clock function for testing

	mu       sync.Mutex // Protects the fields below
	value    string     // The current authentication value (prefix + file content)
	loadedAt time.Time  // When the value was last loaded
}

func newFileAuthTransport(cfg *fileAuthConfig, base http.RoundTripper) (*fileAuthTransport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("file auth: missing configuration")
	}
	if base == nil {
		base = http.DefaultTransport
	}
	transport := &fileAuthTransport{
		header:  cfg.headerName(),
		prefix:  cfg.Prefix,
		path:    cfg.Path,
		refresh: cfg.refreshInterval(),
		base:    base,
		clock:   time.Now,
	}
	if err := transport.initialise(); err != nil {
		return nil, err
	}
	return transport, nil
}

func (t *fileAuthTransport) initialise() error {
	return t.refreshValue(t.clock())
}

func (t *fileAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	value, err := t.currentValue()
	if err != nil {
		return nil, err
	}
	// Clone the request so we comply with http.RoundTripper requirements and avoid
	// mutating caller-visible fields such as Header.
	req := r.Clone(r.Context())
	if req.Header == nil {
		req.Header = make(http.Header)
	} else {
		req.Header = req.Header.Clone()
	}
	req.Header.Set(t.header, value)
	return t.base.RoundTrip(req)
}

func (t *fileAuthTransport) currentValue() (string, error) {
	now := t.clock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if now.Sub(t.loadedAt) >= t.refresh {
		if err := t.refreshValue(now); err != nil {
			return "", err
		}
	}
	return t.value, nil
}

// refreshValue reloads the authentication value from disk.
// It must be called with t.mu held.
func (t *fileAuthTransport) refreshValue(now time.Time) error {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return fmt.Errorf("file auth: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return fmt.Errorf("file auth: file %q is empty", t.path)
	}
	t.value = t.prefix + value
	t.loadedAt = now
	return nil
}

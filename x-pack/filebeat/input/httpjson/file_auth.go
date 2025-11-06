// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type fileAuthTransport struct {
	header  string
	prefix  string
	path    string
	refresh time.Duration
	base    http.RoundTripper
	clock   func() time.Time

	mu       sync.Mutex
	value    string
	loadedAt time.Time
}

func newFileAuthTransport(cfg *fileAuthConfig, base http.RoundTripper) (*fileAuthTransport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("file auth: missing configuration")
	}
	if base == nil {
		base = http.DefaultTransport
	}
	tr := &fileAuthTransport{
		header:  cfg.headerName(),
		prefix:  cfg.Prefix,
		path:    cfg.Path,
		refresh: cfg.refreshInterval(),
		base:    base,
		clock:   time.Now,
	}
	if err := tr.initialise(); err != nil {
		return nil, err
	}
	return tr, nil
}

func (t *fileAuthTransport) initialise() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.refreshLocked(t.clock())
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
	if t.value == "" || t.refresh <= 0 || now.Sub(t.loadedAt) >= t.refresh {
		if err := t.refreshLocked(now); err != nil {
			return "", err
		}
	}
	return t.value, nil
}

func (t *fileAuthTransport) refreshLocked(now time.Time) error {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return fmt.Errorf("file auth: failed reading %q: %w", t.path, err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return fmt.Errorf("file auth: file %q is empty", t.path)
	}
	t.value = t.prefix + value
	t.loadedAt = now
	return nil
}

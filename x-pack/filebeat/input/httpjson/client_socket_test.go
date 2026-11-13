// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package httpjson

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestOAuth2ClientKeepsUnixSocketDialer(t *testing.T) {
	// The socket path must stay short enough for the platform's sun_path limit.
	dir, err := os.MkdirTemp("", "hj")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s")

	l, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socket)
	require.NoError(t, err)
	srv := &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "socket-token", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		assert.Equal(t, "Bearer socket-token", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"ok":true}`)
	})}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { srv.Close() })

	cfg := conf.MustNewConfigFrom(map[string]any{ //nolint:gosec // G101: not a credential, an inert test fixture
		"request.url":                "http+unix://" + socket,
		"request.retry.max_attempts": 1,
		"auth.oauth2.client.id":      "client",
		"auth.oauth2.client.secret":  "secret",
		"auth.oauth2.token_url":      "http://token.invalid/token",
	})
	c := defaultConfig()
	require.NoError(t, cfg.Unpack(&c))

	client, err := newHTTPClient(t.Context(), c.Auth, c.Request, noopReporter{}, logptest.NewTestingLogger(t, ""), nil, nil)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, c.Request.URL.String(), nil)
	require.NoError(t, err)
	resp, err := client.client.Do(req)
	require.NoError(t, err, "the API request must be sent over the configured socket")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

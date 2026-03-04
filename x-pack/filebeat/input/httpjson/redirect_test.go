// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/cel"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRedirect_EndToEnd(t *testing.T) {
	log := logp.NewNopLogger()
	store := newTestStore()

	httpjsonPlugin := v2.Plugin{
		Name:      "httpjson",
		Stability: feature.Stable,
		Manager:   NewInputManager(log, store),
	}
	celPlugin := cel.Plugin(log, store)

	loader, err := v2.NewLoader(log, []v2.Plugin{httpjsonPlugin, celPlugin}, "type", "")
	require.NoError(t, err)

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"type":        "httpjson",
		"interval":    "60s",
		"run_as_cel":  true,
		"request.url": "https://api.example.com/events",
		"cel.program": `{"events":[{"message":"Hello, World!"}]}`,
		"cel.state":   map[string]interface{}{},
	})

	input, err := loader.Configure(cfg)
	require.NoError(t, err)
	require.NotNil(t, input)
}

func TestRedirect_NoRedirectWhenFlagAbsent(t *testing.T) {
	log := logp.NewNopLogger()
	store := newTestStore()

	httpjsonPlugin := v2.Plugin{
		Name:      "httpjson",
		Stability: feature.Stable,
		Manager:   NewInputManager(log, store),
	}

	loader, err := v2.NewLoader(log, []v2.Plugin{httpjsonPlugin}, "type", "")
	require.NoError(t, err)

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"type":        "httpjson",
		"interval":    "60s",
		"request.url": "https://api.example.com/events",
	})

	input, err := loader.Configure(cfg)
	require.NoError(t, err)
	require.NotNil(t, input)
}

func TestRedirect_ErrorWithoutProgram(t *testing.T) {
	log := logp.NewNopLogger()
	store := newTestStore()

	httpjsonPlugin := v2.Plugin{
		Name:      "httpjson",
		Stability: feature.Stable,
		Manager:   NewInputManager(log, store),
	}

	loader, err := v2.NewLoader(log, []v2.Plugin{httpjsonPlugin}, "type", "")
	require.NoError(t, err)

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"type":        "httpjson",
		"interval":    "60s",
		"request.url": "https://api.example.com/events",
		"run_as_cel":  true,
	})

	_, err = loader.Configure(cfg)
	require.Error(t, err)
}

func TestConvertHttpjsonToCel(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `bytes(resp.Body).decode_json()`,
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		typ, err := out.String("type", -1)
		require.NoError(t, err)
		require.Equal(t, "cel", typ)

		url, err := out.String("resource.url", -1)
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com/events", url)

		interval, err := out.String("interval", -1)
		require.NoError(t, err)
		require.Equal(t, "60s", interval)

		program, err := out.String("program", -1)
		require.NoError(t, err)
		require.Equal(t, `bytes(resp.Body).decode_json()`, program)
	})

	t.Run("passthrough_id", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"id":          "my-input",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		id, err := out.String("id", -1)
		require.NoError(t, err)
		require.Equal(t, "my-input", id)
	})

	t.Run("auth_block", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":                "httpjson",
			"interval":            "60s",
			"request.url":         "https://api.example.com/events",
			"cel.program":         `true`,
			"auth.basic.user":     "testuser",
			"auth.basic.password": "testpass",
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		user, err := out.String("auth.basic.user", -1)
		require.NoError(t, err)
		require.Equal(t, "testuser", user)

		pass, err := out.String("auth.basic.password", -1)
		require.NoError(t, err)
		require.Equal(t, "testpass", pass)
	})

	t.Run("retry_block", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":                       "httpjson",
			"interval":                   "60s",
			"request.url":                "https://api.example.com/events",
			"cel.program":                `true`,
			"request.retry.max_attempts": 3,
			"request.retry.wait_min":     "1s",
			"request.retry.wait_max":     "30s",
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("resource.retry", -1)
		require.NoError(t, err)
		require.True(t, has)

		sub, err := out.Child("resource.retry", -1)
		require.NoError(t, err)

		v, err := sub.Int("max_attempts", -1)
		require.NoError(t, err)
		require.Equal(t, int64(3), v)
	})

	t.Run("redirect_block", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":                             "httpjson",
			"interval":                         "60s",
			"request.url":                      "https://api.example.com/events",
			"cel.program":                      `true`,
			"request.redirect.forward_headers": true,
			"request.redirect.max_redirects":   5,
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("resource.redirect", -1)
		require.NoError(t, err)
		require.True(t, has)

		sub, err := out.Child("resource.redirect", -1)
		require.NoError(t, err)

		fwd, err := sub.Bool("forward_headers", -1)
		require.NoError(t, err)
		require.True(t, fwd)
	})

	t.Run("keep_alive_block", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"request.keep_alive.max_idle_connections":          10,
			"request.keep_alive.max_idle_connections_per_host": 2,
			"request.keep_alive.idle_connection_timeout":       "30s",
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("resource.keep_alive", -1)
		require.NoError(t, err)
		require.True(t, has)

		sub, err := out.Child("resource.keep_alive", -1)
		require.NoError(t, err)

		v, err := sub.Int("max_idle_connections", -1)
		require.NoError(t, err)
		require.Equal(t, int64(10), v)
	})

	t.Run("tracer_block", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":                    "httpjson",
			"interval":                "60s",
			"request.url":             "https://api.example.com/events",
			"cel.program":             `true`,
			"request.tracer.filename": "/tmp/trace.ndjson",
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("resource.tracer", -1)
		require.NoError(t, err)
		require.True(t, has)

		v, err := out.String("resource.tracer.filename", -1)
		require.NoError(t, err)
		require.Equal(t, "/tmp/trace.ndjson", v)
	})

	t.Run("transport_ssl", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":                          "httpjson",
			"interval":                      "60s",
			"request.url":                   "https://api.example.com/events",
			"cel.program":                   `true`,
			"request.ssl.verification_mode": "none",
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("resource.ssl", -1)
		require.NoError(t, err)
		require.True(t, has)

		v, err := out.String("resource.ssl.verification_mode", -1)
		require.NoError(t, err)
		require.Equal(t, "none", v)
	})

	t.Run("transport_timeout", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":            "httpjson",
			"interval":        "60s",
			"request.url":     "https://api.example.com/events",
			"cel.program":     `true`,
			"request.timeout": "45s",
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		v, err := out.String("resource.timeout", -1)
		require.NoError(t, err)
		require.Equal(t, "45s", v)
	})

	t.Run("transport_proxy", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":                  "httpjson",
			"interval":              "60s",
			"request.url":           "https://api.example.com/events",
			"cel.program":           `true`,
			"request.proxy_url":     "http://proxy.example.com:8080",
			"request.proxy_disable": true,
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		v, err := out.String("resource.proxy_url", -1)
		require.NoError(t, err)
		require.Equal(t, "http://proxy.example.com:8080", v)

		b, err := out.Bool("resource.proxy_disable", -1)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("cel_max_executions", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":               "httpjson",
			"interval":           "60s",
			"request.url":        "https://api.example.com/events",
			"cel.program":        `true`,
			"cel.max_executions": 500,
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		v, err := out.Int("max_executions", -1)
		require.NoError(t, err)
		require.Equal(t, int64(500), v)
	})

	t.Run("cel_state", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.state":   map[string]interface{}{"cursor": map[string]interface{}{"ts": "2024-01-01T00:00:00Z"}},
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("state", -1)
		require.NoError(t, err)
		require.True(t, has)

		v, err := out.String("state.cursor.ts", -1)
		require.NoError(t, err)
		require.Equal(t, "2024-01-01T00:00:00Z", v)
	})

	t.Run("cel_regexp", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.regexp":  map[string]interface{}{"link_next": `<([^>]+)>;\s*rel="next"`},
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		v, err := out.String("regexp.link_next", -1)
		require.NoError(t, err)
		require.Equal(t, `<([^>]+)>;\s*rel="next"`, v)
	})

	t.Run("cel_xsd", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.xsd":     map[string]interface{}{"evt": "<xs:schema/>"},
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		v, err := out.String("xsd.evt", -1)
		require.NoError(t, err)
		require.Equal(t, "<xs:schema/>", v)
	})

	t.Run("cel_redact", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.redact":  map[string]interface{}{"fields": []string{"auth_token"}, "delete": true},
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("redact", -1)
		require.NoError(t, err)
		require.True(t, has)

		b, err := out.Bool("redact.delete", -1)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("httpjson_only_fields_excluded", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":           "httpjson",
			"interval":       "60s",
			"request.url":    "https://api.example.com/events",
			"request.method": "POST",
			"cel.program":    `true`,
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		has, err := out.Has("request", -1)
		require.NoError(t, err)
		require.False(t, has)

		has, err = out.Has("cel", -1)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("realistic_full_config", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"id":          "okta-system-log",
			"interval":    "120s",
			"request.url": "https://dev-123456.okta.com/api/v1/logs",

			"auth.oauth2.client.id":     "0oa1234567890abcdef",
			"auth.oauth2.client.secret": "client-secret-value",
			"auth.oauth2.token_url":     "https://dev-123456.okta.com/oauth2/v1/token",
			"auth.oauth2.scopes":        []string{"okta.logs.read"},

			"request.retry.max_attempts": 5,
			"request.retry.wait_min":     "2s",
			"request.retry.wait_max":     "60s",

			"request.redirect.forward_headers": true,
			"request.redirect.max_redirects":   3,

			"request.keep_alive.max_idle_connections": 5,

			"request.tracer.filename": "/tmp/okta-trace.ndjson",

			"request.ssl.verification_mode": "full",
			"request.timeout":               "30s",
			"request.proxy_url":             "http://corp-proxy:3128",

			"cel.program": `
state.url.with({
    "Header": {"Accept": ["application/json"]},
}).as(req, request("GET", req).as(resp,
    bytes(resp.Body).decode_json().as(body, {
        "events": body.map(e, {"message": e.encode_json()}),
        "cursor": {"after": body[body.size()-1].published},
    })
))`,
			"cel.max_executions": 100,
			"cel.state":          map[string]interface{}{"cursor": map[string]interface{}{"after": ""}},
			"cel.regexp":         map[string]interface{}{"link": `<([^>]+)>;\s*rel="next"`},
			"cel.redact":         map[string]interface{}{"fields": []string{"auth.oauth2.client.secret"}},
		})

		out, err := convertHttpjsonToCel(cfg)
		require.NoError(t, err)

		typ, err := out.String("type", -1)
		require.NoError(t, err)
		require.Equal(t, "cel", typ)

		id, err := out.String("id", -1)
		require.NoError(t, err)
		require.Equal(t, "okta-system-log", id)

		interval, err := out.String("interval", -1)
		require.NoError(t, err)
		require.Equal(t, "120s", interval)

		url, err := out.String("resource.url", -1)
		require.NoError(t, err)
		require.Equal(t, "https://dev-123456.okta.com/api/v1/logs", url)

		// Auth transferred
		clientID, err := out.String("auth.oauth2.client.id", -1)
		require.NoError(t, err)
		require.Equal(t, "0oa1234567890abcdef", clientID)

		// Retry transferred
		retrySub, err := out.Child("resource.retry", -1)
		require.NoError(t, err)
		maxAttempts, err := retrySub.Int("max_attempts", -1)
		require.NoError(t, err)
		require.Equal(t, int64(5), maxAttempts)

		// Redirect transferred
		has, err := out.Has("resource.redirect", -1)
		require.NoError(t, err)
		require.True(t, has)

		// Keep alive transferred
		has, err = out.Has("resource.keep_alive", -1)
		require.NoError(t, err)
		require.True(t, has)

		// Tracer transferred
		tracerFile, err := out.String("resource.tracer.filename", -1)
		require.NoError(t, err)
		require.Equal(t, "/tmp/okta-trace.ndjson", tracerFile)

		// Transport transferred
		sslMode, err := out.String("resource.ssl.verification_mode", -1)
		require.NoError(t, err)
		require.Equal(t, "full", sslMode)

		timeout, err := out.String("resource.timeout", -1)
		require.NoError(t, err)
		require.Equal(t, "30s", timeout)

		proxyURL, err := out.String("resource.proxy_url", -1)
		require.NoError(t, err)
		require.Equal(t, "http://corp-proxy:3128", proxyURL)

		// CEL fields transferred
		program, err := out.String("program", -1)
		require.NoError(t, err)
		require.Contains(t, program, "state.url.with")

		maxExec, err := out.Int("max_executions", -1)
		require.NoError(t, err)
		require.Equal(t, int64(100), maxExec)

		has, err = out.Has("state", -1)
		require.NoError(t, err)
		require.True(t, has)

		has, err = out.Has("regexp", -1)
		require.NoError(t, err)
		require.True(t, has)

		has, err = out.Has("redact", -1)
		require.NoError(t, err)
		require.True(t, has)

		// httpjson-only fields absent
		has, err = out.Has("request", -1)
		require.NoError(t, err)
		require.False(t, has)

		has, err = out.Has("cel", -1)
		require.NoError(t, err)
		require.False(t, has)
	})
}

var _ statestore.States = (*testStore)(nil)

type testStore struct {
	registry *statestore.Registry
}

func newTestStore() *testStore {
	return &testStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testStore) Close()                                     { s.registry.Close() }
func (s *testStore) StoreFor(string) (*statestore.Store, error) { return s.registry.Get("filebeat") }
func (s *testStore) CleanupInterval() time.Duration             { return 0 }

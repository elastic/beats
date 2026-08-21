// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
<<<<<<< HEAD
=======
	"encoding/json"
	"fmt"
	"sync"
>>>>>>> eb40647a7 (x-pack/filebeat/input/httpjson: call SetID before cursor migration store read (#52759))
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
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
	require.Equal(t, "cel", input.Name())
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

func TestMigrateCursor(t *testing.T) {
	t.Run("injects_stored_cursor", func(t *testing.T) {
		store := newTestStore()
		s, err := store.StoreFor("httpjson")
		require.NoError(t, err)
		err = s.Set("httpjson::my-input::https://api.example.com/events", map[string]interface{}{
			"ttl":     0,
			"updated": time.Now(),
			"cursor":  map[string]interface{}{"timestamp": "2025-06-15T10:30:00Z"},
		})
		require.NoError(t, err)
		s.Close()

		mgr := NewInputManager(logp.NewNopLogger(), store)
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"id":          "my-input",
			"interval":    "60s",
			"run_as_cel":  true,
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.state":   map[string]interface{}{"cursor": map[string]interface{}{"timestamp": ""}},
		})

		_, newCfg, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		v, err := newCfg.String("state.cursor.timestamp", -1)
		require.NoError(t, err)
		require.Equal(t, "2025-06-15T10:30:00Z", v)
	})

	t.Run("no_entry", func(t *testing.T) {
		store := newTestStore()
		mgr := NewInputManager(logp.NewNopLogger(), store)
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"id":          "my-input",
			"interval":    "60s",
			"run_as_cel":  true,
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.state":   map[string]interface{}{"cursor": map[string]interface{}{"timestamp": "default"}},
		})

		_, newCfg, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		v, err := newCfg.String("state.cursor.timestamp", -1)
		require.NoError(t, err)
		require.Equal(t, "default", v)
	})

	t.Run("no_id", func(t *testing.T) {
		store := newTestStore()
		s, err := store.StoreFor("httpjson")
		require.NoError(t, err)
		err = s.Set("httpjson::https://api.example.com/events", map[string]interface{}{
			"ttl":     0,
			"updated": time.Now(),
			"cursor":  map[string]interface{}{"page": "42"},
		})
		require.NoError(t, err)
		s.Close()

		mgr := NewInputManager(logp.NewNopLogger(), store)
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"interval":    "60s",
			"run_as_cel":  true,
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.state":   map[string]interface{}{},
		})

		_, newCfg, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		v, err := newCfg.String("state.cursor.page", -1)
		require.NoError(t, err)
		require.Equal(t, "42", v)
	})

	t.Run("no_state_in_config", func(t *testing.T) {
		store := newTestStore()
		s, err := store.StoreFor("httpjson")
		require.NoError(t, err)
		err = s.Set("httpjson::no-state::https://api.example.com/events", map[string]interface{}{
			"ttl":     0,
			"updated": time.Now(),
			"cursor":  map[string]interface{}{"offset": "100"},
		})
		require.NoError(t, err)
		s.Close()

		mgr := NewInputManager(logp.NewNopLogger(), store)
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"id":          "no-state",
			"interval":    "60s",
			"run_as_cel":  true,
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
		})

		_, newCfg, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		v, err := newCfg.String("state.cursor.offset", -1)
		require.NoError(t, err)
		require.Equal(t, "100", v)
	})

	t.Run("idempotent", func(t *testing.T) {
		store := newTestStore()
		s, err := store.StoreFor("httpjson")
		require.NoError(t, err)
		err = s.Set("httpjson::idem::https://api.example.com/events", map[string]interface{}{
			"ttl":     0,
			"updated": time.Now(),
			"cursor":  map[string]interface{}{"seq": "99"},
		})
		require.NoError(t, err)
		s.Close()

		mgr := NewInputManager(logp.NewNopLogger(), store)
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type":        "httpjson",
			"id":          "idem",
			"interval":    "60s",
			"run_as_cel":  true,
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.state":   map[string]interface{}{},
		})

		_, first, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		_, second, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		v1, err := first.String("state.cursor.seq", -1)
		require.NoError(t, err)
		v2, err := second.String("state.cursor.seq", -1)
		require.NoError(t, err)
		require.Equal(t, v1, v2)
	})

	t.Run("set_id_scoped_store", func(t *testing.T) {
		// Simulate an Elasticsearch-backed store where SetID routes to a different
		// index. The cursor was previously written under the "my-input" namespace;
		// migrateCursor must call SetID so it reads from the right partition.
		b := newNamespacedMemBackend()
		b.partitions["my-input"] = map[string]any{
			"httpjson::my-input::https://api.example.com/events": map[string]any{
				"ttl":     0,
				"updated": time.Now(),
				"cursor":  map[string]any{"timestamp": "2025-06-15T10:30:00Z"},
			},
		}

		store := newNamespacedTestStore(b)
		mgr := NewInputManager(logp.NewNopLogger(), store)
		cfg := conf.MustNewConfigFrom(map[string]any{
			"type":        "httpjson",
			"id":          "my-input",
			"interval":    "60s",
			"run_as_cel":  true,
			"request.url": "https://api.example.com/events",
			"cel.program": `true`,
			"cel.state":   map[string]any{"cursor": map[string]any{"timestamp": "default_timestamp_should_not_be_in_redirected_cursor"}},
		})

		_, newCfg, err := mgr.Redirect(cfg)
		require.NoError(t, err)

		v, err := newCfg.String("state.cursor.timestamp", -1)
		require.NoError(t, err)
		require.Equal(t, "2025-06-15T10:30:00Z", v)
	})
}

func TestCursorKey(t *testing.T) {
	require.Equal(t, "httpjson::my-id::https://example.com", cursorKey("httpjson", "my-id", "https://example.com"))
	require.Equal(t, "cel::https://example.com", cursorKey("cel", "", "https://example.com"))
	require.Equal(t, "httpjson::https://example.com/path", cursorKey("httpjson", "", "https://example.com/path"))
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

// namespacedMemBackend is a backend.Registry whose stores partition keys by
// namespace, simulating the Elasticsearch state store where SetID routes
// operations to a different index.
type namespacedMemBackend struct {
	mu         sync.Mutex
	partitions map[string]map[string]any
}

// newNamespacedMemBackend returns an empty namespacedMemBackend.
func newNamespacedMemBackend() *namespacedMemBackend {
	return &namespacedMemBackend{partitions: make(map[string]map[string]any)}
}

// Access returns a new namespacedMemStore sharing the back end's partition map.
// The store family name is ignored; it selects which backing store to open,
// but we are a singleton in this setting.
func (b *namespacedMemBackend) Access(string) (backend.Store, error) {
	return &namespacedMemStore{b: b}, nil
}

// Close is a no-op.
func (b *namespacedMemBackend) Close() error { return nil }

// namespacedMemStore is a backend.Store that routes all key operations to the
// partition currently selected by SetID. The default namespace (before any
// SetID call) is the empty string. ns is protected by b.mu.
type namespacedMemStore struct {
	b  *namespacedMemBackend
	ns string
}

// SetID sets the active namespace; all subsequent key operations target that
// partition, mirroring how the Elasticsearch back end switches index on SetID.
func (s *namespacedMemStore) SetID(id string) {
	s.b.mu.Lock()
	s.ns = id
	s.b.mu.Unlock()
}

// Has reports whether key exists in the current namespace's partition.
func (s *namespacedMemStore) Has(key string) (bool, error) {
	s.b.mu.Lock()
	_, ok := s.b.partitions[s.ns][key]
	s.b.mu.Unlock()
	return ok, nil
}

// Get decodes the value stored under key in the current namespace into value
// via a JSON round-trip.
func (s *namespacedMemStore) Get(key string, val any) error {
	s.b.mu.Lock()
	v, ok := s.b.partitions[s.ns][key]
	s.b.mu.Unlock()
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, val)
}

// Set stores value under key in the current namespace's partition, creating
// the partition if it does not yet exist.
func (s *namespacedMemStore) Set(key string, val any) error {
	s.b.mu.Lock()
	if s.b.partitions[s.ns] == nil {
		s.b.partitions[s.ns] = make(map[string]any)
	}
	s.b.partitions[s.ns][key] = val
	s.b.mu.Unlock()
	return nil
}

// Remove deletes key from the current namespace's partition.
func (s *namespacedMemStore) Remove(key string) error {
	s.b.mu.Lock()
	delete(s.b.partitions[s.ns], key)
	s.b.mu.Unlock()
	return nil
}

// Each snapshots the current namespace's partition under the lock, then
// iterates over the snapshot without holding the lock, calling fn for each
// key-value pair. Iteration stops when fn returns false or a non-nil error.
func (s *namespacedMemStore) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.b.mu.Lock()
	p := s.b.partitions[s.ns]
	keys := make([]string, 0, len(p))
	vals := make([]any, 0, len(p))
	for k, v := range p {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	s.b.mu.Unlock()
	for i, k := range keys {
		cont, err := fn(k, jsonValueDecoder{vals[i]})
		if err != nil || !cont {
			return err
		}
	}
	return nil
}

// Close is a no-op; the store holds no resources beyond the shared backend.
func (s *namespacedMemStore) Close() error { return nil }

// jsonValueDecoder implements backend.ValueDecoder via a JSON round-trip,
// allowing arbitrary in-memory values to be decoded into typed targets.
type jsonValueDecoder struct{ v any }

// Decode marshals the held value to JSON and unmarshals it into to.
func (d jsonValueDecoder) Decode(dst any) error {
	buf, err := json.Marshal(d.v)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, dst)
}

var _ statestore.States = (*namespacedTestStore)(nil)

// namespacedTestStore implements statestore.States backed by a
// namespacedMemBackend, for use in tests that require SetID-aware storage.
type namespacedTestStore struct {
	registry *statestore.Registry
}

// newNamespacedTestStore returns a namespacedTestStore backed by b.
func newNamespacedTestStore(b *namespacedMemBackend) *namespacedTestStore {
	return &namespacedTestStore{registry: statestore.NewRegistry(b)}
}

// StoreFor returns the shared "filebeat" store from the underlying registry.
func (s *namespacedTestStore) StoreFor(_ string) (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *namespacedTestStore) StoreKey() string               { return fmt.Sprintf("namespaced:%p", s.registry) }
func (s *namespacedTestStore) CleanupInterval() time.Duration { return 0 }

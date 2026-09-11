// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchclient

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type reportingHost struct {
	component.Host
	mu     sync.Mutex
	events []*componentstatus.Event
}

func (h *reportingHost) Report(e *componentstatus.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, e)
}

func (h *reportingHost) lastStatus() componentstatus.Status {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.events) == 0 {
		return componentstatus.StatusNone
	}
	return h.events[len(h.events)-1].Status()
}

func newTestExtension(esCfg map[string]any, logger *logp.Logger) *elasticsearchClient {
	return &elasticsearchClient{
		cfg:    &Config{ElasticsearchConfig: esCfg},
		logger: logger,
		info: beat.Info{
			Beat:   componentType,
			Logger: logger,
		},
	}
}

func pingOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)
}

func newFakeES(t *testing.T, handle func(http.ResponseWriter, *http.Request) bool) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			pingOK(w)
			return
		}
		if handle != nil && handle(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func startExtension(t *testing.T, ext *elasticsearchClient, host component.Host) {
	t.Helper()
	err := ext.Start(t.Context(), host)
	require.NoError(t, err, "extension Start must succeed")
	// Shutdown during t.Cleanup runs after t.Context() is cancelled, so a live
	// background context is required for Collector lifecycle cleanup.
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })
}

func TestFactoryType(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, "elasticsearchclient", factory.Type().String(), "OTel component type must be elasticsearchclient")
	cfg, ok := factory.CreateDefaultConfig().(*Config)
	require.True(t, ok, "default config must be *Config")
	assert.Nil(t, cfg.ElasticsearchConfig, "default remain-config must be empty")
}

func TestFactoryCreate_InvalidConfigType(t *testing.T) {
	_, err := createExtension(t.Context(), extensiontest.NewNopSettings(Type), struct{}{})
	require.Error(t, err, "createExtension must reject a non-*Config value")
	assert.Contains(t, err.Error(), "elasticsearchclient config", "error must identify the expected config type")
}

func TestExtension_StartRequestShutdown(t *testing.T) {
	const responseBody = `{"hits":{"total":{"value":1},"hits":[{"_source":{"state":{"id":"abc"}}}]}}`
	var gotMethod, gotPath, gotPipeline, gotParam string
	var gotBody []byte

	srv := newFakeES(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost && r.URL.Path == "/synthetics-*/_search" {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotPipeline = r.URL.Query().Get("pipeline")
			gotParam = r.URL.Query().Get("size")
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, responseBody)
			return true
		}
		return false
	})

	host := &reportingHost{Host: componenttest.NewNopHost()}
	ext := newTestExtension(map[string]any{
		"hosts":    []string{srv.URL},
		"username": "elastic",
		"password": "changeme",
	}, logp.NewNopLogger())
	startExtension(t, ext, host)
	assert.Equal(t, componentstatus.StatusOK, host.lastStatus(), "successful Start must report StatusOK")

	status, body, err := ext.Request("POST", "/synthetics-*/_search", "my-pipeline", map[string]string{"size": "1"}, map[string]any{
		"query": map[string]any{"match_all": map[string]any{}},
	})
	require.NoError(t, err, "forwarded request must succeed")
	assert.Equal(t, http.StatusOK, status, "forwarded request must return HTTP 200")
	assert.JSONEq(t, responseBody, string(body), "response body must be forwarded intact")
	assert.Equal(t, http.MethodPost, gotMethod, "request method must be forwarded")
	assert.Equal(t, "/synthetics-*/_search", gotPath, "request path must be forwarded")
	assert.Equal(t, "my-pipeline", gotPipeline, "pipeline query parameter must be forwarded")
	assert.Equal(t, "1", gotParam, "request params must be forwarded")
	assert.Contains(t, string(gotBody), `"match_all"`, "request body must be forwarded")
}

func TestExtension_BasicAuth(t *testing.T) {
	var sawUser, sawPass string
	var sawBasic bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		sawBasic = ok
		sawUser, sawPass = u, p
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			pingOK(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)

	ext := newTestExtension(map[string]any{
		"hosts":    []string{srv.URL},
		"username": "beats-user",
		"password": "beats-pass",
	}, logp.NewNopLogger())
	startExtension(t, ext, componenttest.NewNopHost())

	status, _, err := ext.Request(http.MethodGet, "/_cluster/health", "", nil, nil)
	require.NoError(t, err, "authenticated GET must succeed")
	assert.Equal(t, http.StatusOK, status, "authenticated GET must return HTTP 200")
	assert.True(t, sawBasic, "client must send HTTP basic auth")
	assert.Equal(t, "beats-user", sawUser, "basic auth username must match config")
	assert.Equal(t, "beats-pass", sawPass, "basic auth password must match config")
}

func TestExtension_APIKey(t *testing.T) {
	const rawKey = "id:secret-api-key"
	wantHeader := "ApiKey " + base64.StdEncoding.EncodeToString([]byte(rawKey))
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			pingOK(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)

	ext := newTestExtension(map[string]any{
		"hosts":   []string{srv.URL},
		"api_key": rawKey,
	}, logp.NewNopLogger())
	startExtension(t, ext, componenttest.NewNopHost())

	status, _, err := ext.Request(http.MethodGet, "/_cluster/health", "", nil, nil)
	require.NoError(t, err, "API-key authenticated GET must succeed")
	assert.Equal(t, http.StatusOK, status, "API-key authenticated GET must return HTTP 200")
	assert.Equal(t, wantHeader, gotAuth, "Authorization header must be the base64-encoded API key")
}

func TestExtension_RequestBeforeStart(t *testing.T) {
	ext := newTestExtension(map[string]any{"hosts": []string{"http://127.0.0.1:1"}}, logp.NewNopLogger())

	status, body, err := ext.Request(http.MethodGet, "/", "", nil, nil)
	assert.ErrorIs(t, err, ErrNotStarted, "Request before Start must return ErrNotStarted")
	assert.Equal(t, 0, status, "Request before Start must not invent an HTTP status")
	assert.Nil(t, body, "Request before Start must not return a body")
}

func TestExtension_RequestAfterShutdown(t *testing.T) {
	srv := newFakeES(t, nil)
	ext := newTestExtension(map[string]any{
		"hosts":    []string{srv.URL},
		"username": "elastic",
		"password": "changeme",
	}, logp.NewNopLogger())
	require.NoError(t, ext.Start(t.Context(), componenttest.NewNopHost()), "Start must succeed")
	require.NoError(t, ext.Shutdown(t.Context()), "Shutdown must succeed")

	status, body, err := ext.Request(http.MethodGet, "/", "", nil, nil)
	assert.ErrorIs(t, err, ErrShutdown, "Request after Shutdown must return ErrShutdown")
	assert.Equal(t, 0, status, "Request after Shutdown must not invent an HTTP status")
	assert.Nil(t, body, "Request after Shutdown must not return a body")
}

func TestExtension_ShutdownIdempotent(t *testing.T) {
	srv := newFakeES(t, nil)
	ext := newTestExtension(map[string]any{
		"hosts":    []string{srv.URL},
		"username": "elastic",
		"password": "changeme",
	}, logp.NewNopLogger())
	require.NoError(t, ext.Start(t.Context(), componenttest.NewNopHost()), "Start must succeed")
	require.NoError(t, ext.Shutdown(t.Context()), "first Shutdown must succeed")
	require.NoError(t, ext.Shutdown(t.Context()), "second Shutdown must be idempotent")
}

func TestExtension_ShutdownCancelsInFlightRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCanceled := make(chan struct{})
	srv := newFakeES(t, func(_ http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/_cluster/health" {
			return false
		}
		close(requestStarted)
		<-r.Context().Done()
		close(requestCanceled)
		return true
	})

	ext := newTestExtension(map[string]any{
		"hosts":   []string{srv.URL},
		"timeout": "1m",
	}, logp.NewNopLogger())
	require.NoError(t, ext.Start(t.Context(), componenttest.NewNopHost()), "Start must succeed")

	requestDone := make(chan error, 1)
	go func() {
		_, _, err := ext.Request(http.MethodGet, "/_cluster/health", "", nil, nil)
		requestDone <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("request did not reach Elasticsearch")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- ext.Shutdown(t.Context())
	}()

	select {
	case err := <-shutdownDone:
		require.NoError(t, err, "Shutdown must complete after canceling the request")
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown did not cancel the in-flight request")
	}

	select {
	case <-requestCanceled:
	case <-time.After(10 * time.Second):
		t.Fatal("Elasticsearch request context was not canceled")
	}

	select {
	case err := <-requestDone:
		require.Error(t, err, "the canceled request must return an error")
	case <-time.After(10 * time.Second):
		t.Fatal("canceled request did not return")
	}
}

func TestExtension_ShutdownNeverStarted(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	ext := newTestExtension(nil, logger)
	require.NoError(t, ext.Shutdown(t.Context()), "Shutdown before Start must succeed")

	_, _, err := ext.Request(http.MethodGet, "/", "", nil, nil)
	assert.ErrorIs(t, err, ErrShutdown, "Request after Shutdown without Start must return ErrShutdown")
}

func TestExtension_StartFailed_MissingHosts(t *testing.T) {
	host := &reportingHost{Host: componenttest.NewNopHost()}
	ext := newTestExtension(map[string]any{}, logp.NewNopLogger())

	err := ext.Start(t.Context(), host)
	require.Error(t, err, "Start without hosts must fail")
	assert.Contains(t, err.Error(), "failed connecting elasticsearch client", "startup failure must wrap the client error")
	assert.Equal(t, componentstatus.StatusPermanentError, host.lastStatus(), "startup failure must report a permanent error")
}

func TestFactoryCreate(t *testing.T) {
	config := &Config{ElasticsearchConfig: map[string]any{"hosts": []string{"https://example.com"}}}
	ext, err := createExtension(t.Context(), extensiontest.NewNopSettings(Type), config)
	require.NoError(t, err, "factory must create the extension")

	client, ok := ext.(*elasticsearchClient)
	require.True(t, ok, "factory-created extension must be *elasticsearchClient")
	assert.Same(t, config, client.cfg, "factory-created extension must retain its typed configuration")
	assert.Equal(t, componentType, client.info.Beat, "factory-created extension must set its Beat identity")
	assert.NotNil(t, client.info.Logger, "factory-created extension must set its logger")
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/v2/apmtest"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/common/productorigin"
	"github.com/elastic/beats/v7/libbeat/management"
)

type elasticsearchAuthTestHost struct {
	extensions map[component.ID]component.Component
}

func (h elasticsearchAuthTestHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

type nopTestExtension struct{}

func (nopTestExtension) Start(context.Context, component.Host) error { return nil }
func (nopTestExtension) Shutdown(context.Context) error              { return nil }

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type closableRoundTripper struct {
	roundTripperFunc
	closed chan struct{}
	once   sync.Once
}

func (r *closableRoundTripper) CloseIdleConnections() {
	r.once.Do(func() { close(r.closed) })
}

type fakeElasticsearchAuthExtension struct {
	endpoints        []string
	roundTripper     http.RoundTripper
	roundTripErr     error
	baseRoundTripper http.RoundTripper
}

func (f *fakeElasticsearchAuthExtension) Start(context.Context, component.Host) error { return nil }
func (f *fakeElasticsearchAuthExtension) Shutdown(context.Context) error              { return nil }
func (f *fakeElasticsearchAuthExtension) Endpoints() []string                         { return f.endpoints }
func (f *fakeElasticsearchAuthExtension) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	f.baseRoundTripper = base
	return f.roundTripper, f.roundTripErr
}

func TestElasticsearchAuthStartHookErrors(t *testing.T) {
	missingID := component.MustNewIDWithName("elasticsearchauth", "missing")
	wrongTypeID := component.MustNewIDWithName("elasticsearchauth", "wrong-type")
	authID := component.MustNewIDWithName("elasticsearchauth", "auth")

	tests := []struct {
		name      string
		reference string
		host      component.Host
		wantError string
	}{
		{
			name:      "malformed component ID",
			reference: "/missing-type",
			host:      elasticsearchAuthTestHost{},
			wantError: `invalid elasticsearch_auth component ID "/missing-type"`,
		},
		{
			name:      "missing extension",
			reference: missingID.String(),
			host:      elasticsearchAuthTestHost{},
			wantError: `elasticsearch_auth extension "elasticsearchauth/missing" not found`,
		},
		{
			name:      "wrong extension type",
			reference: wrongTypeID.String(),
			host: elasticsearchAuthTestHost{extensions: map[component.ID]component.Component{
				wrongTypeID: nopTestExtension{},
			}},
			wantError: "does not implement HTTP authentication and endpoint discovery",
		},
		{
			name:      "missing captured heartbeat",
			reference: authID.String(),
			host: elasticsearchAuthTestHost{extensions: map[component.ID]component.Component{
				authID: &fakeElasticsearchAuthExtension{endpoints: []string{"http://localhost:9200"}},
			}},
			wantError: `heartbeat instance was not captured for elasticsearch_auth extension "elasticsearchauth/auth"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := elasticsearchAuthStartHook(t.Context(), test.reference, nil, "", func(*esClient) {})(test.host)
			require.Error(t, err, "start hook should reject an invalid Elasticsearch authentication extension")
			assert.Contains(t, err.Error(), test.wantError, "start hook should return the expected resolver error")
		})
	}
}

func TestElasticsearchAuthStartHookEmptyReference(t *testing.T) {
	require.NoError(t, elasticsearchAuthStartHook(t.Context(), "", nil, "", func(*esClient) {})(nil), "empty Elasticsearch auth reference should be a no-op")
}

func TestESClientTransportComposition(t *testing.T) {
	returnedTransport := &closableRoundTripper{
		roundTripperFunc: func(*http.Request) (*http.Response, error) {
			return nil, nil
		},
		closed: make(chan struct{}),
	}
	auth := &fakeElasticsearchAuthExtension{
		endpoints:    []string{"http://example.test"},
		roundTripper: returnedTransport,
	}

	client, err := newESClient(t.Context(), auth, "Heartbeat/test-agent")
	require.NoError(t, err, "client creation")
	require.NotNil(t, auth.baseRoundTripper, "elasticsearchauth should receive a base transport")
	assert.NotSame(t, http.DefaultTransport, auth.baseRoundTripper, "the base transport should be wrapped")
	assert.Same(t, returnedTransport, client.client.Transport, "the transport returned by elasticsearchauth should be installed directly")

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()

	_, spans, apmErrors := apmtest.WithTransaction(func(ctx context.Context) {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/_search", nil)
		require.NoError(t, requestErr, "creating request for the base transport")
		response, roundTripErr := auth.baseRoundTripper.RoundTrip(request)
		require.NoError(t, roundTripErr, "APM-wrapped base transport should complete the request")
		require.NoError(t, response.Body.Close(), "closing the instrumented response body")
	})
	assert.Empty(t, apmErrors, "APM instrumentation should not report errors")
	require.Len(t, spans, 1, "the base transport should create exactly one APM span")
	assert.Equal(t, "db", spans[0].Type, "the base transport should create a database span")
	assert.Equal(t, "elasticsearch", spans[0].Subtype, "the base transport should use Elasticsearch APM instrumentation")
}

func TestESClientRequest(t *testing.T) {
	var receivedRequest *http.Request
	auth := &fakeElasticsearchAuthExtension{
		endpoints: []string{"http://example.test/base"},
		roundTripper: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			receivedRequest = request.Clone(request.Context())
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(`{"hits":{"hits":[]}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client, err := newESClient(t.Context(), auth, "Heartbeat/test-agent")
	require.NoError(t, err, "client creation")
	assert.Equal(t, elasticsearchRequestTimeout, client.client.Timeout, "Heartbeat must own the Elasticsearch request deadline")
	status, body, err := client.Request(http.MethodPost, "/_search?size=1", "pipeline", map[string]string{"routing": "monitor"}, map[string]string{"query": "state"})
	require.NoError(t, err, "request should succeed")
	assert.Equal(t, http.StatusOK, status, "status")
	assert.JSONEq(t, `{"hits":{"hits":[]}}`, string(body), "unexpected body")
	require.NotNil(t, receivedRequest, "returned request should be valid")
	assert.Equal(t, http.MethodPost, receivedRequest.Method, "unexpected method")
	assert.Equal(t, "/base/_search", receivedRequest.URL.Path, "unexpected path")
	assert.Equal(t, "1", receivedRequest.URL.Query().Get("size"), "unexpected size query")
	assert.Equal(t, "pipeline", receivedRequest.URL.Query().Get("pipeline"), "unexpected pipeline query")
	assert.Equal(t, "monitor", receivedRequest.URL.Query().Get("routing"), "unexpected routing query")
	assert.Equal(t, "application/json", receivedRequest.Header.Get("Content-Type"), "unexpected content type")
	assert.Equal(t, "application/json", receivedRequest.Header.Get("Accept"), "unexpected Accept header")
	assert.Equal(t, productorigin.Beats, receivedRequest.Header.Get(productorigin.Header), "unexpected product origin header")
	assert.Equal(t, "Heartbeat/test-agent", receivedRequest.Header.Get("User-Agent"), "unexpected User-Agent")
}

func TestESClientRequestConfiguredHeadersOverrideDefaults(t *testing.T) {
	var receivedRequest *http.Request
	auth := &fakeElasticsearchAuthExtension{
		endpoints: []string{"http://example.test"},
		roundTripper: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			// elasticsearchauth applies configured headers in its transport, after
			// the requester has supplied its defaults.
			request = request.Clone(request.Context())
			request.Header.Set("Accept", "application/vnd.elasticsearch+json;compatible-with=8")
			request.Header.Set(productorigin.Header, "custom-origin")
			receivedRequest = request
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client, err := newESClient(t.Context(), auth, "Heartbeat/test-agent")
	require.NoError(t, err, "client creation")
	_, _, err = client.Request(http.MethodGet, "/", "", nil, nil)
	require.NoError(t, err, "request should succeed")
	require.NotNil(t, receivedRequest, "returned request should be valid")
	assert.Equal(t, "application/vnd.elasticsearch+json;compatible-with=8", receivedRequest.Header.Get("Accept"), "configured Accept header should override the default")
	assert.Equal(t, "custom-origin", receivedRequest.Header.Get(productorigin.Header), "configured product origin should override the default")
}

func TestESClientRequestNon2xx(t *testing.T) {
	var receivedUserAgent string
	auth := &fakeElasticsearchAuthExtension{
		endpoints: []string{"http://example.test"},
		roundTripper: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			receivedUserAgent = request.Header.Get("User-Agent")
			return &http.Response{
				StatusCode: http.StatusTeapot,
				Status:     "418 I'm a teapot",
				Body:       io.NopCloser(bytes.NewBufferString(`Brewing error`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client, err := newESClient(t.Context(), auth, "")
	require.NoError(t, err, "client creation")
	status, body, err := client.Request(http.MethodGet, "/", "", nil, nil)
	require.EqualError(t, err, `418 I'm a teapot: Brewing error`, "unexpected error message")
	assert.Equal(t, http.StatusTeapot, status, "unexpected status code")
	assert.Equal(t, "Brewing error", string(body), "unexpected response body")
	assert.Contains(t, receivedUserAgent, "Heartbeat/", "fallback User-Agent should identify Heartbeat")
}

func TestElasticsearchAuthStartHookInjectsBeforeRun(t *testing.T) {
	previousUnderAgent := management.UnderAgent()
	t.Cleanup(func() {
		management.SetUnderAgent(previousUnderAgent)
	})

	requested := make(chan struct{})
	var once sync.Once
	transport := &closableRoundTripper{
		closed: make(chan struct{}),
		roundTripperFunc: func(*http.Request) (*http.Response, error) {
			once.Do(func() { close(requested) })
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"hits":{"hits":[]}}`)),
				Header:     make(http.Header),
			}, nil
		},
	}
	extensionID := component.MustNewIDWithName("elasticsearchauth", "_agent-component/default")
	extension := &fakeElasticsearchAuthExtension{
		endpoints:    []string{"http://example.test"},
		roundTripper: transport,
	}
	host := elasticsearchAuthTestHost{extensions: map[component.ID]component.Component{
		extensionID: extension,
	}}
	cfg := &Config{
		ElasticsearchAuth: extensionID.String(),
		Beatconfig: map[string]any{
			"heartbeat": map[string]any{
				"monitors": []map[string]any{
					{
						"type":     "tcp",
						"id":       "elasticsearch-auth-hook",
						"schedule": "@every 100ms",
						"hosts":    []string{"localhost:0"},
					},
				},
			},
			"management.otel.enabled": true,
			"path.home":               t.TempDir(),
			"queue.mem.flush.timeout": "0s",
		},
	}
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	settings := receiver.Settings{
		ID: component.NewIDWithName(factory.Type(), "elasticsearch-auth-hook"),
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}

	rec, err := factory.CreateLogs(t.Context(), settings, cfg, consumertest.NewNop())
	require.NoError(t, err, "creating the Heartbeat receiver should succeed")

	require.NoError(t, rec.Start(t.Context(), host), "starting the Heartbeat receiver should succeed")
	select {
	case <-requested:
	case <-time.After(10 * time.Second):
		t.Fatal("Heartbeat did not use the injected Elasticsearch requester")
	}

	shutdownCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	require.NoError(t, rec.Shutdown(shutdownCtx), "shutting down the Heartbeat receiver should succeed")
	select {
	case <-transport.closed:
	case <-time.After(time.Second):
		t.Fatal("Heartbeat did not close idle Elasticsearch connections")
	}
}

func TestElasticsearchAuthShutdownCancelsInFlightRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	requestDone := make(chan error, 1)
	var startOnce sync.Once
	transport := &closableRoundTripper{
		closed: make(chan struct{}),
		roundTripperFunc: func(req *http.Request) (*http.Response, error) {
			startOnce.Do(func() { close(requestStarted) })
			<-req.Context().Done()
			select {
			case requestDone <- req.Context().Err():
			default:
			}
			return nil, req.Context().Err()
		},
	}
	extensionID := component.MustNewIDWithName("elasticsearchauth", "_agent-component/default")
	extension := &fakeElasticsearchAuthExtension{
		endpoints:    []string{"http://example.test"},
		roundTripper: transport,
	}
	host := elasticsearchAuthTestHost{extensions: map[component.ID]component.Component{
		extensionID: extension,
	}}
	cfg := &Config{
		ElasticsearchAuth: extensionID.String(),
		Beatconfig: map[string]any{
			"heartbeat": map[string]any{
				"monitors": []map[string]any{
					{
						"type":     "tcp",
						"id":       "elasticsearch-auth-cancel",
						"schedule": "@every 100ms",
						"hosts":    []string{"localhost:0"},
					},
				},
			},
			"management.otel.enabled": true,
			"path.home":               t.TempDir(),
			"queue.mem.flush.timeout": "0s",
		},
	}
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	settings := receiver.Settings{
		ID: component.NewIDWithName(factory.Type(), "elasticsearch-auth-cancel"),
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}

	rec, err := factory.CreateLogs(t.Context(), settings, cfg, consumertest.NewNop())
	require.NoError(t, err, "creating the Heartbeat receiver should succeed")

	require.NoError(t, rec.Start(t.Context(), host), "starting the Heartbeat receiver should succeed")
	select {
	case <-requestStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("Heartbeat did not start the in-flight Elasticsearch request")
	}

	shutdownStart := time.Now()
	shutdownCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, rec.Shutdown(shutdownCtx), "shutting down the Heartbeat receiver should succeed")
	shutdownDuration := time.Since(shutdownStart)

	assert.Less(t, shutdownDuration, 5*time.Second, "shutdown should release the request promptly rather than waiting for the HTTP timeout")

	select {
	case err := <-requestDone:
		assert.ErrorIs(t, err, context.Canceled, "in-flight request should be released due to context cancellation")
	case <-time.After(time.Second):
		t.Fatal("in-flight request did not record completion")
	}

	select {
	case <-transport.closed:
	case <-time.After(time.Second):
		t.Fatal("Heartbeat did not close idle Elasticsearch connections")
	}
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

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
	endpoints    []string
	roundTripper http.RoundTripper
	roundTripErr error
}

func (f *fakeElasticsearchAuthExtension) Start(context.Context, component.Host) error { return nil }
func (f *fakeElasticsearchAuthExtension) Shutdown(context.Context) error              { return nil }
func (f *fakeElasticsearchAuthExtension) Endpoints() []string                         { return f.endpoints }
func (f *fakeElasticsearchAuthExtension) RoundTripper(http.RoundTripper) (http.RoundTripper, error) {
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
			err := elasticsearchAuthStartHook(test.reference, nil, func(*esClient) {})(test.host)
			require.Error(t, err, "start hook should reject an invalid Elasticsearch authentication extension")
			assert.Contains(t, err.Error(), test.wantError, "start hook should return the expected resolver error")
		})
	}
}

func TestElasticsearchAuthStartHookEmptyReference(t *testing.T) {
	require.NoError(t, elasticsearchAuthStartHook("", nil, func(*esClient) {})(nil), "empty Elasticsearch auth reference should be a no-op")
}

func TestESClientRequest(t *testing.T) {
	var receivedRequest *http.Request
	auth := &fakeElasticsearchAuthExtension{
		endpoints: []string{"http://example.test/base"},
		roundTripper: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			receivedRequest = request.Clone(request.Context())
			return &http.Response{
				StatusCode: http.StatusTeapot,
				Status:     "418 I'm a teapot",
				Body:       io.NopCloser(bytes.NewBufferString(`Brewing error`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client, err := newESClient(auth)
	require.NoError(t, err)
	assert.Equal(t, elasticsearchRequestTimeout, client.client.Timeout, "Heartbeat must own the Elasticsearch request deadline")
	status, body, err := client.Request(http.MethodPost, "/_search?size=1", "pipeline", map[string]string{"routing": "monitor"}, map[string]string{"query": "state"})
	require.EqualError(t, err, `418 I'm a teapot: Brewing error`)
	assert.Equal(t, http.StatusTeapot, status)
	assert.JSONEq(t, `{"hits":{"hits":[]}}`, string(body))
	require.NotNil(t, receivedRequest)
	assert.Equal(t, http.MethodPost, receivedRequest.Method)
	assert.Equal(t, "/base/_search", receivedRequest.URL.Path)
	assert.Equal(t, "1", receivedRequest.URL.Query().Get("size"))
	assert.Equal(t, "pipeline", receivedRequest.URL.Query().Get("pipeline"))
	assert.Equal(t, "monitor", receivedRequest.URL.Query().Get("routing"))
	assert.Equal(t, "application/json", receivedRequest.Header.Get("Content-Type"))
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

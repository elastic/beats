// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/logp"
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
			err := elasticsearchAuthStartHook(t.Context(), test.reference, nil, "", logp.NewNopLogger(), func(*eslegclient.Connection) {})(test.host)
			require.Error(t, err, "start hook should reject an invalid Elasticsearch authentication extension")
			assert.Contains(t, err.Error(), test.wantError, "start hook should return the expected resolver error")
		})
	}
}

func TestElasticsearchAuthStartHookEmptyReference(t *testing.T) {
	require.NoError(t, elasticsearchAuthStartHook(t.Context(), "", nil, "", logp.NewNopLogger(), func(*eslegclient.Connection) {})(nil), "empty Elasticsearch auth reference should be a no-op")
}

func TestNewESClientConnectivityFailureIsBestEffort(t *testing.T) {
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("Elasticsearch is unavailable")
	})
	extension := &fakeElasticsearchAuthExtension{
		endpoints:    []string{"http://example.test"},
		roundTripper: transport,
	}

	client, err := newESClient(t.Context(), extension, "", logp.NewNopLogger())
	require.NoError(t, err, "temporary Elasticsearch connectivity failures must not prevent client creation")
	require.NotNil(t, client, "a client must be returned so state loading can retry after connectivity returns")

	_, _, err = client.Request("GET", "/_search", "", nil, nil)
	require.Error(t, err, "the fallback client must preserve the connectivity error for state loading")
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
		roundTripperFunc: func(request *http.Request) (*http.Response, error) {
			if strings.HasSuffix(request.URL.Path, "_search") {
				once.Do(func() { close(requested) })
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(bytes.NewBufferString(`{"hits":{"hits":[]}}`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(`{"version":{"number":"8.10.0","build_flavor":"default"}}`)),
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
			if !strings.HasSuffix(req.URL.Path, "_search") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(bytes.NewBufferString(`{"version":{"number":"8.10.0","build_flavor":"default"}}`)),
					Header:     make(http.Header),
				}, nil
			}
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

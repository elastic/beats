// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
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

type elasticsearchClientTestHost struct {
	extensions map[component.ID]component.Component
}

func (h elasticsearchClientTestHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

type nopTestExtension struct{}

func (nopTestExtension) Start(context.Context, component.Host) error { return nil }
func (nopTestExtension) Shutdown(context.Context) error              { return nil }

type fakeElasticsearchClientExtension struct {
	requested chan struct{}
	once      sync.Once
}

func (f *fakeElasticsearchClientExtension) Start(context.Context, component.Host) error {
	return nil
}

func (f *fakeElasticsearchClientExtension) Shutdown(context.Context) error {
	return nil
}

func (f *fakeElasticsearchClientExtension) Request(
	_, _, _ string,
	_ map[string]string,
	_ any,
) (int, []byte, error) {
	f.once.Do(func() {
		close(f.requested)
	})
	return 200, []byte(`{"hits":{"hits":[]}}`), nil
}

func TestElasticsearchClientStartHookErrors(t *testing.T) {
	missingID := component.MustNewIDWithName("elasticsearchclient", "missing")
	wrongTypeID := component.MustNewIDWithName("elasticsearchclient", "wrong-type")
	requesterID := component.MustNewIDWithName("elasticsearchclient", "requester")

	tests := []struct {
		name      string
		reference string
		host      component.Host
		wantError string
	}{
		{
			name:      "malformed component ID",
			reference: "/missing-type",
			host:      elasticsearchClientTestHost{},
			wantError: `invalid elasticsearch_client component ID "/missing-type"`,
		},
		{
			name:      "missing extension",
			reference: missingID.String(),
			host:      elasticsearchClientTestHost{},
			wantError: `elasticsearch_client extension "elasticsearchclient/missing" not found`,
		},
		{
			name:      "wrong extension type",
			reference: wrongTypeID.String(),
			host: elasticsearchClientTestHost{extensions: map[component.ID]component.Component{
				wrongTypeID: nopTestExtension{},
			}},
			wantError: "does not implement monitorstate.ElasticsearchRequester",
		},
		{
			name:      "missing captured heartbeat",
			reference: requesterID.String(),
			host: elasticsearchClientTestHost{extensions: map[component.ID]component.Component{
				requesterID: &fakeElasticsearchClientExtension{requested: make(chan struct{})},
			}},
			wantError: `heartbeat instance was not captured for elasticsearch_client extension "elasticsearchclient/requester"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := elasticsearchClientStartHook(test.reference, nil)(test.host)
			require.Error(t, err, "start hook should reject an invalid Elasticsearch client extension")
			assert.Contains(t, err.Error(), test.wantError, "start hook should return the expected resolver error")
		})
	}
}

func TestElasticsearchClientStartHookEmptyReference(t *testing.T) {
	err := elasticsearchClientStartHook("", nil)(nil)
	require.NoError(t, err, "empty Elasticsearch client reference should be a no-op")
}

func TestElasticsearchClientStartHookInjectsBeforeRun(t *testing.T) {
	previousUnderAgent := management.UnderAgent()
	t.Cleanup(func() {
		management.SetUnderAgent(previousUnderAgent)
	})

	extensionID := component.MustNewIDWithName("elasticsearchclient", "_agent-component/default")
	extension := &fakeElasticsearchClientExtension{requested: make(chan struct{})}
	host := elasticsearchClientTestHost{extensions: map[component.ID]component.Component{
		extensionID: extension,
	}}
	cfg := &Config{
		ElasticsearchClient: extensionID.String(),
		Beatconfig: map[string]any{
			"heartbeat": map[string]any{
				"monitors": []map[string]any{
					{
						"type":     "tcp",
						"id":       "elasticsearch-client-hook",
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
		ID: component.NewIDWithName(factory.Type(), "elasticsearch-client-hook"),
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}

	rec, err := factory.CreateLogs(t.Context(), settings, cfg, consumertest.NewNop())
	require.NoError(t, err, "creating the Heartbeat receiver should succeed")

	require.NoError(t, rec.Start(t.Context(), host), "starting the Heartbeat receiver should succeed")
	select {
	case <-extension.requested:
	case <-time.After(10 * time.Second):
		t.Fatal("Heartbeat did not use the injected Elasticsearch requester")
	}

	shutdownCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	require.NoError(t, rec.Shutdown(shutdownCtx), "shutting down the Heartbeat receiver should succeed")
}

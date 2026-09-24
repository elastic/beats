// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

func TestHeartbeatOTelElasticsearchAuthLoadsMonitorState(t *testing.T) {
	integration.EnsureESIsRunning(t)

	esHost := integration.GetESAdminURL(t, "http")
	esUser := esHost.User.Username()
	esPassword, _ := esHost.User.Password()
	esURL := fmt.Sprintf("%s://%s", esHost.Scheme, esHost.Host)
	es := integration.GetESAdminClient(t, "http")

	namespace := uuid.Must(uuid.NewV4()).String()
	stateIndex := "heartbeat-auth-state-" + namespace
	eventIndex := "logs-heartbeat-auth-" + namespace
	monitorID := "auth-state-test-" + namespace
	previousStateID := "previous-state-" + namespace

	t.Cleanup(func() {
		_, err := es.Indices.Delete([]string{stateIndex})
		require.NoError(t, err, "failed to delete seeded monitor state index")

		_, err = es.Indices.DeleteDataStream([]string{eventIndex})
		require.NoError(t, err, "failed to delete emitted event data stream")
	})

	previousState := map[string]any{
		"@timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"monitor": map[string]any{
			"id":   monitorID,
			"type": "http",
		},
		"state": map[string]any{
			"id":           previousStateID,
			"started_at":   time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339Nano),
			"duration_ms":  "300000",
			"status":       "down",
			"checks":       5,
			"up":           0,
			"down":         5,
			"flap_history": []string{},
		},
	}
	previousStateJSON, err := json.Marshal(previousState)
	require.NoError(t, err, "failed to encode seeded monitor state")

	indexResponse, err := es.Index(
		stateIndex,
		bytes.NewReader(previousStateJSON),
		es.Index.WithDocumentID(previousStateID),
		es.Index.WithRefresh("true"),
	)
	require.NoError(t, err, "failed to seed monitor state")
	require.False(t, indexResponse.IsError(), "seeding monitor state returned %s", indexResponse.Status())
	require.NoError(t, indexResponse.Body.Close(), "failed to close state indexing response")

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)

	esBackend := esHost
	esBackend.User = nil
	tlsElasticsearch := httptest.NewTLSServer(httputil.NewSingleHostReverseProxy(&esBackend))
	t.Cleanup(tlsElasticsearch.Close)

	proxy := &recordingConnectProxy{}
	proxyServer := httptest.NewServer(proxy)
	t.Cleanup(proxyServer.Close)

	collectorConfig := fmt.Sprintf(`extensions:
  beatsauth/state:
    proxy_url: %s
    idle_connection_timeout: 1s
    ssl:
      verification_mode: none
  elasticsearchauth/state:
    endpoints:
      - %s
    user: %s
    password: %s
    auth:
      authenticator: beatsauth/state
receivers:
  heartbeatreceiver:
    elasticsearch_auth: elasticsearchauth/state
    heartbeat:
      monitors:
        - type: http
          id: %s
          name: %s
          enabled: true
          schedule: "@every 1s"
          timeout: 3s
          urls:
            - %s
          check.response.status: 200
    logging:
      level: info
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
    management.otel.enabled: true
exporters:
  elasticsearch/log:
    endpoints:
      - %s
    user: %s
    password: %s
    compression: none
    logs_index: %s
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  extensions:
    - beatsauth/state
    - elasticsearchauth/state
  pipelines:
    logs:
      receivers:
        - heartbeatreceiver
      exporters:
        - elasticsearch/log
  telemetry:
    logs:
      level: info
    metrics:
      level: none
`,
		proxyServer.URL,
		tlsElasticsearch.URL,
		esUser,
		esPassword,
		monitorID,
		monitorID,
		target.URL,
		t.TempDir(),
		esURL,
		esUser,
		esPassword,
		eventIndex,
	)

	oteltestcol.New(t, collectorConfig)

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{"match": map[string]any{"monitor.id": monitorID}},
					{"exists": map[string]any{"field": "state.ends.id"}},
				},
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	require.EventuallyWithTf(t, func(ct *assert.CollectT) {
		findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer findCancel()

		docs, err := estools.PerformQueryForRawQuery(findCtx, query, ".ds-"+eventIndex+"*", es)
		if !assert.NoError(ct, err, "failed to query emitted Heartbeat events") {
			return
		}

		found := false
		for _, hit := range docs.Hits.Hits {
			fields := mapstr.M(hit.Source).Flatten()
			if fields["state.status"] == "up" &&
				fields["state.ends.id"] == previousStateID &&
				fields["state.ends.status"] == "down" {
				found = true
				break
			}
		}

		assert.True(
			ct,
			found,
			"expected an emitted Heartbeat event to contain the loaded previous state in state.ends; got %v",
			docs.Hits.Hits,
		)
	},
		2*time.Minute,
		time.Second,
		"timed out waiting for Heartbeat to load and use monitor state",
	)

	assert.Positive(t, proxy.connects.Load(), "Heartbeat state loading must connect to Elasticsearch through beatsauth's configured proxy")
}

type recordingConnectProxy struct {
	connects atomic.Uint64
}

func (p *recordingConnectProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodConnect {
		http.Error(writer, "only CONNECT is supported", http.StatusMethodNotAllowed)
		return
	}

	dialCtx, dialCancel := context.WithTimeout(request.Context(), 10*time.Second)
	defer dialCancel()

	destination, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", request.Host)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusServiceUnavailable)
		return
	}

	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		_ = destination.Close()
		http.Error(writer, "hijacking is unsupported", http.StatusInternalServerError)
		return
	}
	client, bufferedClient, err := hijacker.Hijack()
	if err != nil {
		_ = destination.Close()
		return
	}

	if _, err := bufferedClient.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		_ = client.Close()
		_ = destination.Close()
		return
	}
	if err := bufferedClient.Flush(); err != nil {
		_ = client.Close()
		_ = destination.Close()
		return
	}
	p.connects.Add(1)

	copyDone := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(destination, bufferedClient)
		copyDone <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, destination)
		copyDone <- struct{}{}
	}()

	<-copyDone
	_ = client.Close()
	_ = destination.Close()
	<-copyDone
}

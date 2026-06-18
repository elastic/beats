// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/testing/testutils"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestFollowStreamReturnsOnCancelWithStalledServer(t *testing.T) {
	testutils.SkipIfFIPSOnly(t, "websocket uses SHA-1.")

	// Server sends one message then goes silent. The client will
	// successfully read and publish the first message, re-enter
	// ReadMessage for the second read, and block. Cancelling at
	// that point exercises the shutdown-while-blocked path.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Send one message so the client completes a full read cycle.
		conn.WriteMessage(websocket.TextMessage, []byte(`{"ts":"2024-01-01T00:00:00Z","data":"hello"}`))
		// Then go silent — hold the connection open until test cleanup.
		<-r.Context().Done()
		conn.Close()
	}))
	defer server.Close()

	config := map[string]interface{}{
		"url": "ws" + server.URL[4:] + "/stall",
		"program": `
			state.response.decode_json().as(body, {
				"events": [body],
			})`,
	}
	cfg := conf.MustNewConfigFrom(config)

	c := defaultConfig()
	c.Redact = &redact{}
	if err := cfg.Unpack(&c); err != nil {
		t.Fatalf("unexpected error unpacking config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v2Ctx := v2.Context{
		Logger:          logptest.NewTestingLogger(t, "websocket_shutdown_test"),
		ID:              "test_id:stalled_shutdown",
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}

	// Cancel after the first event is published. Publish sends on
	// an unbuffered channel; by the time the test goroutine receives
	// and calls cancel, the client has returned from processing and
	// re-entered ReadMessage (a blocking syscall that yields the
	// goroutine).
	published := make(chan struct{})
	var client publisher
	client.done = func() {
		select {
		case published <- struct{}{}:
		default:
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- input{}.run(v2Ctx, &source{c}, nil, &client)
	}()

	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first event publish")
	}
	cancel()

	select {
	case err := <-done:
		if err != nil && err != context.Canceled { //nolint:errorlint // ctx.Err() is never wrapped
			t.Errorf("input.run() error = %v; want nil or context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("input.run() did not return after context cancellation; ReadMessage is likely stuck")
	}
}

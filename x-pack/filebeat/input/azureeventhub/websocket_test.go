// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
)

func TestWebSocketConnectionSurvivesDialContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"amqp"}})
		if err != nil {
			return
		}
		defer ws.CloseNow()

		messageType, payload, err := ws.Read(r.Context())
		if err == nil {
			_ = ws.Write(r.Context(), messageType, payload)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	dial := newWebSocketDialer(httpcommon.HTTPClientProxySettings{})
	conn, err := dial(ctx, azeventhubs.WebSocketConnParams{Host: "ws" + strings.TrimPrefix(server.URL, "http")})
	require.NoError(t, err, "the WebSocket handshake should succeed")
	defer conn.Close()
	cancel()

	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)), "the exchange should have a deadline")
	payload := []byte("probe")
	_, err = conn.Write(payload)
	require.NoError(t, err, "canceling the dial context should leave the established connection writable")
	echo := make([]byte, len(payload))
	_, err = io.ReadFull(conn, echo)
	require.NoError(t, err, "canceling the dial context should leave the established connection readable")
	require.Equal(t, payload, echo, "the server should echo the probe")
}

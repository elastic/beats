// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package osqdcli

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryCancellationInterruptsTransportRead(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "query.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err, "test RPC server must listen")
	defer listener.Close()
	request := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		data := make([]byte, 128)
		if _, err := connection.Read(data); err != nil {
			return
		}
		close(request)
		// Hold the request until client cancellation closes the connection.
		for {
			if _, err := connection.Read(data); err != nil {
				return
			}
		}
	}()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := New(socket, WithTimeout(time.Minute))
	done := make(chan error, 1)
	go func() { _, err := client.Query(ctx, "SELECT 1", time.Minute); done <- err }()
	select {
	case <-request:
	case <-time.After(time.Second):
		t.Fatal("RPC request was not sent")
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "cancellation must interrupt the transport before its minute timeout")
	case <-time.After(time.Second):
		t.Fatal("RPC remained blocked after cancellation")
	}
	select {
	case <-serverDone:
	case <-time.After(time.Second):
		t.Fatal("RPC connection was not closed")
	}
}

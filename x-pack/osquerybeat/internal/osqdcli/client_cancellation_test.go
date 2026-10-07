// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package osqdcli

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	genosquery "github.com/osquery/osquery-go/gen/osquery"
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

type cancellationColumnServer struct {
	genosquery.ExtensionManager
	entered chan struct{}
	release chan struct{}
}

func (s *cancellationColumnServer) GetQueryColumns(_ context.Context, sql string) (*genosquery.ExtensionResponse, error) {
	if sql == "SELECT cancelled" {
		close(s.entered)
		<-s.release
	}
	return &genosquery.ExtensionResponse{Status: &genosquery.ExtensionStatus{Code: 0}, Response: []map[string]string{{"value": "INTEGER"}}}, nil
}

func TestCancelledColumnLookupDoesNotPoisonScheduledResolution(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "columns.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err, "test column server must listen")
	handler := &cancellationColumnServer{entered: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(handler.release) }) }
	var serving sync.WaitGroup
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			serving.Go(func() {
				transport := thrift.NewTSocketFromConnTimeout(conn, time.Minute)
				defer transport.Close()
				input := thrift.NewTBinaryProtocolFactoryDefault().GetProtocol(transport)
				output := thrift.NewTBinaryProtocolFactoryDefault().GetProtocol(transport)
				processor := genosquery.NewExtensionManagerProcessor(handler)
				for {
					if _, err := processor.Process(t.Context(), input, output); err != nil {
						return
					}
				}
			})
		}
	}()
	client := New(socket)
	defer func() { client.Close(); listener.Close(); <-stopped; serving.Wait() }()
	require.NoError(t, client.Connect(t.Context()), "daemon client must connect once")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client.ResolveResult(ctx, "SELECT cancelled", []map[string]string{{"value": "1"}})
		done <- err
	}()
	select {
	case <-handler.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("column RPC was not started")
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "short-lived lookup must respect cancellation")
	case <-time.After(time.Second):
		t.Fatal("column lookup did not cancel")
	}
	release()
	hits, err := client.ResolveResult(t.Context(), "SELECT scheduled", []map[string]string{{"value": "2"}})
	require.NoError(t, err, "cancelled action must not break subsequent scheduled resolution without daemon reconnect")
	assert.Equal(t, []map[string]any{{"value": int64(2)}}, hits, "subsequent schedule must still resolve types")
	client.Close()
	_, err = client.ResolveResult(t.Context(), "SELECT scheduled", nil)
	assert.ErrorIs(t, err, ErrClientClosed, "intentional client Close must still reject scheduled resolution")
}

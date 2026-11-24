// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package osqdcli

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	genosquery "github.com/osquery/osquery-go/gen/osquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cancellationTestSocket(t *testing.T) string {
	t.Helper()
	// macOS's default temp directory can exceed the Unix socket path limit.
	dir, err := os.MkdirTemp("/tmp", "osqd-")
	require.NoError(t, err, "create short private socket directory")
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(dir), "remove socket directory") })
	return filepath.Join(dir, "rpc.sock")
}

type cancellationServer struct {
	genosquery.ExtensionManager
	entered chan struct{}
	release chan struct{}
}

func (s *cancellationServer) Query(_ context.Context, sql string) (*genosquery.ExtensionResponse, error) {
	if sql == "SELECT blocked_query" {
		close(s.entered)
		<-s.release
	}
	return &genosquery.ExtensionResponse{Status: &genosquery.ExtensionStatus{}, Response: []map[string]string{{"value": "2"}}}, nil
}

func (s *cancellationServer) GetQueryColumns(_ context.Context, sql string) (*genosquery.ExtensionResponse, error) {
	if sql == "SELECT blocked_columns" {
		close(s.entered)
		<-s.release
	}
	return &genosquery.ExtensionResponse{Status: &genosquery.ExtensionStatus{}, Response: []map[string]string{{"value": "INTEGER"}}}, nil
}

func startCancellationServer(t *testing.T) (string, *cancellationServer) {
	t.Helper()
	socket := cancellationTestSocket(t)
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socket)
	require.NoError(t, err, "listen for test RPC clients")
	handler := &cancellationServer{entered: make(chan struct{}), release: make(chan struct{})}
	var serving sync.WaitGroup
	var connectionsMu sync.Mutex
	var connections []net.Conn
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connectionsMu.Lock()
			connections = append(connections, conn)
			connectionsMu.Unlock()
			serving.Go(func() {
				transport := thrift.NewTSocketFromConnConf(conn, &thrift.TConfiguration{SocketTimeout: time.Minute})
				defer transport.Close()
				input := thrift.NewTBinaryProtocolConf(transport, nil)
				output := thrift.NewTBinaryProtocolConf(transport, nil)
				processor := genosquery.NewExtensionManagerProcessor(handler)
				for {
					if _, err := processor.Process(t.Context(), input, output); err != nil {
						return
					}
				}
			})
		}
	}()
	t.Cleanup(func() {
		close(handler.release)
		listener.Close()
		<-stopped
		connectionsMu.Lock()
		for _, conn := range connections {
			conn.Close()
		}
		connectionsMu.Unlock()
		serving.Wait()
	})
	return socket, handler
}

func awaitCancellation(t *testing.T, done <-chan error, expected error) {
	t.Helper()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, expected, "RPC admission or transport must return the context error")
	case <-time.After(time.Second):
		t.Fatal("RPC did not return promptly after its context ended")
	}
}

func cancelQueuedCall(t *testing.T, call func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		done <- call(ctx)
	}()
	<-started
	select {
	case err := <-done:
		t.Fatalf("queued RPC returned before its context ended: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	awaitCancellation(t, done, context.Canceled)
}

func TestQueryCancellationInterruptsTransportRead(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		name := "cancel"
		if deadline {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			socket, handler := startCancellationServer(t)
			client := New(socket, WithTimeout(time.Minute))
			ctx, cancel := context.WithCancel(t.Context())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), time.Second)
			}
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := client.Query(ctx, "SELECT blocked_query", time.Minute); done <- err }()
			select {
			case <-handler.entered:
			case <-time.After(2 * time.Second):
				t.Fatal("query RPC was not started")
			}
			expected := context.Canceled
			if deadline {
				<-ctx.Done()
				expected = context.DeadlineExceeded
			} else {
				cancel()
			}
			awaitCancellation(t, done, expected)
		})
	}
}

func TestCancelledColumnLookupDoesNotPoisonFutureResolution(t *testing.T) {
	socket, handler := startCancellationServer(t)
	client := New(socket, WithTimeout(time.Minute))
	require.NoError(t, client.Connect(t.Context()), "connect lifecycle client once")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client.ResolveResult(ctx, "SELECT blocked_columns", []map[string]string{{"value": "1"}})
		done <- err
	}()
	select {
	case <-handler.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("column RPC was not started")
	}
	cancel()
	awaitCancellation(t, done, context.Canceled)
	columns, err := client.cli.GetQueryColumnsContext(t.Context(), "SELECT lifecycle")
	require.NoError(t, err, "per-operation cancellation must leave the retained lifecycle connection usable")
	assert.Equal(t, "INTEGER", columns.Response[0]["value"], "retained client must still receive column types")
	hits, err := client.ResolveResult(t.Context(), "SELECT scheduled", []map[string]string{{"value": "2"}})
	require.NoError(t, err, "subsequent cache-miss resolution must succeed without reconnecting the daemon client")
	assert.Equal(t, []map[string]any{{"value": int64(2)}}, hits, "subsequent results must retain numeric types")
	client.Close()
	_, err = client.ResolveResult(t.Context(), "SELECT scheduled", nil)
	assert.ErrorIs(t, err, ErrClientClosed, "intentional Close must still reject result resolution")
}

func TestCancelledCallDoesNotWaitForPendingConnect(t *testing.T) {
	client := New(cancellationTestSocket(t), WithConnectRetries(0))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	connected := make(chan error, 1)
	go func() { connected <- client.Connect(ctx) }()
	require.Eventually(t, func() bool {
		if client.mx.TryLock() {
			client.mx.Unlock()
			return false
		}
		return true
	}, time.Second, time.Millisecond, "initial Connect must be waiting for its missing socket")
	for _, call := range []struct {
		name string
		fn   func(context.Context) error
	}{
		{"connect", client.Connect},
		{"query", func(ctx context.Context) error { _, err := client.Query(ctx, "SELECT 1", time.Minute); return err }},
		{"resolve", func(ctx context.Context) error { _, err := client.ResolveResult(ctx, "SELECT 1", nil); return err }},
	} {
		t.Run(call.name, func(t *testing.T) {
			cancelQueuedCall(t, call.fn)
		})
	}
	cancel()
	awaitCancellation(t, connected, context.Canceled)
}

func TestCancelledCallDoesNotWaitForActiveQuery(t *testing.T) {
	socket, handler := startCancellationServer(t)
	client := New(socket, WithTimeout(time.Minute))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	queryDone := make(chan error, 1)
	go func() { _, err := client.Query(ctx, "SELECT blocked_query", time.Minute); queryDone <- err }()
	select {
	case <-handler.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("initial query RPC was not started")
	}
	for _, call := range []struct {
		name string
		fn   func(context.Context) error
	}{
		{"connect", client.Connect},
		{"query", func(ctx context.Context) error { _, err := client.Query(ctx, "SELECT 1", time.Minute); return err }},
		{"resolve", func(ctx context.Context) error { _, err := client.ResolveResult(ctx, "SELECT 1", nil); return err }},
	} {
		t.Run(call.name, func(t *testing.T) {
			cancelQueuedCall(t, call.fn)
		})
	}
	cancel()
	awaitCancellation(t, queryDone, context.Canceled)
}

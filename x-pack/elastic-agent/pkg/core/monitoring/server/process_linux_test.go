// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessProxyRequest(t *testing.T) {
	sock := "/tmp/elastic-agent-test.sock"
	defer os.Remove(sock)

	endpoint := "http+unix://" + sock
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write the path to the client so they can verify the request
		// was correct
		w.Write([]byte(r.URL.Path))
	}))

	// Mimic subprocesses and listen on a unix socket
	l, err := net.Listen("unix", sock)
	require.NoError(t, err)
	server.Listener = l
	server.Start()
	defer server.Close()

	for _, path := range []string{"stats", "", "state"} {
		respBytes, _, err := processMetrics(context.Background(), endpoint, path)
		require.NoError(t, err)
		// Verify that the server saw the path we tried to request
		assert.Equal(t, "/"+path, string(respBytes))
	}
}

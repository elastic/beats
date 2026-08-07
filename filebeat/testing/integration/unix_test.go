// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration

// Tests for the Unix socket input, ported from test_unix.py.

package integration

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// shortUnixSockDir creates a temporary directory with a short path suitable
// for Unix socket paths (max 103 chars on macOS, 107 on Linux).
func shortUnixSockDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "fb-unix-")
	if err != nil {
		t.Fatalf("shortUnixSockDir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// sendUnixStreamEvents is the shared helper for Unix stream socket delimiter tests.
func sendUnixStreamEvents(t *testing.T, delimiter string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := shortUnixSockDir(t)
	sockPath := fmt.Sprintf("%s/filebeat.sock", dir)

	config := UnixInputConfig(sockPath, UnixInputOptions{LineDelimiter: delimiter})
	test := NewTest(t, TestOptions{Config: config})

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "Hello World")

	test.ExpectJSONFields(mapstr.M{"input.type": "unix"})
	test.ExpectOutput("Hello World: 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	WaitForUnixStreamSocket(t, sockPath)

	msgs := [][]byte{
		[]byte("Hello World: 0" + delimiter),
		[]byte("Hello World: 1" + delimiter),
	}
	SendUnixStream(t, sockPath, msgs)

	test.Wait()
	require.EqualValues(t, 2, eventCount.Load(), "expected 2 Unix stream events")
}

// TestUnixNewlineDelimiter verifies the Unix input with the default newline delimiter.
func TestUnixNewlineDelimiter(t *testing.T) {
	sendUnixStreamEvents(t, "\n")
}

// TestUnixCustomCharDelimiter verifies the Unix input with a single-character delimiter.
func TestUnixCustomCharDelimiter(t *testing.T) {
	sendUnixStreamEvents(t, ";")
}

// TestUnixCustomWordDelimiter verifies the Unix input with a multi-character delimiter.
func TestUnixCustomWordDelimiter(t *testing.T) {
	sendUnixStreamEvents(t, "<END>")
}

// TestUnixDatagramSocket verifies the Unix input in datagram mode.
func TestUnixDatagramSocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := shortUnixSockDir(t)
	sockPath := fmt.Sprintf("%s/filebeat.sock", dir)

	config := UnixInputConfig(sockPath, UnixInputOptions{SocketType: "datagram"})
	test := NewTest(t, TestOptions{Config: config})

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "Hello World")

	// Datagram mode: each datagram is one event; the delimiter is kept in the message.
	test.ExpectJSONFields(mapstr.M{"input.type": "unix"})
	test.ExpectOutput("Hello World: 1;")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	WaitForUnixDatagramSocket(t, sockPath)

	msgs := [][]byte{
		[]byte("Hello World: 0;"),
		[]byte("Hello World: 1;"),
	}
	SendUnixDatagram(t, sockPath, msgs)

	test.Wait()
	require.EqualValues(t, 2, eventCount.Load(), "expected 2 Unix datagram events")
}

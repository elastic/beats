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

// Tests for the TCP input, ported from test_tcp.py.

package integration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func sendTCPEvents(t *testing.T, bindAddr string, delimiter string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	port := AllocatePort(t)
	listenAddr := fmt.Sprintf("%s:%d", bindAddr, port)
	sendAddr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(listenAddr, TCPInputOptions{LineDelimiter: delimiter})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "Hello World")

	test.ExpectJSONFields(mapstr.M{"input.type": "tcp"})
	test.ExpectOutput("Hello World: 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting TCP connections")

	msgs := [][]byte{
		[]byte("Hello World: 0" + delimiter),
		[]byte("Hello World: 1" + delimiter),
	}
	SendTCP(t, sendAddr, msgs)

	test.Wait()
	require.EqualValues(t, 2, eventCount.Load(), "expected 2 TCP events")
}

// TestTCPNewlineDelimiter verifies the TCP input with the default newline delimiter.
func TestTCPNewlineDelimiter(t *testing.T) {
	sendTCPEvents(t, "127.0.0.1", "\n")
}

// TestTCPCustomCharDelimiter verifies the TCP input with a single-character custom delimiter.
func TestTCPCustomCharDelimiter(t *testing.T) {
	sendTCPEvents(t, "127.0.0.1", ";")
}

// TestTCPCustomWordDelimiter verifies the TCP input with a multi-character custom delimiter.
func TestTCPCustomWordDelimiter(t *testing.T) {
	sendTCPEvents(t, "127.0.0.1", "<END>")
}

// TestTCPWildcardAddress verifies that binding to 0.0.0.0 accepts connections.
func TestTCPWildcardAddress(t *testing.T) {
	sendTCPEvents(t, "0.0.0.0", "\n")
}

// sendTCPEventsRFC6587 sends two RFC 6587 framed messages over TCP.
// mode is the client-side encoding: "non-transparent" (newline-delimited) or
// "octet" (octet-counting prefix). The Filebeat config always uses rfc6587 framing.
func sendTCPEventsRFC6587(t *testing.T, mode string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		LineDelimiter: "\n",
		Framing:       "rfc6587",
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "Hello World")

	test.ExpectJSONFields(mapstr.M{"input.type": "tcp"})
	test.ExpectOutput("Hello World: 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting TCP connections")

	var msgs [][]byte
	for n := range 2 {
		switch mode {
		case "non-transparent":
			msgs = append(msgs, fmt.Appendf(nil, "Hello World: %d\n", n))
		case "octet":
			msg := fmt.Sprintf("Hello World: %d", n)
			msgs = append(msgs, fmt.Appendf(nil, "%d %s", len(msg), msg))
		}
	}
	SendTCP(t, addr, msgs)

	test.Wait()
	require.EqualValues(t, 2, eventCount.Load(), "expected 2 TCP events")
}

// TestTCPRFC6587NonTransparent verifies RFC 6587 non-transparent framing.
func TestTCPRFC6587NonTransparent(t *testing.T) {
	sendTCPEventsRFC6587(t, "non-transparent")
}

// TestTCPRFC6587Octet verifies RFC 6587 octet-counting framing.
func TestTCPRFC6587Octet(t *testing.T) {
	sendTCPEventsRFC6587(t, "octet")
}

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

// Tests for the UDP input, ported from test_udp.py.

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

func sendUDPEvents(t *testing.T, bindAddr string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	port := AllocatePort(t)
	listenAddr := fmt.Sprintf("%s:%d", bindAddr, port)
	sendAddr := fmt.Sprintf("127.0.0.1:%d", port)

	config := UDPInputConfig(listenAddr)
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, udpReadyMsg)

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "Hello World")

	test.ExpectJSONFields(mapstr.M{"input.type": "udp"})
	test.ExpectOutput("Hello World: 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start listening for UDP connections")

	msgs := [][]byte{
		[]byte("Hello World: 0"),
		[]byte("Hello World: 1"),
	}
	SendUDP(t, sendAddr, msgs)

	test.Wait()
	require.EqualValues(t, 2, eventCount.Load(), "expected 2 UDP events")
}

// TestUDP verifies the UDP input binding to 127.0.0.1.
func TestUDP(t *testing.T) {
	sendUDPEvents(t, "127.0.0.1")
}

// TestUDPWildcardAddress verifies the UDP input binding to 0.0.0.0.
func TestUDPWildcardAddress(t *testing.T) {
	sendUDPEvents(t, "0.0.0.0")
}

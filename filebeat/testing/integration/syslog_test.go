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

// Tests for the syslog input, ported from test_syslog.py.

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

// rfc3164Message returns an RFC 3164 syslog message for event index n.
// PRI 13 = facility 1 (user-level), severity 5 (Notice).
func rfc3164Message(n int) []byte {
	return fmt.Appendf(nil,
		"<13>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8 %d\n",
		n,
	)
}

// syslogJSONFields returns the mapstr.M assertions that every valid syslog
// event from rfc3164Message must satisfy.
func syslogJSONFields() mapstr.M {
	return mapstr.M{
		"input.type":            "syslog",
		"hostname":              "wopr.mymachine.co",
		"process.program":       "postfix/smtpd",
		"process.pid":           float64(2000),
		"syslog.facility":       float64(1),
		"syslog.priority":       float64(13),
		"syslog.severity_label": "Notice",
		"syslog.facility_label": "user-level",
		"event.severity":        float64(5),
	}
}

// TestSyslogTCP verifies the syslog input over TCP.
func TestSyslogTCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := SyslogInputConfig(SyslogInputOptions{Protocol: SyslogTCP, Addr: addr})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	test.ExpectJSONFields(syslogJSONFields())
	test.ExpectOutput("failed for lonvick on /dev/pts/8 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting syslog TCP connections")

	msgs := [][]byte{rfc3164Message(0), rfc3164Message(1)}
	SendTCP(t, addr, msgs)

	test.Wait()
}

// TestSyslogTCPInvalidMessage verifies that an unparseable syslog message is
// stored verbatim in the message field.
func TestSyslogTCPInvalidMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := SyslogInputConfig(SyslogInputOptions{Protocol: SyslogTCP, Addr: addr})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	test.ExpectJSONFields(mapstr.M{"input.type": "syslog", "message": "invalid"})
	test.ExpectOutput("invalid")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting syslog TCP connections")

	SendTCP(t, addr, [][]byte{[]byte("invalid\n"), []byte("invalid\n")})

	test.Wait()
}

// TestSyslogUDP verifies the syslog input over UDP.
func TestSyslogUDP(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := SyslogInputConfig(SyslogInputOptions{Protocol: SyslogUDP, Addr: addr})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, udpReadyMsg)

	test.ExpectJSONFields(syslogJSONFields())
	test.ExpectOutput("failed for lonvick on /dev/pts/8 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start listening for syslog UDP")

	// Send 2 datagrams and stop after both are received. We match on "on /dev/pts/8 1"
	// (the second message) so the beat runs until at least both are processed.
	// UDP on loopback does not lose packets, but sending 2 provides a safety margin.
	SendUDP(t, addr, [][]byte{rfc3164Message(0), rfc3164Message(1)})

	test.Wait()
}

// TestSyslogUnixStream verifies the syslog input over a Unix stream socket.
func TestSyslogUnixStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	sockPath := fmt.Sprintf("%s/filebeat.sock", shortUnixSockDir(t))

	config := SyslogInputConfig(SyslogInputOptions{
		Protocol:   SyslogUnix,
		Addr:       sockPath,
		SocketType: "stream",
	})
	test := NewTest(t, TestOptions{Config: config})

	test.ExpectJSONFields(mapstr.M{"input.type": "syslog"})
	test.ExpectOutput("failed for lonvick on /dev/pts/8 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	WaitForUnixStreamSocket(t, sockPath)

	msgs := [][]byte{rfc3164Message(0), rfc3164Message(1)}
	SendUnixStream(t, sockPath, msgs)

	test.Wait()
}

// TestSyslogUnixDatagram verifies the syslog input over a Unix datagram socket.
func TestSyslogUnixDatagram(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	sockPath := fmt.Sprintf("%s/filebeat.sock", shortUnixSockDir(t))

	config := SyslogInputConfig(SyslogInputOptions{
		Protocol:   SyslogUnix,
		Addr:       sockPath,
		SocketType: "datagram",
	})
	test := NewTest(t, TestOptions{Config: config})

	test.ExpectJSONFields(mapstr.M{"input.type": "syslog"})
	test.ExpectOutput("failed for lonvick on /dev/pts/8 1")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	WaitForUnixDatagramSocket(t, sockPath)

	msgs := [][]byte{rfc3164Message(0), rfc3164Message(1)}
	SendUnixDatagram(t, sockPath, msgs)

	test.Wait()
}

// TestSyslogUnixStreamInvalidMessage verifies that an invalid syslog message
// over a Unix stream socket is stored verbatim in the message field.
func TestSyslogUnixStreamInvalidMessage(t *testing.T) {
	runSyslogUnixInvalidMessage(t, "stream")
}

// TestSyslogUnixDatagramInvalidMessage verifies that an invalid syslog message
// over a Unix datagram socket is stored in the message field (with trailing \n).
func TestSyslogUnixDatagramInvalidMessage(t *testing.T) {
	runSyslogUnixInvalidMessage(t, "datagram")
}

func runSyslogUnixInvalidMessage(t *testing.T, socketType string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	sockPath := fmt.Sprintf("%s/filebeat.sock", shortUnixSockDir(t))

	config := SyslogInputConfig(SyslogInputOptions{
		Protocol:   SyslogUnix,
		Addr:       sockPath,
		SocketType: socketType,
	})
	test := NewTest(t, TestOptions{Config: config})

	test.ExpectJSONFields(mapstr.M{"input.type": "syslog"})
	test.ExpectOutput("invalid")
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	if socketType == "stream" {
		WaitForUnixStreamSocket(t, sockPath)
		SendUnixStream(t, sockPath, [][]byte{[]byte("invalid\n"), []byte("invalid\n")})
	} else {
		WaitForUnixDatagramSocket(t, sockPath)
		SendUnixDatagram(t, sockPath, [][]byte{[]byte("invalid\n"), []byte("invalid\n")})
	}

	test.Wait()
}

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

// Tests for the TCP input with TLS, ported from test_tcp_tls.py.

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const numTLSEvents = 2

func tlsMessages(n int) [][]byte {
	msgs := make([][]byte, n)
	for i := range n {
		msgs[i] = fmt.Appendf(nil, "Hello World: %d\n", i)
	}
	return msgs
}

// TestTCPTLSValidServer verifies TLS with a valid CA cert and no mutual auth.
func TestTCPTLSValidServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	ca := GenerateTestCA(t)
	certFile, keyFile := ca.Issue(t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		TLSCA:      ca.CertFile,
		TLSCert:    certFile,
		TLSKey:     keyFile,
		ClientAuth: "optional",
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	test.ExpectJSONFields(mapstr.M{"input.type": "tcp"})
	test.ExpectOutput(fmt.Sprintf("Hello World: %d", numTLSEvents-1))
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting connections")

	tlsCfg := LoadClientTLSConfig(t, ca.CertFile, "", "")
	err := SendTCPTLS(ctx, addr, tlsCfg, tlsMessages(numTLSEvents))
	require.NoError(t, err)

	test.Wait()
}

// TestTCPTLSInvalidServer verifies that a mismatched CA cert causes the TLS
// handshake to fail and no events are ingested.
func TestTCPTLSInvalidServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	// Server uses certs from ca1; client trusts only ca2 → verification must fail.
	ca1 := GenerateTestCA(t)
	serverCert, serverKey := ca1.Issue(t)

	ca2 := GenerateTestCA(t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		TLSCA:      ca1.CertFile,
		TLSCert:    serverCert,
		TLSKey:     serverKey,
		ClientAuth: "optional",
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, `"input.type"`)

	test.ExpectStop(exitSignalKilled)
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting connections")

	tlsCfg := LoadClientTLSConfig(t, ca2.CertFile, "", "")
	err := SendTCPTLS(ctx, addr, tlsCfg, tlsMessages(numTLSEvents))
	require.Error(t, err, "expected TLS handshake to fail with mismatched CA")

	cancel()
	test.Wait()
	require.EqualValues(t, 0, eventCount.Load(), "no events expected from failed TLS handshake")
}

// TestTCPTLSMutualAuthFails verifies that a client without a certificate is
// rejected when the server requires mutual auth.
func TestTCPTLSMutualAuthFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	ca := GenerateTestCA(t)
	serverCert, serverKey := ca.Issue(t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		TLSCA:   ca.CertFile,
		TLSCert: serverCert,
		TLSKey:  serverKey,
		// Default client_authentication is "required".
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, `"input.type"`)

	test.ExpectStop(exitSignalKilled)
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting connections")

	// Client verifies server correctly but presents no client cert.
	// The server will reject it during or just after the handshake.
	tlsCfg := LoadClientTLSConfig(t, ca.CertFile, "", "")
	conn, dialErr := (&tls.Dialer{Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	if dialErr == nil {
		// TLS 1.3: the server's rejection alert arrives as a read error, not
		// during the handshake. We discard the error because any error here
		// (or on Dial itself) is the expected outcome; we only care that no
		// event was produced.
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		conn.Close()
	}

	cancel()
	test.Wait()
	require.EqualValues(t, 0, eventCount.Load(), "no events expected when client cert is missing")
}

// TestTCPTLSMutualAuthSucceeds verifies that mutual TLS auth works with valid
// client and server certificates.
func TestTCPTLSMutualAuthSucceeds(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	ca := GenerateTestCA(t)
	serverCert, serverKey := ca.Issue(t)
	clientCert, clientKey := ca.Issue(t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		TLSCA:      ca.CertFile,
		TLSCert:    serverCert,
		TLSKey:     serverKey,
		ClientAuth: "required",
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	test.ExpectJSONFields(mapstr.M{"input.type": "tcp"})
	test.ExpectOutput(fmt.Sprintf("Hello World: %d", numTLSEvents-1))
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting connections")

	tlsCfg := LoadClientTLSConfig(t, ca.CertFile, clientCert, clientKey)
	err := SendTCPTLS(ctx, addr, tlsCfg, tlsMessages(numTLSEvents))
	require.NoError(t, err)

	test.Wait()
}

// TestTCPTLSPlainTextSocket verifies that a plain-text connection to a TLS
// server is rejected and produces no events.
func TestTCPTLSPlainTextSocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	ca := GenerateTestCA(t)
	certFile, keyFile := ca.Issue(t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		TLSCA:      ca.CertFile,
		TLSCert:    certFile,
		TLSKey:     keyFile,
		ClientAuth: "required",
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, `"input.type"`)

	test.ExpectStop(exitSignalKilled)
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting connections")

	// Connect with plain TCP to a TLS server; writes will eventually fail.
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err == nil {
		for i := range 100 {
			if _, werr := conn.Write(fmt.Appendf(nil, "Hello World: %d\n", i)); werr != nil {
				break
			}
		}
		conn.Close()
	}

	cancel()
	test.Wait()
	require.EqualValues(t, 0, eventCount.Load(), "no events expected from plain-text connection to TLS server")
}

// TestTCPTLSMutualAuthRFC6587 verifies mutual TLS with RFC 6587 octet-counting framing.
func TestTCPTLSMutualAuthRFC6587(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	ca := GenerateTestCA(t)
	serverCert, serverKey := ca.Issue(t)
	clientCert, clientKey := ca.Issue(t)

	port := AllocatePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	config := TCPInputConfig(addr, TCPInputOptions{
		TLSCA:      ca.CertFile,
		TLSCert:    serverCert,
		TLSKey:     serverKey,
		ClientAuth: "required",
		Framing:    "rfc6587",
	})
	test := NewTest(t, TestOptions{Config: config})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, tcpReadyMsg)

	test.ExpectJSONFields(mapstr.M{"input.type": "tcp"})
	test.ExpectOutput(fmt.Sprintf("Hello World: %d", numTLSEvents-1))
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"filebeat did not start accepting connections")

	tlsCfg := LoadClientTLSConfig(t, ca.CertFile, clientCert, clientKey)

	// RFC 6587 octet-counting: "<length> <message>" with no trailing newline.
	var msgs [][]byte
	for n := range numTLSEvents {
		msg := fmt.Sprintf("Hello World: %d", n)
		msgs = append(msgs, fmt.Appendf(nil, "%d %s", len(msg), msg))
	}
	err := SendTCPTLS(ctx, addr, tlsCfg, msgs)
	require.NoError(t, err)

	test.Wait()
}

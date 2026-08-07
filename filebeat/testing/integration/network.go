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

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// AllocatePort returns a free TCP port on 127.0.0.1.
// There is a small TOCTOU window between returning and Filebeat binding to it.
func AllocatePort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("AllocatePort: unexpected address type %T", l.Addr())
	}
	port := tcpAddr.Port
	if err := l.Close(); err != nil {
		t.Fatalf("AllocatePort: close: %v", err)
	}
	return port
}

// TCPInputOptions configures a TCP input.
type TCPInputOptions struct {
	// LineDelimiter separates events. Empty means the default (newline).
	LineDelimiter string
	// Framing, when non-empty, enables the named framing mode (e.g. "rfc6587").
	Framing string
	// TLS cert paths; all three must be set to enable TLS.
	TLSCA, TLSCert, TLSKey string
	// ClientAuth sets ssl.client_authentication ("optional" or "required").
	ClientAuth string
}

// TCPInputConfig returns a Filebeat YAML config for a TCP input on addr.
func TCPInputConfig(addr string, opts TCPInputOptions) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "filebeat.inputs:\n  - type: tcp\n    host: %q\n    enabled: true\n", addr)
	if opts.LineDelimiter != "" {
		fmt.Fprintf(&sb, "    line_delimiter: %q\n", opts.LineDelimiter)
	}
	if opts.Framing != "" {
		fmt.Fprintf(&sb, "    framing: %s\n", opts.Framing)
	}
	if opts.TLSCA != "" {
		fmt.Fprintf(&sb, "    ssl.certificate_authorities: %s\n", opts.TLSCA)
		fmt.Fprintf(&sb, "    ssl.certificate: %s\n", opts.TLSCert)
		fmt.Fprintf(&sb, "    ssl.key: %s\n", opts.TLSKey)
	}
	if opts.ClientAuth != "" {
		fmt.Fprintf(&sb, "    ssl.client_authentication: %s\n", opts.ClientAuth)
	}
	sb.WriteString("\noutput.console:\n  enabled: true\n")
	return sb.String()
}

// UDPInputConfig returns a Filebeat YAML config for a UDP input on addr.
func UDPInputConfig(addr string) string {
	return fmt.Sprintf(
		"filebeat.inputs:\n  - type: udp\n    host: %q\n    enabled: true\n\noutput.console:\n  enabled: true\n",
		addr,
	)
}

// UnixInputOptions configures a Unix socket input.
type UnixInputOptions struct {
	// SocketType is "stream" (default) or "datagram".
	SocketType string
	// LineDelimiter separates events in stream mode. Empty means the default (newline).
	LineDelimiter string
}

// UnixInputConfig returns a Filebeat YAML config for a Unix socket input at sockPath.
func UnixInputConfig(sockPath string, opts UnixInputOptions) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "filebeat.inputs:\n  - type: unix\n    path: %s\n    enabled: true\n", sockPath)
	if opts.SocketType != "" {
		fmt.Fprintf(&sb, "    socket_type: %s\n", opts.SocketType)
	}
	if opts.LineDelimiter != "" {
		fmt.Fprintf(&sb, "    line_delimiter: %q\n", opts.LineDelimiter)
	}
	sb.WriteString("\noutput.console:\n  enabled: true\n")
	return sb.String()
}

// SyslogProtocol identifies the network protocol used for a syslog input.
type SyslogProtocol string

const (
	SyslogTCP  SyslogProtocol = "tcp"
	SyslogUDP  SyslogProtocol = "udp"
	SyslogUnix SyslogProtocol = "unix"
)

// SyslogInputOptions configures a syslog input.
type SyslogInputOptions struct {
	// Protocol is one of SyslogTCP, SyslogUDP, or SyslogUnix.
	Protocol SyslogProtocol
	// Addr is "host:port" for TCP/UDP, or a socket path for Unix.
	Addr string
	// SocketType is "stream" or "datagram" (Unix only).
	SocketType string
}

// SyslogInputConfig returns a Filebeat YAML config for a syslog input.
func SyslogInputConfig(opts SyslogInputOptions) string {
	var sb strings.Builder
	sb.WriteString("filebeat.inputs:\n  - type: syslog\n    protocol:\n")
	switch opts.Protocol {
	case SyslogTCP, SyslogUDP:
		fmt.Fprintf(&sb, "      %s:\n        host: %q\n", opts.Protocol, opts.Addr)
	case SyslogUnix:
		fmt.Fprintf(&sb, "      unix:\n        path: %s\n", opts.Addr)
		if opts.SocketType != "" {
			fmt.Fprintf(&sb, "        socket_type: %s\n", opts.SocketType)
		}
	default:
		panic(fmt.Sprintf("SyslogInputConfig: unknown protocol %q", opts.Protocol))
	}
	sb.WriteString("\noutput.console:\n  enabled: true\n")
	return sb.String()
}

// StdinInputConfig returns a Filebeat YAML config that reads from stdin.
func StdinInputConfig() string {
	return "filebeat.inputs:\n  - type: stdin\n    enabled: true\n\noutput.console:\n  enabled: true\n"
}

// LoadClientTLSConfig builds a *tls.Config for a test client that trusts caFile
// and (optionally) presents certFile/keyFile as its client certificate.
// Pass empty strings for certFile/keyFile when no client certificate is needed.
func LoadClientTLSConfig(t *testing.T, caFile, certFile, keyFile string) *tls.Config {
	t.Helper()
	caData, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatalf("LoadClientTLSConfig: read CA %q: %v", caFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		t.Fatalf("LoadClientTLSConfig: AppendCertsFromPEM failed for %q", caFile)
	}
	cfg := &tls.Config{
		RootCAs: pool,
	}
	if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			t.Fatalf("LoadClientTLSConfig: LoadX509KeyPair(%q, %q): %v", certFile, keyFile, err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg
}

// SendTCP connects to addr over plain TCP and sends each byte slice in messages.
func SendTCP(t *testing.T, addr string, messages [][]byte) {
	t.Helper()
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("SendTCP: dial %s: %v", addr, err)
	}
	defer conn.Close()
	for _, msg := range messages {
		if _, err := conn.Write(msg); err != nil {
			t.Fatalf("SendTCP: write: %v", err)
		}
	}
}

// SendTCPTLS connects to addr over TLS using tlsConfig and sends messages.
// Returns the first error encountered (connection or write).
func SendTCPTLS(ctx context.Context, addr string, tlsConfig *tls.Config, messages [][]byte) error {
	conn, err := (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	for _, msg := range messages {
		if _, err := conn.Write(msg); err != nil {
			return fmt.Errorf("SendTCPTLS write: %w", err)
		}
	}
	return nil
}

// SendUDP sends each byte slice in messages as a UDP datagram to addr.
func SendUDP(t *testing.T, addr string, messages [][]byte) {
	t.Helper()
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "udp", addr)
	if err != nil {
		t.Fatalf("SendUDP: dial %s: %v", addr, err)
	}
	defer conn.Close()
	for _, msg := range messages {
		if _, err := conn.Write(msg); err != nil {
			t.Fatalf("SendUDP: write: %v", err)
		}
	}
}

// SendUnixStream connects to sockPath via a Unix stream socket and sends messages.
func SendUnixStream(t *testing.T, sockPath string, messages [][]byte) {
	t.Helper()
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "unix", sockPath)
	if err != nil {
		t.Fatalf("SendUnixStream: dial %s: %v", sockPath, err)
	}
	defer conn.Close()
	for _, msg := range messages {
		if _, err := conn.Write(msg); err != nil {
			t.Fatalf("SendUnixStream: write: %v", err)
		}
	}
}

// WaitForUnixStreamSocket polls until a connection to the Unix stream socket at
// sockPath succeeds, confirming that both bind and listen have completed.
func WaitForUnixStreamSocket(t *testing.T, sockPath string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := (&net.Dialer{}).DialContext(t.Context(), "unix", sockPath)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 30*time.Second, 50*time.Millisecond, "unix stream socket not ready: %s", sockPath)
}

// WaitForUnixDatagramSocket polls until the socket file at sockPath exists,
// confirming that bind has completed and the datagram socket is ready to receive.
func WaitForUnixDatagramSocket(t *testing.T, sockPath string) {
	t.Helper()
	require.Eventually(t, func() bool {
		_, err := os.Stat(sockPath)
		return err == nil
	}, 30*time.Second, 50*time.Millisecond, "unix datagram socket not ready: %s", sockPath)
}

// SendUnixDatagram sends each byte slice in messages as a Unix datagram to sockPath.
func SendUnixDatagram(t *testing.T, sockPath string, messages [][]byte) {
	t.Helper()
	addr := &net.UnixAddr{Name: sockPath, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		t.Fatalf("SendUnixDatagram: dial %s: %v", sockPath, err)
	}
	defer conn.Close()
	for _, msg := range messages {
		if _, err := conn.Write(msg); err != nil {
			t.Fatalf("SendUnixDatagram: write: %v", err)
		}
	}
}

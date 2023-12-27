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

package tcp

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	"github.com/elastic/beats/v7/libbeat/beat"
	btesting "github.com/elastic/beats/v7/libbeat/testing"
)

func testTCPCheck(t *testing.T, host string, port uint16) *beat.Event {
	config := mapstr.M{
		"hosts":   host,
		"ports":   port,
		"timeout": "1s",
	}
	return testTCPConfigCheck(t, config)
}

// TestUpEndpointJob tests an up endpoint configured using either direct lookups or IPs
func TestUpEndpointJob(t *testing.T) {
	// Test with domain, IPv4 and IPv6
	scenarios := []struct {
		name       string
		hostname   string
		isIP       bool
		expectedIP string
	}{
		{
			name:       "localhost",
			hostname:   "localhost",
			isIP:       false,
			expectedIP: "127.0.0.1",
		},
		{
			name:       "ipv4",
			hostname:   "127.0.0.1",
			isIP:       true,
			expectedIP: "127.0.0.1",
		},
		{
			name:     "ipv6",
			hostname: "::1",
			isIP:     true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			server, port, err := setupServer(t, func(handler http.Handler) (*httptest.Server, error) {
				return newHostTestServer(handler, scenario.hostname)
			})
			// Some machines don't have ipv6 setup correctly, so we ignore the test
			// if we can't bind to the port / setup the server.
			if err != nil && scenario.hostname == "::1" {
				return
			}
			require.NoError(t, err)

			defer server.Close()

			hostURL := &url.URL{Scheme: "tcp", Host: net.JoinHostPort(scenario.hostname, strconv.Itoa(int(port)))}

			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			event := testTCPCheck(t, hostURL.String(), port)

			validators := []validator.Validator{
				hbtest.BaseChecks(serverURL.Hostname(), "up", "tcp"),
				hbtest.SummaryStateChecks(1, 0),
				hbtest.URLChecks(t, hostURL),
				hbtest.RespondingTCPChecks(),
			}

			if !scenario.isIP {
				validators = append(validators, hbtest.ResolveChecks(scenario.expectedIP))
			}

			testslike.Test(
				t,
				lookslike.Strict(lookslike.Compose(validators...)),
				event.Fields,
			)
		})
	}
}

func TestConnectionRefusedEndpointJob(t *testing.T) {
	ip := "127.0.0.1"
	port, err := btesting.AvailableTCP4Port()
	require.NoError(t, err)

	event := testTCPCheck(t, ip, port)

	dialErr := fmt.Sprintf("dial tcp %s:%d", ip, port)
	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.SummaryStateChecks(0, 1),
			hbtest.SimpleURLChecks(t, "tcp", ip, port),
			hbtest.ECSErrCodeChecks(ecserr.CODE_NET_COULD_NOT_CONNECT, dialErr),
		)),
		event.Fields,
	)
}

func TestUnreachableEndpointJob(t *testing.T) {
	ip := "203.0.113.1"
	port := uint16(1234)
	event := testTCPCheck(t, ip, port)

	dialErr := fmt.Sprintf("dial tcp %s:%d", ip, port)
	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.SummaryStateChecks(0, 1),
			hbtest.SimpleURLChecks(t, "tcp", ip, port),
			hbtest.ECSErrCodeChecks(ecserr.CODE_NET_COULD_NOT_CONNECT, dialErr),
		)),
		event.Fields,
	)
}

func TestCheckUp(t *testing.T) {
	host, port, ip, closeEcho, err := startEchoServer(t)
	require.NoError(t, err)
	defer closeEcho() //nolint:errcheck // not needed in test

	configMap := mapstr.M{
		"hosts":         host,
		"ports":         port,
		"timeout":       "1s",
		"check.receive": "echo123",
		"check.send":    "echo123",
	}

	event := testTCPConfigCheck(t, configMap)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks(ip, "up", "tcp"),
			hbtest.RespondingTCPChecks(),
			hbtest.SimpleURLChecks(t, "tcp", host, port),
			hbtest.SummaryStateChecks(1, 0),
			hbtest.ResolveChecks(ip),
			lookslike.MustCompile(map[string]interface{}{
				"tcp": map[string]interface{}{
					"rtt.validate.us": hbtestllext.IsInt64,
				},
			}),
		)),
		event.Fields,
	)
}

func TestCheckDown(t *testing.T) {
	host, port, ip, closeEcho, err := startEchoServer(t)
	require.NoError(t, err)
	defer closeEcho() //nolint:errcheck // not needed in test

	configMap := mapstr.M{
		"hosts":         host,
		"ports":         port,
		"timeout":       "1s",
		"check.receive": "BOOM", // should fail
		"check.send":    "echo123",
	}
	event := testTCPConfigCheck(t, configMap)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.RespondingTCPChecks(),
			hbtest.SimpleURLChecks(t, "tcp", host, port),
			hbtest.SummaryStateChecks(0, 1),
			hbtest.ResolveChecks(ip),
			lookslike.MustCompile(map[string]interface{}{
				"tcp": map[string]interface{}{
					"rtt.validate.us": hbtestllext.IsInt64,
				},
				"error": map[string]interface{}{
					"type":    "validate",
					"message": "received string mismatch",
				},
			}),
		)), event.Fields)
}

func TestNXDomainJob(t *testing.T) {
	host := "notadomainatallforsure.notadomain.notatldreally"
	port := uint16(1234)
	event := testTCPCheck(t, host, port)

	dialErr := fmt.Sprintf("lookup %s", host)
	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks("", "down", "tcp"),
			hbtest.SummaryStateChecks(0, 1),
			hbtest.SimpleURLChecks(t, "tcp", host, port),
			hbtest.ErrorChecks(dialErr, "io"),
		)),
		event.Fields,
	)
}

// startEchoServer starts a simple TCP echo server for testing. Only handles a single connection once.
// Note you MUST connect to this server exactly once to avoid leaking a goroutine. This is only useful
// for the specific tests used here.
func startEchoServer(t *testing.T) (host string, port uint16, ip string, close func() error, err error) {
	// Simple echo server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", 0, "", nil, err
	}

	go func() {
		conn, err := listener.Accept()
		require.NoError(t, err)
		buf := make([]byte, 1024)
		rlen, err := conn.Read(buf)
		require.NoError(t, err)
		wlen, err := conn.Write(buf[:rlen])
		require.NoError(t, err)
		// Normally we'd retry partial writes, but for tests this is OK
		require.Equal(t, wlen, rlen)
		conn.Close()
	}()

	ip, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		listener.Close()
		return "", 0, "", nil, err
	}
	portUint64, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		listener.Close()
		return "", 0, "", nil, err
	}

	return "localhost", uint16(portUint64), ip, listener.Close, nil
}

// StaticResolver allows for a custom in-memory mapping of hosts to IPs, it ignores network names
// and zones.
type StaticResolver struct {
	mapping map[string][]net.IP
}

func NewStaticResolver(mapping map[string][]net.IP) StaticResolver {
	return StaticResolver{mapping}
}

func (s StaticResolver) ResolveIPAddr(network string, host string) (*net.IPAddr, error) {
	found, err := s.LookupIP(host)
	if err != nil {
		return nil, err
	}
	return &net.IPAddr{IP: found[0]}, nil
}

func (s StaticResolver) LookupIP(host string) ([]net.IP, error) {
	if found, ok := s.mapping[host]; ok {
		return found, nil
	} else {
		return nil, makeStaticNXDomainErr(host)
	}
}

func makeStaticNXDomainErr(host string) *net.DNSError {
	return &net.DNSError{
		IsNotFound: true,
		Err:        fmt.Sprintf("Hostname '%s' not found in static resolver", host),
	}
}

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
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	btesting "github.com/elastic/beats/libbeat/testing"
)

func testTCPCheck(t *testing.T, host string, port uint16) *beat.Event {
	config := common.MapStr{
		"hosts":   host,
		"ports":   port,
		"timeout": "1s",
	}
	return testTCPConfigCheck(t, config, host, port)
}

func testTCPConfigCheck(t *testing.T, configMap common.MapStr, host string, port uint16) *beat.Event {
	config, err := common.NewConfigFrom(configMap)
	require.NoError(t, err)

	jobs, endpoints, err := create("tcp", config)
	require.NoError(t, err)

	job := wrappers.WrapCommon(jobs, "test", "", "tcp")[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	require.Equal(t, 1, endpoints)

	return event
}

func testTLSTCPCheck(t *testing.T, host string, port uint16, certFileName string) *beat.Event {
	config, err := common.NewConfigFrom(common.MapStr{
		"hosts":   host,
		"ports":   int64(port),
		"ssl":     common.MapStr{"certificate_authorities": certFileName},
		"timeout": "1s",
	})
	require.NoError(t, err)

	jobs, endpoints, err := create("tcp", config)
	require.NoError(t, err)

	job := wrappers.WrapCommon(jobs, "test", "", "tcp")[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	require.Equal(t, 1, endpoints)

	return event
}

func setupServer(t *testing.T, serverCreator func(http.Handler) *httptest.Server) (*httptest.Server, uint16) {
	server := serverCreator(hbtest.HelloWorldHandler(200))

	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	return server, port
}

func TestUpEndpointJob(t *testing.T) {
	server, port := setupServer(t, httptest.NewServer)
	defer server.Close()

	event := testTCPCheck(t, "localhost", port)

	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.BaseChecks("127.0.0.1", "up", "tcp"),
			hbtest.SummaryChecks(1, 0),
			hbtest.SimpleURLChecks(t, "tcp", "localhost", port),
			hbtest.RespondingTCPChecks(),
			mapval.MustCompile(mapval.Map{
				"resolve": mapval.Map{
					"ip":     "127.0.0.1",
					"rtt.us": mapval.IsDuration,
				},
			}),
		)),
		event.Fields,
	)
}

func TestTLSConnection(t *testing.T) {
	// Start up a TLS Server
	server, port := setupServer(t, httptest.NewTLSServer)
	defer server.Close()

	// Parse its URL
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Determine the IP address the server's hostname resolves to
	ips, err := net.LookupHost(serverURL.Hostname())
	require.NoError(t, err)
	require.Len(t, ips, 1)
	ip := ips[0]

	// Parse the cert so we can test against it
	cert, err := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	require.NoError(t, err)

	// Save the server's cert to a file so heartbeat can use it
	certFile := hbtest.CertToTempFile(t, cert)
	require.NoError(t, certFile.Close())
	defer os.Remove(certFile.Name())

	event := testTLSTCPCheck(t, ip, port, certFile.Name())
	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.TLSChecks(0, 0, cert),
			hbtest.RespondingTCPChecks(),
			hbtest.BaseChecks(ip, "up", "tcp"),
			hbtest.SummaryChecks(1, 0),
			hbtest.SimpleURLChecks(t, "ssl", serverURL.Hostname(), port),
		)),
		event.Fields,
	)
}

func TestConnectionRefusedEndpointJob(t *testing.T) {
	ip := "127.0.0.1"
	port, err := btesting.AvailableTCP4Port()
	require.NoError(t, err)

	event := testTCPCheck(t, ip, port)

	dialErr := fmt.Sprintf("dial tcp %s:%d", ip, port)
	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.SummaryChecks(0, 1),
			hbtest.SimpleURLChecks(t, "tcp", ip, port),
			hbtest.ErrorChecks(dialErr, "io"),
		)),
		event.Fields,
	)
}

func TestUnreachableEndpointJob(t *testing.T) {
	ip := "203.0.113.1"
	port := uint16(1234)
	event := testTCPCheck(t, ip, port)

	dialErr := fmt.Sprintf("dial tcp %s:%d", ip, port)
	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.SummaryChecks(0, 1),
			hbtest.SimpleURLChecks(t, "tcp", ip, port),
			hbtest.ErrorChecks(dialErr, "io"),
		)),
		event.Fields,
	)
}

func TestCheckUp(t *testing.T) {
	host, port, ip, closeEcho, err := startEchoServer(t)
	require.NoError(t, err)
	defer closeEcho()

	configMap := common.MapStr{
		"hosts":         host,
		"ports":         port,
		"timeout":       "1s",
		"check.receive": "echo123",
		"check.send":    "echo123",
	}

	event := testTCPConfigCheck(t, configMap, host, port)

	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.BaseChecks(ip, "up", "tcp"),
			hbtest.RespondingTCPChecks(),
			hbtest.SimpleURLChecks(t, "tcp", host, port),
			hbtest.SummaryChecks(1, 0),
			mapval.MustCompile(mapval.Map{
				"resolve": mapval.Map{
					"ip":     ip,
					"rtt.us": mapval.IsDuration,
				},
				"tcp": mapval.Map{
					"rtt.validate.us": mapval.IsDuration,
				},
			}),
		)),
		event.Fields,
	)
}

func TestCheckDown(t *testing.T) {
	host, port, ip, closeEcho, err := startEchoServer(t)
	require.NoError(t, err)
	defer closeEcho()

	configMap := common.MapStr{
		"hosts":         host,
		"ports":         port,
		"timeout":       "1s",
		"check.receive": "BOOM", // should fail
		"check.send":    "echo123",
	}

	event := testTCPConfigCheck(t, configMap, host, port)

	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.RespondingTCPChecks(),
			hbtest.SimpleURLChecks(t, "tcp", host, port),
			hbtest.SummaryChecks(0, 1),
			mapval.MustCompile(mapval.Map{
				"resolve": mapval.Map{
					"ip":     ip,
					"rtt.us": mapval.IsDuration,
				},
				"tcp": mapval.Map{
					"rtt.validate.us": mapval.IsDuration,
				},
				"error": mapval.Map{
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
	mapval.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.BaseChecks("", "down", "tcp"),
			hbtest.SummaryChecks(0, 1),
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
	}()

	ip, portStr, err := net.SplitHostPort(listener.Addr().String())
	portUint64, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		listener.Close()
		return "", 0, "", nil, err
	}

	return "localhost", uint16(portUint64), ip, listener.Close, nil
}

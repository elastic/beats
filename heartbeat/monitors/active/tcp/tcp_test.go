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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	btesting "github.com/elastic/beats/libbeat/testing"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func testTCPCheck(t *testing.T, host string, port uint16) *beat.Event {
	config, err := common.NewConfigFrom(common.MapStr{
		"hosts":   host,
		"ports":   port,
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

func tcpMonitorChecks(host string, ip string, port uint16, status string) mapval.Validator {
	return hbtest.BaseChecks(ip, status, "tcp")
}

func TestUpEndpointJob(t *testing.T) {
	server, port := setupServer(t, httptest.NewServer)
	defer server.Close()

	event := testTCPCheck(t, "localhost", port)

	mapvaltest.Test(
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
	mapvaltest.Test(
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
	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			tcpMonitorChecks(ip, ip, port, "down"),
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
	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			tcpMonitorChecks(ip, ip, port, "down"),
			hbtest.SummaryChecks(0, 1),
			hbtest.SimpleURLChecks(t, "tcp", ip, port),
			hbtest.ErrorChecks(dialErr, "io"),
		)),
		event.Fields,
	)
}

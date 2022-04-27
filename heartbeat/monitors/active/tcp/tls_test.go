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
	"crypto/tls"
	"crypto/x509"
	"math/bits"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
)

// Tests that we can check a TLS connection with a cert for a SAN IP
func TestTLSSANIPConnection(t *testing.T) {
	if runtime.GOOS == "windows" && bits.UintSize == 32 {
		t.Skip("flaky test: https://github.com/elastic/beats/issues/25857")
	}
	ip, port, cert, certFile, teardown := setupTLSTestServer(t)
	defer teardown()

	event := testTLSTCPCheck(t, ip, port, certFile.Name(), monitors.NewStdResolver())
	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.TLSChecks(0, 0, cert),
			hbtest.RespondingTCPChecks(),
			hbtest.BaseChecks(ip, "up", "tcp"),
			hbtest.SummaryChecks(1, 0),
			hbtest.SimpleURLChecks(t, "ssl", ip, port),
		)),
		event.Fields,
	)
}

func TestTLSHostname(t *testing.T) {
	ip, port, cert, certFile, teardown := setupTLSTestServer(t)
	defer teardown()

	hostname := cert.DNSNames[0] // Should be example.com
	resolver := NewStaticResolver(map[string][]net.IP{hostname: []net.IP{net.ParseIP(ip)}})
	event := testTLSTCPCheck(t, hostname, port, certFile.Name(), resolver)
	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.TLSChecks(0, 0, cert),
			hbtest.RespondingTCPChecks(),
			hbtest.BaseChecks(ip, "up", "tcp"),
			hbtest.SummaryChecks(1, 0),
			hbtest.SimpleURLChecks(t, "ssl", hostname, port),
			hbtest.ResolveChecks(ip),
		)),
		event.Fields,
	)
}

func TestTLSInvalidCert(t *testing.T) {
	ip, port, cert, certFile, teardown := setupTLSTestServer(t)
	defer teardown()

	mismatchedHostname := "notadomain.elastic.co"
	resolver := NewStaticResolver(
		map[string][]net.IP{
			cert.DNSNames[0]:   {net.ParseIP(ip)},
			mismatchedHostname: {net.ParseIP(ip)},
		},
	)
	event := testTLSTCPCheck(t, mismatchedHostname, port, certFile.Name(), resolver)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.RespondingTCPChecks(),
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.SummaryChecks(0, 1),
			hbtest.SimpleURLChecks(t, "ssl", mismatchedHostname, port),
			hbtest.ResolveChecks(ip),
			lookslike.MustCompile(map[string]interface{}{
				"error": map[string]interface{}{
					"message": x509.HostnameError{Certificate: cert, Host: mismatchedHostname}.Error(),
					"type":    "io",
				},
			}),
		)),
		event.Fields,
	)
}

func TestTLSExpiredCert(t *testing.T) {
	certFile := "../fixtures/expired.cert"
	tlsCert, err := tls.LoadX509KeyPair(certFile, "../fixtures/expired.key")
	require.NoError(t, err)

	ip, portStr, cert, closeSrv := hbtest.StartHTTPSServer(t, tlsCert)
	defer closeSrv()

	portInt, err := strconv.Atoi(portStr)
	port := uint16(portInt)
	require.NoError(t, err)

	host := "localhost"
	event := testTLSTCPCheck(t, host, port, certFile, monitors.NewStdResolver())

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.RespondingTCPChecks(),
			hbtest.BaseChecks(ip, "down", "tcp"),
			hbtest.SummaryChecks(0, 1),
			hbtest.SimpleURLChecks(t, "ssl", host, port),
			hbtest.ResolveChecks(ip),
			hbtest.ExpiredCertChecks(cert),
		)),
		event.Fields,
	)
}

func setupTLSTestServer(t *testing.T) (ip string, port uint16, cert *x509.Certificate, certFile *os.File, teardown func()) {
	// Start up a TLS Server
	server, port, err := setupServer(t, func(handler http.Handler) (*httptest.Server, error) {
		return httptest.NewTLSServer(handler), nil
	})
	require.NoError(t, err)

	// Parse its URL
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Determine the IP address the server's hostname resolves to
	ips, err := net.LookupHost(serverURL.Hostname())
	require.NoError(t, err)
	require.Len(t, ips, 1)
	ip = ips[0]

	// Parse the cert so we can test against it
	cert, err = x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	require.NoError(t, err)

	// Save the server's cert to a file so heartbeat can use it
	certFile = hbtest.CertToTempFile(t, cert)
	require.NoError(t, certFile.Close())

	return ip, port, cert, certFile, func() {
		defer server.Close()
		err := os.Remove(certFile.Name())
		require.NoError(t, err)
	}
}

func testTLSTCPCheck(t *testing.T, host string, port uint16, certFileName string, resolver monitors.Resolver) *beat.Event {
	config, err := conf.NewConfigFrom(common.MapStr{
		"hosts":   host,
		"ports":   int64(port),
		"ssl":     common.MapStr{"certificate_authorities": certFileName},
		"timeout": "1s",
	})
	require.NoError(t, err)

	p, err := createWithResolver(config, resolver)
	require.NoError(t, err)

	sched := schedule.MustParse("@every 1s")
	job := wrappers.WrapCommon(p.Jobs, stdfields.StdMonitorFields{ID: "test", Type: "tcp", Schedule: sched, Timeout: 1})[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	require.Equal(t, 1, p.Endpoints)

	return event
}

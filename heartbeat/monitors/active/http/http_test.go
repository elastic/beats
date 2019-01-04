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

package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/elastic/beats/libbeat/common/file"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	btesting "github.com/elastic/beats/libbeat/testing"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func testRequest(t *testing.T, testURL string) *beat.Event {
	return testTLSRequest(t, testURL, nil)
}

// testTLSRequest tests the given request. certPath is optional, if given
// an empty string no cert will be set.
func testTLSRequest(t *testing.T, testURL string, extraConfig map[string]interface{}) *beat.Event {
	configSrc := map[string]interface{}{
		"urls":    testURL,
		"timeout": "1s",
	}

	if extraConfig != nil {
		for k, v := range extraConfig {
			configSrc[k] = v
		}
	}

	config, err := common.NewConfigFrom(configSrc)
	require.NoError(t, err)

	jobs, endpoints, err := create("tls", config)
	require.NoError(t, err)

	job := jobs[0]

	event := &beat.Event{}
	_, err = job.Run(event)
	require.NoError(t, err)

	require.Equal(t, 1, endpoints)

	return event
}

func checkServer(t *testing.T, handlerFunc http.HandlerFunc) (*httptest.Server, *beat.Event) {
	server := httptest.NewServer(handlerFunc)
	defer server.Close()
	event := testRequest(t, server.URL)

	return server, event
}

// The minimum response is just the URL. Only to be used for unreachable server
// tests.
func httpBaseChecks(url string) mapval.Validator {
	return mapval.MustCompile(mapval.Map{
		"http.url": url,
	})
}

func respondingHTTPChecks(url string, statusCode int) mapval.Validator {
	return mapval.Compose(
		httpBaseChecks(url),
		mapval.MustCompile(mapval.Map{
			"http": mapval.Map{
				"response.status_code":   statusCode,
				"rtt.content.us":         mapval.IsDuration,
				"rtt.response_header.us": mapval.IsDuration,
				"rtt.total.us":           mapval.IsDuration,
				"rtt.validate.us":        mapval.IsDuration,
				"rtt.write_request.us":   mapval.IsDuration,
			},
		}),
	)
}

var upStatuses = []int{
	// 1xx
	http.StatusContinue,
	http.StatusSwitchingProtocols,
	http.StatusProcessing,
	// 2xx
	http.StatusOK,
	http.StatusCreated,
	http.StatusAccepted,
	http.StatusNonAuthoritativeInfo,
	http.StatusNoContent,
	http.StatusResetContent,
	http.StatusPartialContent,
	http.StatusMultiStatus,
	http.StatusAlreadyReported,
	http.StatusIMUsed,
	// 3xx
	http.StatusMultipleChoices,
	http.StatusMovedPermanently,
	http.StatusFound,
	http.StatusSeeOther,
	http.StatusNotModified,
	http.StatusUseProxy,
	http.StatusTemporaryRedirect,
	http.StatusPermanentRedirect,
}

var downStatuses = []int{
	//4xx
	http.StatusBadRequest,
	http.StatusUnauthorized,
	http.StatusPaymentRequired,
	http.StatusForbidden,
	http.StatusNotFound,
	http.StatusMethodNotAllowed,
	http.StatusNotAcceptable,
	http.StatusProxyAuthRequired,
	http.StatusRequestTimeout,
	http.StatusConflict,
	http.StatusGone,
	http.StatusLengthRequired,
	http.StatusPreconditionFailed,
	http.StatusRequestEntityTooLarge,
	http.StatusRequestURITooLong,
	http.StatusUnsupportedMediaType,
	http.StatusRequestedRangeNotSatisfiable,
	http.StatusExpectationFailed,
	http.StatusTeapot,
	http.StatusUnprocessableEntity,
	http.StatusLocked,
	http.StatusFailedDependency,
	http.StatusUpgradeRequired,
	http.StatusPreconditionRequired,
	http.StatusTooManyRequests,
	http.StatusRequestHeaderFieldsTooLarge,
	http.StatusUnavailableForLegalReasons,
	//5xx,
	http.StatusInternalServerError,
	http.StatusNotImplemented,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
	http.StatusHTTPVersionNotSupported,
	http.StatusVariantAlsoNegotiates,
	http.StatusInsufficientStorage,
	http.StatusLoopDetected,
	http.StatusNotExtended,
	http.StatusNetworkAuthenticationRequired,
}

func TestUpStatuses(t *testing.T) {
	for _, status := range upStatuses {
		status := status
		t.Run(fmt.Sprintf("Test OK HTTP status %d", status), func(t *testing.T) {
			server, event := checkServer(t, hbtest.HelloWorldHandler(status))
			port, err := hbtest.ServerPort(server)
			require.NoError(t, err)

			mapvaltest.Test(
				t,
				mapval.Strict(mapval.Compose(
					hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "http", "up"),
					hbtest.RespondingTCPChecks(port),
					respondingHTTPChecks(server.URL, status),
				)),
				event.Fields,
			)
		})
	}
}

func TestDownStatuses(t *testing.T) {
	for _, status := range downStatuses {
		status := status
		t.Run(fmt.Sprintf("test down status %d", status), func(t *testing.T) {
			server, event := checkServer(t, hbtest.HelloWorldHandler(status))
			port, err := hbtest.ServerPort(server)
			require.NoError(t, err)

			mapvaltest.Test(
				t,
				mapval.Strict(mapval.Compose(
					hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "http", "down"),
					hbtest.RespondingTCPChecks(port),
					respondingHTTPChecks(server.URL, status),
					hbtest.ErrorChecks(fmt.Sprintf("%d", status), "validate"),
				)),
				event.Fields,
			)
		})
	}
}

func TestLargeResponse(t *testing.T) {
	server := httptest.NewServer(hbtest.SizedResponseHandler(1024 * 1024))
	defer server.Close()

	configSrc := map[string]interface{}{
		"urls":                server.URL,
		"timeout":             "1s",
		"check.response.body": "x",
	}

	config, err := common.NewConfigFrom(configSrc)
	require.NoError(t, err)

	jobs, _, err := create("largeresp", config)
	require.NoError(t, err)

	job := jobs[0]

	event := &beat.Event{}
	_, err = job.Run(event)
	require.NoError(t, err)

	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "http", "up"),
			hbtest.RespondingTCPChecks(port),
			respondingHTTPChecks(server.URL, 200),
		)),
		event.Fields,
	)
}

func runHTTPSServerCheck(
	t *testing.T,
	server *httptest.Server,
	reqExtraConfig map[string]interface{}) {
	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	// Parse the cert so we can test against it.
	cert, err := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	require.NoError(t, err)

	// Write the cert to a tempfile so heartbeat can use it in its config.
	certFile := hbtest.CertToTempFile(t, cert)
	require.NoError(t, certFile.Close())
	defer os.Remove(certFile.Name())

	mergedExtraConfig := map[string]interface{}{"ssl.certificate_authorities": certFile.Name()}
	for k, v := range reqExtraConfig {
		mergedExtraConfig[k] = v
	}

	event := testTLSRequest(t, server.URL, mergedExtraConfig)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "https", "up"),
			hbtest.RespondingTCPChecks(port),
			hbtest.TLSChecks(0, 0, cert),
			respondingHTTPChecks(server.URL, http.StatusOK),
		)),
		event.Fields,
	)
}

func TestHTTPSServer(t *testing.T) {
	server := httptest.NewTLSServer(hbtest.HelloWorldHandler(http.StatusOK))

	runHTTPSServerCheck(t, server, nil)
}

func TestHTTPSx509Auth(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	clientKeyPath := path.Join(wd, "testdata", "client_key.pem")
	clientCertPath := path.Join(wd, "testdata", "client_cert.pem")

	certReader, err := file.ReadOpen(clientCertPath)
	require.NoError(t, err)

	clientCertBytes, err := ioutil.ReadAll(certReader)
	require.NoError(t, err)

	clientCerts := x509.NewCertPool()
	certAdded := clientCerts.AppendCertsFromPEM(clientCertBytes)
	require.True(t, certAdded)

	tlsConf := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCerts,
		MinVersion: tls.VersionTLS12,
	}
	tlsConf.BuildNameToCertificate()

	server := httptest.NewUnstartedServer(hbtest.HelloWorldHandler(http.StatusOK))
	server.TLS = tlsConf
	server.StartTLS()
	defer server.Close()

	runHTTPSServerCheck(
		t,
		server,
		map[string]interface{}{
			"ssl.certificate": clientCertPath,
			"ssl.key":         clientKeyPath,
		},
	)
}

func TestConnRefusedJob(t *testing.T) {
	ip := "127.0.0.1"
	port, err := btesting.AvailableTCP4Port()
	require.NoError(t, err)

	url := fmt.Sprintf("http://%s:%d", ip, port)

	event := testRequest(t, url)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+url, url, ip, "http", "down"),
			hbtest.TCPBaseChecks(port),
			hbtest.ErrorChecks(url, "io"),
			httpBaseChecks(url),
		)),
		event.Fields,
	)
}

func TestUnreachableJob(t *testing.T) {
	// 203.0.113.0/24 is reserved for documentation so should not be routable
	// See: https://tools.ietf.org/html/rfc6890
	ip := "203.0.113.1"
	// Port 80 is sometimes omitted in logs a non-standard one is easier to validate
	port := 1234
	url := fmt.Sprintf("http://%s:%d", ip, port)

	event := testRequest(t, url)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+url, url, ip, "http", "down"),
			hbtest.TCPBaseChecks(uint16(port)),
			hbtest.ErrorChecks(url, "io"),
			httpBaseChecks(url),
		)),
		event.Fields,
	)
}

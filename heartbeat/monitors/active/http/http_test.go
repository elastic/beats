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
	"io"
	"io/ioutil"
	"math/bits"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	btesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"
)

func sendSimpleTLSRequest(t *testing.T, testURL string, useUrls bool) *beat.Event {
	return sendTLSRequest(t, testURL, useUrls, nil)
}

// sendTLSRequest tests the given request. certPath is optional, if given
// an empty string no cert will be set.
func sendTLSRequest(t *testing.T, testURL string, useUrls bool, extraConfig map[string]interface{}) *beat.Event {
	configSrc := map[string]interface{}{
		"timeout": "1s",
	}

	if useUrls {
		configSrc["urls"] = testURL
	} else {
		configSrc["hosts"] = testURL
	}

	if extraConfig != nil {
		for k, v := range extraConfig {
			configSrc[k] = v
		}
	}

	config, err := common.NewConfigFrom(configSrc)
	require.NoError(t, err)

	p, err := create("tls", config)
	require.NoError(t, err)

	sched := schedule.MustParse("@every 1s")
	job := wrappers.WrapCommon(p.Jobs, stdfields.StdMonitorFields{ID: "tls", Type: "http", Schedule: sched, Timeout: 1})[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	require.Equal(t, 1, p.Endpoints)

	return event
}

func checkServer(t *testing.T, handlerFunc http.HandlerFunc, useUrls bool) (*httptest.Server, *beat.Event) {
	server := httptest.NewServer(handlerFunc)
	defer server.Close()
	event := sendSimpleTLSRequest(t, server.URL, useUrls)

	return server, event
}

// The minimum response is just the URL. Only to be used for unreachable server
// tests.
func urlChecks(urlStr string) validator.Validator {
	u, _ := url.Parse(urlStr)
	return lookslike.MustCompile(map[string]interface{}{
		"url": wrappers.URLFields(u),
	})
}

func respondingHTTPChecks(url, mimeType string, statusCode int) validator.Validator {
	return lookslike.Compose(
		minimalRespondingHTTPChecks(url, mimeType, statusCode),
		respondingHTTPStatusAndTimingChecks(statusCode),
		respondingHTTPHeaderChecks(),
	)
}

func respondingHTTPStatusAndTimingChecks(statusCode int) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"http": map[string]interface{}{
			"response.status_code":   statusCode,
			"rtt.content.us":         isdef.IsDuration,
			"rtt.response_header.us": isdef.IsDuration,
			"rtt.total.us":           isdef.IsDuration,
			"rtt.validate.us":        isdef.IsDuration,
			"rtt.write_request.us":   isdef.IsDuration,
		},
	})
}

func minimalRespondingHTTPChecks(url, mimeType string, statusCode int) validator.Validator {
	return lookslike.Compose(
		urlChecks(url),
		httpBodyChecks(),
		lookslike.MustCompile(map[string]interface{}{
			"http": map[string]interface{}{
				"response.mime_type":   mimeType,
				"response.status_code": statusCode,
				"rtt.total.us":         isdef.IsDuration,
			},
		}),
	)
}

func httpBodyChecks() validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"http.response.body.bytes": isdef.IsIntGt(-1),
		"http.response.body.hash":  isdef.IsString,
	})
}

func respondingHTTPBodyChecks(body string) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"http.response.body.content": body,
		"http.response.body.bytes":   len(body),
	})
}

func respondingHTTPHeaderChecks() validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"http.response.headers": map[string]interface{}{
			"Date":           isdef.IsString,
			"Content-Length": isdef.Optional(isdef.IsString),
			"Content-Type":   isdef.Optional(isdef.IsString),
			"Location":       isdef.Optional(isdef.IsString),
		},
	})
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
	for _, useURLs := range []bool{true, false} {
		for _, status := range upStatuses {
			status := status

			field := "hosts"
			if useURLs {
				field = "urls"
			}

			testName := fmt.Sprintf("Test OK HTTP status %d using %s config field", status, field)
			t.Run(testName, func(t *testing.T) {
				server, event := checkServer(t, hbtest.HelloWorldHandler(status), useURLs)

				testslike.Test(
					t,
					lookslike.Strict(lookslike.Compose(
						hbtest.BaseChecks("127.0.0.1", "up", "http"),
						hbtest.RespondingTCPChecks(),
						hbtest.SummaryChecks(1, 0),
						respondingHTTPChecks(server.URL, "text/plain; charset=utf-8", status),
					)),
					event.Fields,
				)
			})
		}
	}
}

func TestHeadersDisabled(t *testing.T) {
	server, event := checkServer(t, hbtest.HelloWorldHandler(200), false)
	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks("127.0.0.1", "up", "http"),
			hbtest.RespondingTCPChecks(),
			hbtest.SummaryChecks(1, 0),
			respondingHTTPChecks(server.URL, "text/plain; charset=utf-8", 200),
		)),
		event.Fields,
	)
}

func TestDownStatuses(t *testing.T) {
	for _, status := range downStatuses {
		status := status
		t.Run(fmt.Sprintf("test down status %d", status), func(t *testing.T) {
			server, event := checkServer(t, hbtest.HelloWorldHandler(status), false)

			testslike.Test(
				t,
				lookslike.Strict(lookslike.Compose(
					hbtest.BaseChecks("127.0.0.1", "down", "http"),
					hbtest.RespondingTCPChecks(),
					hbtest.SummaryChecks(0, 1),
					respondingHTTPChecks(server.URL, "text/plain; charset=utf-8", status),
					hbtest.ErrorChecks(fmt.Sprintf("%d", status), "validate"),
					respondingHTTPBodyChecks("hello, world!"),
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
		"hosts":               server.URL,
		"timeout":             "1s",
		"check.response.body": "x",
	}

	config, err := common.NewConfigFrom(configSrc)
	require.NoError(t, err)

	p, err := create("largeresp", config)
	require.NoError(t, err)

	sched, _ := schedule.Parse("@every 1s")
	job := wrappers.WrapCommon(p.Jobs, stdfields.StdMonitorFields{ID: "test", Type: "http", Schedule: sched, Timeout: 1})[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks("127.0.0.1", "up", "http"),
			hbtest.RespondingTCPChecks(),
			hbtest.SummaryChecks(1, 0),
			respondingHTTPChecks(server.URL, "text/plain; charset=utf-8", 200),
		)),
		event.Fields,
	)
}

func runHTTPSServerCheck(
	t *testing.T,
	server *httptest.Server,
	reqExtraConfig map[string]interface{}) {

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

	// Sometimes the test server can take a while to start. Since we're only using this to test up statuses,
	// we give it a few attempts to see if the server can come up before we run the real assertions.
	var event *beat.Event
	for i := 0; i < 10; i++ {
		event = sendTLSRequest(t, server.URL, false, mergedExtraConfig)
		if v, err := event.GetValue("monitor.status"); err == nil && reflect.DeepEqual(v, "up") {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}

	// When connecting through a proxy, the following fields are missing.
	if _, isProxy := reqExtraConfig["proxy_url"]; isProxy {
		missing := map[string]interface{}{
			"http.rtt.response_header.us": time.Duration(0),
			"http.rtt.content.us":         time.Duration(0),
			"monitor.ip":                  "127.0.0.1",
			"tcp.rtt.connect.us":          time.Duration(0),
			"http.rtt.validate.us":        time.Duration(0),
			"http.rtt.write_request.us":   time.Duration(0),
			"tls.rtt.handshake.us":        time.Duration(0),
		}
		for k, v := range missing {
			if found, err := event.Fields.HasKey(k); !found || err != nil {
				event.Fields.Put(k, v)
			}
		}
	}

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks("127.0.0.1", "up", "http"),
			hbtest.RespondingTCPChecks(),
			hbtest.TLSChecks(0, 0, cert),
			hbtest.SummaryChecks(1, 0),
			respondingHTTPChecks(server.URL, "text/plain; charset=utf-8", http.StatusOK),
		)),
		event.Fields,
	)
}

func TestHTTPSServer(t *testing.T) {
	if runtime.GOOS == "windows" && bits.UintSize == 32 {
		t.Skip("flaky test: https://github.com/elastic/beats/issues/25857")
	}
	server := httptest.NewTLSServer(hbtest.HelloWorldHandler(http.StatusOK))

	runHTTPSServerCheck(t, server, nil)
}

func TestExpiredHTTPSServer(t *testing.T) {
	tlsCert, err := tls.LoadX509KeyPair("../fixtures/expired.cert", "../fixtures/expired.key")
	require.NoError(t, err)
	host, port, cert, closeSrv := hbtest.StartHTTPSServer(t, tlsCert)
	defer closeSrv()
	u := &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}

	extraConfig := map[string]interface{}{"ssl.certificate_authorities": "../fixtures/expired.cert"}
	event := sendTLSRequest(t, u.String(), true, extraConfig)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks("127.0.0.1", "down", "http"),
			hbtest.RespondingTCPChecks(),
			hbtest.SummaryChecks(0, 1),
			hbtest.ExpiredCertChecks(cert),
			hbtest.URLChecks(t, &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}),
			// No HTTP fields expected because we fail at the TCP level
		)),
		event.Fields,
	)
}

func TestHTTPSx509Auth(t *testing.T) {
	if runtime.GOOS == "windows" && bits.UintSize == 32 {
		t.Skip("flaky test: https://github.com/elastic/beats/issues/25857")
	}
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

	event := sendSimpleTLSRequest(t, url, false)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks(ip, "down", "http"),
			hbtest.SummaryChecks(0, 1),
			hbtest.ErrorChecks(url, "io"),
			urlChecks(url),
		)),
		event.Fields,
	)
}

func TestUnreachableJob(t *testing.T) {
	// 203.0.113.0/24 is reserved for documentation so should not be routable
	// See: https://tools.ietf.org/html/rfc6890
	ip := "203.0.113.1"
	// Port 80 is sometimes omitted in logs a non-standard one is easier to validate
	port := uint16(1234)
	url := fmt.Sprintf("http://%s:%d", ip, port)

	event := sendSimpleTLSRequest(t, url, false)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks(ip, "down", "http"),
			hbtest.SummaryChecks(0, 1),
			hbtest.ErrorChecks(url, "io"),
			urlChecks(url),
		)),
		event.Fields,
	)
}

func TestRedirect(t *testing.T) {
	redirectingPaths := map[string]string{
		"/redirect_one": "/redirect_two",
		"/redirect_two": "/",
	}
	expectedBody := "TargetBody"
	server := httptest.NewServer(hbtest.RedirectHandler(redirectingPaths, expectedBody))
	defer server.Close()

	testURL := server.URL + "/redirect_one"
	configSrc := map[string]interface{}{
		"urls":                testURL,
		"timeout":             "1s",
		"check.response.body": expectedBody,
		"max_redirects":       10,
	}

	config, err := common.NewConfigFrom(configSrc)
	require.NoError(t, err)

	p, err := create("redirect", config)
	require.NoError(t, err)

	sched, _ := schedule.Parse("@every 1s")
	job := wrappers.WrapCommon(p.Jobs, stdfields.StdMonitorFields{ID: "test", Type: "http", Schedule: sched, Timeout: 1})[0]

	// Run this test multiple times since in the past we had an issue where the redirects
	// list was added onto by each request. See https://github.com/elastic/beats/pull/15944
	for i := 0; i < 10; i++ {
		event := &beat.Event{}
		_, err = job(event)
		require.NoError(t, err)

		testslike.Test(
			t,
			lookslike.Strict(lookslike.Compose(
				hbtest.BaseChecks("", "up", "http"),
				hbtest.SummaryChecks(1, 0),
				minimalRespondingHTTPChecks(testURL, "text/plain; charset=utf-8", 200),
				respondingHTTPHeaderChecks(),
				lookslike.MustCompile(map[string]interface{}{
					// For redirects that are followed we shouldn't record this header because there's no sensible
					// value
					"http.response.headers.Location": isdef.KeyMissing,
					"http.response.redirects": []string{
						server.URL + redirectingPaths["/redirect_one"],
						server.URL + redirectingPaths["/redirect_two"],
					},
				}),
			)),
			event.Fields,
		)
	}
}

func TestNoHeaders(t *testing.T) {
	server := httptest.NewServer(hbtest.HelloWorldHandler(200))
	defer server.Close()

	configSrc := map[string]interface{}{
		"urls":                     server.URL,
		"response.include_headers": false,
	}

	config, err := common.NewConfigFrom(configSrc)
	require.NoError(t, err)

	p, err := create("http", config)
	require.NoError(t, err)

	sched, _ := schedule.Parse("@every 1s")
	job := wrappers.WrapCommon(p.Jobs, stdfields.StdMonitorFields{ID: "test", Type: "http", Schedule: sched, Timeout: 1})[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	testslike.Test(
		t,
		lookslike.Strict(lookslike.Compose(
			hbtest.BaseChecks("127.0.0.1", "up", "http"),
			hbtest.SummaryChecks(1, 0),
			hbtest.RespondingTCPChecks(),
			respondingHTTPStatusAndTimingChecks(200),
			minimalRespondingHTTPChecks(server.URL, "text/plain; charset=utf-8", 200),
			lookslike.MustCompile(map[string]interface{}{
				"http.response.headers": isdef.KeyMissing,
			}),
		)),
		event.Fields,
	)
}

func TestNewRoundTripper(t *testing.T) {
	configs := map[string]Config{
		"Plain":      {Timeout: time.Second},
		"With Proxy": {Timeout: time.Second, ProxyURL: "http://localhost:1234"},
	}

	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			transp, err := newRoundTripper(&config, &tlscommon.TLSConfig{})
			require.NoError(t, err)

			if config.ProxyURL == "" {
				require.Nil(t, transp.Proxy)
			} else {
				require.NotNil(t, transp.Proxy)
			}

			// It's hard to compare func types in tests
			require.NotNil(t, transp.Dial)
			require.NotNil(t, transport.TLSDialer)

			expected := (&tlscommon.TLSConfig{}).ToConfig()
			require.Equal(t, expected.InsecureSkipVerify, transp.TLSClientConfig.InsecureSkipVerify)
			// When we remove support for the legacy common name treatment
			// this test has to be adjusted, as we will not depend on our
			// VerifyConnection callback.
			require.NotNil(t, transp.TLSClientConfig.VerifyConnection)
			require.True(t, transp.DisableKeepAlives)
		})
	}

}

func TestProxy(t *testing.T) {
	if runtime.GOOS == "windows" && bits.UintSize == 32 {
		t.Skip("flaky test: https://github.com/elastic/beats/issues/25857")
	}
	server := httptest.NewTLSServer(hbtest.HelloWorldHandler(http.StatusOK))
	proxy := httptest.NewServer(http.HandlerFunc(httpConnectTunnel))
	runHTTPSServerCheck(t, server, map[string]interface{}{
		"proxy_url": proxy.URL,
	})
}

func TestTLSProxy(t *testing.T) {
	if runtime.GOOS == "windows" && bits.UintSize == 32 {
		t.Skip("flaky test: https://github.com/elastic/beats/issues/25857")
	}
	server := httptest.NewTLSServer(hbtest.HelloWorldHandler(http.StatusOK))
	proxy := httptest.NewTLSServer(http.HandlerFunc(httpConnectTunnel))
	runHTTPSServerCheck(t, server, map[string]interface{}{
		"proxy_url": proxy.URL,
	})
}

func httpConnectTunnel(writer http.ResponseWriter, request *http.Request) {
	// This method is adapted from code by Michał Łowicki @mlowicki (CC BY 4.0)
	// See https://medium.com/@mlowicki/http-s-proxy-in-golang-in-less-than-100-lines-of-code-6a51c2f2c38c
	if request.Method != http.MethodConnect {
		http.Error(writer, "Only CONNECT method is supported", http.StatusMethodNotAllowed)
		return
	}
	destConn, err := net.DialTimeout("tcp", request.Host, 10*time.Second)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writer.WriteHeader(http.StatusOK)
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		http.Error(writer, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientReadWriter, err := hijacker.Hijack()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusServiceUnavailable)
	}
	defer destConn.Close()
	defer clientConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		io.Copy(destConn, clientReadWriter)
		wg.Done()
	}()
	go func() {
		io.Copy(clientConn, destConn)
		wg.Done()
	}()
	wg.Wait()
}

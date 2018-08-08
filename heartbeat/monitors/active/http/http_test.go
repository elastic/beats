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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	btesting "github.com/elastic/beats/libbeat/testing"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func testRequest(t *testing.T, url string) beat.Event {
	config := common.NewConfig()
	config.SetString("urls", 0, url)

	jobs, err := create(monitors.Info{}, config)
	require.NoError(t, err)

	job := jobs[0]

	event, _, err := job.Run()
	require.NoError(t, err)

	return event
}

func checkServer(t *testing.T, handlerFunc http.HandlerFunc) (*httptest.Server, beat.Event) {
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

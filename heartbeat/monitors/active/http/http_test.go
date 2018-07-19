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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"fmt"

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
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
	return mapval.Schema(mapval.Map{
		"http.url": url,
	})
}

func respondingHTTPChecks(url string, statusCode int) mapval.Validator {
	return mapval.Compose(
		httpBaseChecks(url),
		mapval.Schema(mapval.Map{
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

func TestOKJob(t *testing.T) {
	server, event := checkServer(t, hbtest.HelloWorldHandler)
	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "http", "up"),
			hbtest.RespondingTCPChecks(port),
			respondingHTTPChecks(server.URL, http.StatusOK),
		)),
		(event.Fields),
	)
}

func TestBadGatewayJob(t *testing.T) {
	server, event := checkServer(t, hbtest.BadGatewayHandler)
	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "http", "down"),
			hbtest.RespondingTCPChecks(port),
			respondingHTTPChecks(server.URL, http.StatusBadGateway),
			hbtest.ErrorChecks("502 Bad Gateway", "validate"),
		)),
		(event.Fields),
	)
}

func TestUnreachableJob(t *testing.T) {
	// 203.0.113.0/24 is reserved for documentation so should not be routable
	// See: https://tools.ietf.org/html/rfc6890
	ip := "203.0.113.1"
	url := "http://" + ip

	event := testRequest(t, url)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+url, url, ip, "http", "down"),
			hbtest.TCPBaseChecks(80),
			hbtest.ErrorChecks(
				fmt.Sprintf(
					"Get http://%s: dial tcp %s:80: i/o timeout (Client.Timeout exceeded while awaiting headers)", ip, ip),
				"io"),
			httpBaseChecks(url),
		)),
		(event.Fields),
	)
}

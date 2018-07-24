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

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func checkServer(t *testing.T, handlerFunc http.HandlerFunc) (*httptest.Server, beat.Event) {
	server := httptest.NewServer(handlerFunc)
	defer server.Close()

	config := common.NewConfig()
	config.SetString("urls", 0, server.URL)

	jobs, err := create(monitors.Info{}, config)
	require.NoError(t, err)

	job := jobs[0]

	event, _, err := job.Run()
	require.NoError(t, err)

	return server, event
}

func httpChecks(urlStr string, statusCode int) mapval.Validator {
	return mapval.Schema(mapval.Map{
		"http": mapval.Map{
			"url": urlStr,
			"response.status_code":   statusCode,
			"rtt.content.us":         mapval.IsDuration,
			"rtt.response_header.us": mapval.IsDuration,
			"rtt.total.us":           mapval.IsDuration,
			"rtt.validate.us":        mapval.IsDuration,
			"rtt.write_request.us":   mapval.IsDuration,
		},
	})
}

func badGatewayChecks() mapval.Validator {
	return mapval.Schema(mapval.Map{
		"error": mapval.Map{
			"message": "502 Bad Gateway",
			"type":    "validate",
		},
	})
}

func TestOKJob(t *testing.T) {
	server, event := checkServer(t, hbtest.HelloWorldHandler)
	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks("http@"+server.URL, server.URL, "127.0.0.1", "http", "up"),
			hbtest.TCPChecks(port),
			httpChecks(server.URL, http.StatusOK),
		))(event.Fields),
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
			hbtest.TCPChecks(port),
			httpChecks(server.URL, http.StatusBadGateway),
			badGatewayChecks(),
		))(event.Fields),
	)
}

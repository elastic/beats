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

package hbtest

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	"net/http/httptest"

	"github.com/elastic/beats/libbeat/common/mapval"
)

// HelloWorldBody is the body of the HelloWorldHandler.
const HelloWorldBody = "hello, world!"

// HelloWorldHandler is a handler for an http server that returns
// HelloWorldBody and a 200 OK status.
var HelloWorldHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, HelloWorldBody)
})

// BadGatewayBody is the body of the BadGatewayHandler.
const BadGatewayBody = "Bad Gateway"

// BadGatewayHandler is a handler for an http server that returns
// BadGatewayBody and a 200 OK status.
var BadGatewayHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, BadGatewayBody)
})

// ServerPort takes an httptest.Server and returns its port as a uint16.
func ServerPort(server *httptest.Server) (uint16, error) {
	u, err := url.Parse(server.URL)
	if err != nil {
		return 0, err
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		return 0, err
	}
	return uint16(p), nil
}

// MonitorChecks creates a skima.Validator that represents the "monitor" field present
// in all heartbeat events.
func MonitorChecks(id string, host string, ip string, scheme string, status string) mapval.Validator {
	return mapval.Schema(mapval.Map{
		"monitor": mapval.Map{
			// TODO: This is only optional because, for some reason, TCP returns
			// this value, but HTTP does not. We should fix this
			"host":        mapval.Optional(mapval.IsEqual(host)),
			"duration.us": mapval.IsDuration,
			"id":          id,
			"ip":          ip,
			"scheme":      scheme,
			"status":      status,
		},
	})
}

// TCPChecks creates a skima.Validator that represents the "tcp" field present
// in all heartbeat events that use a Tcp connection as part of their DialChain
func TCPChecks(port uint16) mapval.Validator {
	return mapval.Schema(mapval.Map{
		"tcp": mapval.Map{
			"port":           port,
			"rtt.connect.us": mapval.IsDuration,
		},
	})
}

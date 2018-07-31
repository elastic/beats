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
func HelloWorldHandler(status int) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if status >= 301 && status <= 303 {
				w.Header().Set("Location", "/somewhere")
			}
			w.WriteHeader(status)
			io.WriteString(w, HelloWorldBody)
		},
	)
}

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
	return mapval.MustCompile(mapval.Map{
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

// TCPBaseChecks checks the minimum TCP response, which is only issued
// without further fields when the endpoint does not respond.
func TCPBaseChecks(port uint16) mapval.Validator {
	return mapval.MustCompile(mapval.Map{"tcp.port": port})
}

// ErrorChecks checks the standard heartbeat error hierarchy, which should
// consist of a message (or a mapval isdef that can match the message) and a type under the error key.
// The message is checked only as a substring since exact string matches can be fragile due to platform differences.
func ErrorChecks(msgSubstr string, errType string) mapval.Validator {
	return mapval.MustCompile(mapval.Map{
		"error": mapval.Map{
			"message": mapval.IsStringContaining(msgSubstr),
			"type":    errType,
		},
	})
}

// RespondingTCPChecks creates a skima.Validator that represents the "tcp" field present
// in all heartbeat events that use a Tcp connection as part of their DialChain
func RespondingTCPChecks(port uint16) mapval.Validator {
	return mapval.Compose(
		TCPBaseChecks(port),
		mapval.MustCompile(mapval.Map{"tcp.rtt.connect.us": mapval.IsDuration}),
	)
}

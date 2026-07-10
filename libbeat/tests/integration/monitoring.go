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

package integration

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
)

// MonitoringEndpointSnippet is the prefix libbeat/api logs when its HTTP
// monitoring server starts (see libbeat/api/server.go). Tests configure
// http.port: 0 and match this snippet to discover the OS-assigned port.
const MonitoringEndpointSnippet = "Metrics endpoint listening on:"

// e.g. "Metrics endpoint listening on: 127.0.0.1:5067 (configured: localhost)".
var reMonitoringEndpoint = regexp.MustCompile(regexp.QuoteMeta(MonitoringEndpointSnippet) + ` (\S+) \(configured:`)

// ParseMonitoringPort extracts the bound port from a MonitoringEndpointSnippet
// log line. It lets tests read the ephemeral port chosen by the OS instead of
// pre-allocating one, which avoids time-of-check/time-of-use port collisions
// when many tests run in parallel.
func ParseMonitoringPort(logLine string) (int, error) {
	matches := reMonitoringEndpoint.FindStringSubmatch(logLine)
	if len(matches) != 2 {
		return 0, fmt.Errorf("no monitoring address found in log line: %q", logLine)
	}
	return portFromHostPort(matches[1])
}

// portFromHostPort parses the port out of a host:port address logged by a Beat.
func portFromHostPort(addr string) (int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("could not split host:port from %q: %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("could not parse port from %q: %w", portStr, err)
	}
	return port, nil
}

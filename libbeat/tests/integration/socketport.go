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
	"regexp"
)

// SocketListeningSnippet is the fragment the tcp and udp inputs log when their
// socket server binds (see filebeat/inputsource/{tcp,udp}/server.go). Tests
// configure host: <ip>:0 and match this snippet to discover the OS-assigned
// port.
const SocketListeningSnippet = "connection on:"

// e.g. "Started listening for TCP connection on: 127.0.0.1:54321". The address
// is the last token of the message, so the capture stops at whitespace or a
// double quote to avoid swallowing the closing quote when the message is
// embedded in a JSON log line.
var reSocketListening = regexp.MustCompile(regexp.QuoteMeta(SocketListeningSnippet) + ` ([^\s"]+)`)

// ParseSocketListeningPort extracts the bound port from a SocketListeningSnippet
// log line. It lets tests read the ephemeral port chosen by the OS instead of
// pre-allocating one, which avoids time-of-check/time-of-use port collisions
// when many tests run in parallel.
func ParseSocketListeningPort(logLine string) (int, error) {
	matches := reSocketListening.FindStringSubmatch(logLine)
	if len(matches) != 2 {
		return 0, fmt.Errorf("no socket listening address found in log line: %q", logLine)
	}
	return portFromHostPort(matches[1])
}

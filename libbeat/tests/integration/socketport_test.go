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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSocketListeningPort(t *testing.T) {
	testCases := map[string]struct {
		logLine  string
		wantPort int
		wantErr  bool
	}{
		"tcp raw message": {
			// The collector observer exposes the bare message.
			logLine:  "Started listening for TCP connection on: 127.0.0.1:54321",
			wantPort: 54321,
		},
		"udp raw message": {
			logLine:  "Started listening for UDP connection on: 127.0.0.1:9999",
			wantPort: 9999,
		},
		"embedded in a JSON log line": {
			// The Beat writes structured logs; the address is followed by the
			// closing quote of the message field, which must not be captured.
			logLine:  `{"log.level":"info","message":"Started listening for TCP connection on: 127.0.0.1:5067","service.name":"filebeat"}`,
			wantPort: 5067,
		},
		"ipv6 address": {
			logLine:  "Started listening for UDP connection on: [::1]:6000",
			wantPort: 6000,
		},
		"no address": {
			logLine: "Started listening for UDP connection",
			wantErr: true,
		},
		"unrelated line": {
			logLine: "some other log message",
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			port, err := ParseSocketListeningPort(tc.logLine)
			if tc.wantErr {
				assert.Error(t, err, "expected an error parsing %q", tc.logLine)
				return
			}
			require.NoError(t, err, "unexpected error parsing %q", tc.logLine)
			assert.Equal(t, tc.wantPort, port, "parsed the wrong port from %q", tc.logLine)
		})
	}
}

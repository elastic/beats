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

package oteltest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	cfg := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          enabled: true
          message: "test message"
          count: 1 
    output:
      otelconsumer:
    logging:
      level: debug
    queue.mem.flush.timeout: 0s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - debug
  telemetry:
    logs:
      level: DEBUG
`
	col := NewCollector(t, cfg)
	require.NotNil(t, col)

	require.Eventually(t, func() bool {
		return col.ObservedLogs().FilterMessageSnippet(`"message": "test message`).Len() == 1
	}, 30*time.Second, 100*time.Millisecond, "Expected debug log with test message not found")
}

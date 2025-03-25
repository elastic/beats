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

//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var esCfg = `
mockbeat:
logging:
  level: debug
  selectors:
    - publisher_pipeline_output
    - esclientleg
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  allow_older_versions: true
  hosts:
    - "%s"
  backoff:
    init: 0.1s
    max: 0.2s
`

func TestESOutputRecoversFromNetworkError(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")

	s, esIPPort, _, mr := StartMockES(t, ":4242", 0, 0, 0, 0, 0)
	esAddr := "http://" + esIPPort

	mockbeat.WriteConfigFile(fmt.Sprintf(esCfg, esAddr))
	mockbeat.Start()

	// 1. Wait for one _bulk call
	waitForEventToBePublished(t, mr)

	// 2. Stop the mock-es server
	if err := s.Close(); err != nil {
		t.Fatalf("cannot close mock-es server: %s", err)
	}

	// 3. Wait for connection error logs
	mockbeat.WaitForLogs(
		fmt.Sprintf(`Get \"%s\": dial tcp %s: connect: connection refused`, esAddr, esIPPort),
		2*time.Second,
		"did not find connection refused error")

	mockbeat.WaitForLogs(
		fmt.Sprintf("Attempting to reconnect to backoff(elasticsearch(%s)) with 2 reconnect attempt(s)", esAddr),
		2*time.Second,
		"did not find two tries to reconnect")

	// 4. Restart mock-es on the same port
	s, esIPPort, _, mr = StartMockES(t, ":4242", 0, 0, 0, 0, 0)

	// 5. Wait for reconnection logs
	mockbeat.WaitForLogs(
		fmt.Sprintf("Connection to backoff(elasticsearch(%s)) established", esAddr),
		5*time.Second, // There is a backoff, so ensure we wait enough
		"did not find re connection confirmation")

	// 6. Ensure one new call to _bulk is made
	waitForEventToBePublished(t, mr)
	s.Close()
}

// waitForEventToBePublished waits for at least one event published
// by inspecting the count for `bulk.create.total` in `mr`. Once
// the counter is > 1, waitForEventToBePublished returns. If that
// does not happen within 10min, then the test fails with a call to
// t.Fatal.
func waitForEventToBePublished(t *testing.T, rdr *sdkmetric.ManualReader) {
	t.Helper()

	require.Eventually(t, func() bool {
		rm := metricdata.ResourceMetrics{}
		err := rdr.Collect(context.Background(), &rm)

		if err != nil {
			t.Fatalf("failed to collect metrics: %v", err)
		}

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "bulk.create.total" {
					total := int64(0)
					for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
						total += dp.Value
					}
					return total >= 1
				}
			}

			return false
		}
		return false
	},
		10*time.Second,
		100*time.Millisecond,
		"at least one bulk request must be made")
}

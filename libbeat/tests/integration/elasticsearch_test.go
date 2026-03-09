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
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
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

	s, esAddr, _, mr := StartMockES(t, ":4242", 0, 0, 0, 0, 0)

	esURL, err := url.Parse(esAddr)
	if err != nil {
		t.Fatalf("cannot parse mockES URL: %s", err)
	}

	mockbeat.WriteConfigFile(fmt.Sprintf(esCfg, esAddr))
	mockbeat.Start()

	// 1. Wait for one _bulk call
	waitForEventToBePublished(t, mr)

	// 2. Stop the mock-es server
	if err := s.Close(); err != nil {
		t.Fatalf("cannot close mock-es server: %s", err)
	}

	// 3. Wait for connection error logs
	mockbeat.WaitLogsContains(
		fmt.Sprintf(`Get \"%s\": dial tcp %s: connect: connection refused`, esAddr, esURL.Host),

		2*time.Second,
		"did not find connection refused error")

	mockbeat.WaitLogsContains(
		fmt.Sprintf("Attempting to reconnect to backoff(elasticsearch(%s)) with 2 reconnect attempt(s)", esAddr),
		2*time.Second,
		"did not find two tries to reconnect")

	// 4. Restart mock-es on the same port
	s, _, _, mr = StartMockES(t, ":4242", 0, 0, 0, 0, 0)

	// 5. Wait for reconnection logs
	mockbeat.WaitLogsContains(
		fmt.Sprintf("Connection to backoff(elasticsearch(%s)) established", esAddr),
		5*time.Second, // There is a backoff, so ensure we wait enough
		"did not find re connection confirmation")

	// 6. Ensure one new call to _bulk is made
	waitForEventToBePublished(t, mr)
	s.Close()
}

func TestReloadCA(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")

	s, esAddr, _, _ := StartMockES(t, ":4242", 0, 0, 0, 0, 0)
	defer s.Close()

	_, _, pair, err := certutil.NewRootCA()
	require.NoError(t, err, "could not generate root CA")
	caPath := filepath.Join(os.TempDir(), "ca.pem")
	err = os.WriteFile(caPath, pair.Cert, 0644)
	require.NoError(t, err, "could not write CA")

	mockbeat.WriteConfigFile(fmt.Sprintf(`
output.elasticsearch:
  allow_older_versions: true
  hosts: ["%s"]
  ssl:
    certificate_authorities: "%s"
    restart_on_cert_change.enabled: true
    restart_on_cert_change.period: 1s
logging.level: debug
`, esAddr, caPath))

	mockbeat.Start()

	// 1. wait mockbeat to start
	mockbeat.WaitLogsContains(
		fmt.Sprint("mockbeat start running"),
		10*time.Second,
		"did not find 'mockbeat start running' log")

	// 2. "rotate" the CA. Just write it again
	err = os.WriteFile(caPath, pair.Cert, 0644)
	require.NoError(t, err, "could not rotate CA")

	// 3. Wait for cert change detection logs
	mockbeat.WaitLogsContains(
		fmt.Sprintf("some of the following files have been modified: [%s]", caPath),
		10*time.Second,
		"did not detect CA rotation")

	// 4. Wait for CA load log
	mockbeat.WaitLogsContains(
		fmt.Sprintf("Successfully loaded CA certificate: %s", caPath),
		10*time.Second,
		"did not find 'Successfully loaded CA' log")

	// 5. wait mockbeat to start again
	mockbeat.WaitLogsContains(
		fmt.Sprint("mockbeat start running"),
		10*time.Second,
		"did not find 'mockbeat start running' log again")
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
					//nolint:errcheck // It's a test
					for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
						total += dp.Value
					}
					return total >= 1
				}
			}
		}

		return false
	},
		10*time.Second,
		100*time.Millisecond,
		"at least one bulk request must be made")
}

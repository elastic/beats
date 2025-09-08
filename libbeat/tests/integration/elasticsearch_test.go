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
<<<<<<< HEAD
	"errors"
	"io"
	"net/http"
=======
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
>>>>>>> 716277334 (libbeat: add 'eventfd2' to default seccomp policy (#46372))
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/require"
<<<<<<< HEAD

	"github.com/elastic/mock-es/pkg/api"
=======
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
>>>>>>> 716277334 (libbeat: add 'eventfd2' to default seccomp policy (#46372))
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
    - "http://localhost:4242"
  backoff:
    init: 0.1s
    max: 0.2s
`

func TestESOutputRecoversFromNetworkError(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(esCfg)

	s, mr := startMockES(t, "localhost:4242")

	mockbeat.Start()

	// 1. Wait for one _bulk call
	waitForEventToBePublished(t, mr)

	// 2. Stop the mock-es server
	if err := s.Close(); err != nil {
		t.Fatalf("cannot close mock-es server: %s", err)
	}

	// 3. Wait for connection error logs
	mockbeat.WaitForLogs(
		`Get \"http://localhost:4242\": dial tcp 127.0.0.1:4242: connect: connection refused`,
		2*time.Second,
		"did not find connection refused error")

	mockbeat.WaitForLogs(
		"Attempting to reconnect to backoff(elasticsearch(http://localhost:4242)) with 2 reconnect attempt(s)",
		2*time.Second,
		"did not find two tries to reconnect")

	// 4. Restart mock-es on the same port
	s, mr = startMockES(t, "localhost:4242")

	// 5. Wait for reconnection logs
	mockbeat.WaitForLogs(
		"Connection to backoff(elasticsearch(http://localhost:4242)) established",
		5*time.Second, // There is a backoff, so ensure we wait enough
		"did not find re connection confirmation")

	// 6. Ensure one new call to _bulk is made
	waitForEventToBePublished(t, mr)
	s.Close()
}

<<<<<<< HEAD
func startMockES(t *testing.T, addr string) (*http.Server, metrics.Registry) {
	uid := uuid.Must(uuid.NewV4())
	mr := metrics.NewRegistry()
	es := api.NewAPIHandler(uid, "foo2", mr, time.Now().Add(24*time.Hour), 0, 0, 0, 0, 0)

	s := http.Server{Addr: addr, Handler: es, ReadHeaderTimeout: time.Second}
	go func() {
		if err := s.ListenAndServe(); !errors.Is(http.ErrServerClosed, err) {
			t.Errorf("could not start mock-es server: %s", err)
		}
	}()

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://" + addr) //nolint: noctx // It's just a test
		if err != nil {
			//nolint: errcheck // We're just draining the body, we can ignore the error
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return false
		}
		return true
	},
		time.Second, time.Millisecond, "mock-es server did not start on '%s'", addr)

	return &s, mr
=======
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
>>>>>>> 716277334 (libbeat: add 'eventfd2' to default seccomp policy (#46372))
}

// waitForEventToBePublished waits for at least one event published
// by inspecting the count for `bulk.create.total` in `mr`. Once
// the counter is > 1, waitForEventToBePublished returns. If that
// does not happen within 10min, then the test fails with a call to
// t.Fatal.
func waitForEventToBePublished(t *testing.T, mr metrics.Registry) {
	t.Helper()
	require.Eventually(t, func() bool {
		total := mr.Get("bulk.create.total")
		if total == nil {
			return false
		}

<<<<<<< HEAD
		sc, ok := total.(*metrics.StandardCounter)
		if !ok {
			t.Fatalf("expecting 'bulk.create.total' to be *metrics.StandardCounter, but got '%T' instead",
				total,
			)
=======
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
>>>>>>> 716277334 (libbeat: add 'eventfd2' to default seccomp policy (#46372))
		}

		return sc.Count() > 1
	},
		10*time.Second, 100*time.Millisecond,
		"at least one bulk request must be made")
}

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
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/require"

	"github.com/elastic/mock-es/pkg/api"
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

		sc, ok := total.(*metrics.StandardCounter)
		if !ok {
			t.Fatalf("expecting 'bulk.create.total' to be *metrics.StandardCounter, but got '%T' instead",
				total,
			)
		}

		return sc.Count() > 1
	},
		10*time.Second, 100*time.Millisecond,
		"at least one bulk request must be made")
}

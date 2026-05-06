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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	testingintegration "github.com/elastic/beats/v7/filebeat/testing/integration"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// TestFilestreamReadUntilEOFOnInputStop exercises the read_until_eof feature
// end-to-end in the realistic scenario it was designed for: the input is
// stopped while Filebeat keeps running (as autodiscover would do on pod
// termination), with the output under backpressure so the harvester is
// mid-read when the stop fires. The assertion uses mock-es's OpenTelemetry
// counter bulk.create.ok, which only increments on accepted bulk creates and
// therefore reflects unique events actually delivered end-to-end.
func TestFilestreamReadUntilEOFOnInputStop(t *testing.T) {
	// percentTooMany=100 makes every bulk.create return 429, so the ES output
	// cannot drain the pipeline queue; the harvester Publish blocks mid-file.
	s, esAddr, es, rdr := integration.StartMockES(t, "", 0, 100, 0, 0, 0)
	defer s.Close()

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	workDir := filebeat.TempDir()

	inputsDir := filepath.Join(workDir, "inputs.d")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	events := 1000
	prefix := strings.Repeat("a", 1024)
	_, files := testingintegration.GenerateLogFiles(t, 1,
		events, testingintegration.NewJSONGenerator(prefix))
	logPath := files[0]

	// Small queue + tight backoff so backpressure kicks in fast and retries
	// stay tight once we release it.
	mainCfg := fmt.Sprintf(`
filebeat.config.inputs:
  path: %s/*.yml
  reload.enabled: true
  reload.period: 100ms
path.home: %s
queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  hosts: ["%s"]
  backoff:
    init: 100ms
    max: 100ms
logging.level: debug
`, inputsDir, workDir, esAddr)
	filebeat.WriteConfigFile(mainCfg)

	inputID := "TestFilestreamReadUntilEOFOnInputStop"
	inputCfg := fmt.Sprintf(`
- type: filestream
  id: %s
  enabled: true
  paths: [%s]
  read_until_eof:
    enabled: true
    timeout: 1m
  close.reader.on_eof: true
`, inputID, logPath)
	inputFile := filepath.Join(inputsDir, "filestream.yml")
	require.NoError(t, os.WriteFile(inputFile, []byte(inputCfg), 0644))

	filebeat.Start()

	filebeat.WaitLogsContains(
		"Starting harvester for file",
		30*time.Second,
		"harvester did not start")

	// Simulate autodiscover removing the input. Reload will invoke
	// runner.Stop, which sends cancellation. Because mock-es is still
	// returning 429, the harvester's current Publish is blocked — the
	// normal read loop cannot exit until the queue drains. The
	// read_until_eof continuation runs as soon as it does.
	require.NoError(t, os.Rename(inputFile, inputFile+".disabled"))

	// "Stopping runner" is logged synchronously by the reload goroutine
	// before it waits for the runner to exit; this confirms cancellation
	// is in flight before we release backpressure.
	filebeat.WaitLogsContains(
		"Stopping runner",
		10*time.Second,
		"reload did not detect input removal")

	// Release backpressure: queue drains, Publish unblocks, the normal
	// read loop exits on the cancelled context, and the read_until_eof
	// continuation reads until EOF.
	require.NoError(t, es.UpdateOdds(0, 0, 0, 0))

	// bulk.create.ok is only incremented for accepted creates (the 429
	// responses went to bulk.create.too_many). If read_until_eof works,
	// all N unique events are accepted exactly once.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		got := bulkCreateOK(t, rdr)
		assert.Equalf(c, int64(events), got,
			"expected mock-es to accept %d events, got %d", events, got)
	}, 60*time.Second, 200*time.Millisecond)

	filebeat.Stop()
}

// TestFilestreamReadUntilEOFWithoutCloseOnEOF verifies that read_until_eof
// works end-to-end even when close.reader.on_eof is not set (i.e. tail mode
// is the user's normal operation). When the input is stopped mid-read, the
// drain logic must revive the reader, switch it to close-on-EOF mode, and
// drain until EOF so all events reach the output.
func TestFilestreamReadUntilEOFWithoutCloseOnEOF(t *testing.T) {
	s, esAddr, es, rdr := integration.StartMockES(t, "", 0, 100, 0, 0, 0)
	defer s.Close()

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	workDir := filebeat.TempDir()

	inputsDir := filepath.Join(workDir, "inputs.d")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	events := 1000
	prefix := strings.Repeat("a", 1024)
	_, files := testingintegration.GenerateLogFiles(t, 1,
		events, testingintegration.NewJSONGenerator(prefix))
	logPath := files[0]

	mainCfg := fmt.Sprintf(`
filebeat.config.inputs:
  path: %s/*.yml
  reload.enabled: true
  reload.period: 100ms
path.home: %s
queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  hosts: ["%s"]
  backoff:
    init: 100ms
    max: 100ms
logging.level: debug
`, inputsDir, workDir, esAddr)
	filebeat.WriteConfigFile(mainCfg)

	inputID := "TestFilestreamReadUntilEOFWithoutCloseOnEOF"
	// Note: close.reader.on_eof intentionally omitted — defaults to false.
	// In that mode the harvester tails the file (blocks in backoff.Wait on
	// current EOF) which is exactly the scenario the drain logic must handle.
	inputCfg := fmt.Sprintf(`
- type: filestream
  id: %s
  enabled: true
  paths: [%s]
  read_until_eof:
    enabled: true
    timeout: 1m
`, inputID, logPath)
	inputFile := filepath.Join(inputsDir, "filestream.yml")
	require.NoError(t, os.WriteFile(inputFile, []byte(inputCfg), 0644))

	filebeat.Start()

	filebeat.WaitLogsContains(
		"Starting harvester for file",
		30*time.Second,
		"harvester did not start")

	require.NoError(t, os.Rename(inputFile, inputFile+".disabled"))

	filebeat.WaitLogsContains(
		"Stopping runner",
		10*time.Second,
		"reload did not detect input removal")
	err := es.UpdateOdds(0, 0, 0, 0)
	require.NoError(t, err, "could not unblock mock-es")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		got := bulkCreateOK(t, rdr)
		assert.Equalf(c, int64(events), got,
			"expected mock-es to accept %d events, got %d", events, got)
	}, 60*time.Second, 200*time.Millisecond)

	filebeat.Stop()
}

func bulkCreateOK(t *testing.T, rdr *sdkmetric.ManualReader) int64 {
	t.Helper()
	rm := metricdata.ResourceMetrics{}
	if err := rdr.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect mock-es metrics: %v", err)
	}
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "bulk.create.ok" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
			for _, dp := range sum.DataPoints {
				total += dp.Value
			}
		}
	}
	return total
}

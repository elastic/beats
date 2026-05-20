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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/mock-es/pkg/api"
)

func TestFilebeatFilestreamInactiveCloseReopenWithSlowOutputLosesData(t *testing.T) {
	const (
		firstBatchSize  = 5000
		secondBatchSize = 8
	)

	server, esAddr, es, _ := integration.StartMockES(t, "", 0, 100, 0, 0, 20000)
	defer server.Close()

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()
	logFile := filepath.Join(workDir, "log.log")

	firstBatch := filestreamCloseReopenLines("before-close", firstBatchSize)
	secondBatch := filestreamCloseReopenLines("after-reopen", secondBatchSize)

	writeFilestreamCloseReopenLines(t, logFile, firstBatch, false)

	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCloseReopenSlowOutputConfig, logFile, workDir, esAddr))
	filebeat.Start()

	msgLogFilePath := logFile
	if os.PathSeparator == '\\' {
		msgLogFilePath = strings.ReplaceAll(logFile, `\`, `\\`)
	}

	filebeat.WaitLogsContains(
		fmt.Sprintf("'%s' is inactive", msgLogFilePath),
		30*time.Second,
		"filestream did not close the harvester due to inactivity")

	writeFilestreamCloseReopenLines(t, logFile, secondBatch, true)
	time.Sleep(2 * time.Second)

	historyOffset := filestreamCloseReopenBulkHistoryCount(es)
	require.NoError(t, es.UpdateOdds(0, 0, 0, 0), "cannot reset mock-es odds")
	waitForFilestreamCloseReopenMessages(t, es, historyOffset, secondBatch, 30*time.Second)

	// Only bulk requests after the mock-es recovery count as delivered. The
	// initial 429 phase proves Filebeat attempted to publish, but those events
	// were not accepted by the output.
	assertNoMissingFilestreamCloseReopenMessages(t, es, historyOffset, append(firstBatch, secondBatch...))
}

const filestreamCloseReopenSlowOutputConfig = `
filebeat.inputs:
  - type: filestream
    id: close-reopen-loss
    paths:
      - %q
    # Leave file_identity unset to exercise the default identity.
    close.on_state_change.inactive: 1s
    close.on_state_change.check_interval: 100ms
    prospector.scanner.check_interval: 100ms
    backoff.init: 10ms
    backoff.max: 10ms

path.home: %q

queue.mem:
  events: 32
  flush.min_events: 1
  flush.timeout: 0s

output.elasticsearch:
  hosts:
    - %q
  allow_older_versions: true
  worker: 1
  bulk_max_size: 1
  backoff:
    init: 100ms
    max: 100ms

logging:
  level: debug
  selectors:
    - input
    - input.filestream
    - input.harvester
    - publisher_pipeline_output
    - esclientleg

metrics:
  enabled: false
`

func filestreamCloseReopenLines(prefix string, count int) []string {
	lines := make([]string, count)
	padding := strings.Repeat("x", 96)
	for i := range lines {
		lines[i] = fmt.Sprintf("%s line %03d %s", prefix, i, padding)
	}
	return lines
}

func writeFilestreamCloseReopenLines(t *testing.T, path string, lines []string, appendData bool) {
	t.Helper()

	flag := os.O_WRONLY | os.O_CREATE
	if appendData {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(path, flag, 0o644)
	require.NoError(t, err, "cannot open log file %q", path)
	defer f.Close()

	_, err = f.WriteString(strings.Join(lines, "\n") + "\n")
	require.NoError(t, err, "cannot write log file %q", path)
}

func waitForFilestreamCloseReopenMessages(
	t *testing.T,
	es *api.APIHandler,
	historyOffset int,
	messages []string,
	timeout time.Duration,
) {
	t.Helper()

	require.Eventuallyf(t, func() bool {
		seen := filestreamCloseReopenMessagesSeen(es, historyOffset, messages)
		for _, msg := range messages {
			if !seen[msg] {
				return false
			}
		}
		return true
	}, timeout, 100*time.Millisecond, "mock-es did not receive expected messages")
}

func assertNoMissingFilestreamCloseReopenMessages(
	t *testing.T,
	es *api.APIHandler,
	historyOffset int,
	expected []string,
) {
	t.Helper()

	seen := filestreamCloseReopenMessagesSeen(es, historyOffset, expected)
	var missing []string
	for _, msg := range expected {
		if !seen[msg] {
			missing = append(missing, msg)
		}
	}

	if len(missing) > 0 {
		sampleSize := min(len(missing), 5)
		require.Failf(
			t,
			"filestream lost messages after inactive close/reopen",
			"lost=%d expected=%d first_missing=%v",
			len(missing),
			len(expected),
			missing[:sampleSize],
		)
	}
}

func filestreamCloseReopenMessagesSeen(es *api.APIHandler, historyOffset int, candidates []string) map[string]bool {
	seen := map[string]bool{}
	for _, record := range filestreamCloseReopenBulkHistory(es)[historyOffset:] {
		for _, msg := range candidates {
			if strings.Contains(record.Body, msg) {
				seen[msg] = true
			}
		}
	}

	return seen
}

func filestreamCloseReopenBulkHistoryCount(es *api.APIHandler) int {
	return len(filestreamCloseReopenBulkHistory(es))
}

func filestreamCloseReopenBulkHistory(es *api.APIHandler) []*api.RequestRecord {
	var history []*api.RequestRecord
	for _, record := range es.RequestHistory() {
		if record == nil || !strings.Contains(record.URI, "_bulk") {
			continue
		}
		history = append(history, record)
	}
	return history
}

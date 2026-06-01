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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestFilestreamScannerMetricsLoggedWithFileOutput(t *testing.T) {
	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()

	keepLog := filepath.Join(tempDir, "keep.log")
	excludedLog := filepath.Join(tempDir, "excluded.log")
	emptyLog := filepath.Join(tempDir, "empty.log")
	oldLog := filepath.Join(tempDir, "old.log")

	integration.WriteLogFileFrom(t, keepLog, 0, 25, false)
	integration.WriteLogFileFrom(t, excludedLog, 25, 25, false)
	integration.WriteLogFileFrom(t, oldLog, 50, 25, false)
	require.NoError(t, os.WriteFile(emptyLog, nil, 0o644), "failed to write empty log")

	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldLog, oldTime, oldTime), "failed to age old log")

	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-filestream-scanner-metrics"
    paths:
      - %s
    prospector.scanner.exclude_files: ['excluded\.log$']
    ignore_older: 1h
    prospector.scanner.check_interval: 200ms

path.home: %s

queue.mem:
  flush.timeout: 0s

output.file:
  path: ${path.home}
  filename: "output-file"

logging:
  level: debug
  metrics:
    enabled: true
    period: 1s
`, filepath.Join(tempDir, "*.log"), tempDir)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitPublishedEvents(30*time.Second, 25)

	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			`"files_matched":4`,          // All files the input is monitoring
			`"files_unique":2`,           // Unique, non-ignored files
			`"files_no_ingest_target":1`, // Empty file has no ingest target
			`"files_ignored":2`,          // Old and inactive files are ignored
			`"files_empty":1`,            // Empty files matched by the scanner
		},
		15*time.Second,
		"filestream scanner metrics were not logged",
	)
}

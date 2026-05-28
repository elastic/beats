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

	require.NoError(t, os.WriteFile(keepLog, []byte("first line\nsecond line\n"), 0o644), "failed to write keep log")
	require.NoError(t, os.WriteFile(excludedLog, []byte("excluded line\n"), 0o644), "failed to write excluded log")
	require.NoError(t, os.WriteFile(emptyLog, nil, 0o644), "failed to write empty log")
	require.NoError(t, os.WriteFile(oldLog, []byte("old line\n"), 0o644), "failed to write old log")
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
    prospector.scanner.fingerprint.enabled: false
    file_identity.native: ~

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"

logging:
  level: info
  metrics:
    enabled: true
    period: 1s
`, filepath.Join(tempDir, "*.log"), tempDir)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	outputFile := waitForOutputFile(t, tempDir, "output-file-*.ndjson")
	integration.WaitLineCountInFile(t, outputFile, 2)

	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			`"files_matched":4`,
			`"files_unique":2`,
			`"files_no_ingest_target":1`,
			`"files_ignored":2`,
		},
		15*time.Second,
		"filestream scanner metrics were not logged",
	)
}

func waitForOutputFile(t *testing.T, dir, pattern string) string {
	t.Helper()

	var outputFile string
	var globErr error
	require.Eventually(t, func() bool {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		globErr = err
		if err != nil || len(matches) == 0 {
			return false
		}
		outputFile = matches[0]
		return true
	}, 30*time.Second, time.Second, "output file %q was not created: %v", pattern, globErr)

	return outputFile
}

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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

var testStoreCfg = `
filebeat.inputs:
  - type: filestream
    id: test-clean-removed
    enabled: true
    clean_removed: true
    close.on_state_change.inactive: 8s
    ignore_older: 9s
    prospector.scanner.check_interval: 1s
    paths:
      - %s

filebeat.registry:
  cleanup_interval: 5s
  flush: 1s

queue.mem:
  flush.min_events: 8
  flush.timeout: 0.1s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
  rotate_every_kb: 10000

logging:
  level: debug
  selectors:
    - input
    - input.filestream
`

func TestStore(t *testing.T) {
	numLogFiles := 10
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Create some log files and write data to them
	logsFolder := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsFolder, 0755); err != nil {
		t.Fatalf("could not create logs folder '%s': %s", logsFolder, err)
	}

	for i := 0; i < numLogFiles; i++ {
		logFile := path.Join(logsFolder, fmt.Sprintf("log-%d.log", i))
		integration.GenerateLogFile(t, logFile, 10, false)
	}
	logsFolderGlob := filepath.Join(logsFolder, "*")
	filebeat.WriteConfigFile(fmt.Sprintf(testStoreCfg, logsFolderGlob, tempDir))

	// 2. Ingest the file and stop Filebeat
	filebeat.Start()

	for i := 0; i < numLogFiles; i++ {
		// Files can be ingested out of order, so we cannot specify their path.
		// There will be more than one log line per file, but that at least gives us
		// some assurance the files were read
		filebeat.WaitForLogs("Closing reader of filestream", 30*time.Second, "Filebeat did not finish reading the log file")
	}

	// 3. Remove files so their state can be cleaned
	if err := os.RemoveAll(logsFolder); err != nil {
		t.Fatalf("could not remove logs folder '%s': %s", logsFolder, err)
	}
	filebeat.WaitForLogs(fmt.Sprintf("%d entries removed", numLogFiles), 30*time.Second, "store entries not removed")
	filebeat.Stop()

	registryLogFile := filepath.Join(tempDir, "data/registry/filebeat/log.json")
	readFilestreamRegistryLog(t, registryLogFile, "remove", 10)
}

func readFilestreamRegistryLog(t *testing.T, path, op string, expectedCount int) {
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("could not open file '%s': %s", path, err)
	}

	s := bufio.NewScanner(file)
	count := 0
	for s.Scan() {
		line := s.Bytes()

		registryOp := struct {
			Op string `json:"op"`
			ID int    `json:"id"`
		}{}

		if err := json.Unmarshal(line, &registryOp); err != nil {
			t.Fatalf("could not read line '%s': %s", string(line), err)
		}

		// Skips registry log entries that are not operation count
		if registryOp.Op == "" {
			continue
		}

		if registryOp.Op == op {
			count++
		}
	}

	if count != expectedCount {
		t.Errorf("expecting %d '%s' operations, got %d instead", expectedCount, op, count)
	}
}

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
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

var filestreamCleanInactiveCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-clean-inactive"
    paths:
      - %s

    clean_inactive: 3s
    ignore_older: 2s
    close.on_state_change.inactive: 1s
    prospector.scanner.check_interval: 1s

filebeat.registry:
  cleanup_interval: 5s
  flush: 1s

queue.mem:
  events: 32
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
  metrics:
    enabled: false
`

func TestFilestreamCleanInactive(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path, but do not write data to it
	logFilePath := path.Join(tempDir, "log.log")

	// 2. Write configuration file ans start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCleanInactiveCfg, logFilePath, tempDir))
	filebeat.Start()

	// 3. Create the log file
	integration.GenerateLogFile(t, logFilePath, 10, false)

	// 4. Wait for Filebeat to start scanning for files
	//
	filebeat.WaitForLogs(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		10*time.Second,
		"Filebeat did not start looking for files to ingest")

	filebeat.WaitForLogs(
		fmt.Sprintf("Reader was closed. Closing. Path='%s", logFilePath),
		10*time.Second, "Filebeat did not close the file")

	// 5. Now that the reader has been closed, nothing is holding the state
	// of the file, so once the TTL of its state expires and the store GC runs,
	// it will be removed from the registry.
	// Wait for the log message stating 1 entry has been removed from the registry
	filebeat.WaitForLogs("1 entries removed", 20*time.Second, "entry was not removed from registtry")

	// 6. Then assess it has been removed in the registry
	registryFile := filepath.Join(filebeat.TempDir(), "data", "registry", "filebeat", "log.json")
	filebeat.WaitFileContains(registryFile, `"op":"remove"`, time.Second)
}

/*
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	esPassword, _ := esURL.User.Password()

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")

	logFile1 := filepath.Join(filebeat.TempDir(), "logfile1.log")
	logFile2 := filepath.Join(filebeat.TempDir(), "logfile2.log")
	// logFile3 := filepath.Join(filebeat.TempDir(), "logfile3.log")

	filebeat.WriteConfigFile(fmt.Sprintf(
		cleanInactiveCfg,
		filebeat.TempDir(), // base path for logs
		esURL.String(),
		esURL.User.Username(),
		esPassword,
		filebeat.TempDir(), // base path for registry files
	))

	writeFile(t, logFile1, "file 1, line 1")
	writeFile(t, logFile2, "file 2, line 1")
	filebeat.Start()

	// Make sure Filebeat correctly stops
	defer func() {
		filebeat.Stop()
		filebeat.WaitForLogs("filebeat stopped", 5*time.Second, "did not find the stop message")
	}()

	filebeat.WaitForLogs("filebeat start running", 20*time.Second, "did not find Filebeat start logs, did Filebeat start correctly?")

	filebeat.WaitForLogs("Connection to backoff(elasticsearch(http://localhost:9200)", 2*time.Second, "did not connect to ES")
	// filebeat.WaitForLogs("events have been published to elasticsearch in", 5*time.Second, "did not publish events to ES")

	filebeat.WaitForLogs("removed state for", 30*time.Second, "did not find log entry about removing state from registry")


*/

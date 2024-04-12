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

var truncationCfg = `
filebeat.inputs:
  - type: filestream
    id: a-unique-filestream-input-id
    enabled: true
    prospector.scanner.check_interval: 30s
    paths:
      - %s
output:
  file:
    enabled: true
    codec.json:
      pretty: false
    path: %s
    filename: "output"
    rotate_on_startup: true
queue.mem:
  flush:
    timeout: 1s
    min_events: 32
filebeat.registry.flush: 1s
path.home: %s
logging:
  level: debug
  selectors:
    - file_watcher
    - input.filestream
    - input.harvester
  metrics:
    enabled: false
`

func TestFilestreamLiveFileTruncation(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	logFile := path.Join(tempDir, "log.log")
	registryLogFile := filepath.Join(tempDir, "data/registry/filebeat/log.json")
	filebeat.WriteConfigFile(fmt.Sprintf(truncationCfg, logFile, tempDir, tempDir))

	// 1. Create a log file and let Filebeat harvest all contents
	integration.GenerateLogFile(t, logFile, 200, false)
	filebeat.Start()
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")

	// 2. Truncate the file and wait Filebeat to close the file
	if err := os.Truncate(logFile, 0); err != nil {
		t.Fatalf("could not truncate log file: %s", err)
	}

	// 3. Ensure Filebeat detected the file truncation
	filebeat.WaitForLogs("File was truncated as offset (10000) > size (0)", 20*time.Second, "file was not truncated")
	filebeat.WaitForLogs("File was truncated, nothing to read", 20*time.Second, "reader loop did not stop")
	filebeat.WaitForLogs("Stopped harvester for file", 20*time.Second, "harvester did not stop")
	filebeat.WaitForLogs("Closing reader of filestream", 20*time.Second, "reader did not close")

	// 4. Now we need to stop Filebeat before the next scan cycle
	filebeat.Stop()

	// Assert we offset in the registry
	assertLastOffset(t, registryLogFile, 10_000)

	// Open for appending because the file has already been truncated
	integration.GenerateLogFile(t, logFile, 10, true)

	// 5. Start Filebeat again.
	filebeat.Start()
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")

	assertLastOffset(t, registryLogFile, 500)
}

func TestFilestreamOfflineFileTruncation(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	logFile := path.Join(tempDir, "log.log")
	registryLogFile := filepath.Join(tempDir, "data/registry/filebeat/log.json")
	filebeat.WriteConfigFile(fmt.Sprintf(truncationCfg, logFile, tempDir, tempDir))

	// 1. Create a log file with some lines
	integration.GenerateLogFile(t, logFile, 10, false)

	// 2. Ingest the file and stop Filebeat
	filebeat.Start()
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")
	filebeat.Stop()

	// 3. Assert the offset is correctly set in the registry
	assertLastOffset(t, registryLogFile, 500)

	// 4. Truncate the file and write some data (less than before)
	if err := os.Truncate(logFile, 0); err != nil {
		t.Fatalf("could not truncate log file: %s", err)
	}
	integration.GenerateLogFile(t, logFile, 5, true)

	// 5. Read the file again and stop Filebeat
	filebeat.Start()
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")
	filebeat.WaitForLogs("End of file reached", 30*time.Second, "Filebeat did not finish reading the log file")
	filebeat.Stop()

	// 6. Assert the registry offset is new, smaller file size.
	assertLastOffset(t, registryLogFile, 250)
}

func assertLastOffset(t *testing.T, path string, offset int) {
	t.Helper()
	entries := readFilestreamRegistryLog(t, path)
	lastEntry := entries[len(entries)-1]
	if lastEntry.Offset != offset {
		t.Errorf("expecting offset %d got %d instead", offset, lastEntry.Offset)
		t.Log("last registry entries:")

		max := len(entries)
		if max > 10 {
			max = 10
		}
		for _, e := range entries[:max] {
			t.Logf("%+v\n", e)
		}

		t.FailNow()
	}
}

type registryEntry struct {
	Key      string
	Offset   int
	Filename string
	TTL      time.Duration
}

func readFilestreamRegistryLog(t *testing.T, path string) []registryEntry {
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("could not open file '%s': %s", path, err)
	}

	entries := []registryEntry{}
	s := bufio.NewScanner(file)

	for s.Scan() {
		line := s.Bytes()

		e := entry{}
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("could not read line '%s': %s", string(line), err)
		}

		// Skips registry log entries containing the operation ID like:
		// '{"op":"set","id":46}'
		if e.Key == "" {
			continue
		}

		entries = append(entries, registryEntry{
			Key:      e.Key,
			Offset:   e.Value.Cursor.Offset,
			TTL:      e.Value.TTL,
			Filename: e.Value.Meta.Source,
		})
	}

	return entries
}

type entry struct {
	Key   string `json:"k"`
	Value struct {
		Cursor struct {
			Offset int `json:"offset"`
		} `json:"cursor"`
		Meta struct {
			Source string `json:"source"`
		} `json:"meta"`
		TTL time.Duration `json:"ttl"`
	} `json:"v"`
}

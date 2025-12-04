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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// Checks all log lines are ingested
// Checks that if a line does not have a line ending, then it is not read.
// Checks that if a file is renamed, its contents are not re-ingested
func TestCrawler(t *testing.T) {

	var filestreamCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-clean-inactive"
    paths:
      - %s

    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
`
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path, but do not write data to it
	logFilePath := filepath.Join(tempDir, "log.log")

	// 2. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCfg, filepath.Join(tempDir, "*.log"), tempDir))
	filebeat.Start()

	// 3. Create the log file
	integration.WriteLogFile(t, filepath.Join(tempDir, "log.log"), 10, false)

	// wait for output file to exist
	var outputFile string
	require.Eventually(t, func() bool {
		matches, err := filepath.Glob(filepath.Join(tempDir, "output-file-*.ndjson"))
		if err != nil || len(matches) == 0 {
			t.Logf("could not find output file %v", err)
			return false
		}
		outputFile = matches[0]
		return true
	}, 2*time.Minute, 10*time.Second)

	// Ensure all log lines are ingested eventually
	integration.WaitLineCountInFile(t, outputFile, 10)

	// append a line without \n and ensure it is not crawled
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("could not open file: %s: %v", logFilePath, err)
	}
	defer logFile.Close()

	_, err = logFile.Write([]byte("Hello World"))
	if err != nil {
		t.Fatalf("coud not append a new line to a log file: %v", err)
	}

	// Ensure number of lines has not increased
	integration.WaitLineCountInFile(t, outputFile, 10)

	// add \n to logfile
	_, err = logFile.Write([]byte("\n"))
	if err != nil {
		t.Fatalf("coud not append a new line to a log file: %v", err)
	}

	// Add one more line to make sure it keeps reading
	integration.WriteLogFile(t, filepath.Join(tempDir, "log.log"), 1, true)

	// Ensure all logs are ingested
	integration.WaitLineCountInFile(t, outputFile, 12)

	// rename the file
	assert.NoError(t, os.Rename(logFilePath, filepath.Join(tempDir, "newlog.log")))

	// using 6 events to have a separate log line that we can
	// grep for.
	integration.WriteLogFile(t, filepath.Join(tempDir, "newlog.log"), 6, true)

	// Ensure all logs are ingested
	integration.WaitLineCountInFile(t, outputFile, 18)

}

// Checks only the log lines defined by include_lines are ingested
func TestIncludeLines(t *testing.T) {

	var filestreamCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-clean-inactive"
    paths:
      - %s
    include_lines: ['^ERR', '^WARN']
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
`
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path, but do not write data to it
	logFilePath := filepath.Join(tempDir, "log.log")

	// 2. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCfg, filepath.Join(tempDir, "*.log"), tempDir))
	filebeat.Start()

	// 3. Create the log file
	iterations := 20
	integration.WriteLogFile(t, logFilePath, iterations, false, "DBG: a simple debug message")
	integration.WriteLogFile(t, logFilePath, iterations, true, "ERR: a simple error message")
	integration.WriteLogFile(t, logFilePath, iterations, true, "WARNING: a simple warning message")

	// wait for output file to exist
	var outputFile string
	require.Eventually(t, func() bool {
		matches, err := filepath.Glob(filepath.Join(tempDir, "output-file-*.ndjson"))
		if err != nil || len(matches) == 0 {
			t.Logf("could not find output file %v", err)
			return false
		}
		outputFile = matches[0]
		return true
	}, 2*time.Minute, 10*time.Second)

	// Ensure include_lines only events are ingested
	integration.WaitLineCountInFile(t, outputFile, 2*iterations)
}

// Checks log lines defined by exclude_lines are excluded
func TestExcludeLines(t *testing.T) {

	var filestreamCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-clean-inactive"
    paths:
      - %s
    exclude_lines: ['^DBG']
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
`
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path, but do not write data to it
	logFilePath := filepath.Join(tempDir, "log.log")

	// 2. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCfg, filepath.Join(tempDir, "*.log"), tempDir))
	filebeat.Start()

	// 3. Create the log file
	iterations := 20
	integration.WriteLogFile(t, logFilePath, iterations, false, "DBG: a simple debug message")
	integration.WriteLogFile(t, logFilePath, iterations, true, "ERR: a simple error message")
	integration.WriteLogFile(t, logFilePath, iterations, true, "WARNING: a simple warning message")

	// wait for output file to exist
	var outputFile string
	require.Eventually(t, func() bool {
		matches, err := filepath.Glob(filepath.Join(tempDir, "output-file-*.ndjson"))
		if err != nil || len(matches) == 0 {
			t.Logf("could not find output file %v", err)
			return false
		}
		outputFile = matches[0]
		return true
	}, 2*time.Minute, 10*time.Second)

	integration.WaitLineCountInFile(t, outputFile, 2*iterations)
}

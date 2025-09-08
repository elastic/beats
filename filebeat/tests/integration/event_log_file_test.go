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
)

var eventsLogFileCfg = `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    parsers:
      - ndjson:
          target: ""
          overwrite_keys: true
          expand_keys: true
          add_error_key: true
          ignore_decoding_error: false
    paths:
      - %s

output:
  elasticsearch:
    hosts:
      - localhost:9200
    protocol: http
    username: admin
    password: testing

logging:
  level: info
  event_data:
    files:
      name: filebeat-my-event-log
`

func TestEventsLoggerESOutput(t *testing.T) {
	// First things first, ensure ES is running and we can connect to it.
	// If ES is not running, the test will timeout and the only way to know
	// what caused it is going through Filebeat's logs.
	integration.EnsureESIsRunning(t)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	logFilePath := filepath.Join(filebeat.TempDir(), "log.log")
	filebeat.WriteConfigFile(fmt.Sprintf(eventsLogFileCfg, logFilePath))

	logFile, err := os.Create(logFilePath)
	if err != nil {
		t.Fatalf("could not create file '%s': %s", logFilePath, err)
	}

	_, _ = logFile.WriteString(`
{"message":"foo bar","int":10,"string":"str"}
{"message":"index failure 1","int":"not a number","string":10}
{"message":"another message","int":20,"string":"str2"}
{"message":"index failure 2","int":"not a number","string":10}
{"message":"index failure 3","int":"not a number","string":10}
`)
	if err := logFile.Sync(); err != nil {
		t.Fatalf("could not sync log file '%s': %s", logFilePath, err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("could not close log file '%s': %s", logFilePath, err)
	}

	filebeat.Start()

	// Wait for a log entry that indicates an entry in the events
	// logger file.
	msg := "Failed to index 3 events in last"
	require.Eventually(t, func() bool {
		return filebeat.LogContains(msg)
	}, time.Minute, 100*time.Millisecond,
		fmt.Sprintf("String '%s' not found on Filebeat logs", msg))

	// The glob here matches the configured value for the filename
	glob := filepath.Join(filebeat.TempDir(), "filebeat-my-event-log*.ndjson")
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("could not read files matching glob '%s': %s", glob, err)
	}
	if len(files) != 1 {
		t.Fatalf("there must be only one file matching the glob '%s', found: %s", glob, files)
	}

	eventsLogFile := files[0]
	data, err := os.ReadFile(eventsLogFile)
	if err != nil {
		t.Fatalf("could not read '%s': %s", eventsLogFile, err)
	}

	strData := string(data)
	eventMsg := `\"int\":\"not a number\"`
	if !strings.Contains(strData, eventMsg) {
		t.Errorf("expecting to find '%s' on '%s'", eventMsg, eventsLogFile)
		t.Errorf("Contents:\n%s", strData)
		t.FailNow()
	}

	// Ensure the normal log file does not contain the event data
	if filebeat.LogContains(eventMsg) {
		t.Fatalf("normal log file must NOT contain event data, '%s' found in the logs", eventMsg)
	}
}

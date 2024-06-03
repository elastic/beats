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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

var filebeatBasicConfig = `
filebeat.inputs:
  - type: filestream
    id: "test-filebeat-can-log"
    paths:
      - %s
path.home: %s
output.discard.enabled: true
`

func TestFilebeatRunsAndLogsJSONToFile(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.RemoveAllCLIArgs()

	tempDir := filebeat.TempDir()

	// 1. Generate the log file path, but do not write data to it
	logFilePath := path.Join(tempDir, "log.log")

	// 2. Create the log file
	integration.GenerateLogFile(t, logFilePath, 10, false)

	// 3. Write configuration file ans start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filebeatBasicConfig, logFilePath, tempDir))
	filebeat.Start()

	// We're not interested in data ingestion, we just want to ensure Filebeat can run
	// and logs to file when no logging is explicitly configured

	// The default logs home path is `path.logs: ${path.home}/logs`
	logFileName := fmt.Sprintf("filebeat-%s.ndjson", time.Now().Format("20060102"))
	filebeatLogFile := filepath.Join(tempDir, "logs", logFileName)

	var f *os.File
	var err error
	// We have to wait until the file is created, so we wait
	// until `os.Open` returns no error.
	require.Eventuallyf(t, func() bool {
		f, err = os.Open(filebeatLogFile)
		return err == nil
	}, 10*time.Second, time.Millisecond, "could not read log file '%s'", filebeatLogFile)
	defer f.Close()

	r := bufio.NewScanner(f)
	count := 0
	for r.Scan() {
		line := r.Bytes()
		m := map[string]any{}
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("line %d is not a valid JSON: %s", count, err)
		}
		count++
	}
}

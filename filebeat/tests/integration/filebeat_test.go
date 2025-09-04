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
	integration.WriteLogFile(t, logFilePath, 10, false)

	// 3. Write configuration file and start Filebeat
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
	}, 10*time.Second, 100*time.Millisecond, "could not read log file '%s'", filebeatLogFile)
	defer f.Close()

	r := bufio.NewScanner(f)
	count := 0
	for r.Scan() {
		line := r.Bytes()
		m := map[string]any{}
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("line %d is not a valid JSON: %s: %s", count, err, string(line))
		}
		count++
	}
}

func TestCleanInactiveValidation(t *testing.T) {
	testCases := map[string]struct {
		cfg      string
		log      string
		exitCode int
	}{
		"clean_inactive smaller than ignore_older plus check_interval": {
			log:      "clean_inactive must be greater than ignore_older + prospector.scanner.check_interval",
			exitCode: 1,
			cfg: `
filebeat.inputs:
- type: filestream
  id: my-filestream-id
  clean_inactive: 5m
  ignore_older: 10m
  paths:
    - /var/log/*.log

output.discard:
  enabled: true
`,
		},
		"clean_inactive can only be used if ignore_older is enabled": {
			log:      "clean_inactive can only be enabled if ignore_older is also enabled",
			exitCode: 1,
			cfg: `
filebeat.inputs:
- type: filestream
  id: my-filestream-id
  clean_inactive: 42h
  paths:
    - /var/log/*.log
output.discard:
  enabled: true
`,
		},
		"correct configuration": {
			log: "Input 'filestream' starting",
			cfg: `
filebeat.inputs:
- type: filestream
  id: my-filestream-id
  clean_inactive: 42h42m
  ignore_older: 42h
  paths:
    - /var/log/*.log

output.discard:
  enabled: true
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)

			filebeat.WriteConfigFile(tc.cfg)

			// Set the expected exit code to 1 if we're expecting
			//Filebeat to exit with an error
			filebeat.SetExpectedErrorCode(tc.exitCode)

			filebeat.Start()

			if tc.log != "" {
				filebeat.WaitLogsContains(tc.log, 10*time.Second)
			}
		})
	}
}

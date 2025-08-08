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

// This test checks all input-reloading related behavior
// 1. Ensures a new input file is correctly harvested
// 2. Disable the input file and make sure the harvester stops
// 3. Add another enabled input file and ensure it is picked up for reloading
func TestFilebeatInputReload(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	configTemplate := `
filebeat.config.inputs:
  path: %s/*.yml
  reload.enabled: true

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
logging.level: debug
`

	inputConfig := `
- type: filestream
  enabled: true
  id: id-filestream
  paths:
   - %s
  file_identity.native: ~
  prospector.scanner.fingerprint.enabled: false	   
`
	inputs := filepath.Join(tempDir, "inputs.d")
	err := os.MkdirAll(inputs, 0777)
	if err != nil {
		t.Fatalf("failed to create a module directory: %v", err)
	}

	// 1. Generate the log file path, but do not write data to it
	logFilePath := filepath.Join(tempDir, "log.log")

	// 2. Write configuration file
	filebeat.WriteConfigFile(fmt.Sprintf(configTemplate, inputs, tempDir))

	// 3. Create the log file
	integration.WriteLogFile(t, logFilePath, 10, false)

	assert.NoError(t, os.WriteFile(filepath.Join(inputs, "filestream.yml"), []byte(fmt.Sprintf(inputConfig, logFilePath)), 0666))

	// 4. Start
	filebeat.Start()

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

	assert.NoError(t, os.Rename(filepath.Join(inputs, "filestream.yml"), filepath.Join(inputs, "filestream.yml.disabled")))
	filebeat.WaitLogsContains("Runner: 'filestream' has stopped", 2*time.Minute)

	logFilePath2 := filepath.Join(tempDir, "log2.log")
	integration.WriteLogFile(t, logFilePath2, 10, false)
	// bring another file up
	assert.NoError(t, os.WriteFile(filepath.Join(inputs, "secondInput.yml"), []byte(fmt.Sprintf(inputConfig, logFilePath2)), 0666))

	// Ensure all log lines are ingested eventually
	integration.WaitLineCountInFile(t, outputFile, 20)
}

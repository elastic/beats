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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// TestGlobalCacheProcessorMultipleInputs tests that a global file-backed cache
// processor works correctly when multiple inputs connect to the pipeline.
func TestGlobalCacheProcessorMultipleInputs(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// Config with multiple inputs and a global file-backed cache processor.
	configTemplate := `
filebeat.inputs:
  - type: filestream
    id: input-1
    enabled: true
    paths:
      - %s
    prospector.scanner.fingerprint.enabled: false

  - type: filestream
    id: input-2
    enabled: true
    paths:
      - %s
    prospector.scanner.fingerprint.enabled: false

  - type: filestream
    id: input-3
    enabled: true
    paths:
      - %s
    prospector.scanner.fingerprint.enabled: false

# Global cache processor - shared by ALL inputs.
# Tests that SetPaths works correctly when multiple inputs connect.
processors:
  - cache:
      backend:
        file:
          id: test-cache
          write_interval: 1s
        capacity: 1000
      put:
        key_field: message
        value_field: message
        ttl: 1h
      ignore_missing: true

path.home: %s

output.file:
  path: ${path.home}
  filename: output
  codec.json:
    pretty: false

logging.level: info
`

	// Create log files for each input
	logFile1 := filepath.Join(tempDir, "input1.log")
	logFile2 := filepath.Join(tempDir, "input2.log")
	logFile3 := filepath.Join(tempDir, "input3.log")
	filebeat.WriteConfigFile(fmt.Sprintf(
		configTemplate,
		logFile1,
		logFile2,
		logFile3,
		tempDir,
	))

	// Write test data to all log files
	integration.WriteLogFile(t, logFile1, 10, false)
	integration.WriteLogFile(t, logFile2, 10, false)
	integration.WriteLogFile(t, logFile3, 10, false)

	filebeat.Start()
	filebeat.WaitPublishedEvents(30*time.Second, 30)

	// Verify the cache file was created in the correct location
	cacheFile := filepath.Join(tempDir, "data", "cache_processor", "test-cache")
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.FileExists(c, cacheFile)
	}, 10*time.Second, 100*time.Millisecond)
}

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
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// harvesterEvent is the subset of a published event this test asserts on.
type harvesterEvent struct {
	Message   string `json:"message"`
	SharedTag string `json:"shared_tag"`
	Log       struct {
		File struct {
			Path string `json:"path"`
		} `json:"file"`
	} `json:"log"`
}

func TestInputProcessorAppliedToAllHarvesters(t *testing.T) {
	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()

	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: shared-proc-input
    paths:
      - %s
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    processors:
      - add_fields:
          target: ""
          fields:
            shared_tag: applied

path.home: %s

output.file:
  path: ${path.home}
  filename: output
  codec.json:
    pretty: false

logging.level: info
`, filepath.Join(tempDir, "*.log"), tempDir)
	filebeat.WriteConfigFile(cfg)

	const numFiles = 5
	const linesPerFile = 10
	const totalEvents = numFiles * linesPerFile
	expectedSources := map[string]int{}
	for i := 0; i < numFiles; i++ {
		p := filepath.Join(tempDir, fmt.Sprintf("f%d.log", i))
		integration.WriteLogFile(t, p, linesPerFile, false)
		expectedSources[p] = linesPerFile
	}

	filebeat.Start()
	filebeat.WaitPublishedEvents(30*time.Second, totalEvents)

	events := integration.GetEventsFromFileOutput[harvesterEvent](filebeat, totalEvents, false)
	assert.Lenf(t, events, totalEvents, "expected one published event per written line")

	// The shared instance must be applied per client
	sources := map[string]int{}
	for i, e := range events {
		assert.Equalf(t, "applied", e.SharedTag,
			"event %d (%q) is missing the shared input processor field", i, e.Message)
		assert.NotEmptyf(t, e.Log.File.Path, "event %d has no source file path", i)
		sources[e.Log.File.Path]++
	}
	assert.Equal(t, expectedSources, sources,
		"every harvester must contribute all its events, with no missing or extra sources")

	filebeat.Stop()
	filebeat.WaitLogsContains("filebeat stopped.", 30*time.Second, "Filebeat did not stop cleanly")
}

func TestBadInputProcessorConfigFailsFast(t *testing.T) {
	cfg := `
filebeat.inputs:
  - type: filestream
    id: bad-processor-input
    paths:
      - /var/log/*.log
    processors:
      - add_fields:
          INVALID_CONFIG_KEY: true
          fields:
            tag: example

output.discard:
  enabled: true
`
	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitLogsContains(
		"Exiting: Failed to start crawler: starting input failed: error while initializing input: "+
			"unexpected INVALID_CONFIG_KEY option in filebeat.inputs.0.processors.0.add_fields",
		30*time.Second,
	)

	require.Error(t, filebeat.Cmd.Wait(), "Filebeat must exit on a broken input processor config")
	assert.Equal(t, 1, filebeat.Cmd.ProcessState.ExitCode())
}

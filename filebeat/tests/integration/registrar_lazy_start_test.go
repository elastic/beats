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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

const registrarLoadedMsg = "States Loaded from registrar"

// TestRegistrarNotStartedWithFilestreamOnly ensures that a Filebeat running
// only V2 inputs never starts the registrar nor scans the registry for it.
func TestRegistrarNotStartedWithFilestreamOnly(t *testing.T) {
	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()

	logFilePath := filepath.Join(tempDir, "log.log")
	integration.WriteLogFile(t, logFilePath, 10, false)

	filebeat.WriteConfigFile(fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: id-filestream
    paths:
      - %s
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false

path.home: %s
output.file:
  path: ${path.home}
  filename: "output-file"
logging.level: debug
`, logFilePath, tempDir))

	filebeat.Start()

	outputGlob := filepath.Join(tempDir, "output-file-*.ndjson")
	require.Eventually(t, func() bool {
		return filebeat.CountFileLines(outputGlob) == 10
	}, 2*time.Minute, time.Second, "filestream did not ingest all events")

	assert.Empty(t, filebeat.GetLogLine(registrarLoadedMsg),
		"the registrar must not load states when only filestream runs")

	filebeat.Stop()

	// Stop must still be clean even though the registrar never ran.
	assert.NotEmpty(t, filebeat.GetLogLine("Registrar stopped"),
		"the registrar must be stopped on shutdown")
	assert.Empty(t, filebeat.GetLogLine(registrarLoadedMsg),
		"the registrar must not load states when only filestream runs")
}

// TestRegistrarStartedWhenLogInputIsReloaded ensures that the registrar, not
// started with filestream only, is started when a V1 input is added later by a
// config reload, and that the state of that input is persisted.
func TestRegistrarStartedWhenLogInputIsReloaded(t *testing.T) {
	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()

	inputs := filepath.Join(tempDir, "inputs.d")
	require.NoError(t, os.MkdirAll(inputs, 0o777), "failed to create inputs.d")

	filebeat.WriteConfigFile(fmt.Sprintf(`
filebeat.config.inputs:
  path: %s/*.yml
  reload.enabled: true
  reload.period: 1s

path.home: %s
output.file:
  path: ${path.home}
  filename: "output-file"
logging.level: debug
`, inputs, tempDir))

	filestreamLog := filepath.Join(tempDir, "filestream.log")
	integration.WriteLogFile(t, filestreamLog, 10, false)
	require.NoError(t, os.WriteFile(filepath.Join(inputs, "filestream.yml"), fmt.Appendf(nil, `
- type: filestream
  id: id-filestream
  paths:
    - %s
  file_identity.native: ~
  prospector.scanner.fingerprint.enabled: false
`, filestreamLog), 0o666))

	filebeat.Start()

	outputGlob := filepath.Join(tempDir, "output-file-*.ndjson")
	require.Eventually(t, func() bool {
		return filebeat.CountFileLines(outputGlob) == 10
	}, 2*time.Minute, time.Second, "filestream did not ingest all events")
	assert.Empty(t, filebeat.GetLogLine(registrarLoadedMsg),
		"the registrar must not be started before a V1 input exists")

	// Reload: add a log input.
	logInputLog := filepath.Join(tempDir, "loginput.log")
	integration.WriteLogFile(t, logInputLog, 10, false)
	require.NoError(t, os.WriteFile(filepath.Join(inputs, "log.yml"), fmt.Appendf(nil, `
- type: log
  id: id-log
  allow_deprecated_use: true
  paths:
    - %s
`, logInputLog), 0o666))

	filebeat.WaitLogsContains(registrarLoadedMsg, time.Minute,
		"the registrar must start when the log input is created")
	require.Eventually(t, func() bool {
		return filebeat.CountFileLines(outputGlob) == 20
	}, 2*time.Minute, time.Second, "log input did not ingest all events")

	// Stop writes the final registry state.
	filebeat.Stop()

	registry := filepath.Join(tempDir, "data", "registry", "filebeat", "log.json")
	assert.Eventually(t, func() bool {
		data, err := os.ReadFile(registry)
		return err == nil && strings.Contains(string(data), filepath.Base(logInputLog))
	}, 10*time.Second, 100*time.Millisecond,
		"the registry %s must contain the state of the log input", registry)
}

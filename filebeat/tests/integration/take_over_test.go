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
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

const (
	takeOverBaseCfg = `
filebeat.inputs:
{{ if eq .Type "log" }}
  - type: log
    allow_deprecated_use: true
    scan_frequency: 1s
{{ end }}

{{ if eq .Type "filestream" }}
  - type: filestream
    id: take-over-test
    take_over: true
    prospector.scanner.check_interval: 1s
{{ end }}
    paths:
      - {{ .LogFile }}

filebeat.registry.flush: 0s
path.home: {{ .WorkDir }}

output.file:
  path: ${path.home}
  filename: output
  rotate_on_startup: false

queue.mem:
  flush.timeout: 0s

logging.level: debug
`
)

type takeOverEvent struct {
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
	Message string `json:"message"`
}

func TestFilebeatTakeOverAfterRestart(t *testing.T) {
	const batchSize = 25

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFile := filepath.Join(tempDir, "log.log")
	integration.WriteLogFile(t, logFile, batchSize, false)

	tmpl := template.Must(template.New("log-Cfg").Parse(takeOverBaseCfg))
	cfg := strings.Builder{}
	values := map[string]string{
		"Type":    "log",
		"WorkDir": filebeat.TempDir(),
		"LogFile": logFile,
	}
	require.NoError(t, tmpl.Execute(&cfg, values), "cannot render Log input configuration")

	filebeat.WriteConfigFile(cfg.String())
	filebeat.Start()

	initialEvents := batchSize
	filebeat.WaitPublishedEvents(30*time.Second, initialEvents)
	filebeat.Stop()

	cfg.Reset()
	values["Type"] = "filestream"
	require.NoError(t, tmpl.Execute(&cfg, values), "cannot render Filestream input configuration")
	filebeat.WriteConfigFile(cfg.String())

	filebeat.Start()
	filebeat.WaitLogsContains(
		"filestream-takeover",
		30*time.Second,
		"filestream takeover migration was not logged",
	)

	registryBackup, err := filepath.Glob(filepath.Join(tempDir, "data", "registry", "filebeat", "*.bak"))
	require.NoError(t, err, "failed to locate registry backups")
	require.NotEmpty(t, registryBackup, "takeover did not create a registry backup")

	integration.WriteLogFile(t, logFile, batchSize, false, "second run")

	totalEvents := batchSize * 2
	filebeat.WaitPublishedEvents(30*time.Second, totalEvents)
	filebeat.Stop()

	events := integration.GetEventsFromFileOutput[takeOverEvent](filebeat, totalEvents, true)
	require.Len(t, events, totalEvents, "unexpected number of published events")

	// First events are from the Log input
	for i := range batchSize {
		if events[i].Input.Type != "log" {
			t.Errorf("Event %02d is not from the Log input", i)
		}
	}

	for i := batchSize; i < totalEvents; i++ {
		if events[i].Input.Type != "filestream" {
			t.Errorf("Event %02d is not from the Filestream input", i)
		}
	}
}

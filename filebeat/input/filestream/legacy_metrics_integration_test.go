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

package filestream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

var fconfig = `
filebeat.inputs:
  - type: filestream
    id: my-filestream-id
    enabled: true
    close.reader.after_interval: 1s
    prospector.scanner.check_interval: 500ms
    paths:
      - %s/*.filestream
  - type: log
    id: my-log-input
    enabled: true
    close_timeout: 1s
    scan_frequency: 500ms
    paths:
      - %s/*.log

output.console:
  codec.json:
    pretty: true

logging:
  level: debug
  selectors: "*"

http:
  enabled: true
`

func TestLegacyMetrics(t *testing.T) {
	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")

	cfg := fmt.Sprintf(fconfig, filebeat.TempDir(), filebeat.TempDir())

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitForLogs("Metrics endpoint listening on:", 10*time.Second)

	// After starting Filebeat all counters must be zero
	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 0,
			Closed:    0,
			Running:   0,
			Started:   0,
		})

	filestreamLogFile := filepath.Join(filebeat.TempDir(), "01.filestream")
	filestreamLog, err := os.Create(filestreamLogFile)
	if err != nil {
		t.Fatalf("could not create log file '%s': %s", filestreamLogFile, err)
	}

	// Write a line in the file harvested by Filestream
	fmt.Fprintln(filestreamLog, "first line")

	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 1,
			Running:   1,
			Started:   1,
			Closed:    0,
		},
		"Filestream input did not start the harvester")

	// Wait for the harvester to close the file
	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 0,
			Running:   0,
			Started:   1,
			Closed:    1,
		},
		"Filestream input did not close the harvester")

	// Write a line in the file harvested by the log input
	logInputLogFileName := filepath.Join(filebeat.TempDir(), "01.log")
	logInputLog, err := os.Create(logInputLogFileName)
	if err != nil {
		t.Fatalf("could not create log file '%s': %s", logInputLogFileName, err)
	}

	fmt.Fprintln(logInputLog, "first line")

	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 1,
			Running:   1,
			Started:   2,
			Closed:    1,
		},
		"Log input did not start harvester")

	// Wait for the log input to close the file
	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 0,
			Running:   0,
			Started:   2,
			Closed:    2,
		},
		"Log input did not close the harvester")

	// Writes one more line to each log file
	fmt.Fprintln(logInputLog, "second line")
	fmt.Fprintln(filestreamLog, "second line")

	// Both harvesters should be running
	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 2,
			Running:   2,
			Started:   4,
			Closed:    2,
		},
		"Two harvesters should be running")

	// Wait for both harvesters to close the file
	waitForMetrics(t,
		LegacyHarvesterMetrics{
			OpenFiles: 0,
			Running:   0,
			Started:   4,
			Closed:    4,
		},
		"All harvesters must be closed")
}

func waitForMetrics(t *testing.T, expect LegacyHarvesterMetrics, msgAndArgs ...any) {
	t.Helper()
	got := LegacyHarvesterMetrics{}
	assert.Eventually(t, func() bool {
		got = getHarvesterMetrics(t)
		return expect == got
	}, 10*time.Second, 100*time.Millisecond, msgAndArgs...)

	if !t.Failed() {
		return
	}

	if expect.Closed != got.Closed {
		t.Logf("expecting 'closed' to be %d, got %d instead", expect.Closed, got.Closed)
	}

	if expect.OpenFiles != got.OpenFiles {
		t.Logf("expecting 'open_files' to be %d, got %d instead", expect.OpenFiles, got.OpenFiles)
	}

	if expect.Running != got.Running {
		t.Logf("expecting 'running' to be %d, got %d instead", expect.Running, got.Running)
	}

	if expect.Started != got.Started {
		t.Logf("expecting 'started' to be %d, got %d instead", expect.Started, got.Started)
	}
}

func compareMetrics(t *testing.T, expect, got LegacyHarvesterMetrics) {
	t.Helper()

	if expect.Closed != got.Closed {
		t.Errorf("expecting 'closed' to be %d, got %d instead", expect.Closed, got.Closed)
	}

	if expect.OpenFiles != got.OpenFiles {
		t.Errorf("expecting 'open_files' to be %d, got %d instead", expect.OpenFiles, got.OpenFiles)
	}

	if expect.Running != got.Running {
		t.Errorf("expecting 'running' to be %d, got %d instead", expect.Running, got.Running)
	}

	if expect.Started != got.Started {
		t.Errorf("expecting 'started' to be %d, got %d instead", expect.Started, got.Started)
	}
}

type LegacyHarvesterMetrics struct {
	Closed    int `json:"closed"`
	OpenFiles int `json:"open_files"`
	Running   int `json:"running"`
	Started   int `json:"started"`
}

func getHarvesterMetrics(t *testing.T) LegacyHarvesterMetrics {
	// The host is ignored because we're connecting via Unix sockets.
	resp, err := http.Get("http://localhost:5066/stats")
	if err != nil {
		t.Fatalf("could not execute HTTP call: %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read request body: %s", err)
	}

	type foo struct {
		F struct {
			H LegacyHarvesterMetrics `json:"harvester"`
		} `json:"filebeat"`
	}

	m := struct {
		F struct {
			H LegacyHarvesterMetrics `json:"harvester"`
		} `json:"filebeat"`
	}{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("could not unmarshal request body: %s", err)
	}

	return m.F.H
}

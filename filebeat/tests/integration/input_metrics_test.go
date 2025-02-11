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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestInputMetricsFromPipeline_Filestream(t *testing.T) {
	var filestreamCfg = `
http:
  enabled: true
filebeat.inputs:
  - type: filestream
    id: a-filestream-id
    enabled: true
    paths:
      - %s
    processors:
      - drop_event:
          when:
            regexp:
              message: "PUT"
    close.reader.after_interval: 10m

queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s

path.home: %s

output.file:
  path: ${path.home}
  filename: output-file
  rotate_every_kb: 10000

logging.level: debug
`

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path

	relativePath := filepath.Join("testdata", "input_metrics.log")
	logFilePath, err := filepath.Abs(relativePath)
	require.NoError(t, err, "Failed to get absolute path for", relativePath)

	// 2. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCfg, logFilePath, tempDir))
	// filebeat.WriteConfigFile(filestreamCfg)
	filebeat.Start()

	// 4. Wait for Filebeat to start scanning for files
	filebeat.WaitForLogs(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		30*time.Second,
		"Filebeat did not start looking for files to ingest")

	filebeat.WaitForLogs(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath),
		10*time.Second, "Filebeat did not close the file")

	// 5. Now that the reader has been closed, the file is ingested.// todo: is it?
	time.Sleep(5 * time.Second)
	resp, err := http.Get("http://localhost:5066/inputs/")
	require.NoError(t, err, "failed fetching input metrics")
	defer resp.Body.Close()

	var inputMetrics []struct {
		EventsDroppedTotal   int    `json:"events_dropped_total"`
		EventsFilteredTotal  int    `json:"events_filtered_total"`
		EventsProcessedTotal int    `json:"events_processed_total"`
		EventsPublishedTotal int    `json:"events_published_total"`
		ID                   string `json:"id"`
		Input                string `json:"input"`
	}

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed reading response body")
	err = json.Unmarshal(body, &inputMetrics)
	require.NoError(t, err, "failed unmarshalling response body")

	require.Len(t, inputMetrics, 1)
	assert.Equal(t, "a-filestream-id", inputMetrics[0].ID)
	assert.Equal(t, "filestream", inputMetrics[0].Input)
	assert.Equal(t, 10, inputMetrics[0].EventsProcessedTotal)
	assert.Equal(t, 9, inputMetrics[0].EventsPublishedTotal)
	assert.Equal(t, 1, inputMetrics[0].EventsFilteredTotal)

	assert.Falsef(t, t.Failed(), "test faild: input metrics response used for the assertions: %s", body)
}

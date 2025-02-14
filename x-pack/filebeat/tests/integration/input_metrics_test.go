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
	"net/http/httptest"
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
            contains:
              message: "PUT"
    close.reader.after_interval: 10m

  - type: cel
    id: a-cel-input-id
    interval: 1s
    resource.url: %s
    program: bytes(get(state.url).Body).as(body,{"events":[body.decode_json()]})
    publisher_pipeline.disable_host: true
    processors:
      - drop_event:
          when:
            equals:
              ip: "1.1.1.1"

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
	celSrv := makeCelServer()
	defer celSrv.Close()

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
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCfg, logFilePath, celSrv.URL, tempDir))
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

	type inputMetric struct {
		EventsPipelineTotal          int `json:"events_pipeline_total"`
		EventsPipelineDroppedTotal   int `json:"events_pipeline_dropped_total"`
		EventsPipelineFilteredTotal  int `json:"events_pipeline_filtered_total"`
		EventsPipelinePublishedTotal int `json:"events_pipeline_published_total"`

		// EventsPublishedTotal is used by: filestream
		EventsProcessedTotal int `json:"events_processed_total"`
		// EventsPublishedTotal is used by: cel
		EventsPublishedTotal int    `json:"events_published_total"`
		ID                   string `json:"id"`
		Input                string `json:"input"`
	}

	inputMetrics := []inputMetric{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed reading response body")
	err = json.Unmarshal(body, &inputMetrics)
	require.NoError(t, err, "failed unmarshalling response body")

	assertionsByInputID := map[string]func(t *testing.T, metrics inputMetric){
		"a-filestream-id": func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "filestream", metrics.Input)
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal+
					metrics.EventsPipelineDroppedTotal)
			assert.Equal(t, metrics.EventsProcessedTotal, metrics.EventsPipelineTotal)
			assert.Equal(t, 10, metrics.EventsProcessedTotal)
			assert.Equal(t, 9, metrics.EventsPipelinePublishedTotal)
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal)
		},
		fmt.Sprintf("a-cel-input-id::%s", celSrv.URL): func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "cel", metrics.Input)
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal+
					metrics.EventsPipelineDroppedTotal)
			assert.Equal(t, metrics.EventsPublishedTotal, metrics.EventsPipelineTotal)
			assert.Equal(t, 2, metrics.EventsPublishedTotal)
			assert.Equal(t, 1, metrics.EventsPipelinePublishedTotal)
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal)

		},
	}

	assert.Len(t, inputMetrics, len(assertionsByInputID),
		"unexpected number of input reporting metrics. Some input"+
			"assertions might have not run")
	for _, inpMetric := range inputMetrics {
		assertions, ok := assertionsByInputID[inpMetric.ID]
		if !ok {
			t.Errorf("no assertions found for input id %s. "+
				"Continuing with other assertions", inpMetric.ID)
			continue
		}
		assertions(t, inpMetric)
	}

	assert.Falsef(t, t.Failed(), "test faild: input metrics response used for the assertions: %s", body)
}

// makeCelServer returns a *httptest.Server to mock a server called by the 'cel'
// input. It reruns 2 successful responses then all following responses are a
// HTTP 500.
func makeCelServer() *httptest.Server {
	celEventsTotal := 2
	celEventsIdx := 0
	celResponses := []string{"{\"ip\":\"0.0.0.0\"}", "{\"ip\":\"1.1.1.1\"}"}
	celSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if celEventsIdx >= celEventsTotal {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("won't send any more events"))
			return
		}
		_, _ = w.Write([]byte(celResponses[celEventsIdx]))
		celEventsIdx++
	}))
	return celSrv
}

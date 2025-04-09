// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputMetricsFromPipeline(t *testing.T) {
	var tmplCfg = `
http:
  enabled: true
  port: {{.port}}
filebeat.inputs:
  # an input without ID, therefore not metrics for it.
  - type: filestream
    enabled: true
    paths:
      - {{.log_path_no_id}}
    processors:
      - drop_event:
          when:
            contains:
              message: "PUT"
    close.reader.after_interval: 10m

  # an input which does not register input metrics
  - type: log
    id: log-input-id
    paths:
      - {{.log_path_no_id}}
    allow_deprecated_use: true

  - type: filestream
    id: {{.filestream_id}}
    enabled: true
    paths:
      - {{.log_path}}
    processors:
      - drop_event:
          when:
            contains:
              message: "PUT"
    close.reader.after_interval: 10m

  - type: cel
    id: {{.cel_id}}
    interval: 1s
    resource.url: {{.cel_resource_url}}
    program: bytes(get(state.url).Body).as(body,{"events":[body.decode_json()]})
    publisher_pipeline.disable_host: true
    processors:
      - drop_event:
          when:
            equals:
              ip: "1.1.1.1"

  - type: httpjson
    id: {{.httpjson_id}}
    interval: 1s
    request.url: {{.httpjson_requestURL}}
    processors:
      - drop_event:
          when:
            contains:
              message: "1.1.1.1"

queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s

path.home: {{.path_home}}

output.file:
  path: ${path.home}
  filename: output-file
  rotate_every_kb: 10000

logging.level: debug
`

	port := randomPort(t)
	celSrv := makeServer()
	defer celSrv.Close()
	httpjsonSrv := makeServer()
	defer httpjsonSrv.Close()

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path

	relativePath := filepath.Join("testdata", "input_metrics.log")
	logFilePath, err := filepath.Abs(relativePath)
	require.NoError(t, err, "Failed to get absolute path for", relativePath)

	relativePath = filepath.Join("testdata", "input_metrics-no-id.log")
	logFileNoIDPath, err := filepath.Abs(relativePath)
	require.NoError(t, err, "Failed to get absolute path for", relativePath)

	// 2. Write configuration file and start Filebeat
	cgfSB := strings.Builder{}
	tmpl, err := template.New("filebeatConfig").Parse(tmplCfg)
	require.NoErrorf(t, err, "Failed to parse config template")

	filestreamInputID := "a-filestream-id"
	celBaseInputID := "a-cel-input-id"
	celInputID := fmt.Sprintf("%s::%s", celBaseInputID, celSrv.URL)
	httpsjonInputID := "a-httpjson-input-id"

	require.NoError(t, tmpl.Execute(&cgfSB, map[string]string{
		"filestream_id":       filestreamInputID,
		"cel_id":              celBaseInputID,
		"httpjson_id":         httpsjonInputID,
		"log_path":            logFilePath,
		"log_path_no_id":      logFileNoIDPath,
		"cel_resource_url":    celSrv.URL,
		"httpjson_requestURL": httpjsonSrv.URL,
		"path_home":           tempDir,
		"port":                port,
	}), "failed to execute config template")

	filebeat.WriteConfigFile(cgfSB.String())
	filebeat.Start()

	// 4. Wait for Filebeat to start scanning for files
	filebeat.WaitForLogs(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		30*time.Second,
		"Filebeat did not start looking for files to ingest")

	filebeat.WaitForLogs(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath),
		10*time.Second, "Filebeat did not close the file")

	// 5. Now that the file was fully read, we can make the assertions.

	type inputMetric struct {
		EventsPipelineTotal          int `json:"events_pipeline_total"`
		EventsPipelineFilteredTotal  int `json:"events_pipeline_filtered_total"`
		EventsPipelinePublishedTotal int `json:"events_pipeline_published_total"`

		// EventsProcessedTotal is used by: filestream
		EventsProcessedTotal int `json:"events_processed_total"`
		// EventsPublishedTotal is used by: cel
		EventsPublishedTotal int    `json:"events_published_total"`
		ID                   string `json:"id"`
		Input                string `json:"input"`
	}

	totalEventsByInput := map[string]int{
		filestreamInputID: 10,
		celInputID:        2,
		httpsjonInputID:   1,
	}
	wantInputMetricsCount := 4
	var inputMetrics []inputMetric
	var body []byte
	errMsg := strings.Builder{}
	defer func() {
		if t.Failed() {
			t.Errorf("test faild: input metrics response used for the assertions:\n%s",
				body)
		}
	}()
	require.Eventuallyf(t, func() bool {
		errMsg.Reset()
		inputMetrics = []inputMetric{}

		//nolint:noctx // on a test, it's ok
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/inputs/", port))
		if err != nil {
			errMsg.WriteString(fmt.Sprintf("request to /inputs/ failed: %v", err))
			return false
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			errMsg.WriteString(fmt.Sprintf("failed to read response body: %v", err))
			return false
		}
		err = json.Unmarshal(body, &inputMetrics)
		if err != nil {
			errMsg.WriteString(fmt.Sprintf("failed unmarshalling response body: %v", err))
			return false
		}

		if len(inputMetrics) != wantInputMetricsCount {
			errMsg.WriteString(
				fmt.Sprintf("want %d inputs, got %d",
					wantInputMetricsCount, len(inputMetrics)))
			return false
		}

		for _, metrics := range inputMetrics {
			want, ok := totalEventsByInput[metrics.ID]
			if !ok {
				continue
			}

			switch metrics.ID {
			case filestreamInputID:
				if want != metrics.EventsProcessedTotal {
					errMsg.WriteString(
						fmt.Sprintf("input %q wants %d events, got %d",
							filestreamInputID, want, metrics.EventsProcessedTotal))

					return false
				}
			case httpsjonInputID:
				if want != metrics.EventsPipelineFilteredTotal {
					errMsg.WriteString(
						fmt.Sprintf("input %q wants %d events, got %d",
							httpsjonInputID, want, metrics.EventsPipelineFilteredTotal))

					return false
				}
			case celInputID:
				if want != metrics.EventsPublishedTotal {
					errMsg.WriteString(
						fmt.Sprintf("input %q wants %d events, got %d",
							celInputID, want, metrics.EventsPublishedTotal))

					return false
				}
			}
		}

		return true
	}, 10*time.Second, 1*time.Second, "did not get necessary input metrics: %s", &errMsg)

	assertionsByInputID := map[string]func(t *testing.T, metrics inputMetric){
		filestreamInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "filestream", metrics.Input)
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal,
				"filestream EventsPipelineTotal != EventsPipelinePublishedTotal+EventsPipelineFilteredTotal")
			assert.Equal(t, metrics.EventsProcessedTotal,
				metrics.EventsPipelineTotal,
				"filestream EventsPipelineTotal != EventsProcessedTotal")
			assert.Equal(t, 10, metrics.EventsProcessedTotal,
				"filestream EventsProcessedTotal")
			assert.Equal(t, 9, metrics.EventsPipelinePublishedTotal,
				"filestream EventsPipelinePublishedTotal")
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal,
				"filestream EventsPipelineFilteredTotal")
		},
		celInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "cel", metrics.Input)
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal,
				"cel EventsPipelineTotal != EventsPipelinePublishedTotal+EventsPipelineFilteredTotal")
			assert.Equal(t, metrics.EventsPublishedTotal,
				metrics.EventsPipelineTotal,
				"cel EventsPublishedTotal != EventsPipelineTotal")
			assert.Equal(t, 2, metrics.EventsPublishedTotal)
			assert.Equal(t, 1, metrics.EventsPipelinePublishedTotal,
				"cel EventsPipelinePublishedTotal")
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal,
				"cel EventsPipelineFilteredTotal")
		},
		httpsjonInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "httpjson", metrics.Input)
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal,
				"httpjson EventsPipelineTotal != EventsPipelinePublishedTotal+EventsPipelineFilteredTotal")
			assert.Equal(t, 1, metrics.EventsPipelinePublishedTotal,
				"httpjson EventsPipelinePublishedTotal")
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal,
				"httpjson EventsPipelineFilteredTotal")
		},
	}

	count := 0
	for _, inpMetric := range inputMetrics {
		assertions, ok := assertionsByInputID[inpMetric.ID]
		if !ok {
			continue
		}
		count++
		assertions(t, inpMetric)
	}
	assert.Equalf(t, len(assertionsByInputID), count,
		"%d assertions should have run, but only %d run",
		len(assertionsByInputID), count)
}

// makeServer returns a *httptest.Server to mock a server called by an input.
// It reruns 2 successful responses then all following responses are an HTTP 500.
func makeServer() *httptest.Server {
	eventsTotal := 2
	eventsIdx := 0
	responses := []string{"{\"ip\":\"0.0.0.0\"}", "{\"ip\":\"1.1.1.1\"}"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if eventsIdx >= eventsTotal {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("won't send any more events"))
			return
		}
		_, _ = w.Write([]byte(responses[eventsIdx]))
		eventsIdx++
	}))
	return srv
}

func randomPort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "could not create nte.Listener to find a free port")
	defer listener.Close()

	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

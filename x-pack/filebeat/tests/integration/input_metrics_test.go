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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/mock-es/pkg/api"
)

type inputMetric struct {
	ID    string `json:"id"`
	Input string `json:"input"`

	// Pipeline metrics
	EventsPipelineTotal          int `json:"events_pipeline_total"`
	EventsPipelineFilteredTotal  int `json:"events_pipeline_filtered_total"`
	EventsPipelinePublishedTotal int `json:"events_pipeline_published_total"`

	// Output metrics
	EventsOutputAckedTotal           int `json:"events_output_acked_total"`
	EventsOutputDeadLetterTotal      int `json:"events_output_dead_letter_total"`
	EventsOutputDroppedTotal         int `json:"events_output_dropped_total"`
	EventsOutputDuplicateEventsTotal int `json:"events_output_duplicate_events_total"`
	EventsOutputErrTooManyTotal      int `json:"events_output_err_too_many_total"`
	EventsOutputRetryableErrorsTotal int `json:"events_output_retryable_errors_total"`
	EventsOutputTotal                int `json:"events_output_total"`

	// EventsProcessedTotal is used by: filestream
	EventsProcessedTotal int `json:"events_processed_total"`
	// EventsPublishedTotal is used by: cel
	EventsPublishedTotal int `json:"events_published_total"`
	// PagesPublishedTotal is used by httpjson
	PagesPublishedTotal int `json:"pages_published_total"`
}

type esEvent struct {
	RespondWith int `json:"respond-with"`
}

func TestInputMetricsFromPipelineAndOutput(t *testing.T) {
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
      - decode_json_fields:
          fields: ["message"]
          target: ""
          overwrite_keys: true
          add_error_key: true
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

output.elasticsearch:
  hosts: ["{{.es_url}}"]
  non_indexable_policy.dead_letter_index.index: "deadletter"

logging.level: debug
`
	port, filebeat, filestreamInputID, celInputID, httpsjonInputID :=
		initInputMetricsTest(t, tmplCfg)

	assertionsByInputID := map[string]func(t *testing.T, metrics inputMetric){
		filestreamInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "filestream", metrics.Input)

			// Assert pipeline metrics
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal,
				"filestream EventsPipelineTotal != EventsPipelinePublishedTotal+EventsPipelineFilteredTotal")
			assert.Equal(t, metrics.EventsProcessedTotal,
				metrics.EventsPipelineTotal,
				"filestream EventsPipelineTotal != EventsProcessedTotal")
			assert.Equal(t, 12, metrics.EventsProcessedTotal,
				"filestream EventsProcessedTotal")
			assert.Equal(t, 11, metrics.EventsPipelinePublishedTotal,
				"filestream EventsPipelinePublishedTotal")
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal,
				"filestream EventsPipelineFilteredTotal")

			// Assert output metrics
			assert.Equal(t,
				metrics.EventsPipelinePublishedTotal+2, // +2 retried events
				metrics.EventsOutputTotal,
				"EventsOutputTotal should equal EventsPipelinePublishedTotal +2 retried events")
			assert.Equal(t,
				9,
				metrics.EventsOutputAckedTotal,
				"EventsOutputAckedTotal should equal EventsPipelinePublishedTotal")
			assert.Equal(t,
				0,
				metrics.EventsOutputDroppedTotal,
				"unexpected EventsOutputDroppedTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputDuplicateEventsTotal,
				"unexpected EventsOutputDuplicateEventsTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputErrTooManyTotal,
				"unexpected EventsOutputErrTooManyTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputDeadLetterTotal,
				"unexpected EventsOutputDeadLetterTotal")

		},
		celInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "cel", metrics.Input)

			// Assert pipeline metrics
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

			// Assert output metrics
			assert.Equal(t, metrics.EventsPipelinePublishedTotal,
				metrics.EventsOutputTotal,
				"EventsOutputTotal should equal EventsPipelinePublishedTotal for %s",
				metrics.ID)
			assert.Equal(t, metrics.EventsPipelinePublishedTotal,
				metrics.EventsOutputAckedTotal,
				"EventsOutputAckedTotal should equal EventsPipelinePublishedTotal for %s",
				metrics.ID)
		},
		httpsjonInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "httpjson", metrics.Input)

			// Assert pipeline metrics
			assert.Equal(t,
				metrics.EventsPipelineTotal,
				metrics.EventsPipelinePublishedTotal+
					metrics.EventsPipelineFilteredTotal,
				"httpjson EventsPipelineTotal != EventsPipelinePublishedTotal+EventsPipelineFilteredTotal")
			assert.Equal(t, 1, metrics.EventsPipelinePublishedTotal,
				"httpjson EventsPipelinePublishedTotal")
			assert.Equal(t, 1, metrics.EventsPipelineFilteredTotal,
				"httpjson EventsPipelineFilteredTotal")

			// Assert output metrics
			assert.Equal(t, metrics.EventsPipelinePublishedTotal,
				metrics.EventsOutputTotal,
				"EventsOutputTotal should equal EventsPipelinePublishedTotal for %s",
				metrics.ID)
			assert.Equal(t, metrics.EventsPipelinePublishedTotal,
				metrics.EventsOutputAckedTotal,
				"EventsOutputAckedTotal should equal EventsPipelinePublishedTotal for %s",
				metrics.ID)
		},
	}

	wantInputMetricsCount := map[string]func(metrics inputMetric) error{
		filestreamInputID: func(metrics inputMetric) error {
			want := 12
			got := metrics.EventsProcessedTotal
			if got != want {
				return fmt.Errorf(
					"%q events_processed_total: want %d, got %d",
					filestreamInputID, want, got)
			}
			return nil
		},
		celInputID: func(metrics inputMetric) error {
			want := 2
			got := metrics.EventsPublishedTotal
			if got != want {
				return fmt.Errorf(
					"%q events_published_total: want %d, got %d",
					celInputID, want, got)
			}
			return nil
		},
		httpsjonInputID: func(metrics inputMetric) error {
			want := 2
			got := metrics.PagesPublishedTotal
			if got != want {
				return fmt.Errorf(
					"%q pages_published_total: want %d, got %d",
					httpsjonInputID, want, got)
			}
			return nil
		},
	}
	var inputMetrics []inputMetric
	var body []byte
	var err error
	errMsg := strings.Builder{}
	defer func() {
		saveInputMetricsOnFailure(t, filebeat, body)
	}()

	require.Eventuallyf(t, func() bool {
		errMsg.Reset()
		err, inputMetrics, body = findInputMetrics(port, wantInputMetricsCount, 1)
		if err != nil {
			errMsg.WriteString(err.Error())
			return false
		}

		return true
	}, 10*time.Second, 1*time.Second, "aaaaa did not get necessary input metrics: %s", &errMsg)

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

// TestInputMetricsFromPipelineAndOutput_Dropped tests verifies the dropped
// metric is correct. Due to ES output design, it either drops events or  it
// indefinitely retries to send the to te dead letter index. Thus, a test for
// each case is required.
func TestInputMetricsFromPipelineAndOutput_Dropped(t *testing.T) {
	var tmplCfg = `
http:
  enabled: true
  port: {{.port}}
filebeat.inputs:
  - type: filestream
    id: {{.filestream_id}}
    enabled: true
    paths:
      - {{.log_path}}
    processors:
      - decode_json_fields:
          fields: ["message"]
          target: ""
          overwrite_keys: true
          add_error_key: true
    close.reader.after_interval: 10m

queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s

path.home: {{.path_home}}

output.elasticsearch:
  hosts: ["{{.es_url}}"]

logging.level: debug
`
	port, filebeat, filestreamInputID, _, _ :=
		initInputMetricsTest(t, tmplCfg)

	assertionsByInputID := map[string]func(t *testing.T, metrics inputMetric){
		filestreamInputID: func(t *testing.T, metrics inputMetric) {
			assert.Equal(t, "filestream", metrics.Input)

			// Assert output metrics
			assert.Equal(t,
				10,
				metrics.EventsOutputAckedTotal,
				"EventsOutputAckedTotal should equal EventsPipelinePublishedTotal")
			assert.Equal(t,
				0,
				metrics.EventsOutputDeadLetterTotal,
				"unexpected EventsOutputDeadLetterTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputDroppedTotal,
				"unexpected EventsOutputDroppedTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputDuplicateEventsTotal,
				"unexpected EventsOutputDuplicateEventsTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputErrTooManyTotal,
				"unexpected EventsOutputErrTooManyTotal")
			assert.Equal(t,
				1,
				metrics.EventsOutputRetryableErrorsTotal,
				"unexpected EventsOutputRetryableErrorsTotal")
			assert.Equal(t,
				metrics.EventsPipelinePublishedTotal+1, // +1 retried event
				metrics.EventsOutputTotal,
				"EventsOutputTotal should equal EventsPipelinePublishedTotal +1 retried event")
		},
	}

	var inputMetrics []inputMetric
	var errMsg strings.Builder
	var body []byte
	var err error
	defer func() {
		saveInputMetricsOnFailure(t, filebeat, body)
	}()

	wantInputMetricsCount := map[string]func(metrics inputMetric) error{
		filestreamInputID: func(metrics inputMetric) error {
			if metrics.EventsProcessedTotal != 12 {
				return fmt.Errorf(
					"%q events_processed_total should be 12, got %d",
					filestreamInputID, 12)
			}
			return nil
		},
	}
	require.Eventuallyf(t, func() bool {
		errMsg.Reset()
		err, inputMetrics, body = findInputMetrics(port, wantInputMetricsCount, 0)
		if err != nil {
			errMsg.WriteString(err.Error())
			return false
		}

		return true
	}, 10*time.Second, 1*time.Second,
		"did not get necessary input metrics: %s", &errMsg)

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

func initInputMetricsTest(t *testing.T, tmplCfg string) (string, *integration.BeatProc, string, string, string) {
	port := randomPort(t)
	celSrv := makeServer()
	t.Cleanup(celSrv.Close)
	httpjsonSrv := makeServer()
	t.Cleanup(httpjsonSrv.Close)

	esMock := newMockESServer(t)

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
		"es_url":              esMock.URL,
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
	return port, filebeat, filestreamInputID, celInputID, httpsjonInputID
}

func saveInputMetricsOnFailure(t *testing.T, filebeat *integration.BeatProc, body []byte) {
	if t.Failed() {
		inputsJSONFile := filepath.Join(filebeat.TempDir(), "inputs.json")
		if err := os.WriteFile(inputsJSONFile, body, 0o644); err != nil {
			t.Logf("failed to save response body to %s: %v",
				inputsJSONFile, err)
		}

		t.Errorf("test failed: input metrics response used for the assertions:\n%s",
			body)
	}
}

func findInputMetrics(
	port string,
	assertInputMetricCount map[string]func(metrics inputMetric) error,
	extraInputsWant int) (error, []inputMetric, []byte) {

	inputMetrics, body, err := fetchInputMetrics(port)
	if err != nil {
		return err, nil, nil
	}

	var extraInputsGot int
	for _, metric := range inputMetrics {
		f, ok := assertInputMetricCount[metric.ID]
		if !ok {
			extraInputsGot++
			continue
		}

		if err = f(metric); err != nil {
			return err, inputMetrics, body
		}
	}

	if extraInputsWant != extraInputsGot {
		return fmt.Errorf("want %d extra inputs, got %d",
			extraInputsWant, extraInputsGot), inputMetrics, body
	}

	return nil, inputMetrics, body
}
func fetchInputMetrics(port string) ([]inputMetric, []byte, error) {
	var inputMetrics []inputMetric

	//nolint:noctx // on a test, it's ok
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/inputs/", port))
	if err != nil {
		return nil, nil, fmt.Errorf("request to /inputs/ failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %v", err)
	}
	err = json.Unmarshal(body, &inputMetrics)
	if err != nil {
		return nil, body, fmt.Errorf("failed unmarshalling response body: %v", err)
	}

	return inputMetrics, body, nil
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
func newMockESServer(t *testing.T) *httptest.Server {
	repliedWith429 := false

	mockESHandler := api.NewDeterministicAPIHandler(
		uuid.Must(uuid.NewV4()),
		"",
		nil,
		time.Now().Add(24*time.Hour),
		0,
		100,
		func(action api.Action, rawEvent []byte) int {
			var meta map[string]any
			if err := json.Unmarshal(action.Meta, &meta); err != nil {
				t.Errorf(
					"newMockESServer: failed to unmarshal action.Meta: %v. Raw meta: %s",
					err, string(action.Meta),
				)
				return http.StatusInternalServerError
			}

			var event esEvent

			err := json.Unmarshal(rawEvent, &event)
			if err != nil {
				t.Errorf(
					"newMockESServer: failed to unmarshal event: %v. Raw event: %s",
					err, string(rawEvent),
				)
				return http.StatusInternalServerError
			}

			var resp int
			if index, ok := meta["_index"].(string); ok && index == "deadletter" {
				// It keeps retrying if dead letter fails, so we always return OK
				resp = http.StatusOK
			} else {
				resp, repliedWith429 = handleEvent(event, repliedWith429)
			}

			return resp
		})
	esMock := httptest.NewServer(mockESHandler)
	t.Cleanup(esMock.Close)
	return esMock
}

func handleEvent(event esEvent, repliedWith429 bool) (int, bool) {
	resp := http.StatusOK
	repWith429 := repliedWith429

	if event.RespondWith == 0 {
		return resp, repWith429
	}

	resp = event.RespondWith
	if event.RespondWith == http.StatusTooManyRequests {
		if repWith429 {
			resp = http.StatusOK
		} else {
			repWith429 = true
		}
	}

	return resp, repWith429
}

func randomPort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "could not create nte.Listener to find a free port")
	defer listener.Close()

	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

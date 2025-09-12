// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/mock-es/pkg/api"
)

func TestFilebeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	numEvents := 1

	var beatsCfgFile = `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %s
output:
  elasticsearch:
    hosts:
      - localhost:9200
    username: admin
    password: testing
    index: %s
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
http.enabled: true
http.host: localhost
http.port: %d
`

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbOtelIndex := "logs-integration-" + namespace
	fbIndex := "logs-filebeat-" + namespace
	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	logFilePath := filepath.Join(filebeatOTel.TempDir(), "log.log")
	filebeatOTel.WriteConfigFile(fmt.Sprintf(beatsCfgFile, logFilePath, fbOtelIndex, 5066))
	writeEventsToLogFile(t, logFilePath, numEvents)
	filebeatOTel.Start()
	defer filebeatOTel.Stop()

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	logFilePath = filepath.Join(filebeat.TempDir(), "log.log")
	writeEventsToLogFile(t, logFilePath, numEvents)
	s := fmt.Sprintf(beatsCfgFile, logFilePath, fbIndex, 5067)

	filebeat.WriteConfigFile(s)
	filebeat.Start()
	defer filebeat.Stop()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	// wait for logs to be published
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+fbOtelIndex+"*")
			require.NoError(t, err)

			filebeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+fbIndex+"*")
			require.NoError(t, err)

			return otelDocs.Hits.Total.Value >= numEvents && filebeatDocs.Hits.Total.Value >= numEvents
		},
		2*time.Minute, 1*time.Second, fmt.Sprintf("Number of hits %d not equal to number of events %d", filebeatDocs.Hits.Total.Value, numEvents))

	var filebeatDoc, otelDoc mapstr.M
	filebeatDoc = filebeatDocs.Hits.Hits[0].Source
	otelDoc = otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.path",
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")

	assert.Equal(t, "filebeatreceiver", otelDoc.Flatten()["agent.otelcol.component.id"], "expected agent.otelcol.component.id field in log record")
	assert.Equal(t, "receiver", otelDoc.Flatten()["agent.otelcol.component.kind"], "expected agent.otelcol.component.kind field in log record")
	assert.NotContains(t, filebeatDoc.Flatten(), "agent.otelcol.component.id", "expected agent.otelcol.component.id field not to be present in filebeat log record")
	assert.NotContains(t, filebeatDoc.Flatten(), "agent.otelcol.component.kind", "expected agent.otelcol.component.kind field not to be present in filebeat log record")
	assertMonitoring(t, 5066)
}

func TestHTTPJSONInputOTel(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	// create a random uuid and make sure it doesn't contain dashes/
	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNameSpace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	type options struct {
		Namespace string
		ESURL     string
		Username  string
		Password  string
	}

	// The request url is a http mock server started using streams
	configFile := `
filebeat.inputs:
  - type: httpjson
    id: httpjson-e2e-otel
    request.url: http://localhost:8090/test

output:
  elasticsearch:
    hosts:
      - {{ .ESURL }}
    username: {{ .Username }}
    password: {{ .Password }}
    index: logs-integration-{{ .Namespace }}

setup.template.enabled: false
queue.mem.flush.timeout: 0s
processors:
   - add_host_metadata: ~
   - add_cloud_metadata: ~
   - add_docker_metadata: ~
   - add_kubernetes_metadata: ~
`

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}

	var configBuffer bytes.Buffer
	optionsValue.Namespace = otelNamespace
	require.NoError(t, template.Must(template.New("config").Parse(configFile)).Execute(&configBuffer, optionsValue))

	filebeatOTel.WriteConfigFile(configBuffer.String())
	filebeatOTel.Start()

	// reset buffer
	configBuffer.Reset()

	optionsValue.Namespace = fbNameSpace
	require.NoError(t, template.Must(template.New("config").Parse(configFile)).Execute(&configBuffer, optionsValue))

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	rawQuery := map[string]any{
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	// wait for logs to be published
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-logs-integration-"+otelNamespace+"*", es)
			assert.NoError(t, err)

			filebeatDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-logs-integration-"+fbNameSpace+"*", es)
			assert.NoError(t, err)

			return otelDocs.Hits.Total.Value >= 1 && filebeatDocs.Hits.Total.Value >= 1
		},
		2*time.Minute, 1*time.Second, fmt.Sprintf("Number of hits %d not equal to number of events %d", filebeatDocs.Hits.Total.Value, 1))

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

func writeEventsToLogFile(t *testing.T, filename string, numEvents int) {
	t.Helper()
	logFile, err := os.Create(filename)
	if err != nil {
		t.Fatalf("could not create file '%s': %s", filename, err)
	}
	// write events to log file
	for i := 0; i < numEvents; i++ {
		msg := fmt.Sprintf("Line %d", i)
		_, err = logFile.Write([]byte(msg + "\n"))
		require.NoErrorf(t, err, "failed to write line %d to temp file", i)
	}

	if err := logFile.Sync(); err != nil {
		t.Fatalf("could not sync log file '%s': %s", filename, err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("could not close log file '%s': %s", filename, err)
	}
}

func assertMonitoring(t *testing.T, port int) {
	address := fmt.Sprintf("http://localhost:%d", port)
	r, err := http.Get(address) //nolint:noctx,bodyclose,gosec // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/stats") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/not-exist") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

func TestFilebeatOTelReceiverE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	wantEvents := 1

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbReceiverIndex := "logs-integration-" + namespace
	filebeatIndex := "logs-filebeat-" + namespace

	otelConfig := struct {
		Index          string
		MonitoringPort int
		InputFile      string
		PathHome       string
	}{
		Index:          fbReceiverIndex,
		MonitoringPort: 5066,
		InputFile:      filepath.Join(filebeatOTel.TempDir(), "log.log"),
		PathHome:       filebeatOTel.TempDir(),
	}

	cfg := `receivers:
  filebeatreceiver/filestream:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-fbreceiver
          enabled: true
          paths:
            - {{.InputFile}}
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    output:
      otelconsumer:
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{.PathHome}}
    http.enabled: true
    http.host: localhost
    http.port: {{.MonitoringPort}}
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: {{.Index}}
    batcher:
      enabled: true
      flush_timeout: 1s
    mapping:
      mode: bodymap
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver/filestream
      exporters:
        - elasticsearch/log
        - debug
`
	var configBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, otelConfig))
	configContents := configBuffer.Bytes()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Config contents:\n%s", configContents)
		}
	})

	filebeatOTel.WriteConfigFile(string(configContents))
	writeEventsToLogFile(t, otelConfig.InputFile, wantEvents)
	filebeatOTel.Start()
	defer filebeatOTel.Stop()

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	beatsCfgFile := `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %s
output:
  elasticsearch:
    hosts:
      - localhost:9200
    username: admin
    password: testing
    index: %s
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
setup.template.name: logs-filebeat-default
setup.template.pattern: logs-filebeat-default
http.enabled: true
http.host: localhost
http.port: %d
`
	logFilePath := filepath.Join(filebeat.TempDir(), "log.log")
	writeEventsToLogFile(t, logFilePath, wantEvents)
	s := fmt.Sprintf(beatsCfgFile, logFilePath, filebeatIndex, 5067)
	filebeat.WriteConfigFile(s)
	filebeat.Start()
	defer filebeat.Stop()

	es := integration.GetESClient(t, "http")

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	// wait for logs to be published
	require.Eventuallyf(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+fbReceiverIndex+"*")
			require.NoError(t, err)

			filebeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+filebeatIndex+"*")
			require.NoError(t, err)

			return otelDocs.Hits.Total.Value >= wantEvents && filebeatDocs.Hits.Total.Value >= wantEvents
		},
		2*time.Minute, 1*time.Second, "expected at least %d events, got filebeat: %d and otel: %d", wantEvents, filebeatDocs.Hits.Total.Value, otelDocs.Hits.Total.Value)

	var filebeatDoc, otelDoc mapstr.M
	filebeatDoc = filebeatDocs.Hits.Hits[0].Source
	otelDoc = otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.path",
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
	assert.Equal(t, "filebeatreceiver/filestream", otelDoc.Flatten()["agent.otelcol.component.id"], "expected agent.otelcol.component.id field in log record")
	assert.Equal(t, "receiver", otelDoc.Flatten()["agent.otelcol.component.kind"], "expected agent.otelcol.component.kind field in log record")
	assert.NotContains(t, filebeatDoc.Flatten(), "agent.otelcol.component.id", "expected agent.otelcol.component.id field not to be present in filebeat log record")
	assert.NotContains(t, filebeatDoc.Flatten(), "agent.otelcol.component.kind", "expected agent.otelcol.component.kind field not to be present in filebeat log record")
	assertMonitoring(t, otelConfig.MonitoringPort)
	assertMonitoring(t, 5067) // filebeat
}

func TestFilebeatOTelMultipleReceiversE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	wantEvents := 100

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	// write events to log file
	logFilePath := filepath.Join(filebeatOTel.TempDir(), "log.log")
	writeEventsToLogFile(t, logFilePath, wantEvents)

	type receiverConfig struct {
		MonitoringPort int
		InputFile      string
		PathHome       string
	}

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	otelConfig := struct {
		Index     string
		Receivers []receiverConfig
	}{
		Index: "logs-integration-" + namespace,
		Receivers: []receiverConfig{
			{
				MonitoringPort: 5066,
				InputFile:      logFilePath,
				PathHome:       filepath.Join(filebeatOTel.TempDir(), "r1"),
			},
			{
				MonitoringPort: 5067,
				InputFile:      logFilePath,
				PathHome:       filepath.Join(filebeatOTel.TempDir(), "r2"),
			},
		},
	}

	cfg := `receivers:
{{range $i, $receiver := .Receivers}}
  filebeatreceiver/{{$i}}:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-fbreceiver
          enabled: true
          paths:
            - {{$receiver.InputFile}}
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    output:
      otelconsumer:
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{$receiver.PathHome}}
{{if $receiver.MonitoringPort}}
    http.enabled: true
    http.host: localhost
    http.port: {{$receiver.MonitoringPort}}
{{end}}
{{end}}
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: {{.Index}}
    batcher:
      enabled: true
      flush_timeout: 1s
    mapping:
      mode: bodymap
service:
  pipelines:
    logs:
      receivers:
{{range $i, $receiver := .Receivers}}
        - filebeatreceiver/{{$i}}
{{end}}
      exporters:
        - debug
        - elasticsearch/log
`
	var configBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, otelConfig))
	configContents := configBuffer.Bytes()

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Config contents:\n%s", configContents)
		}
	})

	filebeatOTel.WriteConfigFile(string(configContents))
	writeEventsToLogFile(t, logFilePath, wantEvents)
	filebeatOTel.Start()
	defer filebeatOTel.Stop()

	es := integration.GetESClient(t, "http")

	var otelDocs estools.Documents
	var err error

	// wait for logs to be published
	wantTotalLogs := wantEvents * len(otelConfig.Receivers)
	require.Eventuallyf(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+otelConfig.Index+"*")
			require.NoError(t, err)

			return otelDocs.Hits.Total.Value >= wantTotalLogs
		},
		2*time.Minute, 100*time.Millisecond, "expected at least %d events, got %d", wantTotalLogs, otelDocs.Hits.Total.Value)
	for _, rec := range otelConfig.Receivers {
		assertMonitoring(t, rec.MonitoringPort)
	}
}
<<<<<<< HEAD
=======

func TestFilebeatOTelInspect(t *testing.T) {
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	var beatsCfgFile = `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - /tmp/log.log
output:
  elasticsearch:
    hosts:
      - localhost:9200
    username: admin
    password: testing
    index: index
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`
	expectedExporter := `exporters:
    elasticsearch:
        compression: gzip
        compression_params:
            level: 1
        endpoints:
            - http://localhost:9200
        idle_conn_timeout: 3s
        logs_index: index
        mapping:
            mode: bodymap
        max_conns_per_host: 1
        password: testing
        retry:
            enabled: true
            initial_interval: 1s
            max_interval: 1m0s
            max_retries: 3
        sending_queue:
            batch:
                max_size: 1600
                min_size: 0
                sizer: items
            block_on_overflow: true
            enabled: true
            num_consumers: 1
            queue_size: 3200
            wait_for_result: true
        timeout: 1m30s
        user: admin`
	expectedReceiver := `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - enabled: true
                  file_identity:
                    native: null
                  id: filestream-input-id
                  paths:
                    - /tmp/log.log
                  prospector:
                    scanner:
                        fingerprint:
                            enabled: false
                  type: filestream`
	expectedService := `service:
    pipelines:
        logs:
            exporters:
                - elasticsearch
            receivers:
                - filebeatreceiver
`
	filebeatOTel.WriteConfigFile(beatsCfgFile)

	filebeatOTel.Start("inspect")
	defer filebeatOTel.Stop()

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		out, err := filebeatOTel.ReadStdout()
		require.NoError(collect, err)
		require.Contains(collect, out, expectedExporter)
		require.Contains(collect, out, expectedReceiver)
		require.Contains(collect, out, expectedService)
	}, 10*time.Second, 500*time.Millisecond, "failed to get output of inspect command")
}

func TestFilebeatOTelDocumentLevelRetries(t *testing.T) {
	tests := []struct {
		name                     string
		maxRetries               int
		failuresPerEvent         int
		bulkErrorCode            string
		eventIDsToFail           []int
		expectedIngestedEventIDs []int
	}{
		{
			name:                     "bulk 429 with retries",
			maxRetries:               3,
			failuresPerEvent:         2,     // Fail 2 times, succeed on 3rd attempt
			bulkErrorCode:            "429", // retryable error
			eventIDsToFail:           []int{1, 3, 5, 7},
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events should eventually be ingested
		},
		{
			name:                     "bulk exhausts retries",
			maxRetries:               3,
			failuresPerEvent:         5, // Fail more than max_retries
			bulkErrorCode:            "429",
			eventIDsToFail:           []int{2, 4, 6, 8},
			expectedIngestedEventIDs: []int{0, 1, 3, 5, 7, 9}, // Only non-failing events should be ingested
		},
		{
			name:                     "bulk with permanent mapping errors",
			maxRetries:               3,
			failuresPerEvent:         0,                          // always fail
			bulkErrorCode:            "400",                      // never retried
			eventIDsToFail:           []int{1, 4, 8},             // Only specific events fail
			expectedIngestedEventIDs: []int{0, 2, 3, 5, 6, 7, 9}, // Only non-failing events should be ingested
		},
	}

	const numTestEvents = 10
	reEventLine := regexp.MustCompile(`"message":"Line (\d+)"`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ingestedTestEvents []string
			var mu sync.Mutex
			eventFailureCounts := make(map[string]int)

			deterministicHandler := func(action api.Action, event []byte) int {
				// Handle non-bulk requests
				if action.Action != "create" {
					return http.StatusOK
				}

				// Extract event ID from the event data
				if matches := reEventLine.FindSubmatch(event); len(matches) > 1 {
					eventIDStr := string(matches[1])
					eventID, err := strconv.Atoi(eventIDStr)
					if err != nil {
						return http.StatusInternalServerError
					}

					eventKey := "Line " + eventIDStr

					mu.Lock()
					defer mu.Unlock()

					isFailingEvent := slices.Contains(tt.eventIDsToFail, eventID)

					var shouldFail bool
					if isFailingEvent {
						// This event is configured to fail
						failureCount := eventFailureCounts[eventKey]

						switch tt.bulkErrorCode {
						case "400":
							// Permanent errors always fail
							shouldFail = true
						case "429":
							// Temporary errors fail until failuresPerEvent threshold
							shouldFail = failureCount < tt.failuresPerEvent
						}
					} else {
						// Events not in the fail list always succeed
						shouldFail = false
					}

					if shouldFail {
						eventFailureCounts[eventKey] = eventFailureCounts[eventKey] + 1
						if tt.bulkErrorCode == "429" {
							return http.StatusTooManyRequests
						} else {
							return http.StatusBadRequest
						}
					}

					// track ingested event
					found := false
					for _, existing := range ingestedTestEvents {
						if existing == eventKey {
							found = true
							break
						}
					}
					if !found {
						ingestedTestEvents = append(ingestedTestEvents, eventKey)
					}
					return http.StatusOK
				}

				return http.StatusOK
			}

			reader := metric.NewManualReader()
			provider := metric.NewMeterProvider(metric.WithReader(reader))

			mux := http.NewServeMux()
			mux.Handle("/", api.NewDeterministicAPIHandler(
				uuid.Must(uuid.NewV4()),
				"",
				provider,
				time.Now().Add(24*time.Hour),
				0,
				0,
				deterministicHandler,
			))

			server := httptest.NewServer(mux)
			defer server.Close()

			filebeatOTel := integration.NewBeat(
				t,
				"filebeat-otel",
				"../../filebeat.test",
				"otel",
			)

			namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
			index := "logs-integration-" + namespace

			beatsConfig := struct {
				Index          string
				InputFile      string
				ESEndpoint     string
				MaxRetries     int
				MonitoringPort int
			}{
				Index:          index,
				InputFile:      filepath.Join(filebeatOTel.TempDir(), "log.log"),
				ESEndpoint:     server.URL,
				MaxRetries:     tt.maxRetries,
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
			}

			cfg := `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - {{.InputFile}}
output:
  elasticsearch:
    hosts:
      - {{.ESEndpoint}}
    username: admin
    password: testing
    index: {{.Index}}
    compression_level: 0
    max_retries: {{.MaxRetries}}
logging.level: debug
queue.mem.flush.timeout: 0s
setup.template.enabled: false
http.enabled: true
http.host: localhost
http.port: {{.MonitoringPort}}
`
			var configBuffer bytes.Buffer
			require.NoError(t,
				template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, beatsConfig))

			filebeatOTel.WriteConfigFile(configBuffer.String())
			writeEventsToLogFile(t, beatsConfig.InputFile, numTestEvents)
			filebeatOTel.Start()
			defer filebeatOTel.Stop()

			// Wait for file input to be fully read
			filebeatOTel.WaitStdErrContains(fmt.Sprintf("End of file reached: %s; Backoff now.", beatsConfig.InputFile), 30*time.Second)

			// Wait for expected events to be ingested
			require.EventuallyWithT(t, func(ct *assert.CollectT) {
				mu.Lock()
				defer mu.Unlock()

				// collect mock-es metrics
				rm := metricdata.ResourceMetrics{}
				err := reader.Collect(context.Background(), &rm)
				assert.NoError(ct, err, "failed to collect metrics from mock-es")
				metrics := make(map[string]int64)
				for _, sm := range rm.ScopeMetrics {
					for _, m := range sm.Metrics {
						if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
							var total int64
							for _, dp := range sum.DataPoints {
								total += dp.Value
							}
							metrics[m.Name] = total
						}
					}
				}
				assert.Equal(ct, int64(len(tt.expectedIngestedEventIDs)), metrics["bulk.create.ok"], "expected bulk.create.ok metric to match ingested events")

				// If we have the right count, validate the specific events
				// Verify we have the correct events ingested
				for _, expectedID := range tt.expectedIngestedEventIDs {
					expectedEventKey := fmt.Sprintf("Line %d", expectedID)
					found := false
					for _, ingested := range ingestedTestEvents {
						if ingested == expectedEventKey {
							found = true
							break
						}
					}
					assert.True(ct, found, "expected _bulk event %s to be ingested", expectedEventKey)
				}

				// Verify we have valid line content for all ingested events
				for _, ingested := range ingestedTestEvents {
					assert.Regexp(ct, `^Line \d+$`, ingested, "unexpected ingested event format: %s", ingested)
				}
			}, 30*time.Second, 1*time.Second, "timed out waiting for expected event processing")

			// Confirm filebeat agreed with our accounting of ingested events
			require.EventuallyWithT(t, func(ct *assert.CollectT) {
				address := fmt.Sprintf("http://localhost:%d", beatsConfig.MonitoringPort)
				r, err := http.Get(address + "/stats") //nolint:noctx,bodyclose // fine for tests
				assert.NoError(ct, err)
				assert.Equal(ct, http.StatusOK, r.StatusCode, "incorrect status code")
				var m mapstr.M
				err = json.NewDecoder(r.Body).Decode(&m)
				assert.NoError(ct, err)

				m = m.Flatten()

				// Currently, otelconsumer either ACKs or fails the entire batch and has no visibility into individual event failures within the exporter.
				// From otelconsumer's perspective, the whole batch is considered successful as long as ConsumeLogs returns no error.
				assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.total"], "expected total events sent to output to match")
				assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.acked"], "expected total events acked to match")
				assert.Equal(ct, float64(0), m["libbeat.output.events.dropped"], "expected total events dropped to match")
			}, 10*time.Second, 100*time.Millisecond, "expected output stats to be available in monitoring endpoint")
		})
	}
}
>>>>>>> 29488865a (otel: add test for otelconsumer + elasticsearchexporter document-level retries (#46413))

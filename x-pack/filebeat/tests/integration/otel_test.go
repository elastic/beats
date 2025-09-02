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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
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

	otelMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))
	filebeatMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	logFilePath := filepath.Join(filebeatOTel.TempDir(), "log.log")
	filebeatOTel.WriteConfigFile(fmt.Sprintf(beatsCfgFile, logFilePath, fbOtelIndex, otelMonitoringPort))
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
	s := fmt.Sprintf(beatsCfgFile, logFilePath, fbIndex, filebeatMonitoringPort)

	filebeat.WriteConfigFile(s)
	filebeat.Start()
	defer filebeat.Stop()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	// wait for logs to be published
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+fbOtelIndex+"*")
			assert.NoError(ct, err)

			filebeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+fbIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, numEvents, "expected at least %d otel events, got %d", numEvents, otelDocs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, filebeatDocs.Hits.Total.Value, numEvents, "expected at least %d filebeat events, got %d", numEvents, filebeatDocs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected at least %d events for both filebeat and otel", numEvents)

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.path",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
	assertMonitoring(t, otelMonitoringPort)
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
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-logs-integration-"+otelNamespace+"*", es)
			assert.NoError(ct, err)

			filebeatDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-logs-integration-"+fbNameSpace+"*", es)
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 otel event, got %d", otelDocs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, filebeatDocs.Hits.Total.Value, 1, "expected at least 1 filebeat event, got %d", filebeatDocs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected at least 1 event for both filebeat and otel")

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
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

	otelMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))
	filebeatMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	otelConfig := struct {
		Index          string
		MonitoringPort int
		InputFile      string
		PathHome       string
	}{
		Index:          fbReceiverIndex,
		MonitoringPort: otelMonitoringPort,
		InputFile:      filepath.Join(filebeatOTel.TempDir(), "log.log"),
		PathHome:       filebeatOTel.TempDir(),
	}

	cfg := `receivers:
  filebeatreceiver:
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
        - filebeatreceiver
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
	s := fmt.Sprintf(beatsCfgFile, logFilePath, filebeatIndex, filebeatMonitoringPort)
	filebeat.WriteConfigFile(s)
	filebeat.Start()
	defer filebeat.Stop()

	es := integration.GetESClient(t, "http")

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	// wait for logs to be published
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+fbReceiverIndex+"*")
			assert.NoError(ct, err)

			filebeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+filebeatIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, wantEvents, "expected at least %d otel events, got %d", wantEvents, otelDocs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, filebeatDocs.Hits.Total.Value, wantEvents, "expected at least %d filebeat events, got %d", wantEvents, filebeatDocs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected at least %d events for both filebeat and otel", wantEvents)

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.path",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
	assertMonitoring(t, otelConfig.MonitoringPort)
	assertMonitoring(t, filebeatMonitoringPort) // filebeat
}

func TestFilebeatOTelMultipleReceiversE2E(t *testing.T) {
	t.Skip("Flaky test: https://github.com/elastic/beats/issues/45631")
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
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
				InputFile:      logFilePath,
				PathHome:       filepath.Join(filebeatOTel.TempDir(), "r1"),
			},
			{
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
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
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+otelConfig.Index+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, wantTotalLogs, "expected at least %d events, got %d", wantTotalLogs, otelDocs.Hits.Total.Value)
		},
		2*time.Minute, 100*time.Millisecond, "expected at least %d events from multiple receivers", wantTotalLogs)
	for _, rec := range otelConfig.Receivers {
		assertMonitoring(t, rec.MonitoringPort)
	}
}

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
        batcher:
            enabled: true
            max_size: 1600
            min_size: 0
        compression: gzip
        compression_params:
            level: 1
        endpoints:
            - http://localhost:9200
        idle_conn_timeout: 3s
        logs_index: index
        mapping:
            mode: bodymap
        password: testing
        retry:
            enabled: true
            initial_interval: 1s
            max_interval: 1m0s
            max_retries: 3
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
		require.NoError(t, err)
		require.Contains(t, out, expectedExporter)
		require.Contains(t, out, expectedReceiver)
		require.Contains(t, out, expectedService)
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
			failuresPerEvent:         0, // Always fail (permanent failure)
			bulkErrorCode:            "400",
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

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Elastic-Product", "Elasticsearch")

				if r.URL.Path != "/_bulk" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{}`))
					return
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				bodyStr := string(body)

				mu.Lock()
				defer mu.Unlock()

				shouldEventFail := func(eventID int) bool {
					for _, failID := range tt.eventIDsToFail {
						if failID == eventID {
							return true
						}
					}
					return false
				}

				var items []string
				for line := range strings.Lines(bodyStr) {
					if strings.Contains(line, `"create":{`) {
						// Ignore metadata lines
						continue
					}
					if matches := reEventLine.FindStringSubmatch(line); len(matches) > 1 {
						eventIDStr := matches[1]
						eventID := 0
						fmt.Sscanf(eventIDStr, "%d", &eventID)
						eventKey := "Line " + eventIDStr

						// Check if this event should fail
						isFailingEvent := shouldEventFail(eventID)

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
							var errorResponse string
							if tt.bulkErrorCode == "429" {
								errorResponse = `{"create":{"_index":"logs","status":429,"error":{"type":"too_many_requests","reason":"queue capacity exceeded"}}}`
							} else {
								errorResponse = `{"create":{"_index":"logs","status":400,"error":{"type":"mapper_parsing_exception","reason":"failed to parse field"}}}`
							}
							items = append(items, errorResponse)
						} else {
							// Success - track ingested event
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
							items = append(items, `{"create":{"_index":"logs","status":201}}`)
						}
					}
				}

				response := fmt.Sprintf(`{"items":[%s]}`, strings.Join(items, ","))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
			}))
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

				actualCount := len(ingestedTestEvents)
				expectedCount := len(tt.expectedIngestedEventIDs)

				assert.Equal(ct, expectedCount, actualCount, "expected _bulk events count to match")

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

				// TODO: Beats stats are not tracking exporter metrics properly in otelconsumer, so it assumes all events were delivered since the batch was acked.
				// There could have been failures within the batch that were retried and then dropped, only way to know for sure is to check the exporter metrics.
				// require.Equal(t, float64(numTestEvents), m["libbeat.output.events.total"], "expected total events sent to output to match")
				// require.Equal(t, float64(len(tt.expectedIngestedEventIDs)), m["libbeat.output.events.acked"], "expected events acked to match ingested count")
				// require.Equal(t, float64(numTestEvents - len(tt.expectedIngestedEventIDs)), m["libbeat.output.events.dropped"], "expected events dropped to match ingested count")
				assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.total"], "expected total events sent to output to match")
				assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.acked"], "expected total events acked to match")
			}, 10*time.Second, 100*time.Millisecond, "expected output stats to be available in monitoring endpoint")
		})
	}
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/gofrs/uuid/v5"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/mock-es/pkg/api"
)

func TestFilebeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	numEvents := 1

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbOtelIndex := "logs-integration-" + namespace
	fbIndex := "logs-filebeat-" + namespace

	otelMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))
	filebeatMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	otelCfgFile := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - %s
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
    http.enabled: true
    http.host: localhost
    http.port: %d
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
    logs_index: %s
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - elasticsearch/log
        - debug
`
	logFilePath := filepath.Join(tmpdir, "log.log")
	writeEventsToLogFile(t, logFilePath, numEvents)
	oteltestcol.New(t, fmt.Sprintf(otelCfgFile, logFilePath, tmpdir, otelMonitoringPort, fbOtelIndex))

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
http.enabled: true
http.host: localhost
http.port: %d
`

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
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
		"log.file.device_id", // changes value between filebeat and otel receiver
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")

	assert.Equal(t, "filebeat", otelDoc.Flatten()["agent.type"], "expected agent.type field to be 'filebeat' in otel docs")
	assert.Equal(t, "filebeat", filebeatDoc.Flatten()["agent.type"], "expected agent.type field to be 'filebeat' in filebeat docs")
	assertMonitoring(t, otelMonitoringPort)
}

func TestFilebeatOTelHTTPJSONInput(t *testing.T) {
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

	otelConfigFile := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: httpjson
          id: httpjson-e2e-otel
          request.url: http://localhost:8090/test
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
exporters:
  elasticsearch:
    auth:
      authenticator: beatsauth
    compression: gzip
    compression_params:
      level: 1
    endpoints:
      - {{ .ESURL }}
    logs_index: logs-integration-{{ .Namespace }}
    max_conns_per_host: 1
    password: {{ .Password }}
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1600
        min_size: 0
        sizer: items
      block_on_overflow: true
      enabled: true
      num_consumers: 1
      queue_size: 3200
      wait_for_result: true
    user: {{ .Username }}
extensions:
  beatsauth:
    idle_connection_timeout: 3s
    proxy_disable: false
    timeout: 1m30s
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - elasticsearch
`

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}

	var configBuffer bytes.Buffer
	optionsValue.Namespace = otelNamespace
	require.NoError(t, template.Must(template.New("config").Parse(otelConfigFile)).Execute(&configBuffer, optionsValue))
	oteltestcol.New(t, configBuffer.String())

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

func writeEventsToFile(t *testing.T, file *os.File, startLine, numEvents int) {
	t.Helper()
	for i := startLine; i < startLine+numEvents; i++ {
		msg := fmt.Sprintf("Line %d", i)
		_, err := file.Write([]byte(msg + "\n"))
		require.NoErrorf(t, err, "failed to write line %d to temp file", i)
	}

	if err := file.Sync(); err != nil {
		t.Fatalf("could not sync log file: %s", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("could not close log file: %s", err)
	}
}

func writeEventsToLogFile(t *testing.T, filename string, numEvents int) {
	t.Helper()
	logFile, err := os.Create(filename)
	if err != nil {
		t.Fatalf("could not create file '%s': %s", filename, err)
	}
	writeEventsToFile(t, logFile, 0, numEvents)
}

func appendEventsToLogFile(t *testing.T, filename string, startLine int, numEvents int) {
	t.Helper()
	logFile, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	writeEventsToFile(t, logFile, startLine, numEvents)
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

	tmpdir := t.TempDir()
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
		InputFile:      filepath.Join(tmpdir, "log.log"),
		PathHome:       tmpdir,
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
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{.PathHome}}
    http.enabled: true
    http.host: localhost
    http.port: {{.MonitoringPort}}
    management.otel.enabled: true
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
    sending_queue:
      enabled: true
      batch:
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

	writeEventsToLogFile(t, otelConfig.InputFile, wantEvents)
	oteltestcol.New(t, configBuffer.String())

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
		"log.file.device_id", // changes value between filebeat and otel receiver
		"container.id",       // only present in filebeat
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
	assert.Equal(t, "filebeat", otelDoc.Flatten()["agent.type"], "expected agent.type field to be 'filebeat' in otel docs")
	assert.Equal(t, "filebeat", filebeatDoc.Flatten()["agent.type"], "expected agent.type field to be 'filebeat' in filebeat docs")
	assertMonitoring(t, otelConfig.MonitoringPort)
	assertMonitoring(t, filebeatMonitoringPort) // filebeat
}

func TestFilebeatOTelMultipleReceiversE2E(t *testing.T) {
	t.Skip("Flaky test: https://github.com/elastic/beats/issues/45631")
	integration.EnsureESIsRunning(t)
	wantEvents := 100

	tmpdir := t.TempDir()
	// write events to log file
	logFilePath := filepath.Join(tmpdir, "log.log")
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
				PathHome:       filepath.Join(tmpdir, "r1"),
			},
			{
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
				InputFile:      logFilePath,
				PathHome:       filepath.Join(tmpdir, "r2"),
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
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
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
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
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

	writeEventsToLogFile(t, logFilePath, wantEvents)

	oteltestcol.New(t, configBuffer.String())

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

func TestFilebeatOTelDocumentLevelRetries(t *testing.T) {
	tests := []struct {
		name                     string
		maxRetries               int
		failuresPerEvent         int
		requestStatusCode        string
		bulkDocStatusCode        string
		retryOnStatus            string
		eventIDsToFail           []int
		expectedIngestedEventIDs []int
		requestLevelFailure      bool
	}{
		{
			name:                     "bulk 200 succeed without retries",
			maxRetries:               0,
			bulkDocStatusCode:        "200",                               // 200 OK for all documents
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events ingested
		},
		{
			name:                     "bulk 429 retries until success",
			maxRetries:               3,
			failuresPerEvent:         2,                                   // Each failing event fails 2 times, succeeds on 3rd attempt
			bulkDocStatusCode:        "429",                               // Document-level 429 errors in bulk response
			eventIDsToFail:           []int{1, 3, 5, 7},                   // These specific events will fail initially
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events eventually ingested after retries
		},
		{
			name:                     "bulk 503 retry_on_status retries until success",
			maxRetries:               3,
			failuresPerEvent:         2,                                   // Each failing event fails 2 times, succeeds on 3rd attempt
			bulkDocStatusCode:        "503",                               // Document-level 503 errors in bulk response
			retryOnStatus:            "503",                               // retry 503 errors
			eventIDsToFail:           []int{1, 3, 5, 7},                   // These specific events will fail initially
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events eventually ingested after retries
		},
		{
			name:                     "bulk 503 retry_on_status exhausts retries",
			maxRetries:               2,
			failuresPerEvent:         3,                             // Each failing event fails 3 times (total attempts = 1 initial + 2 retries = 3)
			bulkDocStatusCode:        "503",                         // Document-level 503 errors in bulk response
			retryOnStatus:            "503",                         // Explicitly enable 503 retries
			eventIDsToFail:           []int{0, 9},                   // First and last events will permanently fail after exhausting retries
			expectedIngestedEventIDs: []int{1, 2, 3, 4, 5, 6, 7, 8}, // Only non-failing events ingested
		},
		{
			name:                     "bulk 429 exhausts retries",
			maxRetries:               2,
			failuresPerEvent:         3,                       // Each failing event fails 3 times
			bulkDocStatusCode:        "429",                   // Document-level 429 errors in bulk response
			eventIDsToFail:           []int{2, 4, 6, 8},       // These events will permanently fail after exhausting retries
			expectedIngestedEventIDs: []int{0, 1, 3, 5, 7, 9}, // Only non-failing events ingested
		},
		{
			name:                     "bulk 400 permanent failure",
			maxRetries:               3,
			failuresPerEvent:         0,                          // Always fail (permanent error)
			bulkDocStatusCode:        "400",                      // Document-level 400 errors in bulk response
			eventIDsToFail:           []int{1, 4, 8},             // These events have permanent mapping errors
			expectedIngestedEventIDs: []int{0, 2, 3, 5, 6, 7, 9}, // Only non-failing events ingested (no retries for 400)
		},
		{
			name:                     "request 429 retries until success",
			maxRetries:               3,
			failuresPerEvent:         2,                                   // Request fails 2 times, succeeds on 3rd attempt
			requestStatusCode:        "429",                               // Entire HTTP request fails with 429
			bulkDocStatusCode:        "200",                               // Documents succeed when forwarded to handler
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events eventually ingested after request retries
			requestLevelFailure:      true,                                // Request-level failures
		},
		{
			name:                     "request 503 retry_on_status retries until success",
			maxRetries:               3,
			failuresPerEvent:         3,                                   // Request fails 2 times, succeeds on 3rd attempt
			requestLevelFailure:      true,                                // Request-level failures
			requestStatusCode:        "503",                               // Entire HTTP request fails with 503
			bulkDocStatusCode:        "200",                               // Documents succeed when forwarded to handler
			retryOnStatus:            "503",                               // Explicitly enable 503 retries
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events eventually ingested after request retries
		},
		{
			name:                     "request 429 exhausts retries",
			maxRetries:               2,
			failuresPerEvent:         3,       // Request fails 3 times
			requestLevelFailure:      true,    // Request-level failures
			requestStatusCode:        "429",   // Entire HTTP request fails with 429
			expectedIngestedEventIDs: []int{}, // No events ingested (exhausted all attempts without success)
		},
		{
			name:                     "request 503 retry_on_status exhausts retries",
			maxRetries:               2,
			failuresPerEvent:         3,       // Request fails 3 times
			requestLevelFailure:      true,    // Request-level failures
			requestStatusCode:        "503",   // Entire HTTP request fails with 503
			retryOnStatus:            "503",   // Explicitly enable 503 retries
			expectedIngestedEventIDs: []int{}, // No events ingested (exhausted all attempts without success)
		},
		{
			name:                     "request 400 permanent failure",
			maxRetries:               0,
			failuresPerEvent:         1,       // fail request once
			requestLevelFailure:      true,    // Request-level failures
			requestStatusCode:        "400",   // Entire HTTP request fails with 400
			expectedIngestedEventIDs: []int{}, // No events ingested
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
						return http.StatusBadRequest
					}

					eventKey := "Line " + eventIDStr

					mu.Lock()
					defer mu.Unlock()

					isFailingEvent := slices.Contains(tt.eventIDsToFail, eventID)

					var shouldFail bool
					if isFailingEvent {
						// This event is configured to fail
						failureCount := eventFailureCounts[eventKey]

						switch tt.bulkDocStatusCode {
						case "400":
							// Permanent errors always fail
							shouldFail = true
						case "429":
							fallthrough
						case "503":
							// Temporary errors fail until failuresPerEvent threshold
							shouldFail = failureCount < tt.failuresPerEvent
						}
					}

					if shouldFail {
						eventFailureCounts[eventKey] = eventFailureCounts[eventKey] + 1
						switch tt.bulkDocStatusCode {
						case "503":
							return http.StatusServiceUnavailable
						case "429":
							return http.StatusTooManyRequests
						default:
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

				return http.StatusBadRequest
			}

			reader := metric.NewManualReader()
			provider := metric.NewMeterProvider(metric.WithReader(reader))

			mux := http.NewServeMux()

			// Create the base deterministic handler
			baseHandler := api.NewDeterministicAPIHandler(
				uuid.Must(uuid.NewV4()),
				"",
				provider,
				time.Now().Add(24*time.Hour),
				0,
				0,
				deterministicHandler,
			)

			// If requestLevelFailure is true, wrap with request-level failure logic
			if tt.requestLevelFailure {
				// Request-level failures: entire HTTP request fails with the specified status code
				var attemptCount int64
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					currentAttempt := atomic.AddInt64(&attemptCount, 1)

					// For retryable status codes (429, 503), fail for failuresPerEvent times, then forward to deterministic handler
					// For non-retryable status codes (400), always fail
					var shouldFail bool
					switch tt.requestStatusCode {
					case "400":
						// 400 is never retryable, always fail
						shouldFail = true
					case "503", "429":
						shouldFail = currentAttempt <= int64(tt.failuresPerEvent)
					}

					if shouldFail {
						status, err := strconv.Atoi(tt.requestStatusCode)
						assert.NoError(t, err)
						http.Error(w, "", status)
						return
					}

					// Success case - forward to the deterministic handler
					baseHandler.ServeHTTP(w, r)
				})
			} else {
				mux.Handle("/", baseHandler)
			}

			server := httptest.NewServer(mux)
			defer server.Close()

			namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
			index := "logs-integration-" + namespace

			beatsConfig := struct {
				Index          string
				InputFile      string
				ESEndpoint     string
				MaxRetries     int
				MonitoringPort int
				RetryOnStatus  string
			}{
				Index:          index,
				InputFile:      filepath.Join(t.TempDir(), "log.log"),
				ESEndpoint:     server.URL,
				MaxRetries:     tt.maxRetries,
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
				RetryOnStatus:  tt.retryOnStatus,
			}

			cfg := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - {{.InputFile}}
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    logging:
      level: debug
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    http.enabled: true
    http.host: localhost
    http.port: {{.MonitoringPort}}
exporters:
  elasticsearch:
    auth:
      authenticator: beatsauth
    compression: none
    endpoints:
      - {{.ESEndpoint}}
    logs_index: {{.Index}}
    max_conns_per_host: 1
    password: testing
    retry:
      enabled: true
      initial_interval: 500ms
      max_interval: 30s
      max_retries: {{.MaxRetries}}
{{if .RetryOnStatus}}
      retry_on_status: [{{.RetryOnStatus}}]
{{end}}
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1
        min_size: 0
        sizer: items
      block_on_overflow: true
      enabled: true
      num_consumers: 1
      queue_size: 3200
      wait_for_result: true
    user: admin
extensions:
  beatsauth:
    idle_connection_timeout: 3s
    proxy_disable: false
    timeout: 1m30s
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - elasticsearch
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
`
			var configBuffer bytes.Buffer
			require.NoError(t,
				template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, beatsConfig))

			collector := oteltestcol.New(t, configBuffer.String())
			writeEventsToLogFile(t, beatsConfig.InputFile, numTestEvents)

			// Wait for file input to be fully read
			require.Eventually(t, func() bool {
				return collector.ObservedLogs().FilterMessageSnippet(fmt.Sprintf("End of file reached: %s; Backoff now.", beatsConfig.InputFile)).Len() == 1
			}, 30*time.Second, 100*time.Millisecond, "timed out waiting for file input to be fully read")

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

				assert.Equal(ct, float64(numTestEvents), m["libbeat.pipeline.events.published"], "expected total events published to pipeline to match")

				// For non-retryable errors like 400, events are dropped by the exporter
				if tt.requestLevelFailure && tt.requestStatusCode == "400" {
					assert.Equal(ct, float64(0), m["libbeat.output.events.acked"], "expected no events to be acked (400 errors drop events)")
				} else {
					// For retryable errors or successful cases, events are eventually acked
					// Currently, otelconsumer either ACKs or fails the entire batch and has no visibility into individual event failures within the exporter.
					// From otelconsumer's perspective, the whole batch is considered successful as long as ConsumeLogs returns no error.
					// events.total can be larger than the acknowledged event count because it includes retrys.
					assert.GreaterOrEqual(ct, m["libbeat.output.events.total"], float64(numTestEvents), "expected total events sent to output include all events")
					assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.acked"], "expected total events acked to match")
					assert.Equal(ct, float64(0), m["libbeat.output.events.dropped"], "expected total events dropped to match")
				}
			}, 10*time.Second, 100*time.Millisecond, "expected output stats to be available in monitoring endpoint")
		})
	}
}

func TestFileBeatKerberos(t *testing.T) {
	wantEvents := 1
	krbURL := "http://localhost:9203" // this is kerberos client - we've hardcoded the URL here
	tempFile := t.TempDir()
	// ES client
	esCfg := elasticsearch.Config{
		Addresses: []string{krbURL},
		Username:  "admin",
		Password:  "testing",
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // this is only for testing
			},
		},
	}

	es, err := elasticsearch.NewClient(esCfg)
	require.NoError(t, err, "could not get elasticsearch client")

	setupRoleMapping(t, es)

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	filebeatIndex := "logs-filebeat.kerberos-" + namespace

	otelConfig := struct {
		Index     string
		InputFile string
		PathHome  string
		Endpoint  string
	}{
		Index:     filebeatIndex,
		InputFile: filepath.Join(tempFile, "log.log"),
		PathHome:  tempFile,
		Endpoint:  krbURL,
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
    queue.mem.flush.timeout: 0s
    management.otel.enabled: true
    path.home: {{.PathHome}}	
extensions:
  beatsauth:
   kerberos: 
     auth_type: "password"
     config_path: "../../../../libbeat/outputs/elasticsearch/testdata/krb5.conf"
     username: "beats"
     password: "testing"
     realm: "elastic"
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - {{.Endpoint}}
    logs_index: {{.Index}}
    auth:
     authenticator: beatsauth
service:
  extensions: 
  - beatsauth
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

	writeEventsToLogFile(t, otelConfig.InputFile, wantEvents)
	oteltestcol.New(t, string(configContents))

	// wait for logs to be published
	require.EventuallyWithT(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+filebeatIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, wantEvents, "expected at least %d events, got %d", wantEvents, otelDocs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second)
}

func TestFilebeatOTelBeatProcessorE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	wantEvents := 1

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	processorIndex := "logs-processor-" + namespace
	receiverIndex := "logs-receiver-" + namespace

	configParameters := struct {
		Index     string
		InputFile string
		PathHome  string
	}{
		Index:     processorIndex,
		InputFile: filepath.Join(t.TempDir(), "log.log"),
		PathHome:  t.TempDir(),
	}

	configTemplate := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      processors:
        - beat
      exporters:
        - elasticsearch/log
        - debug
  telemetry:
    metrics:
      level: none # Disable collector's own metrics to prevent conflict on port 8888. We don't use those metrics anyway.
receivers:
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
    # Clear the list of default processors
    processors: []
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{.PathHome}}
processors:
  beat:
    processors:
      - add_cloud_metadata:
      - add_docker_metadata:
      - add_fields:
          fields:
            custom_field: "CustomValue"
      - add_host_metadata:
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
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
`
	var renderedConfig bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(configTemplate)).Execute(&renderedConfig, configParameters))
	configContents := renderedConfig.Bytes()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Processor config:\n%s", configContents)
		}
	})

	writeEventsToLogFile(t, configParameters.InputFile, wantEvents)
	oteltestcol.New(t, string(configContents))

	receiverConfig := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - elasticsearch/log
        - debug
  telemetry:
    metrics:
      level: none # Disable collector's own metrics to prevent conflict on port 8888. We don't use those metrics anyway.
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-fbreceiver
          enabled: true
          paths:
            - %s
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    processors:
      - add_cloud_metadata:
      - add_docker_metadata:
      - add_fields:
          fields:
            custom_field: "CustomValue"
      - add_host_metadata:
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: %s
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
    logs_index: %s
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
`
	logFilePath := filepath.Join(t.TempDir(), "log.log")
	writeEventsToLogFile(t, logFilePath, wantEvents)
	receiverRenderedConfig := fmt.Sprintf(receiverConfig,
		logFilePath,
		t.TempDir(),
		receiverIndex,
	)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Receiver config:\n%s", receiverRenderedConfig)
		}
	})
	oteltestcol.New(t, receiverRenderedConfig)

	es := integration.GetESClient(t, "http")

	var processorDocuments estools.Documents
	var receiverDocuments estools.Documents
	var err error

	// wait for logs to be published
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			processorDocuments, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+processorIndex+"*")
			assert.NoError(ct, err)

			receiverDocuments, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+receiverIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, processorDocuments.Hits.Total.Value, wantEvents, "expected at least %d otel events, got %d", wantEvents, processorDocuments.Hits.Total.Value)
			assert.GreaterOrEqual(ct, receiverDocuments.Hits.Total.Value, wantEvents, "expected at least %d filebeat events, got %d", wantEvents, receiverDocuments.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected at least %d events for both filebeat and otel", wantEvents)

	processorDoc := processorDocuments.Hits.Hits[0].Source
	receiverDoc := receiverDocuments.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between the agents
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.path",
	}

	oteltest.AssertMapsEqual(t, receiverDoc, processorDoc, ignoredFields, "expected documents to be equal")
}

func TestNoDuplicates(t *testing.T) {
	integration.EnsureESIsRunning(t)

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbOtelIndex := "logs-integration-" + namespace

	otelMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	otelCfgFile := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - %s
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
    http.enabled: true
    http.host: localhost
    http.port: %d
    management.otel.enabled: true
exporters:
  elasticsearch/log:
    endpoints:
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: %s
    tls:
      insecure_skip_verify: true
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - elasticsearch/log
`
	logFilePath := filepath.Join(tmpdir, "log.log")
	writenLines := make([]string, 0)
	stopChan := make(chan struct{}, 1)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		// create a log file and keep writing to it until the test finishes.
		// This is to ensure that the filebeat receiver is continuously processing
		// new lines and creating new events, which increases the chances of
		// hitting edge cases that could cause duplicates on restart.
		defer wg.Done()
		logFile, err := os.Create(logFilePath)
		if err != nil {
			require.NoErrorf(t, err, "could not create file '%s'", logFilePath)
		}
		defer logFile.Close()

		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for range ticker.C {
			select {
			case <-stopChan:
				return
			default:
			}
			msg := fmt.Sprintf("This is spam message %d: %v", i, uuid.Must(uuid.NewV4()))
			_, err := logFile.Write([]byte(msg + "\n"))
			require.NoErrorf(t, err, "failed to write line %d to temp file", i)
			writenLines = append(writenLines, msg)
			i++
		}
	}()
	t.Cleanup(func() {
		close(stopChan)
		wg.Wait()
	})
	collector := oteltestcol.New(t, fmt.Sprintf(otelCfgFile, logFilePath, tmpdir, otelMonitoringPort, fbOtelIndex))

	require.EventuallyWithT(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			otelDocs, err := estools.GetAllLogsForIndexWithContext(findCtx, integration.GetESClient(t, "http"), ".ds-"+fbOtelIndex+"*")
			assert.NoError(ct, err)
			assert.Greater(ct, otelDocs.Hits.Total.Value, 100)
		},
		1*time.Minute, 1*time.Second, "expected more than 0 events, got none",
	)

	collector.Shutdown()

	// wait for 8888 port to be free (an indication that previous collector has exited)
	require.Eventually(t,
		func() bool {
			ln, err := net.Listen("tcp", "localhost:8888")
			if err != nil {
				return false
			}
			ln.Close()
			return true
		},
		10*time.Second,
		100*time.Millisecond,
		"port 8888 never became available",
	)

	// restart the collector process
	collector = oteltestcol.New(t, fmt.Sprintf(otelCfgFile, logFilePath, tmpdir, otelMonitoringPort, fbOtelIndex))
	t.Cleanup(func() {
		collector.Shutdown()
	})

	// wait for more docs to be published.
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			otelDocs, err := estools.GetAllLogsForIndexWithContext(findCtx, integration.GetESClient(t, "http"), ".ds-"+fbOtelIndex+"*")
			assert.NoError(ct, err)
			assert.Greater(ct, otelDocs.Hits.Total.Value, 300)
		},
		1*time.Minute, 1*time.Second, "expected more than 300 events, got less",
	)
	checkDuplicates(t, ".ds-"+fbOtelIndex+"*")
}

func checkDuplicates(t *testing.T, index string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// duplicate check
	rawQuery := map[string]any{
		"runtime_mappings": map[string]any{
			"log.offset": map[string]any{
				"type": "keyword",
			},
			"log.file.fingerprint": map[string]any{
				"type": "keyword",
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"match": map[string]any{"_index": index}},
				},
			},
		},
		"aggs": map[string]any{
			"duplicates": map[string]any{
				"multi_terms": map[string]any{
					"size":          500,
					"min_doc_count": 2,
					"terms": []map[string]any{
						{"field": "log.file.fingerprint"},
						{"field": "log.offset"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(rawQuery)
	require.NoError(t, err)

	es := esapi.New(integration.GetESClient(t, "http"))
	res, err := es.Search(
		es.Search.WithIndex(index),
		es.Search.WithSize(0),
		es.Search.WithBody(&buf),
		es.Search.WithPretty(),
		es.Search.WithContext(ctx),
	)
	require.NoError(t, err)
	require.Falsef(t, (res.StatusCode >= http.StatusMultipleChoices || res.StatusCode < http.StatusOK), "status should be 2xx was: %d", res.StatusCode)
	resultBuf, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	aggResults := map[string]any{}
	err = json.Unmarshal(resultBuf, &aggResults)
	require.NoError(t, err)
	aggs, ok := aggResults["aggregations"].(map[string]any)
	require.Truef(t, ok, "'aggregations' wasn't a map[string]any, result was %s", string(resultBuf))
	dups, ok := aggs["duplicates"].(map[string]any)
	require.Truef(t, ok, "'duplicates' wasn't a map[string]any, result was %s", string(resultBuf))
	buckets, ok := dups["buckets"].([]any)
	require.Truef(t, ok, "'buckets' wasn't a []any, result was %s", string(resultBuf))

	hits, ok := aggResults["hits"].(map[string]any)
	require.Truef(t, ok, "'hits' wasn't a map[string]any, result was %s", string(resultBuf))
	total, ok := hits["total"].(map[string]any)
	require.Truef(t, ok, "'total' wasn't a map[string]any, result was %s", string(resultBuf))
	value, ok := total["value"].(float64)
	require.Truef(t, ok, "'total' wasn't an int, result was %s", string(resultBuf))

	require.Emptyf(t, buckets, "len(buckets): %d, hits.total.value: %d, result was %s", len(buckets), value, string(resultBuf))
}

// setupRoleMapping sets up role mapping for the Kerberos user beats@elastic
func setupRoleMapping(t *testing.T, client *elasticsearch.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// prepare to query ES
	roleMappingURL := "http://localhost:9203/_security/role_mapping/kerbrolemapping"

	body := map[string]interface{}{
		"roles":   []string{"superuser"},
		"enabled": true,
		"rules": map[string]interface{}{
			"field": map[string]interface{}{
				"username": "beats@elastic",
			},
		},
	}

	jsonData, err := json.Marshal(body)
	require.NoError(t, err, "could not marshal role mapping body to json")

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		roleMappingURL,
		bytes.NewReader(jsonData))
	require.NoError(t, err, "could not create role mapping request")

	// Set content type header
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	require.NoError(t, err, "could not perform role mapping request")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "incorrect response code")
}

func TestFilebeatOTelNoEventLossDuringESOutage(t *testing.T) {
	const numTestEvents = 100

	tmpdir := t.TempDir()
	logFilePath := filepath.Join(tmpdir, "log.log")
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace

	serverPort := int(libbeattesting.MustAvailableTCP4Port(t))
	serverURL := fmt.Sprintf("http://localhost:%d", serverPort)

	var ingestedEvents []string
	var mu sync.Mutex

	handler := func(action api.Action, event []byte) int {
		if action.Action == "create" {
			mu.Lock()
			defer mu.Unlock()

			// Extract message from event
			if matches := regexp.MustCompile(`"message":"([^"]+)"`).FindSubmatch(event); len(matches) > 1 {
				message := string(matches[1])
				ingestedEvents = append(ingestedEvents, message)
			}
		}
		return http.StatusOK
	}

	createMockServer := func() *httptest.Server {
		reader := metric.NewManualReader()
		provider := metric.NewMeterProvider(metric.WithReader(reader))

		mockServer := httptest.NewUnstartedServer(
			api.NewDeterministicAPIHandler(
				uuid.Must(uuid.NewV4()),
				"",
				provider,
				time.Now().Add(24*time.Hour),
				0,
				0,
				handler,
			),
		)

		l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", serverPort))
		require.NoError(t, err)
		mockServer.Listener = l
		mockServer.Start()
		return mockServer
	}

	beatsConfig := struct {
		Index          string
		InputFile      string
		ESEndpoint     string
		MonitoringPort int
	}{
		Index:          index,
		InputFile:      logFilePath,
		ESEndpoint:     serverURL,
		MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
	}

	cfg := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - {{.InputFile}}
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    logging:
      level: debug
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    http.enabled: true
    http.host: localhost
    http.port: {{.MonitoringPort}}
exporters:
  elasticsearch:
    auth:
      authenticator: beatsauth
    compression: none
    endpoints:
      - {{.ESEndpoint}}
    logs_index: {{.Index}}
    max_conns_per_host: 1
    password: testing
    retry:
      enabled: true
      initial_interval: 100ms
      max_interval: 30s
      max_retries: 100
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1
        min_size: 0
        sizer: items
      block_on_overflow: true
      enabled: true
      num_consumers: 1
      queue_size: 3200
      wait_for_result: true
    user: admin
extensions:
  beatsauth:
    idle_connection_timeout: 3s
    proxy_disable: false
    timeout: 1m30s
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - elasticsearch
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
`

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, beatsConfig))

	writeEventsToLogFile(t, logFilePath, numTestEvents)

	collector := oteltestcol.New(t, configBuffer.String())

	t.Run("delivers events when Elasticsearch is unavailable at startup", func(t *testing.T) {
		// Wait for filebeat to read the file
		require.Eventually(t, func() bool {
			return collector.ObservedLogs().FilterMessageSnippet(fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath)).Len() >= 1
		}, 30*time.Second, 100*time.Millisecond, "timed out waiting for file input to be fully read")

		// Wait for a connection refused when the exporter tries to connect with ES and fails
		require.Eventually(t, func() bool {
			return collector.ObservedLogs().FilterMessageSnippet("connection refused").
				FilterMessageSnippet(fmt.Sprintf(":%d", serverPort)).Len() >= 1
		}, 30*time.Second, 100*time.Millisecond, "timed out waiting for connection refused error")

		// Verify no events were ingested yet (server down)
		assert.Empty(t, ingestedEvents, "expected no events to be ingested while server is down")

		// Mock ES starts
		mockServer := createMockServer()
		defer mockServer.Close()

		// Wait for all events to be delivered
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			mu.Lock()
			defer mu.Unlock()
			assert.Len(ct, ingestedEvents, numTestEvents, "expected all events to be delivered after server starts")

			// Verify we got the expected event content
			for i := 0; i < numTestEvents; i++ {
				expectedMsg := fmt.Sprintf("Line %d", i)
				found := false
				for _, ingested := range ingestedEvents {
					if ingested == expectedMsg {
						found = true
						break
					}
				}
				assert.True(ct, found, "expected to find event: %s", expectedMsg)
			}
		}, 30*time.Second, 1*time.Second, "timed out waiting for events to be delivered")
	})

	t.Run("continues delivering events after Elasticsearch failure", func(t *testing.T) {
		// reset observed logs
		collector.ObservedLogs().TakeAll()

		// Append events to log file
		appendEventsToLogFile(t, logFilePath, numTestEvents, numTestEvents)

		// Confirm filebeat read the new lines
		require.Eventually(t, func() bool {
			return collector.ObservedLogs().FilterMessageSnippet(fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath)).Len() == 1
		}, 30*time.Second, 100*time.Millisecond, "timed out waiting for file input to read new lines")

		// Confirm connection refused error
		require.Eventually(t, func() bool {
			return collector.ObservedLogs().FilterMessageSnippet("connection refused").
				FilterMessageSnippet(fmt.Sprintf(":%d", serverPort)).Len() >= 1
		}, 30*time.Second, 100*time.Millisecond, "timed out waiting for second connection refused error")

		// Mock ES restarts
		mockServer := createMockServer()
		defer mockServer.Close()

		// wait for new events to be delivered
		totalExpectedEvents := numTestEvents * 2
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			mu.Lock()
			defer mu.Unlock()
			assert.Len(ct, ingestedEvents, totalExpectedEvents, "expected all events (original + additional) to be delivered after server restarts")

			for i := 0; i < totalExpectedEvents; i++ {
				expectedMsg := fmt.Sprintf("Line %d", i)
				found := false
				for _, ingested := range ingestedEvents {
					if ingested == expectedMsg {
						found = true
						break
					}
				}
				assert.True(ct, found, "expected to find event: %s", expectedMsg)
			}
		}, 30*time.Second, 1*time.Second, "timed out waiting for all events to be delivered")
	})
}

func BenchmarkFilebeatOTelCollector(b *testing.B) {
	numReceivers := 4

	for b.Loop() {
		b.StopTimer()
		tmpDir := b.TempDir()

		type receiverConfig struct {
			Index    int
			PathHome string
		}

		configData := struct {
			Receivers []receiverConfig
		}{
			Receivers: make([]receiverConfig, numReceivers),
		}

		for i := range numReceivers {
			configData.Receivers[i] = receiverConfig{
				Index:    i + 1,
				PathHome: filepath.Join(tmpDir, strconv.Itoa(i+1)),
			}
		}

		cfgTemplate := `receivers:
{{range .Receivers}}
  filebeatreceiver/{{.Index}}:
    filebeat:
      inputs:
        - type: benchmark
          enabled: true
          count: 1
    path.home: {{.PathHome}}
    queue.mem.flush.timeout: 0s
{{end}}
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
{{range .Receivers}}
        - filebeatreceiver/{{.Index}}
{{end}}
      exporters:
        - debug
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
`

		var configBuffer bytes.Buffer
		require.NoError(b, template.Must(template.New("config").Parse(cfgTemplate)).Execute(&configBuffer, configData))

		b.StartTimer()

		col := oteltestcol.New(b, configBuffer.String())
		require.NotNil(b, col)
		require.Eventually(b, func() bool {
			return col.ObservedLogs().
				FilterMessageSnippet("Publish event").Len() == numReceivers
		}, 30*time.Second, 1*time.Millisecond, "expected all receivers to publish events")
		col.Shutdown()
	}
}

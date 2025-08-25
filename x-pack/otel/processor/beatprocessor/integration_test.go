// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package beatprocessor

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/testing/estools"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOtelBeatProcessorE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	wantEvents := 1

	// Start Filebeat receiver with Beat processors in a separate OTel processor
	processorFilebeat := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../../filebeat/filebeat.test",
		"otel",
	)

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	processorIndex := "logs-processor-" + namespace
	receiverIndex := "logs-receiver-" + namespace

	processor := int(libbeattesting.MustAvailableTCP4Port(t))
	receiverMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	configParameters := struct {
		Index          string
		MonitoringPort int
		InputFile      string
		PathHome       string
	}{
		Index:          processorIndex,
		MonitoringPort: processor,
		InputFile:      filepath.Join(processorFilebeat.TempDir(), "log.log"),
		PathHome:       processorFilebeat.TempDir(),
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
    output:
      otelconsumer:
    processors:
      # Configure a processor to prevent enabling default processors
      - add_fields:
          fields:
            custom_field: "custom_value"
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{.PathHome}}
    http.enabled: true
    http.host: localhost
    http.port: {{.MonitoringPort}}
processors:
  beat:
    processors:
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
    batcher:
      enabled: true
      flush_timeout: 1s
    mapping:
      mode: bodymap
`
	var renderedConfig bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(configTemplate)).Execute(&renderedConfig, configParameters))
	configContents := renderedConfig.Bytes()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Processor config:\n%s", configContents)
		}
	})

	processorFilebeat.WriteConfigFile(string(configContents))
	writeEventsToLogFile(t, configParameters.InputFile, wantEvents)
	processorFilebeat.Start()
	defer processorFilebeat.Stop()

	// start Filebeat receiver with processors embedded in receiver's configuration
	filebeatWithReceiver := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../../filebeat/filebeat.test",
		"otel",
	)

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
      - add_fields:
          fields:
            custom_field: "custom_value"
      - add_host_metadata:
    output:
      otelconsumer:
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: %s
    http.enabled: true
    http.host: localhost
    http.port: %v
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
    batcher:
      enabled: true
      flush_timeout: 1s
    mapping:
      mode: bodymap
`
	logFilePath := filepath.Join(filebeatWithReceiver.TempDir(), "log.log")
	writeEventsToLogFile(t, logFilePath, wantEvents)
	receiverRenderedConfig := fmt.Sprintf(receiverConfig,
		logFilePath,
		filebeatWithReceiver.TempDir(),
		receiverMonitoringPort,
		receiverIndex,
	)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Receiver config:\n%s", receiverRenderedConfig)
		}
	})
	filebeatWithReceiver.WriteConfigFile(receiverRenderedConfig)
	filebeatWithReceiver.Start()
	defer filebeatWithReceiver.Stop()

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
	assertMonitoring(t, configParameters.MonitoringPort)
	assertMonitoring(t, receiverMonitoringPort)
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

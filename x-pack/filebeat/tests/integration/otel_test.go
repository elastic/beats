// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofrs/uuid/v5"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/go-elasticsearch/v8"
)

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
    protocol: http
    username: admin
    password: testing
    index: %s
queue.mem.flush.timeout: 0s
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
http.enabled: true
http.host: localhost
http.port: %d
`

func TestFilebeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	numEvents := 1
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

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	logFilePath = filepath.Join(filebeat.TempDir(), "log.log")
	writeEventsToLogFile(t, logFilePath, numEvents)
	s := fmt.Sprintf(beatsCfgFile, logFilePath, fbIndex, filebeatMonitoringPort)
	s = s + `
setup.template.name: logs-filebeat-default
setup.template.pattern: logs-filebeat-default
`

	filebeat.WriteConfigFile(s)
	filebeat.Start()

	// prepare to query ES
	esCfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		Username:  "admin",
		Password:  "testing",
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // this is only for testing
			},
		},
	}
	es, err := elasticsearch.NewClient(esCfg)
	require.NoError(t, err)

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
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

	assertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
	assertMonitoring(t, otelMonitoringPort)
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

func assertMapsEqual(t *testing.T, m1, m2 mapstr.M, ignoredFields []string, msg string) {
	t.Helper()

	flatM1 := m1.Flatten()
	flatM2 := m2.Flatten()
	for _, f := range ignoredFields {
		hasKeyM1, _ := flatM1.HasKey(f)
		hasKeyM2, _ := flatM2.HasKey(f)

		if !hasKeyM1 && !hasKeyM2 {
			assert.Failf(t, msg, "ignored field %q does not exist in either map, please remove it from the ignored fields", f)
		}

		flatM1.Delete(f)
		flatM2.Delete(f)
	}
	require.Equal(t, "", cmp.Diff(flatM1, flatM2), "expected maps to be equal")
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

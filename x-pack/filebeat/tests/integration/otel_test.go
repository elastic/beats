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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	logFilePath := filepath.Join(filebeatOTel.TempDir(), "log.log")
	filebeatOTel.WriteConfigFile(fmt.Sprintf(beatsCfgFile, logFilePath, "logs-integration-default", 5066))
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
	s := fmt.Sprintf(beatsCfgFile, logFilePath, "logs-filebeat-default", 5067)
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
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-logs-integration-default*")
			require.NoError(t, err)

			filebeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-logs-filebeat-default*")
			require.NoError(t, err)

			return otelDocs.Hits.Total.Value >= numEvents && filebeatDocs.Hits.Total.Value >= numEvents
		},
		2*time.Minute, 1*time.Second, fmt.Sprintf("Number of hits %d not equal to number of events for %d", filebeatDocs.Hits.Total.Value, numEvents))

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
	assertMonitoring(t)
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

func assertMonitoring(t *testing.T) {
	r, err := http.Get("http://localhost:5066") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get("http://localhost:5066/stats") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get("http://localhost:5066/not-exist") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
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
`

func TestFilebeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	logFilePath := filepath.Join(filebeatOTel.TempDir(), "log.log")
	filebeatOTel.WriteConfigFile(fmt.Sprintf(beatsCfgFile, logFilePath, "logs-integration-default"))

	logFile, err := os.Create(logFilePath)
	if err != nil {
		t.Fatalf("could not create file '%s': %s", logFilePath, err)
	}

	numEvents := 5

	// write events to log file
	for i := 0; i < numEvents; i++ {
		msg := fmt.Sprintf("Line %d", i)
		_, err = logFile.Write([]byte(msg + "\n"))
		require.NoErrorf(t, err, "failed to write line %d to temp file", i)
	}

	if err := logFile.Sync(); err != nil {
		t.Fatalf("could not sync log file '%s': %s", logFilePath, err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("could not close log file '%s': %s", logFilePath, err)
	}

	filebeatOTel.Start()

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

	actualHits := &struct{ Hits int }{}
	// wait for logs to be published
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			OTelDocs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-logs-integration-default*")
			require.NoError(t, err)

			actualHits.Hits = OTelDocs.Hits.Total.Value
			return actualHits.Hits == numEvents
		},
		2*time.Minute, 1*time.Second, numEvents, actualHits.Hits)

}

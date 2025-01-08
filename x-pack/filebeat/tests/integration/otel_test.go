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

var eventsLogFileCfg = `
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
    index: logs-integration-default
queue.mem.flush.timeout: 0s
`

func TestFilebeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	filebeat := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	logFilePath := filepath.Join(filebeat.TempDir(), "log.log")
	filebeat.WriteConfigFile(fmt.Sprintf(eventsLogFileCfg, logFilePath))

	logFile, err := os.Create(logFilePath)
	if err != nil {
		t.Fatalf("could not create file '%s': %s", logFilePath, err)
	}

	numEvents := 10
	var msg string
	var originalMessage = make(map[string]bool)

	// write events to log file
	for i := 0; i < numEvents; i++ {
		msg = fmt.Sprintf("Line %d", i)
		originalMessage[msg] = false
		_, err = logFile.Write([]byte(msg + "\n"))
		require.NoErrorf(t, err, "failed to write line %d to temp file", i)
	}

	if err := logFile.Sync(); err != nil {
		t.Fatalf("could not sync log file '%s': %s", logFilePath, err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("could not close log file '%s': %s", logFilePath, err)
	}

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

	actualHits := &struct{ Hits int }{}
	allRetrieved := false

	// wait for logs to be published
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			docs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-logs-integration-default*")
			require.NoError(t, err)

			// Mark retrieved messages
			for _, hit := range docs.Hits.Hits {
				message := hit.Source["Body"].(map[string]interface{})["message"].(string) //nolint:errcheck // err check not required on accessing each doc

				if _, exists := originalMessage[message]; exists {
					originalMessage[message] = true // Mark as found
				}
			}

			// Check for missing messages
			for _, retrieved := range originalMessage {
				if !retrieved {
					allRetrieved = false
					break
				}
				allRetrieved = true
			}

			actualHits.Hits = docs.Hits.Total.Value
			return (actualHits.Hits == numEvents) && allRetrieved
		},
		3*time.Minute, 1*time.Second, fmt.Sprintf("actual hits: %d; expected hits: %d; and all messages retrieved: %t", actualHits.Hits, numEvents, allRetrieved))

}

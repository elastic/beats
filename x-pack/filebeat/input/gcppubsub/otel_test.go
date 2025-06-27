// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package gcppubsub

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/go-elasticsearch/v8"
)

func TestGCPInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	gcpConfig := `filebeat.inputs:
- type: gcp-pubsub
  project_id: test-project-id
  topic: test-topic-foo
  subscription.name: test-subscription-bar
  credentials_file: "testdata/fake.json"

output:
  elasticsearch:
    hosts:
      - localhost:9200
    username: admin
    password: testing

queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	filebeatOTel.WriteConfigFile(gcpConfig)
	// Create pubsub client for setting up and communicating to emulator.
	client, clientCancel := testSetup(t)
	defer clientCancel()
	defer client.Close()

	createTopic(t, client)
	const numMsgs = 10
	publishMessages(t, client, numMsgs)

	filebeatOTel.Start()

	// prepare to query ES
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

	rawQuery := map[string]any{
		"query": map[string]any{
			"match_phrase": map[string]any{
				"input.type": "gcppubsub",
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	var otelDocs estools.Documents

	// wait for logs to be published
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, "filebeat-9.1.0*", es)
			assert.NoError(t, err)

			return otelDocs.Hits.Total.Value >= 1
		},
		3*time.Minute, 1*time.Second, fmt.Sprintf("Number of hits %d not equal to number of events %d", otelDocs.Hits.Total.Value, 1))

}

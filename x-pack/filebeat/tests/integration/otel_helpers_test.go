// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v8"
)

func deleteDataStreamsFromES(t *testing.T, es *elasticsearch.Client, dataStreams []string) {
	t.Helper()

	_, err := es.Indices.DeleteDataStream(dataStreams)
	require.NoError(t, err, "failed to delete data streams")
}

const otelElasticsearchExporterYAML = `exporters:
    elasticsearch:
        auth:
            authenticator: beatsauth
        compression: gzip
        compression_params:
            level: 1
        endpoints:
            - {{ .ESURL }}
        logs_dynamic_pipeline:
            enabled: true
        logs_index: {{ .Index }}
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
`

const otelElasticsearchServiceYAML = `service:
    extensions:
        - beatsauth
    pipelines:
        logs:
            exporters:
                - elasticsearch
            receivers:
                - filebeatreceiver
    telemetry:
        metrics:
            level: none
`

func otelE2ERawQueryForInputTypeAndMessage(inputType, message string) map[string]any {
	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match_phrase": map[string]any{
							"input.type": inputType,
						},
					},
					{
						"match_phrase": map[string]any{
							"message": message,
						},
					},
				},
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}
}

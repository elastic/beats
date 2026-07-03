// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/net/nettest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

const (
	tcpInputTestMsg = "tcp-input-otel-e2e-test-event"
	udpInputTestMsg = "udp-input-otel-e2e-test-event"
)

func TestTCPInputOTelE2E(t *testing.T) {
	otelServerAddr := "127.0.0.1:9042"
	fbServerAddr := "127.0.0.1:9043"

	runSocketInputOTelE2E(
		t,
		"tcp",
		tcpInputTestMsg,
		otelServerAddr,
		fbServerAddr,
		nettest.RunTCPClient,
	)
}

func TestUDPInputOTelE2E(t *testing.T) {
	otelServerAddr := "127.0.0.1:9042"
	fbServerAddr := "127.0.0.1:9043"

	runSocketInputOTelE2E(
		t,
		"udp",
		udpInputTestMsg,
		otelServerAddr,
		fbServerAddr,
		nettest.RunUDPClient,
	)
}

type socketClientFn func(t *testing.T, address string, data []string)

func runSocketInputOTelE2E(
	t *testing.T,
	inputType, testMessage, otelAddress, fbAddress string,
	runClient socketClientFn,
) {
	t.Helper()
	integration.EnsureESIsRunning(t)

	otelHome := t.TempDir()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	data := []string{testMessage}

	type options struct {
		InputType string
		Index     string
		ESURL     string
		Username  string
		Password  string
		Host      string
		PathHome  string
	}

	filebeatConfig := `filebeat.inputs:
- type: {{ .InputType }}
  id: {{ .InputType }}-input-e2e
  host: {{ .Host }}

output:
  elasticsearch:
    hosts:
      - {{ .ESURL }}
    username: {{ .Username }}
    password: {{ .Password }}
    index: {{ .Index }}

queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`

	otelConfig := `exporters:
    elasticsearch:
        auth:
            authenticator: beatsauth
        compression: gzip
        compression_params:
            level: 1
        endpoints:
            - {{ .ESURL }}
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
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: {{ .InputType }}
                  id: {{ .InputType }}-input-e2e
                  host: {{ .Host }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        path.home: {{ .PathHome }}
service:
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

	optionsValue := options{
		InputType: inputType,
		ESURL:     fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:  user,
		Password:  password,
		PathHome:  otelHome,
	}

	var configBuffer bytes.Buffer
	optionsValue.Host = otelAddress
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(otelConfig)).Execute(&configBuffer, optionsValue))

	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	optionsValue.Host = fbAddress
	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(filebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	defer filebeat.Stop()

	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			"filebeat start running",
		},
		20*time.Second,
		"filebeat did not run",
	)

	go runClient(t, otelAddress, data)
	go runClient(t, fbAddress, data)

	es := integration.GetESClient(t, "http")

	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{
			otelIndex,
			fbIndex,
		})
		require.NoError(t, err, "failed to delete data streams")
	})

	rawQuery := map[string]any{
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
							"message": testMessage,
						},
					},
				},
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 900*time.Millisecond)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-"+otelIndex+"*", es)
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 otel document, got %d", otelDocs.Hits.Total.Value)

			filebeatDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-"+fbIndex+"*", es)
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, filebeatDocs.Hits.Total.Value, 1, "expected at least 1 filebeat document, got %d", filebeatDocs.Hits.Total.Value)
		},
		3*time.Minute, 1*time.Second, "expected at least 1 document for both filebeat and otel modes")

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.source.address",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

// HostAddress returns the host:port address used by net input integration tests.
func hostAddress(port uint16) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

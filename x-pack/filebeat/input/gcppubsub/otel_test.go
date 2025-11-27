// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package gcppubsub_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcppubsub/testutil"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/oteltestcol"

	"github.com/elastic/elastic-agent-libs/testing/estools"
)

func TestGCPInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	// Create pubsub client for setting up and communicating to emulator.
	client, clientCancel := testutil.TestSetup(t)
	defer func() {
		clientCancel()
		client.Close()
	}()

	testutil.CreateTopic(t, client)
	testutil.CreateSubscription(t, "test-subscription-otel", client)
	testutil.CreateSubscription(t, "test-subscription-fb", client)
	const numMsgs = 10
	testutil.PublishMessages(t, client, numMsgs)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	// create a random uuid and make sure it doesn't contain dashes/
	otelNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))
	fbNameSpace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))

	type options struct {
		Namespace    string
		ESURL        string
		Username     string
		Password     string
		Subscription string
	}

	gcpFilebeatConfig := `filebeat.inputs:
- type: gcp-pubsub
  project_id: test-project-id
  topic: test-topic-foo
  subscription.name:  {{ .Subscription }}
  credentials_file: "testdata/fake.json"

output:
  elasticsearch:
    hosts:
      - {{ .ESURL }}
    username: {{ .Username }}
    password: {{ .Password }}
    index: logs-integration-{{ .Namespace }}

queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`

	gcpOTelConfig := `exporters:
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
        logs_index: logs-integration-{{ .Namespace }}
        mapping:
            mode: bodymap
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
                - credentials_file: "testdata/fake.json"
                  project_id: test-project-id
                  subscription:
                    name: {{ .Subscription }}
                  topic: test-topic-foo
                  type: gcp-pubsub
        output:
            otelconsumer:
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
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
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}

	var configBuffer bytes.Buffer
	optionsValue.Namespace = otelNamespace
	optionsValue.Subscription = "test-subscription-otel"
	require.NoError(t, template.Must(template.New("config").Parse(gcpOTelConfig)).Execute(&configBuffer, optionsValue))

	oteltestcol.New(t, configBuffer.String())

	// reset buffer
	configBuffer.Reset()

	optionsValue.Namespace = fbNameSpace
	optionsValue.Subscription = "test-subscription-fb"
	require.NoError(t, template.Must(template.New("config").Parse(gcpFilebeatConfig)).Execute(&configBuffer, optionsValue))

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	defer filebeat.Stop()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	rawQuery := map[string]any{
		"query": map[string]any{
			"match_phrase": map[string]any{
				"input.type": "gcp-pubsub",
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error
	msg := &strings.Builder{}

	// wait for logs to be published
	require.Eventuallyf(t,
		func() bool {
			msg.Reset()
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-logs-integration-"+otelNamespace+"*", es)
			msg.WriteString(fmt.Sprintf("failed to query ES for beat documents: %v", err))

			filebeatDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-logs-integration-"+fbNameSpace+"*", es)
			msg.WriteString(fmt.Sprintf("failed to query ES for beat documents: %v", err))

			return otelDocs.Hits.Total.Value >= 1 && filebeatDocs.Hits.Total.Value >= 1
		},
		3*time.Minute, 1*time.Second, "document indexed by fb-otel: %d, by fb-classic: %d: expected atleast one document by both modes: %s", otelDocs.Hits.Total.Value, filebeatDocs.Hits.Total.Value, msg)

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")

}

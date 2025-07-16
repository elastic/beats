// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package gcppubsub

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

	"github.com/elastic/elastic-agent-libs/testing/estools"
)

func TestGCPInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	// Create pubsub client for setting up and communicating to emulator.
	client, clientCancel := testSetup(t)
	defer func() {
		clientCancel()
		client.Close()
	}()

	createTopic(t, client)
	createSubscription(t, "test-subscription-otel", client)
	createSubscription(t, "test-subscription-fb", client)
	const numMsgs = 10
	publishMessages(t, client, numMsgs)

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

	gcpConfig := `filebeat.inputs:
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

	// start filebeat in otel mode
	filebeatOTel := integration.NewBeat(
		t,
		"filebeat-otel",
		"../../filebeat.test",
		"otel",
	)

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}

	var configBuffer bytes.Buffer
	optionsValue.Namespace = otelNamespace
	optionsValue.Subscription = "test-subscription-otel"
	require.NoError(t, template.Must(template.New("config").Parse(gcpConfig)).Execute(&configBuffer, optionsValue))

	filebeatOTel.WriteConfigFile(configBuffer.String())

	filebeatOTel.Start()
	defer filebeatOTel.Stop()

	// reset buffer
	configBuffer.Reset()

	optionsValue.Namespace = fbNameSpace
	optionsValue.Subscription = "test-subscription-fb"
	require.NoError(t, template.Must(template.New("config").Parse(gcpConfig)).Execute(&configBuffer, optionsValue))

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
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")

}

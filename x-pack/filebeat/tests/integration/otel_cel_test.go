// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"

	"github.com/elastic/elastic-agent-libs/testing/estools"
)

const celProgram = `get(state.url).Body.as(body,{"events":[body.decode_json()]})`

func TestCELInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	celSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"cel-test-event","ip":"10.0.0.1"}`))
	}))
	t.Cleanup(celSrv.Close)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))
	fbNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	tempDir := t.TempDir()

	type options struct {
		Index       string
		ESURL       string
		Username    string
		Password    string
		ResourceURL string
		Program     string
		PathHome    string
	}

	celFilebeatConfig := `filebeat.inputs:
- type: cel
  id: cel-input-e2e
  interval: 1s
  resource.url: {{ .ResourceURL }}
  program: {{ .Program }}

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

	celOTelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: cel
                  id: cel-input-e2e
                  interval: 1s
                  resource.url: {{ .ResourceURL }}
                  program: {{ .Program }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
        path.home: {{ .PathHome }}
` + otelElasticsearchServiceYAML

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(celOTelConfig)).Execute(&configBuffer, options{
		ESURL:       fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:    user,
		Password:    password,
		ResourceURL: celSrv.URL,
		Program:     celProgram,
		Index:       otelIndex,
		PathHome:    tempDir,
	}))

	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	require.NoError(t, template.Must(template.New("config").Parse(celFilebeatConfig)).Execute(&configBuffer, options{
		ESURL:       fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:    user,
		Password:    password,
		ResourceURL: celSrv.URL,
		Program:     celProgram,
		Index:       fbIndex,
	}))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	defer filebeat.Stop()

	es := integration.GetESClient(t, "http")

	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := map[string]any{
		"query": map[string]any{
			"match_phrase": map[string]any{
				"input.type": "cel",
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
		2*time.Minute, 1*time.Second, "expected at least 1 document for both filebeat and otel modes")

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

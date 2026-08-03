// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	gcsmock "github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/mock"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
)

func TestCometdInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		PathHome string
	}

	cometdFilebeatConfig := `filebeat.inputs:
- type: cometd
  channel_name: /event/LoginEventStream
  auth.oauth2:
    client.id: client.id
    client.secret: client.secret
    user: user
    password: password
    token_url: http://localhost:8080/token
` + filebeatOutputYAML

	cometdOTelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: cometd
                  channel_name: /event/LoginEventStream
                  auth.oauth2:
                    client.id: client.id
                    client.secret: client.secret
                    user: user
                    password: password
                    token_url: http://localhost:8080/token
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        path.home: {{ .PathHome }}	
` + otelElasticsearchServiceYAML

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		PathHome: t.TempDir(),
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(cometdOTelConfig)).Execute(&configBuffer, optionsValue))

	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(cometdFilebeatConfig)).Execute(&configBuffer, optionsValue))

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
		deleteDataStreamsFromES(t, es, []string{
			otelIndex,
			fbIndex,
		})
	})

	rawQuery := map[string]any{
		"query": map[string]any{
			"match_phrase": map[string]any{
				"cometd.channel_name": "/event/LoginEventStream",
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

func TestGCSInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	gcsMock := gcsmock.GCSServer()
	gcsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/storage/v1") {
			http.StripPrefix("/storage/v1", gcsMock).ServeHTTP(w, r)
			return
		}
		gcsMock.ServeHTTP(w, r)
	}))
	t.Cleanup(gcsSrv.Close)

	otelHome := t.TempDir()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		MockURL  string
		PathHome string
	}

	gcsFilebeatConfig := `filebeat.inputs:
- type: gcs
  id: gcs-input-e2e
  project_id: elastic-sa
  alternative_host: {{ .MockURL }}
  auth.credentials_json.account_key: '{"type":"service_account"}'
  poll: false
  max_workers: 1
  file_selectors:
    - regex: '^ata\.json$'
  buckets:
    - name: gcs-test-new
` + filebeatOutputYAML

	gcsOTelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: gcs
                  id: gcs-input-e2e
                  project_id: elastic-sa
                  alternative_host: {{ .MockURL }}
                  auth.credentials_json.account_key: '{"type":"service_account"}'
                  poll: false
                  max_workers: 1
                  file_selectors:
                    - regex: '^ata\.json$'
                  buckets:
                    - name: gcs-test-new
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        path.home: {{ .PathHome }}
        management.otel.enabled: true
` + otelElasticsearchServiceYAML

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		MockURL:  gcsSrv.URL,
		PathHome: otelHome,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(gcsOTelConfig)).Execute(&configBuffer, optionsValue))

	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(gcsFilebeatConfig)).Execute(&configBuffer, optionsValue))

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
		deleteDataStreamsFromES(t, es, []string{
			otelIndex,
			fbIndex,
		})
	})

	rawQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match_phrase": map[string]any{
							"input.type": "gcs",
						},
					},
					{
						"match_phrase": map[string]any{
							"gcs.storage.object.name": "ata.json",
						},
					},
				},
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

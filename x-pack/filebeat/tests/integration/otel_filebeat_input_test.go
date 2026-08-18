// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mqtttestutil "github.com/elastic/beats/v9/filebeat/input/mqtt/testutil"
	redistestutil "github.com/elastic/beats/v9/filebeat/input/redis/testutil"
	"github.com/elastic/beats/v9/libbeat/tests/integration"
	azureblobmock "github.com/elastic/beats/v9/x-pack/filebeat/input/azureblobstorage/mock"
	"github.com/elastic/beats/v9/x-pack/filebeat/input/entityanalytics/provider/activedirectory/testactivedirectory"
	"github.com/elastic/beats/v9/x-pack/filebeat/input/entityanalytics/provider/azuread/testazuread"
	"github.com/elastic/beats/v9/x-pack/filebeat/input/entityanalytics/provider/jamf/testjamf"
	"github.com/elastic/beats/v9/x-pack/filebeat/input/entityanalytics/provider/okta/testokta"
	gcppubsubtestutil "github.com/elastic/beats/v9/x-pack/filebeat/input/gcppubsub/testutil"
	gcsmock "github.com/elastic/beats/v9/x-pack/filebeat/input/gcs/mock"
	"github.com/elastic/beats/v9/x-pack/otel/oteltest"
	"github.com/elastic/beats/v9/x-pack/otel/oteltestcol"
)

const (
	azureBlobTestAccountName = "beatsblobnew"
	azureBlobTestAccountKey  = "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q=="
	azureBlobTestContainer   = "beatscontainer"
	azureBlobTestBlob        = "ata.json"
	azureBlobTestMessage     = "iPhone 9"
)

func startAzureBlobMockStorageServer(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(azureblobmock.AzureStorageServer())
	t.Cleanup(srv.Close)

	return srv.URL + "/"
}

func TestCELInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	celProgram := `get(state.url).Body.as(body,{"events":[body.decode_json()]})`
	celSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"cel-test-event","ip":"10.0.0.1"}`))
	}))
	t.Cleanup(celSrv.Close)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	type options struct {
		Index       string
		ESURL       string
		Username    string
		Password    string
		ResourceURL string
		Program     string
	}

	celFilebeatConfig := `filebeat.inputs:
- type: cel
  id: cel-input-e2e
  interval: 1s
  resource.url: {{ .ResourceURL }}
  program: {{ .Program }}
` + filebeatOutputYAML

	celOTelConfig := otelElasticsearchExporterYAML +
		`
receivers:
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
` + otelElasticsearchServiceYAML

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(celOTelConfig)).Execute(&configBuffer, options{
		ESURL:       fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:    user,
		Password:    password,
		ResourceURL: celSrv.URL,
		Program:     celProgram,
		Index:       otelIndex,
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
				"input.type": "cel",
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

func TestFilebeatOTelHTTPJSONInput(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	// create a random uuid and make sure it doesn't contain dashes/
	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNameSpace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNameSpace

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
	}

	// The request url is a http mock server started using streams
	configFile := `
filebeat.inputs:
  - type: httpjson
    id: httpjson-e2e-otel
    request.url: http://localhost:8090/test
` + filebeatOutputYAML

	otelConfigFile := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: httpjson
          id: httpjson-e2e-otel
          request.url: http://localhost:8090/test
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
` + otelElasticsearchExporterYAML + otelElasticsearchServiceYAML

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(otelConfigFile)).Execute(&configBuffer, optionsValue))
	oteltestcol.New(t, configBuffer.String())

	// reset buffer
	configBuffer.Reset()

	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(configFile)).Execute(&configBuffer, optionsValue))

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	rawQuery := map[string]any{
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

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

func TestRedisInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	otelHome := t.TempDir()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-filebeat-" + fbNamespace

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		Host     string
		PathHome string
	}

	emitCtx, emitCancel := context.WithCancel(t.Context())
	t.Cleanup(emitCancel)

	redisClient := redistestutil.CreateClient(t)
	redistestutil.ConfigureSlowlog(t, redisClient)
	redistestutil.EmitInputData(t, emitCtx, redisClient)

	// Standalone config
	redisFilebeatConfig := `filebeat.inputs:
- type: redis
  id: redis-input-e2e
  hosts:
    - {{ .Host }}
  maxconn: 10
  idle_timeout: 60s
  scan_frequency: 1s
  network: tcp
` + filebeatOutputYAML

	// OTel config
	redisOTelConfig := `exporters:
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
                - type: redis
                  id: redis-input-e2e
                  hosts:
                    - {{ .Host }}
                  maxconn: 10
                  idle_timeout: 60s
                  scan_frequency: 1s
                  network: tcp
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        path.home: {{ .PathHome }}
        management.otel.enabled: true
service:
    extensions:
        - beatsauth
    pipelines:
        logs:
            exporters:
                - elasticsearch
            receivers:
                - filebeatreceiver
`

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		Host:     redistestutil.HostPort(),
		PathHome: otelHome,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(redisOTelConfig)).Execute(&configBuffer, optionsValue))

	// 1. Start redis input in OTel mode
	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(redisFilebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())

	// 2. Start filebeat standalone mode
	filebeat.Start()

	es := integration.GetESClient(t, "http")

	t.Cleanup(func() {
		// delete data streams after the test is done
		deleteDataStreamsFromES(t, es, []string{
			otelIndex,
			fbIndex,
		})
	})

	// query to get redis data from elasticsearch
	rawQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match_phrase": map[string]any{
							"input.type": "redis",
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
		"event.created",
		"message",

		"redis.slowlog.id",
		"redis.slowlog.args",
		"redis.slowlog.clientAddr",
		"redis.slowlog.cmd",
		"redis.slowlog.duration.us",
		"redis.slowlog.key",
	}

	// compare docs
	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

func TestMQTTInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	otelHome := t.TempDir()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))
	fbNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace
	mqttInputTestMsg := "mqtt-input-otel-e2e-test-event"

	topic := fmt.Sprintf("test-mqtt-input-%s", uuid.Must(uuid.NewV4()).String())

	emitCtx, emitCancel := context.WithCancel(t.Context())
	t.Cleanup(emitCancel)

	publisher := mqtttestutil.CreatePublisher(t, "mqtt-test-publisher")
	mqtttestutil.EmitMessages(t, emitCtx, publisher, topic, mqttInputTestMsg)

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		Broker   string
		Topic    string
		ClientID string
		PathHome string
	}

	mqttFilebeatConfig := `filebeat.inputs:
- type: mqtt
  id: mqtt-input-e2e
  hosts:
    - {{ .Broker }}
  topics:
    - {{ .Topic }}
  client_id: {{ .ClientID }}
` + filebeatOutputYAML

	mqttOTelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: mqtt
                  id: mqtt-input-e2e
                  hosts:
                    - {{ .Broker }}
                  topics:
                    - {{ .Topic }}
                  client_id: {{ .ClientID }}
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
		Broker:   mqtttestutil.HostPort(),
		Topic:    topic,
		PathHome: otelHome,
	}

	var configBuffer bytes.Buffer
	optionsValue.ClientID = "otel-mqtt-input-e2e"
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(mqttOTelConfig)).Execute(&configBuffer, optionsValue))

	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	optionsValue.ClientID = "fb-mqtt-input-e2e"
	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(mqttFilebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	es := integration.GetESClient(t, "http")

	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{
			otelIndex,
			fbIndex,
		})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("mqtt", mqttInputTestMsg)

	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"mqtt.message_id",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

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

func TestAzureBlobStorageInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	storageURL := startAzureBlobMockStorageServer(t)
	otelHome := t.TempDir()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	type options struct {
		Index       string
		ESURL       string
		Username    string
		Password    string
		StorageURL  string
		PathHome    string
		AccountName string
		AccountKey  string
	}

	filebeatConfig := `filebeat.inputs:
- type: azure-blob-storage
  id: azure-blob-storage-input-e2e
  account_name: {{ .AccountName }}
  storage_url: {{ .StorageURL }}
  auth:
    shared_credentials:
      account_key: {{ .AccountKey }}
  poll: false
  max_workers: 1
  containers:
    - name: ` + azureBlobTestContainer + `
  file_selectors:
    - regex: '^` + azureBlobTestBlob + `$'
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: azure-blob-storage
                  id: azure-blob-storage-input-e2e
                  account_name: {{ .AccountName }}
                  storage_url: {{ .StorageURL }}
                  auth:
                    shared_credentials:
                      account_key: {{ .AccountKey }}
                  poll: false
                  max_workers: 1
                  containers:
                    - name: ` + azureBlobTestContainer + `
                  file_selectors:
                    - regex: '^` + azureBlobTestBlob + `$'
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        path.home: {{ .PathHome }}
` + otelElasticsearchServiceYAML

	optionsValue := options{
		ESURL:       fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:    user,
		Password:    password,
		StorageURL:  storageURL,
		PathHome:    otelHome,
		AccountName: azureBlobTestAccountName,
		AccountKey:  azureBlobTestAccountKey,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(otelConfig)).Execute(&configBuffer, optionsValue))
	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()
	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(filebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	filebeat.WaitLogsContains("filebeat start running", 20*time.Second, "filebeat did not run")

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("azure-blob-storage", azureBlobTestMessage)
	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}

	oteltest.AssertMapsEqual(t, filebeatDocs.Hits.Hits[0].Source, otelDocs.Hits.Hits[0].Source, ignoredFields, "expected documents to be equal")
}

func TestGCPInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	client, clientCancel := gcppubsubtestutil.TestSetup(t)
	defer func() {
		clientCancel()
		client.Close()
	}()

	gcppubsubtestutil.CreateTopic(t, client)
	gcppubsubtestutil.CreateSubscription(t, "test-subscription-otel", client)
	gcppubsubtestutil.CreateSubscription(t, "test-subscription-fb", client)
	const numMsgs = 10
	gcppubsubtestutil.PublishMessages(t, client, numMsgs)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))
	fbNamespace := fmt.Sprintf("%x", uuid.Must(uuid.NewV4()))

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	type options struct {
		Index           string
		ESURL           string
		Username        string
		Password        string
		Subscription    string
		CredentialsFile string
	}

	gcpFilebeatConfig := `filebeat.inputs:
- type: gcp-pubsub
  project_id: test-project-id
  topic: test-topic-foo
  subscription.name:  {{ .Subscription }}
  credentials_file: "{{ .CredentialsFile }}"
` + filebeatOutputYAML

	gcpOTelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - credentials_file: "{{ .CredentialsFile }}"
                  project_id: test-project-id
                  subscription:
                    name: {{ .Subscription }}
                  topic: test-topic-foo
                  type: gcp-pubsub
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
` + otelElasticsearchServiceYAML

	//nolint:gosec // G101: CredentialsFile is a path to a test fixture, not a secret
	optionsValue := options{
		ESURL:           fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:        user,
		Password:        password,
		CredentialsFile: "testdata/gcp_pubsub_fake_credentials.json",
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	optionsValue.Subscription = "test-subscription-otel"
	require.NoError(t, template.Must(template.New("config").Parse(gcpOTelConfig)).Execute(&configBuffer, optionsValue))
	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()
	optionsValue.Index = fbIndex
	optionsValue.Subscription = "test-subscription-fb"
	require.NoError(t, template.Must(template.New("config").Parse(gcpFilebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

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

	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
	}

	oteltest.AssertMapsEqual(t, filebeatDocs.Hits.Hits[0].Source, otelDocs.Hits.Hits[0].Source, ignoredFields, "expected documents to be equal")
}

func TestHTTPEndpointInputOTelE2E(t *testing.T) {
	httpEndpointInputTestMsg := "http-endpoint-otel-e2e-test-event"

	integration.EnsureESIsRunning(t)

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
		Host     string
		Port     string
		PathHome string
	}

	filebeatConfig := `filebeat.inputs:
- type: http_endpoint
  id: http-endpoint-input-e2e
  listen_address: {{ .Host }}
  listen_port: {{ .Port }}
  url: /events
  prefix: json
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: http_endpoint
                  id: http-endpoint-input-e2e
                  listen_address: {{ .Host }}
                  listen_port: {{ .Port }}
                  url: /events
                  prefix: json
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
		Host:     "127.0.0.1",
		// Bind to an ephemeral port and read the OS-assigned port back from
		// the logs
		Port:     "0",
		PathHome: otelHome,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(otelConfig)).Execute(&configBuffer, optionsValue))

	col := oteltestcol.New(t, configBuffer.String())
	otelPort := col.SocketListeningPort(t)

	configBuffer.Reset()

	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(filebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	filebeat.WaitLogsContainsAnyOrder(
		[]string{"filebeat start running"},
		20*time.Second,
		"filebeat did not run",
	)
	fbPort := filebeat.SocketListeningPort(20 * time.Second)

	payload := fmt.Sprintf(`{"message":%q}`, httpEndpointInputTestMsg)
	postHTTPEndpointEvent(t, fmt.Sprintf("http://127.0.0.1:%d/events", otelPort), payload)
	postHTTPEndpointEvent(t, fmt.Sprintf("http://127.0.0.1:%d/events", fbPort), payload)

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match_phrase": map[string]any{
							"input.type": "http_endpoint",
						},
					},
					{
						"match_phrase": map[string]any{
							"json.message": httpEndpointInputTestMsg,
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

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}
	oteltest.AssertMapsEqual(t, filebeatDocs.Hits.Hits[0].Source, otelDocs.Hits.Hits[0].Source, ignoredFields, "expected documents to be equal")
}

func TestNetflowInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	netflowSourceIP := "172.16.32.100"

	otelHome := t.TempDir()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace

	packet, err := os.ReadFile(filepath.Join("..", "..", "input", "netflow", "testdata", "dat", "netflow9_test_valid01.dat"))
	require.NoError(t, err, "failed to read netflow test packet")

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		Host     string
		PathHome string
	}

	filebeatConfig := `filebeat.inputs:
- type: netflow
  id: netflow-input-e2e
  host: {{ .Host }}
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: netflow
                  id: netflow-input-e2e
                  host: {{ .Host }}
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
		PathHome: otelHome,
	}

	ephemeralHost := hostAddress(0)

	var configBuffer bytes.Buffer
	optionsValue.Host = ephemeralHost
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(otelConfig)).Execute(&configBuffer, optionsValue))

	col := oteltestcol.New(t, configBuffer.String())
	otelAddress := hostAddress(col.SocketListeningPort(t))

	configBuffer.Reset()

	optionsValue.Host = ephemeralHost
	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(filebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	filebeat.WaitLogsContainsAnyOrder(
		[]string{"filebeat start running"},
		20*time.Second,
		"filebeat did not run",
	)
	fbAddress := hostAddress(filebeat.SocketListeningPort(20 * time.Second))

	go sendUDPPacket(t, otelAddress, packet)
	go sendUDPPacket(t, fbAddress, packet)

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match_phrase": map[string]any{
							"input.type": "netflow",
						},
					},
					{
						"match_phrase": map[string]any{
							"source.ip": netflowSourceIP,
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

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
		"netflow.exporter.address",
		"observer.ip",
	}
	oteltest.AssertMapsEqual(t, filebeatDocs.Hits.Hits[0].Source, otelDocs.Hits.Hits[0].Source, ignoredFields, "expected documents to be equal")
}

func postHTTPEndpointEvent(t *testing.T, url, payload string) {
	t.Helper()
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(payload))
		if !assert.NoError(ct, err, "failed to create request") {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if !assert.NoError(ct, err, "failed to POST to http_endpoint") {
			return
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		assert.Equal(ct, http.StatusOK, resp.StatusCode, "unexpected http_endpoint status")
	}, 20*time.Second, 100*time.Millisecond, "http_endpoint did not accept event at %s", url)
}

func sendUDPPacket(t *testing.T, address string, packet []byte) {
	t.Helper()
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		conn, err := net.Dial("udp", address) //nolint:noctx // test helper
		if !assert.NoError(ct, err, "failed to dial udp %s", address) {
			return
		}
		defer conn.Close()
		n, err := conn.Write(packet)
		assert.NoError(ct, err, "failed to write udp packet to %s", address)
		assert.Equal(ct, len(packet), n, "short udp write to %s", address)
	}, 20*time.Second, 100*time.Millisecond, "udp endpoint %s was not ready", address)
}

func TestEntityAnalyticsOktaInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	oktaSrv := testokta.StartServer(t)
	oktaDomain := strings.TrimPrefix(oktaSrv.URL, "https://")

	fbConfig := `filebeat.inputs:
- type: entity-analytics
  id: entity-analytics-okta-e2e
  enabled: true
  provider: okta
  dataset: users
  enrich_with: ["none"]
  sync_interval: 24h
  update_interval: 12h
  okta_domain: {{ .OktaDomain }}
  okta_token: test-token
  limit_fixed: 1000
  request.ssl.verification_mode: none
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: entity-analytics
                  id: entity-analytics-okta-e2e
                  enabled: true
                  provider: okta
                  dataset: users
                  enrich_with: ["none"]
                  sync_interval: 24h
                  update_interval: 12h
                  okta_domain: {{ .OktaDomain }}
                  okta_token: test-token
                  limit_fixed: 1000
                  request.ssl.verification_mode: none
        path.home: {{ .PathHome }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
` + otelElasticsearchServiceYAML

	runEntityAnalyticsOTelE2E(
		t,
		entityAnalyticsE2ECase{
			name:       "okta",
			fbConfig:   fbConfig,
			otelConfig: otelConfig,
			templateData: map[string]any{
				"OktaDomain": oktaDomain,
			},
			queryMust: []map[string]any{
				{"match_phrase": map[string]any{"input.type": "entity-analytics"}},
				{"match_phrase": map[string]any{"event.action": "user-discovered"}},
				{"match_phrase": map[string]any{"user.id": "u1"}},
			},
		})
}

func TestEntityAnalyticsJamfInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	jamfTenant, username, password, _ := testjamf.StartServer(t)

	fbConfig := `filebeat.inputs:
- type: entity-analytics
  id: entity-analytics-jamf-e2e
  enabled: true
  provider: jamf
  sync_interval: 24h
  update_interval: 12h
  jamf_tenant: {{ .JamfTenant }}
  jamf_username: {{ .JamfUsername }}
  jamf_password: {{ .JamfPassword }}
  request.ssl.verification_mode: none
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: entity-analytics
                  id: entity-analytics-jamf-e2e
                  enabled: true
                  provider: jamf
                  sync_interval: 24h
                  update_interval: 12h
                  jamf_tenant: {{ .JamfTenant }}
                  jamf_username: {{ .JamfUsername }}
                  jamf_password: {{ .JamfPassword }}
                  request.ssl.verification_mode: none
        path.home: {{ .PathHome }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
` + otelElasticsearchServiceYAML

	runEntityAnalyticsOTelE2E(t, entityAnalyticsE2ECase{
		name:       "jamf",
		fbConfig:   fbConfig,
		otelConfig: otelConfig,
		templateData: map[string]any{
			"JamfTenant":   jamfTenant,
			"JamfUsername": username,
			"JamfPassword": password,
		},
		queryMust: []map[string]any{
			{"match_phrase": map[string]any{"input.type": "entity-analytics"}},
			{"match_phrase": map[string]any{"event.action": "device-discovered"}},
			{"match_phrase": map[string]any{"device.id": testjamf.DeviceUDID}},
		},
	})
}

func TestEntityAnalyticsAzureADInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	azureSrv := testazuread.StartGraphServer(t)

	fbConfig := `filebeat.inputs:
- type: entity-analytics
  id: entity-analytics-azure-ad-e2e
  enabled: true
  use_minimal_state: true
  provider: azure-ad
  dataset: users
  enrich_with: ["none"]
  sync_interval: 24h
  update_interval: 12h
  tenant_id: test-tenant
  client_id: test-client
  secret: test-secret
  login_endpoint: {{ .LoginEndpoint }}
  api_endpoint: {{ .APIEndpoint }}
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: entity-analytics
                  id: entity-analytics-azure-ad-e2e
                  enabled: true
                  use_minimal_state: true
                  provider: azure-ad
                  dataset: users
                  enrich_with: ["none"]
                  sync_interval: 24h
                  update_interval: 12h
                  tenant_id: test-tenant
                  client_id: test-client
                  secret: test-secret
                  login_endpoint: {{ .LoginEndpoint }}
                  api_endpoint: {{ .APIEndpoint }}
        path.home: {{ .PathHome }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
` + otelElasticsearchServiceYAML

	runEntityAnalyticsOTelE2E(t, entityAnalyticsE2ECase{
		name: "azure-ad",
		// use_minimal_state exposes login_endpoint/api_endpoint so Graph + token
		// traffic can be pointed at httptest (plain HTTP).
		fbConfig:   fbConfig,
		otelConfig: otelConfig,
		templateData: map[string]any{
			"LoginEndpoint": azureSrv.URL,
			"APIEndpoint":   azureSrv.URL + "/v1.0",
		},
		queryMust: []map[string]any{
			{"match_phrase": map[string]any{"input.type": "entity-analytics"}},
			{"match_phrase": map[string]any{"event.action": "user-discovered"}},
			{"match_phrase": map[string]any{"user.id": testazuread.UserID}},
		},
	})
}

func TestEntityAnalyticsActiveDirectoryInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	ldapURL := testactivedirectory.StartLDAPServer(t)

	fbConfig := `filebeat.inputs:
- type: entity-analytics
  id: entity-analytics-ad-e2e
  enabled: true
  use_minimal_state: true
  provider: activedirectory
  dataset: users
  sync_interval: 24h
  update_interval: 12h
  ad_url: {{ .ADURL }}
  ad_base_dn: DC=example,DC=com
  ad_user: cn=admin,dc=example,dc=com
  ad_password: pass
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `
receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: entity-analytics
                  id: entity-analytics-ad-e2e
                  enabled: true
                  use_minimal_state: true
                  provider: activedirectory
                  dataset: users
                  sync_interval: 24h
                  update_interval: 12h
                  ad_url: {{ .ADURL }}
                  ad_base_dn: DC=example,DC=com
                  ad_user: cn=admin,dc=example,dc=com
                  ad_password: pass
        path.home: {{ .PathHome }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
` + otelElasticsearchServiceYAML

	runEntityAnalyticsOTelE2E(t, entityAnalyticsE2ECase{
		name:       "activedirectory",
		fbConfig:   fbConfig,
		otelConfig: otelConfig,
		templateData: map[string]any{
			"ADURL": ldapURL,
		},
		queryMust: []map[string]any{
			{"match_phrase": map[string]any{"input.type": "entity-analytics"}},
			{"match_phrase": map[string]any{"event.action": "user-discovered"}},
			{"match_phrase": map[string]any{"user.id": testactivedirectory.UserDN}},
		},
	})
}

type entityAnalyticsE2ECase struct {
	name         string
	fbConfig     string
	otelConfig   string
	templateData map[string]any
	queryMust    []map[string]any
}

func runEntityAnalyticsOTelE2E(t *testing.T, tc entityAnalyticsE2ECase) {
	t.Helper()

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-integration-" + fbNamespace
	otelHome := t.TempDir()

	data := map[string]any{
		"Index":    otelIndex,
		"ESURL":    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		"Username": user,
		"Password": password,
		"PathHome": otelHome,
	}

	maps.Copy(data, tc.templateData)

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("otel-"+tc.name).Parse(tc.otelConfig)).Execute(&configBuffer, data))
	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()
	data["Index"] = fbIndex
	require.NoError(t, template.Must(template.New("fb-"+tc.name).Parse(tc.fbConfig)).Execute(&configBuffer, data))

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": tc.queryMust,
			},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}

	oteltest.AssertMapsEqual(
		t,
		filebeatDocs.Hits.Hits[0].Source,
		otelDocs.Hits.Hits[0].Source,
		ignoredFields,
		fmt.Sprintf("expected entity-analytics/%s documents to be equal between classic and otel modes", tc.name),
	)
}

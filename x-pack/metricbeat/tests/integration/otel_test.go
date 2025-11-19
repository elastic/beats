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
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

func TestMetricbeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	// create a random uuid and make sure it doesn't contain dashes/
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	mbIndex := "logs-integration-mb-" + namespace
	mbReceiverIndex := "logs-integration-mbreceiver-" + namespace

	otelMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))
	metricbeatMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	otelConfig := struct {
		Index          string
		ESURL          string
		Username       string
		Password       string
		MonitoringPort int
	}{
		Index:          mbReceiverIndex,
		ESURL:          fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:       user,
		Password:       password,
		MonitoringPort: otelMonitoringPort,
	}

	cfg := `receivers:
  metricbeatreceiver:
    metricbeat:
      modules:
       - module: system
         enabled: true
         period: 1s
         processes:
          - '.*'
         metricsets:
          - cpu
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    http.enabled: true
    http.host: localhost
    http.port: {{.MonitoringPort}}
    management.otel.enabled: true
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - {{.ESURL}}
    compression: none
    user: {{.Username}}
    password: {{.Password}}
    logs_index: {{.Index}}
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
    mapping:
      mode: bodymap
service:
  pipelines:
    logs:
      receivers:
        - metricbeatreceiver
      exporters:
        - elasticsearch/log
        - debug
`
	var configBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, otelConfig))
	configContents := configBuffer.Bytes()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Config contents:\n%s", configContents)
		}
	})

	oteltestcol.New(t, configBuffer.String())

	var beatsCfgFile = `
metricbeat:
   modules:
   - module: system
     enabled: true
     period: 1s
     processes:
      - '.*'
     metricsets:
      - cpu
output:
  elasticsearch:
    hosts:
      - {{.ESURL}}
    username: {{.Username}}
    password: {{.Password}}
    index: {{.Index}}
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
http.enabled: true
http.host: localhost
http.port: {{.MonitoringPort}}
`

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{
			mbIndex,
			mbReceiverIndex,
		})
		require.NoError(t, err, "failed to delete indices")
	})

	var mbConfigBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&mbConfigBuffer,
		struct {
			Index          string
			ESURL          string
			Username       string
			Password       string
			MonitoringPort int
		}{
			Index:          mbIndex,
			ESURL:          fmt.Sprintf("%s://%s", host.Scheme, host.Host),
			Username:       user,
			Password:       password,
			MonitoringPort: metricbeatMonitoringPort,
		}))

	metricbeat := integration.NewBeat(t, "metricbeat", "../../metricbeat.test")
	metricbeat.WriteConfigFile(mbConfigBuffer.String())
	metricbeat.Start()
	defer metricbeat.Stop()

	// Make sure find the logs
	var metricbeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+mbReceiverIndex+"*")
			assert.NoError(ct, err)

			metricbeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+mbIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 log for otel receiver, got %d", otelDocs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, metricbeatDocs.Hits.Total.Value, 1, "expected at least 1 log for metricbeat, got %d", metricbeatDocs.Hits.Total.Value)
		},
		1*time.Minute, 1*time.Second, "expected at least 1 log for metricbeat and otel receiver")

	var metricbeatDoc, otelDoc mapstr.M
	otelDoc = otelDocs.Hits.Hits[0].Source
	metricbeatDoc = metricbeatDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}
	assert.Equal(t, "metricbeatreceiver", otelDoc.Flatten()["agent.otelcol.component.id"], "expected agent.otelcol.component.id field in log record")
	assert.Equal(t, "receiver", otelDoc.Flatten()["agent.otelcol.component.kind"], "expected agent.otelcol.component.kind field in log record")
	assert.NotContains(t, metricbeatDoc.Flatten(), "agent.otelcol.component.id", "expected agent.otelcol.component.id field not to be present in metricbeat log record")
	assert.NotContains(t, metricbeatDoc.Flatten(), "agent.otelcol.component.kind", "expected agent.otelcol.component.kind field not to be present in metricbeat log record")
	assertMapstrKeysEqual(t, otelDoc, metricbeatDoc, ignoredFields, "expected documents keys to be equal")
	assertMonitoring(t, metricbeatMonitoringPort)
}

func assertMonitoring(t *testing.T, port int) {
	address := fmt.Sprintf("http://localhost:%d", port)
	r, err := http.Get(address) //nolint:noctx,bodyclose,gosec // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/stats") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/not-exist") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

func TestMetricbeatOTelReceiverE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	es := integration.GetESClient(t, "http")

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	mbReceiverIndex := "logs-integration-mbreceiver-" + namespace
	mbIndex := "logs-integration-mb-" + namespace
	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{
			mbIndex,
			mbReceiverIndex,
		})
		require.NoError(t, err, "failed to delete indices")
	})

	cfg := `receivers:
  metricbeatreceiver:
    metricbeat:
      modules:
       - module: system
         enabled: true
         period: 1s
         processes:
          - '.*'
         metricsets:
          - cpu
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    management.otel.enabled: true
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - {{.ESURL}}
    compression: none
    user: {{.Username}}
    password: {{.Password}}
    logs_index: {{.Index}}
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
    mapping:
      mode: bodymap
service:
  pipelines:
    logs:
      receivers:
        - metricbeatreceiver
      exporters:
        - elasticsearch/log
        - debug
`

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, struct {
		Index    string
		ESURL    string
		Username string
		Password string
	}{
		Index:    mbReceiverIndex,
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}))
	configContents := configBuffer.Bytes()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Config contents:\n%s", configContents)
		}
	})

	oteltestcol.New(t, configBuffer.String())

	var beatsCfgFile = `
metricbeat:
   modules:
   - module: system
     enabled: true
     period: 1s
     processes:
      - '.*'
     metricsets:
      - cpu
output:
  elasticsearch:
    hosts:
      - {{.ESURL}}
    username: {{.Username}}
    password: {{.Password}}
    index: {{.Index}}
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`
	var beatsCfgBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&beatsCfgBuffer, struct {
		Index    string
		ESURL    string
		Username string
		Password string
	}{
		Index:    mbIndex,
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}))

	metricbeat := integration.NewBeat(t, "metricbeat", "../../metricbeat.test")
	metricbeat.WriteConfigFile(beatsCfgBuffer.String())
	metricbeat.Start()
	defer metricbeat.Stop()

	var metricbeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+mbReceiverIndex+"*")
			assert.NoError(ct, err)

			metricbeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+mbIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 log for otel receiver, got %d", otelDocs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, metricbeatDocs.Hits.Total.Value, 1, "expected at least 1 log for metricbeat receiver, got %d", metricbeatDocs.Hits.Total.Value)
		},
		1*time.Minute, 1*time.Second, "expected at least a single log for metricbeat and otel mode")
	otelDoc := otelDocs.Hits.Hits[0]
	metricbeatDoc := metricbeatDocs.Hits.Hits[0]
	ignoredFields := []string{
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}
	assertMapstrKeysEqual(t, otelDoc.Source, metricbeatDoc.Source, ignoredFields, "expected documents keys to be equal")
}

func TestMetricbeatOTelMultipleReceiversE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	type receiverConfig struct {
		MonitoringPort int
		InputFile      string
		PathHome       string
	}

	es := integration.GetESClient(t, "http")
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace
	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{
			index,
		})
		require.NoError(t, err, "failed to delete indices")
	})

	tmpDir := t.TempDir()
	otelConfig := struct {
		Index     string
		ESURL     string
		Username  string
		Password  string
		Receivers []receiverConfig
	}{
		Index:    index,
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		Receivers: []receiverConfig{
			{
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
				PathHome:       filepath.Join(tmpDir, "r1"),
			},
			{
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
				PathHome:       filepath.Join(tmpDir, "r2"),
			},
		},
	}

	cfg := `receivers:
{{range $i, $receiver := .Receivers}}
  metricbeatreceiver/{{$i}}:
    metricbeat:
      modules:
       - module: system
         enabled: true
         period: 1s
         processes:
          - '.*'
         metricsets:
          - cpu
    processors:
      - add_fields:
          target: ''
          fields:
            receiverid: "{{$i}}"
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{$receiver.PathHome}}
    management.otel.enabled: true
{{if $receiver.MonitoringPort}}
    http.enabled: true
    http.host: localhost
    http.port: {{$receiver.MonitoringPort}}
{{end}}
{{end}}
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - {{.ESURL}}
    compression: none
    user: {{.Username}}
    password: {{.Password}}
    logs_index: {{.Index}}
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
    mapping:
      mode: bodymap
service:
  pipelines:
    logs:
      receivers:
{{range $i, $receiver := .Receivers}}
        - metricbeatreceiver/{{$i}}
{{end}}
      exporters:
        - debug
        - elasticsearch/log
`
	var configBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, otelConfig))
	configContents := configBuffer.Bytes()

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Config contents:\n%s", configContents)
		}
	})

	oteltestcol.New(t, string(configContents))

	var r0Docs, r1Docs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			r0Docs, err = estools.PerformQueryForRawQuery(findCtx, map[string]any{
				"query": map[string]any{
					"match": map[string]any{
						"receiverid": "0",
					},
				},
			}, ".ds-"+otelConfig.Index+"*", es)
			assert.NoError(ct, err, "failed to query for receiver 0 logs")

			r1Docs, err = estools.PerformQueryForRawQuery(findCtx, map[string]any{
				"query": map[string]any{
					"match": map[string]any{
						"receiverid": "1",
					},
				},
			}, ".ds-"+otelConfig.Index+"*", es)
			assert.NoError(ct, err, "failed to query for receiver 1 logs")

			assert.GreaterOrEqualf(ct, r0Docs.Hits.Total.Value, 1, "expected at least 1 log for receiver 0, got %d", r0Docs.Hits.Total.Value)
			assert.GreaterOrEqualf(ct, r1Docs.Hits.Total.Value, 1, "expected at least 1 log for receiver 1, got %d", r1Docs.Hits.Total.Value)
		},
		1*time.Minute, 100*time.Millisecond, "expected at least 1 log for each receiver")
	ignoredFields := []string{
		// only present in beats receivers
		"agent.otelcol.component.id",
		"agent.otelcol.component.kind",
	}
	assertMapstrKeysEqual(t, r0Docs.Hits.Hits[0].Source, r1Docs.Hits.Hits[0].Source, ignoredFields, "expected documents keys to be equal")
	for _, rec := range otelConfig.Receivers {
		assertMonitoring(t, rec.MonitoringPort)
	}
}

func assertMapstrKeysEqual(t *testing.T, m1, m2 mapstr.M, ignoredFields []string, msg string) {
	t.Helper()
	// Delete all ignored fields.
	for _, f := range ignoredFields {
		_ = m1.Delete(f)
		_ = m2.Delete(f)
	}

	flatM1 := m1.Flatten()
	flatM2 := m2.Flatten()

	for k := range flatM1 {
		flatM1[k] = ""
	}
	for k := range flatM2 {
		flatM2[k] = ""
	}

	require.Zero(t, cmp.Diff(flatM1, flatM2), msg)
}

func TestMetricbeatOTelInspect(t *testing.T) {
	mbOTel := integration.NewBeat(
		t,
		"metricbeat-otel",
		"../../metricbeat.test",
		"otel",
	)

	var beatsCfgFile = `
metricbeat:
   modules:
   - module: system
     enabled: true
     period: 1s
     processes:
      - '.*'
     metricsets:
      - cpu
output:
  elasticsearch:
    hosts:
      - localhost:9200
    username: admin
    password: testing
    index: index
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`
	expectedExporter := `exporters:
    elasticsearch:
        auth:
            authenticator: beatsauth
        compression: gzip
        compression_params:
            level: 1
        endpoints:
            - http://localhost:9200
        logs_dynamic_pipeline:
            enabled: true
        logs_index: index
        mapping:
            mode: bodymap
        max_conns_per_host: 1
        password: testing
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
        user: admin
extensions:
    beatsauth:
        idle_connection_timeout: 3s
        proxy_disable: false
        timeout: 1m30s
`
	expectedReceiver := `receivers:
    metricbeatreceiver:
        logging:
            files:
                rotateeverybytes: 104857600
                rotateonstartup: false
            to_files: true
        metricbeat:
            modules:
                - enabled: true
                  metricsets:
                    - cpu
                  module: system
                  period: 1s
                  processes:
                    - .*
`

	expectedService := `service:
    extensions:
        - beatsauth
    pipelines:
        logs:
            exporters:
                - elasticsearch
            receivers:
                - metricbeatreceiver
`
	mbOTel.WriteConfigFile(beatsCfgFile)

	mbOTel.Start("inspect")
	defer mbOTel.Stop()

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		out, err := mbOTel.ReadStdout()
		require.NoError(collect, err)
		require.Contains(collect, out, expectedExporter)
		require.Contains(collect, out, expectedReceiver)
		require.Contains(collect, out, expectedService)
	}, 10*time.Second, 500*time.Millisecond, "failed to get output of inspect command")
}

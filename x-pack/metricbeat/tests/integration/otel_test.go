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

	"github.com/elastic/beats/v7/libbeat/tests/integration"
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

	type options struct {
		Index          string
		ESURL          string
		Username       string
		Password       string
		MonitoringPort int
	}

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
http.host: localhost
http.port: {{.MonitoringPort}}	
`

	// start metricbeat in otel mode
	metricbeatOTel := integration.NewBeat(
		t,
		"metricbeat-otel",
		"../../metricbeat.test",
		"otel",
	)

	optionsValue := options{
		ESURL:          fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:       user,
		Password:       password,
		MonitoringPort: 5078,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = "logs-integration-mbreceiver-" + namespace
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&configBuffer, optionsValue))

	metricbeatOTel.WriteConfigFile(configBuffer.String())
	metricbeatOTel.Start()
	defer metricbeatOTel.Stop()

	var mbConfigBuffer bytes.Buffer
	optionsValue.Index = "logs-integration-mb-" + namespace
	optionsValue.MonitoringPort = 5079
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&mbConfigBuffer, optionsValue))
	metricbeat := integration.NewBeat(t, "metricbeat", "../../metricbeat.test")
	metricbeat.WriteConfigFile(mbConfigBuffer.String())
	metricbeat.Start()
	defer metricbeat.Stop()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	// Make sure find the logs
	var metricbeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-logs-integration-mbreceiver-"+namespace+"*")
			require.NoError(t, err)

			metricbeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-logs-integration-mb-"+namespace+"*")
			require.NoError(t, err)

			return otelDocs.Hits.Total.Value >= 1 && metricbeatDocs.Hits.Total.Value >= 1
		},
		2*time.Minute, 1*time.Second, "Expected at least one ingested metric event, got metricbeat: %d, otel: %d", metricbeatDocs.Hits.Total.Value, otelDocs.Hits.Total.Value)

	otelDoc := otelDocs.Hits.Hits[0]
	metricbeatDoc := metricbeatDocs.Hits.Hits[0]
	assertMapstrKeysEqual(t, otelDoc.Source, metricbeatDoc.Source, []string{}, "expected documents keys to be equal")
	assertMonitoring(t, optionsValue.MonitoringPort)
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

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	mbReceiverIndex := "logs-integration-mbreceiver-" + namespace
	mbIndex := "logs-metricbeat-mb-" + namespace

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
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
    output:
      otelconsumer:
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
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
    batcher:
      enabled: true
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

	// start metricbeat in otel mode
	metricbeatOTel := integration.NewBeat(
		t,
		"metricbeat-otel",
		"../../metricbeat.test",
		"otel",
	)

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, options{
		Index:    mbReceiverIndex,
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}))

	metricbeatOTel.WriteConfigFile(configBuffer.String())
	metricbeatOTel.Start()
	defer metricbeatOTel.Stop()

	var beatsCfgFile = `receivers:
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
	var mbConfigBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&mbConfigBuffer, options{
		Index:    mbIndex,
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
	}))
	metricbeat := integration.NewBeat(t, "metricbeat", "../../metricbeat.test")
	metricbeat.WriteConfigFile(mbConfigBuffer.String())
	metricbeat.Start()
	defer metricbeat.Stop()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	var metricbeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	require.Eventuallyf(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+mbReceiverIndex+"*")
			require.NoError(t, err)

			metricbeatDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+mbIndex+"*")
			require.NoError(t, err)

			return otelDocs.Hits.Total.Value >= 1 && metricbeatDocs.Hits.Total.Value >= 1
		},
		2*time.Minute, 1*time.Second, "expected at least 1 log")
	otelDoc := otelDocs.Hits.Hits[0]
	metricbeatDoc := metricbeatDocs.Hits.Hits[0]
	assertMapstrKeysEqual(t, otelDoc.Source, metricbeatDoc.Source, []string{}, "expected documents keys to be equal")
}

func TestMetricbeatOTelMultipleReceiversE2E(t *testing.T) {
	t.Skip("Flaky test, see https://github.com/elastic/beats/issues/45631")
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	metricbeatOTel := integration.NewBeat(
		t,
		"metricbeat-otel",
		"../../metricbeat.test",
		"otel",
	)

	type receiverConfig struct {
		MonitoringPort int
		InputFile      string
		PathHome       string
	}

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	otelConfig := struct {
		Index     string
		Username  string
		Password  string
		Receivers []receiverConfig
	}{
		Index:    "logs-integration-" + namespace,
		Username: user,
		Password: password,
		Receivers: []receiverConfig{
			{
				MonitoringPort: 5066,
				PathHome:       filepath.Join(metricbeatOTel.TempDir(), "r1"),
			},
			{
				MonitoringPort: 5067,
				PathHome:       filepath.Join(metricbeatOTel.TempDir(), "r2"),
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
    output:
      otelconsumer:
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{$receiver.PathHome}}
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
      - http://localhost:9200
    compression: none
    user: {{.Username}}
    password: {{.Password}}
    logs_index: {{.Index}}
    batcher:
      enabled: true
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

	metricbeatOTel.WriteConfigFile(string(configContents))
	metricbeatOTel.Start()
	defer metricbeatOTel.Stop()

	es := integration.GetESClient(t, "http")

	var r0Docs, r1Docs estools.Documents
	var err error

	require.Eventuallyf(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			r0Docs, err = estools.PerformQueryForRawQuery(findCtx, map[string]any{
				"query": map[string]any{
					"match": map[string]any{
						"receiverid": "0",
					},
				},
			}, ".ds-"+otelConfig.Index+"*", es)
			require.NoError(t, err)

			r1Docs, err = estools.PerformQueryForRawQuery(findCtx, map[string]any{
				"query": map[string]any{
					"match": map[string]any{
						"receiverid": "1",
					},
				},
			}, ".ds-"+otelConfig.Index+"*", es)
			require.NoError(t, err)

			return r0Docs.Hits.Total.Value >= 1 && r1Docs.Hits.Total.Value >= 1
		},
		1*time.Minute, 100*time.Millisecond, "expected at least 1 log for each receiver")
	assertMapstrKeysEqual(t, r0Docs.Hits.Hits[0].Source, r1Docs.Hits.Hits[0].Source, []string{}, "expected documents keys to be equal")
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

func TestmbOTelInspect(t *testing.T) {
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
	expecteExporter := `exporters:
    elasticsearch:
        batcher:
            enabled: true
            max_size: 1600
            min_size: 0
        compression: gzip
        compression_params:
            level: 1
        endpoints:
            - http://localhost:9200
        idle_conn_timeout: 3s
        logs_index: index
        mapping:
            mode: bodymap
        password: testing
        retry:
            enabled: true
            initial_interval: 1s
            max_interval: 1m0s
            max_retries: 3
        timeout: 1m30s
        user: admin`
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
                    - .*`
	expectedService := `service:
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
		fmt.Println(out)
		require.NoError(collect, err)
		require.Contains(collect, out, expecteExporter)
		require.Contains(collect, out, expectedReceiver)
		require.Contains(collect, out, expectedService)
	}, 10*time.Second, 500*time.Millisecond, "failed to get output of inspect command")
}

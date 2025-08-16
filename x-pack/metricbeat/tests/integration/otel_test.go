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
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

func TestMetricbeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	es := integration.GetESClient(t, "http")

	// create a random uuid and make sure it doesn't contain dashes/
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	mbIndex := "logs-integration-mb-" + namespace
	mbReceiverIndex := "logs-integration-mbreceiver-" + namespace
	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{
			mbIndex,
			mbReceiverIndex,
		})
		require.NoError(t, err, "failed to delete indices")
	})

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
		MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = mbReceiverIndex
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&configBuffer, optionsValue))

	metricbeatOTel.WriteConfigFile(configBuffer.String())
	metricbeatOTel.Start()
	defer metricbeatOTel.Stop()

	var mbConfigBuffer bytes.Buffer
	optionsValue.Index = mbIndex
	optionsValue.MonitoringPort = int(libbeattesting.MustAvailableTCP4Port(t))
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&mbConfigBuffer, optionsValue))
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
		require.NoError(collect, err)
		require.Contains(collect, out, expectedExporter)
		require.Contains(collect, out, expectedReceiver)
		require.Contains(collect, out, expectedService)
	}, 10*time.Second, 500*time.Millisecond, "failed to get output of inspect command")
}

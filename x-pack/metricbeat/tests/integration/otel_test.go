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

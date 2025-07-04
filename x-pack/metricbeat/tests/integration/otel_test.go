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
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

func TestMetricbeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	// create a random uuid and make sure it doesn't contain dashes/
	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	type options struct {
		Namespace string
		ESURL     string
		Username  string
		Password  string
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
    index: logs-integration-{{ .Namespace }}
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`

	// start metricbeat in otel mode
	metricbeatOTel := integration.NewBeat(
		t,
		"metricbeat-otel",
		"../../metricbeat.test",
		"otel",
	)

	var configBuffer bytes.Buffer
	require.NoError(t, template.Must(template.New("config").Parse(beatsCfgFile)).Execute(&configBuffer, options{
		Namespace: otelNamespace,
		ESURL:     fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username:  user,
		Password:  password,
	}))

	metricbeatOTel.WriteConfigFile(configBuffer.String())
	metricbeatOTel.Start()
	defer metricbeatOTel.Stop()

	// prepare to query ES
	es := integration.GetESClient(t, "http")

	// Make sure find the logs
	actualHits := &struct{ Hits int }{}
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err := estools.GetLogsForIndexWithContext(findCtx, es, ".ds-logs-integration-"+otelNamespace+"*", map[string]interface{}{
				"metricset.name": "cpu",
			})
			require.NoError(t, err)

			actualHits.Hits = otelDocs.Hits.Total.Value
			return actualHits.Hits >= 1
		},
		2*time.Minute, 1*time.Second,
		"Expected at least %d logs, got %v", 1, actualHits)

}

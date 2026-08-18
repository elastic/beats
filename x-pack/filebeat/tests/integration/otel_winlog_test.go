// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && windows

package integration

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/tests/integration"
	"github.com/elastic/beats/v9/x-pack/otel/oteltest"
	"github.com/elastic/beats/v9/x-pack/otel/oteltestcol"
)

func TestWinlogInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelHome := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	otelIndex := "logs-integration-" + namespace
	fbIndex := "logs-filebeat-" + namespace

	evtxPath, err := filepath.Abs(filepath.Join("windows", "testdata", "1100.evtx"))
	require.NoError(t, err, "failed to resolve evtx path")

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		PathHome string
		EVTXPath string
	}

	filebeatConfig := `filebeat.inputs:
  - type: winlog
    id: winlog-input-e2e
    enabled: true
    name: {{ .EVTXPath }}

path.home: {{ .PathHome }}
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: winlog
                  id: winlog-input-e2e
                  enabled: true
                  name: {{ .EVTXPath }}
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
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		PathHome: otelHome,
		EVTXPath: evtxPath,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(otelConfig)).Execute(&configBuffer, optionsValue))
	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()
	optionsValue.Index = fbIndex
	optionsValue.PathHome = t.TempDir()
	require.NoError(t, template.Must(template.New("config").Parse(filebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := NewFilebeat(t)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	t.Cleanup(filebeat.Stop)

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("winlog", "The event logging service has shut down.")
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
		"expected documents to be equal",
	)
}

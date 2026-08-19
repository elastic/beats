// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && linux

package integration

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/tests/integration"
	"github.com/elastic/beats/v9/x-pack/otel/oteltest"
	"github.com/elastic/beats/v9/x-pack/otel/oteltestcol"
)

func TestJournaldInputOTelE2E(t *testing.T) {
	if _, err := exec.LookPath("systemd-cat"); err != nil {
		t.Skipf("systemd-cat not available: %v", err)
	}

	if _, err := exec.LookPath("journalctl"); err != nil {
		t.Skipf("journalctl not available: %v", err)
	}

	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelHome := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	syslogID := "journald-e2e-" + namespace
	testMessage := "otel journald e2e " + namespace

	otelIndex := "logs-integration-" + namespace
	fbIndex := "logs-filebeat-" + namespace

	type options struct {
		Index    string
		ESURL    string
		Username string
		Password string
		PathHome string
		SyslogID string
	}

	filebeatConfig := `filebeat.inputs:
  - type: journald
    id: journald-input-e2e
    enabled: true
    syslog_identifiers:
      - {{ .SyslogID }}

path.home: {{ .PathHome }}
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: journald
                  id: journald-input-e2e
                  enabled: true
                  syslog_identifiers:
                    - {{ .SyslogID }}
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
		SyslogID: syslogID,
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

	generateJournaldE2ELogs(t, syslogID, []string{testMessage})

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("journald", testMessage)
	filebeatDocs, otelDocs := getFilebeatOTelDocs(t, fbIndex, otelIndex, rawQuery)

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"event.created",
	}

	oteltest.AssertMapsEqual(
		t,
		filebeatDocs.Hits.Hits[0].Source,
		otelDocs.Hits.Hits[0].Source,
		ignoredFields,
		"expected documents to be equal",
	)
}

func generateJournaldE2ELogs(t *testing.T, syslogID string, messages []string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "systemd-cat", "-t", syslogID)
	w, err := cmd.StdinPipe()
	require.NoError(t, err, "failed creating systemd-cat stdin pipe")
	require.NoError(t, cmd.Start(), "failed starting systemd-cat")

	for _, message := range messages {
		_, err := fmt.Fprintln(w, message)
		require.NoError(t, err, "failed writing journald message")
		time.Sleep(10 * time.Millisecond)
	}

	require.NoError(t, w.Close(), "failed closing systemd-cat stdin")
	require.NoError(t, cmd.Wait(), "systemd-cat exited unsuccessfully")
}

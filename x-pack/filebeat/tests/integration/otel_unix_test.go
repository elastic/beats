// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !windows

package integration

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/tests/integration"
	"github.com/elastic/beats/v9/x-pack/otel/oteltest"
	"github.com/elastic/beats/v9/x-pack/otel/oteltestcol"
)

const unixInputTestMsg = "unix-input-otel-e2e-test-event"

func TestUnixInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()

	otelHome := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	shortNamespace := namespace[:12]

	otelIndex := "logs-integration-" + namespace
	fbIndex := "logs-filebeat-" + namespace

	// We use os.TempDir() here instead of t.TempDir() because there is a hard limit on how long the socket name cane be
	// t.TempDir() blows that up.
	otelSocketPath := filepath.Join(os.TempDir(), "otel-unix-"+shortNamespace+".sock")
	fbSocketPath := filepath.Join(os.TempDir(), "fb-unix-"+shortNamespace+".sock")
	t.Cleanup(func() {
		_ = os.Remove(otelSocketPath)
		_ = os.Remove(fbSocketPath)
	})

	type options struct {
		Index      string
		ESURL      string
		Username   string
		Password   string
		PathHome   string
		SocketPath string
	}

	filebeatConfig := `filebeat.inputs:
  - type: unix
    id: unix-input-e2e
    path: {{ .SocketPath }}

path.home: {{ .PathHome }}
` + filebeatOutputYAML

	otelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: unix
                  id: unix-input-e2e
                  path: {{ .SocketPath }}
        path.home: {{ .PathHome }}
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false		
` + otelElasticsearchServiceYAML

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		PathHome: otelHome,
	}

	var configBuffer bytes.Buffer
	optionsValue.Index = otelIndex
	optionsValue.SocketPath = otelSocketPath
	require.NoError(t, template.Must(template.New("config").Parse(otelConfig)).Execute(&configBuffer, optionsValue))
	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()
	optionsValue.Index = fbIndex
	optionsValue.PathHome = t.TempDir()
	optionsValue.SocketPath = fbSocketPath
	require.NoError(t, template.Must(template.New("config").Parse(filebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := NewFilebeat(t)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	t.Cleanup(filebeat.Stop)

	go sendUnixSocketMessages(t, otelSocketPath, []string{unixInputTestMsg})
	go sendUnixSocketMessages(t, fbSocketPath, []string{unixInputTestMsg})

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("unix", unixInputTestMsg)
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

func sendUnixSocketMessages(t *testing.T, socketPath string, messages []string) {
	t.Helper()

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		conn, err := net.Dial("unix", socketPath) //nolint:noctx // test helper
		if !assert.NoError(ct, err, "failed to dial unix socket %s", socketPath) {
			return
		}
		defer conn.Close()

		for _, message := range messages {
			_, err := fmt.Fprintln(conn, message)
			if !assert.NoError(ct, err, "failed to write to unix socket %s", socketPath) {
				return
			}
		}
	}, 20*time.Second, 100*time.Millisecond, "unix socket %s was not ready", socketPath)
}

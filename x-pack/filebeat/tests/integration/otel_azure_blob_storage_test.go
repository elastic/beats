// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/mock"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/testing/estools"
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

	srv := httptest.NewServer(mock.AzureStorageServer())
	t.Cleanup(srv.Close)

	return srv.URL + "/"
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

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	defer filebeat.Stop()

	filebeat.WaitLogsContains(
		"filebeat start running",
		20*time.Second,
		"filebeat did not run",
	)

	es := integration.GetESClient(t, "http")

	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{otelIndex, fbIndex})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("azure-blob-storage", azureBlobTestMessage)

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 900*time.Millisecond)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-"+otelIndex+"*", es)
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 otel document, got %d", otelDocs.Hits.Total.Value)

			filebeatDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-"+fbIndex+"*", es)
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, filebeatDocs.Hits.Total.Value, 1, "expected at least 1 filebeat document, got %d", filebeatDocs.Hits.Total.Value)
		},
		3*time.Minute, 1*time.Second, "expected at least 1 document for both filebeat and otel modes")

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

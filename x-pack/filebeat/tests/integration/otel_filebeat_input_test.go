// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/activedirectory/testactivedirectory"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/testazuread"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/jamf/testjamf"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/testokta"
	gcsmock "github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/mock"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
)

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
	defer filebeat.Stop()

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
	defer filebeat.Stop()

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

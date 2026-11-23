// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
)

func TestWorkloadIdentityConfig(t *testing.T) {
	tokenFilePath := filepath.Join(t.TempDir(), "token")
	for _, env := range []string{"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_FEDERATED_TOKEN_FILE"} {
		t.Setenv(env, "")
	}
	tests := []struct {
		name                 string
		settings             map[string]any
		env                  map[string]string
		wantError            string
		wantConnectionString string
	}{
		{name: "explicit settings"},
		{name: "storage key ignored", settings: map[string]any{"storage_account_key": "unused-key"}},
		{name: "storage connection string ignored", settings: map[string]any{"storage_account_connection_string": "unused-connection-string"}, wantConnectionString: "unused-connection-string"},
		{
			name:     "environment settings",
			settings: map[string]any{"tenant_id": "", "client_id": "", "federated_token_file": ""},
			env:      map[string]string{"AZURE_TENANT_ID": "tenant", "AZURE_CLIENT_ID": "client", "AZURE_FEDERATED_TOKEN_FILE": tokenFilePath},
		},
		{
			name:     "mixed settings",
			settings: map[string]any{"client_id": ""},
			env:      map[string]string{"AZURE_CLIENT_ID": "client"},
		},
		{
			name: "missing tenant", settings: map[string]any{"tenant_id": ""},
			wantError: "tenant_id or AZURE_TENANT_ID is required when using workload_identity authentication",
		},
		{
			name: "missing client", settings: map[string]any{"client_id": ""},
			wantError: "client_id or AZURE_CLIENT_ID is required when using workload_identity authentication",
		},
		{
			name: "missing token file", settings: map[string]any{"federated_token_file": ""},
			wantError: "federated_token_file or AZURE_FEDERATED_TOKEN_FILE is required when using workload_identity authentication",
		},
		{
			name: "missing namespace", settings: map[string]any{"eventhub_namespace": ""},
			wantError: "eventhub_namespace is required when using workload_identity authentication",
		},
		{
			name: "processor v1 rejected", settings: map[string]any{"processor_version": "v1"},
			wantError: "workload_identity authentication requires processor_version v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			settings := map[string]any{
				"auth_type": "workload_identity", "eventhub": "test-hub",
				"eventhub_namespace": "test.servicebus.windows.net", "storage_account": "teststorage",
				"tenant_id": "tenant", "client_id": "client", "federated_token_file": tokenFilePath,
			}
			maps.Copy(settings, tt.settings)
			cfg := defaultConfig()
			err := config.MustNewConfigFrom(settings).Unpack(&cfg)
			if tt.wantError != "" {
				assert.ErrorContains(t, err, tt.wantError, "invalid federation settings should fail while unpacking")
				return
			}
			require.NoError(t, err, "federation should not require any keys or client secrets")
			assert.Equal(t, tt.wantConnectionString, cfg.SAConnectionString, "validation must leave an unused storage connection string unchanged")
			assert.True(t, cfg.MigrateCheckpoint, "default checkpoint migration must remain enabled")
			namespace, err := cfg.GetFullyQualifiedEventHubNamespace()
			require.NoError(t, err, "checkpoint migration must resolve the namespace for federation")
			assert.Equal(t, "test.servicebus.windows.net", namespace, "migration must use the configured namespace")
		})
	}
}

// A per-credential transport keeps SDK exchanges isolated from other HTTP clients.
type workloadIdentityTestTransport func(*http.Request) (*http.Response, error)

func (f workloadIdentityTestTransport) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestWorkloadIdentityTokenExchange(t *testing.T) {
	for _, authority := range []string{"", "https://login.microsoftonline.us", "https://login.chinacloudapi.cn"} {
		t.Run(authority, func(t *testing.T) {
			for _, source := range []string{"configuration", "environment", "mixed"} {
				t.Run(source, func(t *testing.T) {
					configuredFile := filepath.Join(t.TempDir(), "configured-token")
					environmentFile := filepath.Join(t.TempDir(), "environment-token")
					require.NoError(t, os.WriteFile(environmentFile, []byte("environment-assertion"), 0600), "environment token must exist")
					t.Setenv("AZURE_TENANT_ID", "environment-tenant")
					t.Setenv("AZURE_CLIENT_ID", "environment-client")
					t.Setenv("AZURE_FEDERATED_TOKEN_FILE", environmentFile)
					cfg := azureInputConfig{TenantID: "configured-tenant", ClientID: "configured-client", FederatedTokenFile: configuredFile, AuthorityHost: authority}
					cfg.Proxy.Disable = true
					tenant, client, assertion := "configured-tenant", "configured-client", "configured-assertion"
					switch source {
					case "environment":
						cfg.TenantID, cfg.ClientID, cfg.FederatedTokenFile = "", "", ""
						tenant, client, assertion = "environment-tenant", "environment-client", "environment-assertion"
					case "mixed":
						cfg.ClientID = ""
						client = "environment-client"
					}
					options := workloadIdentityCredentialOptions(&cfg)
					proxyClient, ok := options.Transport.(*http.Client)
					require.True(t, ok, "proxy_disable must produce a dedicated HTTP client")
					proxyTransport, ok := proxyClient.Transport.(*http.Transport)
					require.True(t, ok, "proxy configuration must use an HTTP transport")
					assert.Nil(t, proxyTransport.Proxy, "proxy_disable must disable environment proxies")
					expectedCloud := getAzureCloud(authority)
					assert.Equal(t, expectedCloud, options.Cloud, "credential options must select the configured Azure cloud")
					authorityHost := strings.TrimSuffix(strings.TrimPrefix(expectedCloud.ActiveDirectoryAuthorityHost, "https://"), "/")
					scopes := map[string]int{}
					// Override only this credential's transport, preserving the configured cloud and identity fields.
					options.Transport = workloadIdentityTestTransport(func(r *http.Request) (*http.Response, error) {
						var body string
						switch {
						case strings.HasSuffix(r.URL.Path, "/instance"):
							body = fmt.Sprintf(`{"tenant_discovery_endpoint":"https://%s/%s/v2.0/.well-known/openid-configuration","metadata":[{"preferred_network":"%s","preferred_cache":"%s","aliases":["%s"]}]}`, authorityHost, tenant, authorityHost, authorityHost, authorityHost)
						case strings.HasSuffix(r.URL.Path, "/openid-configuration"):
							assert.Equal(t, authorityHost, r.URL.Host, "discovery must use the configured cloud")
							assert.Contains(t, r.URL.Path, "/"+tenant+"/", "discovery must use the resolved tenant")
							body = fmt.Sprintf(`{"token_endpoint":"https://%s/%s/oauth2/v2.0/token","authorization_endpoint":"https://%s/%s/oauth2/v2.0/authorize","issuer":"https://%s/%s/v2.0"}`, authorityHost, tenant, authorityHost, tenant, authorityHost, tenant)
						case strings.HasSuffix(r.URL.Path, "/token"):
							assert.Equal(t, authorityHost, r.URL.Host, "token exchange must use the configured cloud")
							assert.Contains(t, r.URL.Path, "/"+tenant+"/", "token exchange must use the resolved tenant")
							assert.NoError(t, r.ParseForm(), "token request must contain a form")
							assert.Equal(t, client, r.FormValue("client_id"), "configuration must override the environment when set")
							assert.Equal(t, assertion, r.FormValue("client_assertion"), "SDK must read the resolved projected token file")
							assert.Equal(t, "client_credentials", r.FormValue("grant_type"), "federation must use the client credentials grant")
							for scope := range strings.FieldsSeq(r.FormValue("scope")) {
								scopes[scope]++
							}
							body = `{"access_token":"federated-access-token","expires_in":3600,"token_type":"Bearer"}`
						default:
							return nil, fmt.Errorf("unexpected workload identity request: %s", r.URL.Redacted())
						}
						return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					})
					credential, err := azidentity.NewWorkloadIdentityCredential(options)
					require.NoError(t, err, "SDK must accept the configured credential options")
					if source != "environment" {
						_, err = credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"https://storage.azure.com/.default"}})
						assert.ErrorContains(t, err, configuredFile, "missing token file must produce an actionable error")
						require.NoError(t, os.WriteFile(configuredFile, []byte(assertion), 0600), "configured token must exist before acquisition")
					}
					for _, scope := range []string{"https://eventhubs.azure.net/.default", "https://storage.azure.com/.default"} {
						token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{scope}})
						require.NoError(t, err, "credential must exchange the resolved assertion for the requested scope")
						assert.Equal(t, "federated-access-token", token.Token, "credential must return the Entra access token")
						assert.Equal(t, 1, scopes[scope], "explicit GetToken call must request the resource-specific scope once")
					}
				})
			}
		})
	}
}

func TestWorkloadIdentityClientFactories(t *testing.T) {
	cfg := defaultConfig()
	cfg.AuthType = AuthTypeWorkloadIdentity
	cfg.TenantID, cfg.ClientID = "tenant", "client"
	cfg.FederatedTokenFile = filepath.Join(t.TempDir(), "token")
	cfg.EventHubNamespace, cfg.EventHubName, cfg.ConsumerGroup = "test.servicebus.windows.net", "test-hub", "$Default"
	cfg.SAName, cfg.SAContainer = "teststorage", "checkpoints"
	cfg.SAKey, cfg.SAConnectionString = "unused-key", "unused-connection-string"
	log := logptest.NewTestingLogger(t, "")
	credential, err := createCredential(&cfg, log)
	require.NoError(t, err, "factory must construct the explicit credential type")
	assert.IsType(t, &azidentity.WorkloadIdentityCredential{}, credential, "workload identity must select the explicit SDK credential")
	// Construction is lazy; AMQP authentication itself is exercised by the live tests.
	consumer, err := CreateEventHubConsumerClient(&cfg, log)
	require.NoError(t, err, "consumer construction must ignore legacy connection strings and keys")
	t.Cleanup(func() { assert.NoError(t, consumer.Close(context.Background()), "consumer must close") })
	blob, err := CreateStorageAccountContainerClient(&cfg, log)
	require.NoError(t, err, "Blob client construction must ignore legacy connection strings and keys")
	assert.Equal(t, "https://teststorage.blob.core.windows.net/checkpoints", blob.URL(), "Blob endpoint must use the configured account and container")
	cfg.TenantID = "invalid/tenant"
	_, err = CreateEventHubConsumerClient(&cfg, log)
	assert.ErrorContains(t, err, "failed to create workload identity credential", "consumer factory must propagate workload credential errors")
	_, err = CreateStorageAccountContainerClient(&cfg, log)
	assert.ErrorContains(t, err, "failed to create workload identity credential", "Blob factory must propagate workload credential errors")
}

func TestWorkloadIdentityProxy(t *testing.T) {
	var requests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodConnect, r.Method, "HTTPS credential traffic must tunnel through the configured proxy")
		assert.Equal(t, "login.microsoftonline.com:443", r.Host, "proxy must receive the authority connection")
		requests.Add(1)
		// Deliberately reject the connection: the test must never contact Entra.
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(proxy.Close)
	proxySettings, err := httpcommon.NewHTTPClientProxySettings(proxy.URL, nil, false)
	require.NoError(t, err, "test proxy URL must parse")
	cfg := azureInputConfig{AuthType: AuthTypeWorkloadIdentity, TenantID: "tenant", ClientID: "client", FederatedTokenFile: filepath.Join(t.TempDir(), "token"), Proxy: *proxySettings}
	require.NoError(t, os.WriteFile(cfg.FederatedTokenFile, []byte("projected-assertion"), 0600), "token file must exist")
	credential, err := createCredential(&cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "factory must accept explicit proxy settings")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://storage.azure.com/.default"}})
	assert.Error(t, err, "rejecting proxy must prevent token acquisition")
	assert.Positive(t, requests.Load(), "credential must contact the explicitly configured proxy")
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestWorkloadIdentityConfig(t *testing.T) {
	tokenFilePath := filepath.Join(t.TempDir(), "token")
	for _, env := range []string{"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_FEDERATED_TOKEN_FILE"} {
		t.Setenv(env, "")
	}
	tests := []struct {
		name      string
		settings  map[string]any
		env       map[string]string
		wantError string
	}{
		{name: "explicit settings"},
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
				require.ErrorContains(t, err, tt.wantError, "invalid federation settings should fail while unpacking")
				return
			}
			require.NoError(t, err, "federation should not require any keys or client secrets")
			assert.Empty(t, cfg.SAConnectionString, "validation must not synthesize a storage connection string")
			assert.True(t, cfg.MigrateCheckpoint, "default checkpoint migration must remain enabled")
			namespace, err := cfg.GetFullyQualifiedEventHubNamespace()
			require.NoError(t, err, "checkpoint migration must resolve the namespace for federation")
			assert.Equal(t, "test.servicebus.windows.net", namespace, "migration must use the configured namespace")
		})
	}
}

func TestWorkloadIdentityClients(t *testing.T) {
	// Invalid environment values ensure explicit YAML settings take precedence in the SDK.
	t.Setenv("AZURE_TENANT_ID", "invalid/tenant")
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "")
	for _, authority := range []string{"", "https://login.microsoftonline.us", "https://login.chinacloudapi.cn"} {
		t.Run(authority, func(t *testing.T) {
			authorityHost := strings.TrimSuffix(strings.TrimPrefix(getAzureCloud(authority).ActiveDirectoryAuthorityHost, "https://"), "/")
			var tokenRequests, blobRequests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case strings.HasSuffix(r.URL.Path, "/instance"):
					_, _ = fmt.Fprintf(w, `{"tenant_discovery_endpoint":"https://%s/tenant/v2.0/.well-known/openid-configuration","metadata":[{"preferred_network":"%s","preferred_cache":"%s","aliases":["%s"]}]}`, authorityHost, authorityHost, authorityHost, authorityHost)
				case strings.HasSuffix(r.URL.Path, "/openid-configuration"):
					assert.Equal(t, authorityHost, r.Host, "discovery should use the configured cloud")
					_, _ = fmt.Fprintf(w, `{"token_endpoint":"https://%s/tenant/oauth2/v2.0/token","authorization_endpoint":"https://%s/tenant/oauth2/v2.0/authorize","issuer":"https://%s/tenant/v2.0"}`, authorityHost, authorityHost, authorityHost)
				case strings.HasSuffix(r.URL.Path, "/token"):
					assert.Equal(t, authorityHost, r.Host, "token exchange should use the configured cloud")
					assert.NoError(t, r.ParseForm(), "token request should contain a form")
					assert.Equal(t, "client", r.FormValue("client_id"), "explicit client ID should override the environment")
					assert.Equal(t, "projected-service-account-token", r.FormValue("client_assertion"), "SDK should exchange the projected token")
					assert.Equal(t, "client_credentials", r.FormValue("grant_type"), "federation should use the client credentials grant")
					assert.Contains(t, []string{"https://eventhubs.azure.net/.default", "https://storage.azure.com/.default"}, strings.Fields(r.FormValue("scope"))[0], "each service should request its own token audience")
					tokenRequests.Add(1)
					_, _ = w.Write([]byte(`{"access_token":"federated-access-token","expires_in":3600,"token_type":"Bearer"}`))
				default:
					assert.Contains(t, r.Host, "teststorage.blob.", "only the configured checkpoint account should be contacted")
					assert.Equal(t, "Bearer federated-access-token", r.Header.Get("Authorization"), "checkpoint requests must use the federated access token")
					blobRequests.Add(1)
				}
			}))
			t.Cleanup(server.Close)
			// Route HTTPS to the local mock while retaining the service hostnames in requests.
			baseTransport, ok := server.Client().Transport.(*http.Transport)
			require.True(t, ok, "test server should expose an HTTP transport")
			transport := baseTransport.Clone()
			transport.TLSClientConfig.ServerName = server.Certificate().DNSNames[0]
			transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
			}
			originalTransport := http.DefaultTransport
			http.DefaultTransport = transport
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
				transport.CloseIdleConnections()
			})
			cfg := defaultConfig()
			cfg.AuthType = AuthTypeWorkloadIdentity
			cfg.EventHubNamespace = "test.servicebus.windows.net"
			cfg.EventHubName = "test-hub"
			cfg.ConsumerGroup = "$Default"
			cfg.SAName = "teststorage"
			cfg.TenantID = "tenant"
			cfg.ClientID = "client"
			cfg.FederatedTokenFile = filepath.Join(t.TempDir(), "missing-token")
			cfg.AuthorityHost = authority
			cfg.Proxy.Disable = true
			require.NoError(t, cfg.Validate(), "explicit configuration should override environment values")
			log := logptest.NewTestingLogger(t, "")
			credential, err := createCredential(&cfg, log)
			require.NoError(t, err, "factory should construct a workload identity credential")
			assert.IsType(t, &azidentity.WorkloadIdentityCredential{}, credential, "federation must select the explicit credential type")
			// No token is read at construction: the SDK must read the projected file at acquisition time.
			_, err = credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"https://storage.azure.com/.default"}})
			require.ErrorContains(t, err, cfg.FederatedTokenFile, "a missing projected token must produce an actionable error")
			require.NoError(t, os.WriteFile(cfg.FederatedTokenFile, []byte("projected-service-account-token"), 0600), "projected token should be available before acquisition")
			token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"https://eventhubs.azure.net/.default"}})
			require.NoError(t, err, "credential should exchange the token file through the configured HTTP transport")
			assert.Equal(t, "federated-access-token", token.Token, "credential should return the Entra token")
			consumer, err := CreateEventHubConsumerClient(&cfg, log)
			require.NoError(t, err, "consumer client should accept federation without a connection string")
			t.Cleanup(func() { assert.NoError(t, consumer.Close(context.Background()), "consumer should close") })
			container, err := CreateStorageAccountContainerClient(&cfg, log)
			require.NoError(t, err, "checkpoint client should accept federation without a storage key")
			assert.Contains(t, container.URL(), "teststorage.blob."+getStorageEndpointSuffix(authority), "checkpoint endpoint should match the selected cloud")
			_, err = container.GetProperties(context.Background(), nil)
			require.NoError(t, err, "checkpoint client should authenticate using federation")
			assert.EqualValues(t, 2, tokenRequests.Load(), "Event Hubs and Blob Storage must acquire resource-specific tokens")
			assert.EqualValues(t, 1, blobRequests.Load(), "checkpoint client should send an authenticated Blob request")
		})
	}
}

func TestWorkloadIdentityCredentialEnvironment(t *testing.T) {
	t.Setenv("AZURE_TENANT_ID", "tenant")
	t.Setenv("AZURE_CLIENT_ID", "client")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", filepath.Join(t.TempDir(), "token"))
	credential, err := createCredential(&azureInputConfig{AuthType: AuthTypeWorkloadIdentity}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "SDK should use environment settings when explicit fields are absent")
	assert.IsType(t, &azidentity.WorkloadIdentityCredential{}, credential, "environment configuration must select workload identity")
}

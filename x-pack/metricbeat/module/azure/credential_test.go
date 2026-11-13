// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

const (
	testTenantID = "00000000-0000-0000-0000-000000000000"
	testClientID = "00000000-0000-0000-0000-000000000001"
	testIDToken  = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature" //nolint:gosec // G101: not a credential, an inert JWT-shaped test fixture
)

// setCloudConnectorsTokenFile points the cloud connectors flow at a token file and
// clears the workload-identity issuer, so the token file path is selected.
func setCloudConnectorsTokenFile(t *testing.T) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "id-token")
	require.NoError(t, os.WriteFile(p, []byte(testIDToken), 0o600), "expected the token file to be written")
	t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
	t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, p)
}

func clearCloudConnectorsEnv(t *testing.T) {
	t.Helper()
	t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
	t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, "")
}

// entraTestServer serves the Entra endpoints for testTenantID on a custom authority
// host and records the form of the last token request.
type entraTestServer struct {
	*httptest.Server
	tokenForm atomic.Value
}

func newEntraTestServer(t *testing.T) *entraTestServer {
	t.Helper()
	s := &entraTestServer{}
	s.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/discovery/instance"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tenant_discovery_endpoint": s.URL + "/" + testTenantID + "/v2.0/.well-known/openid-configuration",
				"metadata":                  []any{},
			})
		case r.URL.Path == "/"+testTenantID+"/v2.0/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                 s.URL + "/" + testTenantID + "/v2.0",
				"authorization_endpoint": s.URL + "/" + testTenantID + "/oauth2/v2.0/authorize",
				"token_endpoint":         s.URL + "/" + testTenantID + "/oauth2/v2.0/token",
			})
		case r.URL.Path == "/"+testTenantID+"/oauth2/v2.0/token":
			assert.NoError(t, r.ParseForm(), "expected a form-encoded token request")
			s.tokenForm.Store(r.PostForm)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "azure-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *entraTestServer) clientOptions() policy.ClientOptions {
	return policy.ClientOptions{
		Cloud:     cloud.Configuration{ActiveDirectoryAuthorityHost: s.URL + "/"},
		Transport: entraTestTransport{server: s.Server},
	}
}

// entraTestTransport sends every request to the test server, including the
// instance discovery request the SDK makes to the public cloud for a custom
// authority host.
type entraTestTransport struct {
	server *httptest.Server
}

func (tr entraTestTransport) Do(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(tr.server.URL)
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = ""
	return tr.server.Client().Do(req) //nolint:gosec // G704: test transport that only sends to the local test server
}

func (s *entraTestServer) lastTokenForm(t *testing.T) url.Values {
	t.Helper()
	form, ok := s.tokenForm.Load().(url.Values)
	require.True(t, ok, "expected a token request to Entra")
	return form
}

func TestNewCredential(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("uses the client secret without cloud connectors", func(t *testing.T) {
		clearCloudConnectorsEnv(t)
		config := Config{TenantId: testTenantID, ClientId: testClientID, ClientSecret: "client-secret"}

		cred, err := NewCredential(config, policy.ClientOptions{}, logger)
		require.NoError(t, err, "expected the client secret credential to be created")
		assert.IsType(t, &azidentity.ClientSecretCredential{}, cred, "expected a client secret credential")
	})

	t.Run("uses identity federation with cloud connectors", func(t *testing.T) {
		setCloudConnectorsTokenFile(t)
		config := Config{TenantId: testTenantID, ClientId: testClientID, UseCloudConnectors: true}

		cred, err := NewCredential(config, policy.ClientOptions{}, logger)
		require.NoError(t, err, "expected the identity federation credential to be created")
		assert.IsType(t, &azidentity.ClientAssertionCredential{}, cred, "expected a client assertion credential")
	})

	t.Run("ignores the client secret with cloud connectors", func(t *testing.T) {
		setCloudConnectorsTokenFile(t)
		config := Config{TenantId: testTenantID, ClientId: testClientID, ClientSecret: "client-secret", UseCloudConnectors: true}

		cred, err := NewCredential(config, policy.ClientOptions{}, logger)
		require.NoError(t, err, "expected the identity federation credential to be created")
		assert.IsType(t, &azidentity.ClientAssertionCredential{}, cred, "expected a client assertion credential")
	})

	t.Run("sends the identity token to the configured authority as the client assertion", func(t *testing.T) {
		setCloudConnectorsTokenFile(t)
		entra := newEntraTestServer(t)
		config := Config{TenantId: testTenantID, ClientId: testClientID, UseCloudConnectors: true}

		cred, err := NewCredential(config, entra.clientOptions(), logger)
		require.NoError(t, err, "expected the identity federation credential to be created")

		token, err := cred.GetToken(t.Context(), policy.TokenRequestOptions{
			Scopes: []string{"https://management.azure.com/.default"},
		})
		require.NoError(t, err, "expected Entra to issue an access token")
		assert.Equal(t, "azure-access-token", token.Token, "expected the access token from Entra")

		form := entra.lastTokenForm(t)
		assert.Equal(t, []string{testClientID}, form["client_id"], "expected the configured client ID")
		assert.Equal(t, []string{"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"}, form["client_assertion_type"], "expected a JWT bearer client assertion")
		assert.Equal(t, []string{testIDToken}, form["client_assertion"], "expected the identity token as the client assertion")
		assert.Empty(t, form["client_secret"], "expected no client secret to be sent")
	})

	t.Run("fails without an identity token source", func(t *testing.T) {
		clearCloudConnectorsEnv(t)
		config := Config{TenantId: testTenantID, ClientId: testClientID, UseCloudConnectors: true}

		_, err := NewCredential(config, policy.ClientOptions{}, logger)
		require.ErrorContains(t, err, "couldn't create identity federation credentials", "expected the identity federation error")
		require.ErrorContains(t, err, "no identity token source configured", "expected the cause to be reported")
	})

	t.Run("fails with an invalid tenant ID", func(t *testing.T) {
		clearCloudConnectorsEnv(t)
		config := Config{TenantId: "invalid tenant", ClientId: testClientID, ClientSecret: "client-secret"}

		_, err := NewCredential(config, policy.ClientOptions{}, logger)
		require.ErrorContains(t, err, "couldn't create client credentials", "expected the client secret error")
	})
}

func TestNewServiceWithCloudConnectors(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	newConfig := func(t *testing.T) Config {
		t.Helper()
		config := Config{
			TenantId:           testTenantID,
			ClientId:           testClientID,
			SubscriptionId:     "subscription",
			UseCloudConnectors: true,
		}
		config.InitDefaults()
		require.NoError(t, config.Validate(), "expected the configuration to be valid")
		return config
	}

	t.Run("creates the service", func(t *testing.T) {
		setCloudConnectorsTokenFile(t)

		service, err := NewService(newConfig(t), logger)
		require.NoError(t, err, "expected the monitor service to be created")
		require.NotNil(t, service.metricsClient, "expected the metrics client to be created")
		require.NotNil(t, service.resourceClient, "expected the resources client to be created")
	})

	t.Run("fails without an identity token source", func(t *testing.T) {
		clearCloudConnectorsEnv(t)

		_, err := NewService(newConfig(t), logger)
		require.ErrorContains(t, err, "no identity token source configured", "expected the identity federation error")
	})
}

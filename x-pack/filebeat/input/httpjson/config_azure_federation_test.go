// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

const (
	testAzureTenantID = "00000000-0000-0000-0000-000000000000"
	testAzureClientID = "00000000-0000-0000-0000-000000000001"
	testAzureIDToken  = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature" //nolint:gosec // G101: not a credential, an inert JWT-shaped test fixture
	testAzureToken    = "azure-access-token"
)

func TestConfigOAuth2AzureCloudConnectorsValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		input   map[string]any
	}{
		{
			name: "valid_with_resource",
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"azure.resource":             "https://management.azure.com/",
				"client.id":                  testAzureClientID,
			},
		},
		{
			name: "valid_with_scopes",
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"scopes":                     []string{"https://graph.microsoft.com/.default"},
				"client.id":                  testAzureClientID,
			},
		},
		{
			name:    "must_have_tenant_id",
			wantErr: errors.New("tenant_id must be provided with azure.use_cloud_connectors accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.resource":             "https://management.azure.com/",
				"client.id":                  testAzureClientID,
			},
		},
		{
			name:    "must_not_have_token_url",
			wantErr: errors.New("token_url can't be used with azure.use_cloud_connectors; use tenant_id accessing 'auth.oauth2'"),
			input: map[string]any{ //nolint:gosec // G101: a test token endpoint URL, not a credential
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"azure.resource":             "https://management.azure.com/",
				"token_url":                  "https://login.example.com/token",
				"client.id":                  testAzureClientID,
			},
		},
		{
			name:    "must_have_client_id",
			wantErr: errors.New("client.id must be provided with azure.use_cloud_connectors accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"azure.resource":             "https://management.azure.com/",
			},
		},
		{
			name:    "must_not_have_client_secret",
			wantErr: errors.New("client.secret can't be used with azure.use_cloud_connectors accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"azure.resource":             "https://management.azure.com/",
				"client.id":                  testAzureClientID,
				"client.secret":              "a_client_secret",
			},
		},
		{
			name:    "must_not_have_user_and_password",
			wantErr: errors.New("user and password can't be used with azure.use_cloud_connectors accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"azure.resource":             "https://management.azure.com/",
				"client.id":                  testAzureClientID,
				"user":                       "a_user",
				"password":                   "a_password",
			},
		},
		{
			name:    "must_not_have_endpoint_params",
			wantErr: errors.New("endpoint_params can't be used with azure.use_cloud_connectors accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"azure.resource":             "https://management.azure.com/",
				"client.id":                  testAzureClientID,
				"endpoint_params":            map[string][]string{"p": {"v"}},
			},
		},
		{
			name:    "must_use_the_azure_provider",
			wantErr: errors.New("azure.use_cloud_connectors can only be used with the azure provider accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "okta",
				"azure.use_cloud_connectors": true,
			},
		},
		{
			name:    "must_have_scopes_or_resource",
			wantErr: errors.New("scopes or azure.resource must be provided with azure.use_cloud_connectors accessing 'auth.oauth2'"),
			input: map[string]any{
				"provider":                   "azure",
				"azure.use_cloud_connectors": true,
				"azure.tenant_id":            testAzureTenantID,
				"client.id":                  testAzureClientID,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := unpackAzureCloudConnectorsConfig(t, test.input)
			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error return from Unpack: got:%v want:%v", err, test.wantErr)
			}
		})
	}
}

func TestOAuth2AzureScopes(t *testing.T) {
	tests := []struct {
		name string
		conf oAuth2Config
		want []string
	}{
		{
			name: "scopes_take_precedence",
			conf: oAuth2Config{Scopes: []string{"a/.default"}, AzureResource: "https://management.azure.com/"},
			want: []string{"a/.default"},
		},
		{
			name: "resource_with_trailing_slash_is_converted_to_a_scope",
			conf: oAuth2Config{AzureResource: "https://management.azure.com/"},
			want: []string{"https://management.azure.com//.default"},
		},
		{
			name: "resource_without_trailing_slash_is_converted_to_a_scope",
			conf: oAuth2Config{AzureResource: "https://management.azure.com"},
			want: []string{"https://management.azure.com/.default"},
		},
		{
			name: "neither",
			conf: oAuth2Config{},
			want: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.conf.azureScopes())
		})
	}
}

func TestOAuth2AzureCloudConnectorsClient(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	input := map[string]any{
		"provider":                   "azure",
		"azure.use_cloud_connectors": true,
		"azure.tenant_id":            testAzureTenantID,
		"azure.resource":             "https://management.azure.com/",
		"client.id":                  testAzureClientID,
	}

	t.Run("authenticates API requests with a federated access token", func(t *testing.T) {
		setAzureCloudConnectorsTokenFile(t)
		srv := newAzureFederationTestServer(t)
		cfg, err := unpackAzureCloudConnectorsConfig(t, input)
		require.NoError(t, err, "expected the configuration to be valid")

		client, err := cfg.Auth.OAuth2.client(t.Context(), srv.client(), srv.client(), logger)
		require.NoError(t, err, "expected the oauth2 client to be created")

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/api", nil)
		require.NoError(t, err, "expected the API request to be created")
		resp, err := client.Do(req)
		require.NoError(t, err, "expected the API request to succeed")
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected the API to accept the access token")

		form := srv.lastTokenForm(t)
		assert.Equal(t, []string{testAzureClientID}, form["client_id"], "expected the configured client ID")
		assert.Equal(t, []string{testAzureIDToken}, form["client_assertion"], "expected the identity token as the client assertion")
		assert.Contains(t, strings.Fields(form.Get("scope")), "https://management.azure.com//.default", "expected the scope derived from azure.resource")
		assert.Empty(t, form["client_secret"], "expected no client secret to be sent")
	})

	t.Run("fails without an identity token source", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
		t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, "")
		cfg, err := unpackAzureCloudConnectorsConfig(t, input)
		require.NoError(t, err, "expected the configuration to be valid")

		_, err = cfg.Auth.OAuth2.client(t.Context(), http.DefaultClient, http.DefaultClient, logger)
		require.ErrorContains(t, err, "error creating Azure identity federation credentials", "expected the client error")
		require.ErrorContains(t, err, "no identity token source configured", "expected the cause to be reported")
	})
}

func TestAzureCloudConnectorsTokenExchangeBypassesAPITransport(t *testing.T) {
	setAzureCloudConnectorsTokenFile(t)
	srv := newAzureFederationTestServer(t)
	cfg, err := unpackAzureCloudConnectorsConfig(t, map[string]any{
		"provider":                   "azure",
		"azure.use_cloud_connectors": true,
		"azure.tenant_id":            testAzureTenantID,
		"azure.resource":             "https://management.azure.com/",
		"client.id":                  testAzureClientID,
	})
	require.NoError(t, err)

	apiClient := srv.client()
	apiClient.Transport = rejectTokenExchangeTransport{next: apiClient.Transport}
	client, err := cfg.Auth.OAuth2.client(t.Context(), apiClient, srv.client(), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/api", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err, "the Entra exchange must not use the API transport")
	resp.Body.Close()
}

func TestAzureCloudConnectorsRejectsCrossOriginRedirect(t *testing.T) {
	setAzureCloudConnectorsTokenFile(t)
	srv := newAzureFederationTestServer(t)
	cfg, err := unpackAzureCloudConnectorsConfig(t, map[string]any{
		"provider":                   "azure",
		"azure.use_cloud_connectors": true,
		"azure.tenant_id":            testAzureTenantID,
		"azure.resource":             "https://management.azure.com/",
		"client.id":                  testAzureClientID,
	})
	require.NoError(t, err)

	client, err := cfg.Auth.OAuth2.client(t.Context(), srv.client(), srv.client(), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/redirect", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.ErrorContains(t, err, "refusing to forward Azure access token across origins")
	assert.Empty(t, srv.redirectAuthorization(), "the redirect target must not receive the Azure access token")
}

func TestAzureCloudConnectorsClientKeepsAPITimeout(t *testing.T) {
	setAzureCloudConnectorsTokenFile(t)
	srv := newAzureFederationTestServer(t)
	cfg, err := unpackAzureCloudConnectorsConfig(t, map[string]any{
		"provider":                   "azure",
		"azure.use_cloud_connectors": true,
		"azure.tenant_id":            testAzureTenantID,
		"azure.resource":             "https://management.azure.com/",
		"client.id":                  testAzureClientID,
	})
	require.NoError(t, err)

	apiClient := srv.client()
	apiClient.Timeout = 42 * time.Second
	client, err := cfg.Auth.OAuth2.client(t.Context(), apiClient, srv.client(), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	assert.Equal(t, 42*time.Second, client.Timeout)
}

func unpackAzureCloudConnectorsConfig(t *testing.T, oauth2 map[string]any) (config, error) {
	t.Helper()
	cfg := conf.MustNewConfigFrom(map[string]any{
		"request.url": "https://localhost",
		"auth.oauth2": oauth2,
	})
	c := defaultConfig()
	err := cfg.Unpack(&c)
	return c, err
}

func setAzureCloudConnectorsTokenFile(t *testing.T) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "id-token")
	require.NoError(t, os.WriteFile(p, []byte(testAzureIDToken), 0o600), "expected the token file to be written")
	t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
	t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, p)
}

// azureFederationTestServer serves the Entra endpoints for testAzureTenantID and an
// API endpoint that requires the Entra access token. It records the form of the
// last token request.
type azureFederationTestServer struct {
	*httptest.Server
	tokenForm          atomic.Value
	redirectAuthHeader atomic.Value
}

func newAzureFederationTestServer(t *testing.T) *azureFederationTestServer {
	t.Helper()
	s := &azureFederationTestServer{}
	s.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/discovery/instance"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tenant_discovery_endpoint": "https://login.microsoftonline.com/" + testAzureTenantID + "/v2.0/.well-known/openid-configuration",
				"metadata":                  []any{},
			})
		case r.URL.Path == "/"+testAzureTenantID+"/v2.0/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                 "https://login.microsoftonline.com/" + testAzureTenantID + "/v2.0",
				"authorization_endpoint": "https://login.microsoftonline.com/" + testAzureTenantID + "/oauth2/v2.0/authorize",
				"token_endpoint":         "https://login.microsoftonline.com/" + testAzureTenantID + "/oauth2/v2.0/token",
			})
		case r.URL.Path == "/"+testAzureTenantID+"/oauth2/v2.0/token":
			assert.NoError(t, r.ParseForm(), "expected a form-encoded token request")
			s.tokenForm.Store(r.PostForm)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": testAzureToken,
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case r.URL.Path == "/api":
			if r.Header.Get("Authorization") != "Bearer "+testAzureToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/redirect":
			http.Redirect(w, r, "https://attacker.example.com/stolen", http.StatusFound)
		case r.URL.Path == "/stolen":
			s.redirectAuthHeader.Store(r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// client returns a client that sends every request to the test server, so both
// the Entra requests and the API requests reach it.
func (s *azureFederationTestServer) client() *http.Client {
	return &http.Client{Transport: redirectTransport{server: s.Server}}
}

func (s *azureFederationTestServer) lastTokenForm(t *testing.T) url.Values {
	t.Helper()
	form, ok := s.tokenForm.Load().(url.Values)
	require.True(t, ok, "expected a token request to Entra")
	return form
}

func (s *azureFederationTestServer) redirectAuthorization() string {
	auth, _ := s.redirectAuthHeader.Load().(string)
	return auth
}

type rejectTokenExchangeTransport struct {
	next http.RoundTripper
}

func (tr rejectTokenExchangeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.Path, "/discovery/instance") ||
		strings.HasSuffix(req.URL.Path, "/.well-known/openid-configuration") ||
		strings.HasSuffix(req.URL.Path, "/oauth2/v2.0/token") {
		return nil, errors.New("token exchange reached API transport")
	}
	return tr.next.RoundTrip(req)
}

type redirectTransport struct {
	server *httptest.Server
}

func (tr redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(tr.server.URL)
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = ""
	return tr.server.Client().Transport.RoundTrip(req)
}

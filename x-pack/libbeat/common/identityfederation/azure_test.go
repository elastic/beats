// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package identityfederation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// fakeJWT is a syntactically valid (3 dot-separated parts) JWT for tests.
const fakeJWT = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature"

func TestAzureParamsValidate(t *testing.T) {
	cases := []struct {
		name    string
		params  AzureParams
		wantErr string
	}{
		{
			name:    "missing all",
			params:  AzureParams{},
			wantErr: "TenantID is required",
		},
		{
			name:    "missing tenant id",
			params:  AzureParams{ClientID: "client", JWTFilePath: "/p"},
			wantErr: "TenantID is required",
		},
		{
			name:    "missing client id",
			params:  AzureParams{TenantID: "tenant", JWTFilePath: "/p"},
			wantErr: "ClientID is required",
		},
		{
			name:    "missing jwt path",
			params:  AzureParams{TenantID: "tenant", ClientID: "client"},
			wantErr: "JWTFilePath is required",
		},
		{
			name:   "all set",
			params: AzureParams{TenantID: "tenant", ClientID: "client", JWTFilePath: "/p"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestAzureReadJWT(t *testing.T) {
	dir := t.TempDir()

	t.Run("happy path", func(t *testing.T) {
		p := filepath.Join(dir, "good")
		require.NoError(t, os.WriteFile(p, []byte("  "+fakeJWT+"\n"), 0o600))
		got, err := AzureReadJWT(p)
		require.NoError(t, err)
		assert.Equal(t, fakeJWT, got)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := AzureReadJWT(filepath.Join(dir, "does-not-exist"))
		require.ErrorContains(t, err, "reading JWT file")
	})

	t.Run("empty file", func(t *testing.T) {
		p := filepath.Join(dir, "empty")
		require.NoError(t, os.WriteFile(p, []byte("   \n"), 0o600))
		_, err := AzureReadJWT(p)
		require.ErrorContains(t, err, "is empty")
	})

	t.Run("malformed jwt", func(t *testing.T) {
		p := filepath.Join(dir, "malformed")
		require.NoError(t, os.WriteFile(p, []byte("not.a-valid-jwt"), 0o600))
		_, err := AzureReadJWT(p)
		require.ErrorContains(t, err, "invalid JWT")
	})
}

func TestAzureNewClientAssertionCredential(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "jwt")
		require.NoError(t, os.WriteFile(p, []byte(fakeJWT), 0o600))

		cred, err := AzureNewClientAssertionCredential(AzureParams{
			TenantID:    "00000000-0000-0000-0000-000000000000",
			ClientID:    "00000000-0000-0000-0000-000000000001",
			JWTFilePath: p,
		})
		require.NoError(t, err)
		require.NotNil(t, cred)
	})

	t.Run("invalid params", func(t *testing.T) {
		_, err := AzureNewClientAssertionCredential(AzureParams{})
		require.ErrorContains(t, err, "invalid Azure identity federation params")
	})
}

func TestAzureWIIParamsValidate(t *testing.T) {
	source := &WIITokenSource{}
	cases := []struct {
		name    string
		params  AzureWIIParams
		wantErr string
	}{
		{
			name:    "missing all",
			params:  AzureWIIParams{},
			wantErr: "TenantID is required",
		},
		{
			name:    "missing tenant id",
			params:  AzureWIIParams{ClientID: "client", TokenSource: source},
			wantErr: "TenantID is required",
		},
		{
			name:    "missing client id",
			params:  AzureWIIParams{TenantID: "tenant", TokenSource: source},
			wantErr: "ClientID is required",
		},
		{
			name:    "missing token source",
			params:  AzureWIIParams{TenantID: "tenant", ClientID: "client"},
			wantErr: "TokenSource is required",
		},
		{
			name:   "all set",
			params: AzureWIIParams{TenantID: "tenant", ClientID: "client", TokenSource: source},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestAzureNewWIIClientAssertionCredential(t *testing.T) {
	t.Run("invalid params", func(t *testing.T) {
		_, err := AzureNewWIIClientAssertionCredential(AzureWIIParams{})
		require.ErrorContains(t, err, "invalid Azure identity federation params")
	})

	t.Run("wii: the WII token is sent to Entra as the client assertion", func(t *testing.T) {
		const (
			tenantID = "00000000-0000-0000-0000-000000000000"
			clientID = "00000000-0000-0000-0000-000000000001"
			wiiToken = "eyJhbGciOiJSUzI1NiJ9.wii-test.sig" //nolint:gosec // G101: not a credential, an inert JWT-shaped test fixture
		)

		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

		var wiiCalls atomic.Int32
		wii := newWIITestServer(t, pki, func(w http.ResponseWriter, r *http.Request) {
			wiiCalls.Add(1)
			assert.Equal(t, "/token", r.URL.Path)
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.JSONEq(t, `{"aud":"api://AzureADTokenExchange"}`, string(body))
			_, _ = w.Write(tokenResponse(wiiToken, time.Now().Add(time.Hour)))
		})

		var entraCalls atomic.Int32
		var entra *httptest.Server
		entra = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/" + tenantID + "/v2.0/.well-known/openid-configuration":
				_ = json.NewEncoder(w).Encode(map[string]string{
					"issuer":                 entra.URL + "/" + tenantID + "/v2.0",
					"authorization_endpoint": entra.URL + "/" + tenantID + "/oauth2/v2.0/authorize",
					"token_endpoint":         entra.URL + "/" + tenantID + "/oauth2/v2.0/token",
				})
			case "/" + tenantID + "/oauth2/v2.0/token":
				entraCalls.Add(1)
				assert.NoError(t, r.ParseForm())
				assert.Equal(t, clientID, r.PostForm.Get("client_id"))
				assert.Equal(t, "urn:ietf:params:oauth:client-assertion-type:jwt-bearer", r.PostForm.Get("client_assertion_type"))
				assert.Equal(t, wiiToken, r.PostForm.Get("client_assertion"))
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "azure-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(entra.Close)

		source, err := NewWIITokenSource(wii.URL, pki.certPath, pki.keyPath, "api://AzureADTokenExchange", logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		cred, err := AzureNewWIIClientAssertionCredential(AzureWIIParams{
			TenantID:    tenantID,
			ClientID:    clientID,
			TokenSource: source,
			Options: &azidentity.ClientAssertionCredentialOptions{
				ClientOptions: azcore.ClientOptions{
					Cloud:     cloud.Configuration{ActiveDirectoryAuthorityHost: entra.URL + "/"},
					Transport: entra.Client(),
				},
				DisableInstanceDiscovery: true,
			},
		})
		require.NoError(t, err)

		token, err := cred.GetToken(t.Context(), policy.TokenRequestOptions{
			Scopes: []string{"https://management.azure.com/.default"},
		})
		require.NoError(t, err)
		require.Equal(t, "azure-access-token", token.Token)
		require.Equal(t, int32(1), wiiCalls.Load())
		require.Equal(t, int32(1), entraCalls.Load())
	})
}

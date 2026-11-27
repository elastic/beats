// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package identityfederation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
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

const (
	testAzureTenantID = "00000000-0000-0000-0000-000000000000"
	testAzureClientID = "00000000-0000-0000-0000-000000000001"
)

// fakeEntra is a Microsoft Entra token endpoint that records the client
// assertion of each token request.
type fakeEntra struct {
	srv *httptest.Server

	mu         sync.Mutex
	assertions []string
}

func newFakeEntra(t *testing.T) *fakeEntra {
	t.Helper()
	e := &fakeEntra{}
	tenantPath := "/" + testAzureTenantID
	e.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case tenantPath + "/v2.0/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                 e.srv.URL + tenantPath + "/v2.0",
				"authorization_endpoint": e.srv.URL + tenantPath + "/oauth2/v2.0/authorize",
				"token_endpoint":         e.srv.URL + tenantPath + "/oauth2/v2.0/token",
			})
		case tenantPath + "/oauth2/v2.0/token":
			assert.NoError(t, r.ParseForm())
			assert.Equal(t, testAzureClientID, r.PostForm.Get("client_id"))
			assert.Equal(t, "urn:ietf:params:oauth:client-assertion-type:jwt-bearer", r.PostForm.Get("client_assertion_type"))
			e.mu.Lock()
			e.assertions = append(e.assertions, r.PostForm.Get("client_assertion"))
			n := len(e.assertions)
			e.mu.Unlock()
			// MSAL treats a token that expires within five minutes as expired, so
			// each GetToken goes back to this endpoint.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("azure-access-token-%d", n),
				"token_type":   "Bearer",
				"expires_in":   60,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(e.srv.Close)
	return e
}

func (e *fakeEntra) options() *azidentity.ClientAssertionCredentialOptions {
	return &azidentity.ClientAssertionCredentialOptions{
		Cloud:                    cloud.Configuration{ActiveDirectoryAuthorityHost: e.srv.URL + "/"},
		Transport:                e.srv.Client(),
		DisableInstanceDiscovery: true,
	}
}

func (e *fakeEntra) receivedAssertions() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.assertions...)
}

// newAzureWIITestServer serves a new JWT on each call. Each JWT expires inside
// the token source's refresh margin, so the next call fetches a new one.
func newAzureWIITestServer(t *testing.T, pki wiiTestPKI, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	return newWIITestServer(t, pki, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"aud":"api://AzureADTokenExchange"}`, string(body))
		_, _ = w.Write(tokenResponse(fmt.Sprintf("eyJ.wii-%d.sig", n), time.Now().Add(wiiRefreshBeforeExpiry/2)))
	})
}

func getAzureToken(t *testing.T, cred azcore.TokenCredential) string {
	t.Helper()
	token, err := cred.GetToken(t.Context(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	require.NoError(t, err)
	return token.Token
}

func TestAzureNewWIIClientAssertionCredential(t *testing.T) {
	t.Run("invalid params", func(t *testing.T) {
		_, err := AzureNewWIIClientAssertionCredential(AzureWIIParams{})
		require.ErrorContains(t, err, "invalid Azure identity federation params")
	})

	t.Run("sends the WII token to Entra and refetches it after expiry", func(t *testing.T) {
		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
		var wiiCalls atomic.Int32
		wii := newAzureWIITestServer(t, pki, &wiiCalls)
		entra := newFakeEntra(t)

		source, err := NewWIITokenSource(wii.URL, pki.certPath, pki.keyPath, AzureTokenExchangeAudience, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		cred, err := AzureNewWIIClientAssertionCredential(AzureWIIParams{
			TenantID:    testAzureTenantID,
			ClientID:    testAzureClientID,
			TokenSource: source,
			Options:     entra.options(),
		})
		require.NoError(t, err)

		require.Equal(t, "azure-access-token-1", getAzureToken(t, cred))
		require.Equal(t, "azure-access-token-2", getAzureToken(t, cred))
		require.Equal(t, []string{"eyJ.wii-1.sig", "eyJ.wii-2.sig"}, entra.receivedAssertions())
		require.Equal(t, int32(2), wiiCalls.Load())
	})
}

func TestAzureNewWIIClientAssertionCredentialFromEnv(t *testing.T) {
	t.Run("missing issuer URL", func(t *testing.T) {
		t.Setenv(WIIIssuerURLEnvVar, "")
		t.Setenv(WIISSLCertFileEnvVar, "/cert")
		t.Setenv(WIISSLKeyFileEnvVar, "/key")

		_, err := AzureNewWIIClientAssertionCredentialFromEnv(testAzureTenantID, testAzureClientID, nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "requires the workload-identity-issuer environment set by the agentless controller")
		require.ErrorContains(t, err, WIIIssuerURLEnvVar)
	})

	for _, missing := range []string{WIISSLCertFileEnvVar, WIISSLKeyFileEnvVar} {
		t.Run("missing "+missing, func(t *testing.T) {
			t.Setenv(WIIIssuerURLEnvVar, "https://wii.example.com")
			t.Setenv(WIISSLCertFileEnvVar, "/cert")
			t.Setenv(WIISSLKeyFileEnvVar, "/key")
			t.Setenv(missing, "")

			_, err := AzureNewWIIClientAssertionCredentialFromEnv(testAzureTenantID, testAzureClientID, nil, logptest.NewTestingLogger(t, ""))
			require.ErrorContains(t, err, "must be set when "+WIIIssuerURLEnvVar+" is configured")
		})
	}

	t.Run("all set", func(t *testing.T) {
		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
		var wiiCalls atomic.Int32
		wii := newAzureWIITestServer(t, pki, &wiiCalls)
		entra := newFakeEntra(t)
		t.Setenv(WIIIssuerURLEnvVar, wii.URL)
		t.Setenv(WIISSLCertFileEnvVar, pki.certPath)
		t.Setenv(WIISSLKeyFileEnvVar, pki.keyPath)

		cred, err := AzureNewWIIClientAssertionCredentialFromEnv(testAzureTenantID, testAzureClientID, entra.options(), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		require.Equal(t, "azure-access-token-1", getAzureToken(t, cred))
		require.Equal(t, []string{"eyJ.wii-1.sig"}, entra.receivedAssertions())
	})
}

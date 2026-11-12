// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package identityfederation

import (
	"context"
	"encoding/json"
	"errors"
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

const (
	testTenantID = "00000000-0000-0000-0000-000000000000"
	testClientID = "00000000-0000-0000-0000-000000000001"
	testWIIToken = "eyJhbGciOiJSUzI1NiJ9.wii-test.sig" //nolint:gosec // G101: not a credential, an inert JWT-shaped test fixture
)

// newEntraTestServer serves the Entra discovery and token endpoints for testTenantID
// and records the client assertion of the last token request.
func newEntraTestServer(t *testing.T) (*httptest.Server, *atomic.Value) {
	t.Helper()
	var assertion atomic.Value
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/" + testTenantID + "/v2.0/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                 srv.URL + "/" + testTenantID + "/v2.0",
				"authorization_endpoint": srv.URL + "/" + testTenantID + "/oauth2/v2.0/authorize",
				"token_endpoint":         srv.URL + "/" + testTenantID + "/oauth2/v2.0/token",
			})
		case "/" + testTenantID + "/oauth2/v2.0/token":
			assert.NoError(t, r.ParseForm())
			assert.Equal(t, testClientID, r.PostForm.Get("client_id"))
			assert.Equal(t, "urn:ietf:params:oauth:client-assertion-type:jwt-bearer", r.PostForm.Get("client_assertion_type"))
			assertion.Store(r.PostForm.Get("client_assertion"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "azure-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &assertion
}

// newShortLivedEntraTestServer serves Entra access tokens that expire in 5 seconds,
// inside the refresh windows of both oauth2.ReuseTokenSource and azidentity, so
// every token request reaches the server. It records the client assertion of
// each token request in order.
func newShortLivedEntraTestServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var (
		mu         sync.Mutex
		assertions []string
	)
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/" + testTenantID + "/v2.0/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                 srv.URL + "/" + testTenantID + "/v2.0",
				"authorization_endpoint": srv.URL + "/" + testTenantID + "/oauth2/v2.0/authorize",
				"token_endpoint":         srv.URL + "/" + testTenantID + "/oauth2/v2.0/token",
			})
		case "/" + testTenantID + "/oauth2/v2.0/token":
			assert.NoError(t, r.ParseForm())
			mu.Lock()
			assertions = append(assertions, r.PostForm.Get("client_assertion"))
			n := len(assertions)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("azure-access-token-%d", n),
				"token_type":   "Bearer",
				"expires_in":   5,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), assertions...)
	}
}

func entraTestOptions(srv *httptest.Server) *azidentity.ClientAssertionCredentialOptions {
	return &azidentity.ClientAssertionCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud:     cloud.Configuration{ActiveDirectoryAuthorityHost: srv.URL + "/"},
			Transport: srv.Client(),
		},
		DisableInstanceDiscovery: true,
	}
}

func getTestToken(t *testing.T, cred azcore.TokenCredential) {
	t.Helper()
	token, err := cred.GetToken(t.Context(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	require.NoError(t, err)
	require.Equal(t, "azure-access-token", token.Token)
}

// newAzureWIITestServer starts a workload-identity issuer that returns testWIIToken
// for the Entra audience, and sets WORKLOAD_IDENTITY_SSL_CA_FILE to trust it.
func newAzureWIITestServer(t *testing.T) (*httptest.Server, wiiTestPKI, *atomic.Int32) {
	t.Helper()
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
	var calls atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, "/token", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"aud":"api://AzureADTokenExchange"}`, string(body))
		_, _ = w.Write(tokenResponse(testWIIToken, time.Now().Add(time.Hour)))
	})
	return srv, pki, &calls
}

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

	t.Run("the WII token is sent to Entra as the client assertion", func(t *testing.T) {
		wii, pki, wiiCalls := newAzureWIITestServer(t)
		entra, assertion := newEntraTestServer(t)

		source, err := NewWIITokenSource(wii.URL, pki.certPath, pki.keyPath, AzureWIIAudience, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		cred, err := AzureNewWIIClientAssertionCredential(AzureWIIParams{
			TenantID:    testTenantID,
			ClientID:    testClientID,
			TokenSource: source,
			Options:     entraTestOptions(entra),
		})
		require.NoError(t, err)

		getTestToken(t, cred)
		require.Equal(t, testWIIToken, assertion.Load())
		require.Equal(t, int32(1), wiiCalls.Load())
	})

	t.Run("cancellation reaches the workload-identity issuer", func(t *testing.T) {
		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
		started := make(chan struct{})
		wii := newWIITestServer(t, pki, func(_ http.ResponseWriter, r *http.Request) {
			// The server detects a client disconnect only after the body is consumed.
			_, _ = io.Copy(io.Discard, r.Body)
			close(started)
			<-r.Context().Done()
		})
		entra, _ := newEntraTestServer(t)

		source, err := NewWIITokenSource(wii.URL, pki.certPath, pki.keyPath, AzureWIIAudience, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		cred, err := AzureNewWIIClientAssertionCredential(AzureWIIParams{
			TenantID:    testTenantID,
			ClientID:    testClientID,
			TokenSource: source,
			Options:     entraTestOptions(entra),
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)
		go func() {
			_, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{"https://management.azure.com//.default"},
			})
			result <- err
		}()

		select {
		case <-started:
		case err := <-result:
			t.Fatalf("the Azure token request ended before reaching the WII issuer: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("the Azure token request did not reach the WII issuer")
		}
		cancel()
		select {
		case err := <-result:
			// azidentity flattens the callback error, so errors.Is cannot see context.Canceled.
			require.ErrorContains(t, err, context.Canceled.Error())
		case <-time.After(time.Second):
			t.Fatal("canceling the Azure token request did not interrupt the WII request")
		}
	})
}

func TestAzureNewCloudConnectorCredential(t *testing.T) {
	writeTokenFile := func(t *testing.T) string {
		p := filepath.Join(t.TempDir(), "jwt")
		require.NoError(t, os.WriteFile(p, []byte(fakeJWT), 0o600))
		return p
	}
	clearEnv := func(t *testing.T) {
		t.Setenv(WIIIssuerURLEnvVar, "")
		t.Setenv(WIISSLCertFileEnvVar, "")
		t.Setenv(WIISSLKeyFileEnvVar, "")
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, "")
	}

	t.Run("uses the workload-identity issuer when it is configured", func(t *testing.T) {
		clearEnv(t)
		wii, pki, wiiCalls := newAzureWIITestServer(t)
		entra, assertion := newEntraTestServer(t)
		t.Setenv(WIIIssuerURLEnvVar, wii.URL)
		t.Setenv(WIISSLCertFileEnvVar, pki.certPath)
		t.Setenv(WIISSLKeyFileEnvVar, pki.keyPath)

		cred, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		getTestToken(t, cred)
		require.Equal(t, testWIIToken, assertion.Load())
		require.Equal(t, int32(1), wiiCalls.Load())
	})

	t.Run("prefers the workload-identity issuer over the token file", func(t *testing.T) {
		clearEnv(t)
		wii, pki, _ := newAzureWIITestServer(t)
		entra, assertion := newEntraTestServer(t)
		t.Setenv(WIIIssuerURLEnvVar, wii.URL)
		t.Setenv(WIISSLCertFileEnvVar, pki.certPath)
		t.Setenv(WIISSLKeyFileEnvVar, pki.keyPath)
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, writeTokenFile(t))

		cred, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		getTestToken(t, cred)
		require.Equal(t, testWIIToken, assertion.Load())
	})

	t.Run("falls back to the token file", func(t *testing.T) {
		clearEnv(t)
		entra, assertion := newEntraTestServer(t)
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, writeTokenFile(t))

		cred, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		getTestToken(t, cred)
		require.Equal(t, fakeJWT, assertion.Load())
	})

	t.Run("reports an invalid workload-identity issuer URL", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(WIIIssuerURLEnvVar, "http://issuer.example.com")
		t.Setenv(WIISSLCertFileEnvVar, "/cert")
		t.Setenv(WIISSLKeyFileEnvVar, "/key")

		_, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "configuring WII token source")
	})

	t.Run("requires tenant and client IDs with the workload-identity issuer", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(WIIIssuerURLEnvVar, "https://issuer.example.com")
		t.Setenv(WIISSLCertFileEnvVar, "/cert")
		t.Setenv(WIISSLKeyFileEnvVar, "/key")

		_, err := AzureNewCloudConnectorCredential("", "", nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "TenantID is required")
	})

	t.Run("reports a workload-identity issuer error when a token is requested", func(t *testing.T) {
		clearEnv(t)
		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
		wii := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		entra, assertion := newEntraTestServer(t)
		t.Setenv(WIIIssuerURLEnvVar, wii.URL)
		t.Setenv(WIISSLCertFileEnvVar, pki.certPath)
		t.Setenv(WIISSLKeyFileEnvVar, pki.keyPath)

		cred, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		_, err = cred.GetToken(t.Context(), policy.TokenRequestOptions{
			Scopes: []string{"https://management.azure.com/.default"},
		})
		require.ErrorContains(t, err, "workload-identity issuer returned HTTP 403")
		require.Nil(t, assertion.Load(), "no token request must reach Entra")
	})

	t.Run("requires the WII client certificate", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(WIIIssuerURLEnvVar, "https://issuer.example.com")

		_, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "are required when WORKLOAD_IDENTITY_ISSUER_URL is set")
	})

	t.Run("requires an identity token source", func(t *testing.T) {
		clearEnv(t)

		_, err := AzureNewCloudConnectorCredential(testTenantID, testClientID, nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "no identity token source configured")
	})

	t.Run("requires tenant and client IDs", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, writeTokenFile(t))

		_, err := AzureNewCloudConnectorCredential("", "", nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "TenantID is required")
	})
}

type fakeTokenCredential struct {
	token  azcore.AccessToken
	err    error
	scopes []string
}

func (c *fakeTokenCredential) GetToken(_ context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.scopes = opts.Scopes
	return c.token, c.err
}

func TestAzureTokenSource(t *testing.T) {
	t.Run("maps the Azure access token to an OAuth 2.0 bearer token", func(t *testing.T) {
		expiry := time.Now().Add(time.Hour).Truncate(time.Second)
		cred := &fakeTokenCredential{token: azcore.AccessToken{Token: "azure-access-token", ExpiresOn: expiry}}
		source := &azureTokenSource{ctx: t.Context(), cred: cred, scopes: []string{"https://graph.microsoft.com/.default"}}

		token, err := source.Token()
		require.NoError(t, err)
		assert.Equal(t, "azure-access-token", token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, expiry, token.Expiry)
		assert.Equal(t, []string{"https://graph.microsoft.com/.default"}, cred.scopes)
	})

	t.Run("reports credential errors", func(t *testing.T) {
		cred := &fakeTokenCredential{err: errors.New("entra unavailable")}
		source := &azureTokenSource{ctx: t.Context(), cred: cred, scopes: []string{"scope"}}

		_, err := source.Token()
		require.ErrorContains(t, err, "getting Azure access token: entra unavailable")
	})
}

func TestAzureNewCloudConnectorTokenSource(t *testing.T) {
	t.Run("serves access tokens from Entra", func(t *testing.T) {
		t.Setenv(WIIIssuerURLEnvVar, "")
		p := filepath.Join(t.TempDir(), "jwt")
		require.NoError(t, os.WriteFile(p, []byte(fakeJWT), 0o600))
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, p)
		entra, assertion := newEntraTestServer(t)

		source, err := AzureNewCloudConnectorTokenSource(t.Context(), testTenantID, testClientID,
			[]string{"https://management.azure.com/.default"}, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		token, err := source.Token()
		require.NoError(t, err)
		assert.Equal(t, "azure-access-token", token.AccessToken)
		assert.True(t, token.Valid(), "expected a token that has not expired")
		assert.Equal(t, fakeJWT, assertion.Load())
	})

	t.Run("sends the rotated token file on refresh", func(t *testing.T) {
		const rotatedJWT = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJyb3RhdGVkIn0.signature"
		t.Setenv(WIIIssuerURLEnvVar, "")
		p := filepath.Join(t.TempDir(), "jwt")
		require.NoError(t, os.WriteFile(p, []byte(fakeJWT), 0o600))
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, p)
		entra, assertions := newShortLivedEntraTestServer(t)

		source, err := AzureNewCloudConnectorTokenSource(t.Context(), testTenantID, testClientID,
			[]string{"https://management.azure.com//.default"}, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		first, err := source.Token()
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(p, []byte(rotatedJWT), 0o600))
		second, err := source.Token()
		require.NoError(t, err)

		assert.Equal(t, []string{fakeJWT, rotatedJWT}, assertions())
		assert.NotEqual(t, first.AccessToken, second.AccessToken)
	})

	t.Run("sends the rotated workload-identity token on refresh", func(t *testing.T) {
		wiiTokens := []string{
			"eyJhbGciOiJSUzI1NiJ9.wii-first.sig",
			"eyJhbGciOiJSUzI1NiJ9.wii-second.sig",
		}
		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
		var wiiCalls atomic.Int32
		wii := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
			n := int(wiiCalls.Add(1))
			// An expiry inside the WII refresh window forces a new issuer request each time.
			_, _ = w.Write(tokenResponse(wiiTokens[min(n, len(wiiTokens))-1], time.Now().Add(30*time.Second)))
		})
		t.Setenv(WIIIssuerURLEnvVar, wii.URL)
		t.Setenv(WIISSLCertFileEnvVar, pki.certPath)
		t.Setenv(WIISSLKeyFileEnvVar, pki.keyPath)
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, "")
		entra, assertions := newShortLivedEntraTestServer(t)

		source, err := AzureNewCloudConnectorTokenSource(t.Context(), testTenantID, testClientID,
			[]string{"https://management.azure.com//.default"}, entraTestOptions(entra), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		first, err := source.Token()
		require.NoError(t, err)
		second, err := source.Token()
		require.NoError(t, err)

		assert.Equal(t, wiiTokens, assertions())
		assert.Equal(t, int32(2), wiiCalls.Load())
		assert.NotEqual(t, first.AccessToken, second.AccessToken)
	})

	t.Run("requires a scope", func(t *testing.T) {
		_, err := AzureNewCloudConnectorTokenSource(t.Context(), testTenantID, testClientID, nil, nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "at least one scope is required")
	})

	t.Run("reports credential errors", func(t *testing.T) {
		t.Setenv(WIIIssuerURLEnvVar, "")
		t.Setenv(CloudConnectorsIDTokenFileEnvVar, "")

		_, err := AzureNewCloudConnectorTokenSource(t.Context(), testTenantID, testClientID, []string{"scope"}, nil, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "no identity token source configured")
	})
}

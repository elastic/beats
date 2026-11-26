// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestAzureNewWIIOAuth2TokenSourceFromEnv(t *testing.T) {
	t.Run("missing WII environment", func(t *testing.T) {
		t.Setenv(WIIIssuerURLEnvVar, "")

		_, err := AzureNewWIIOAuth2TokenSourceFromEnv(t.Context(), "https://login.example.com/token", "client", nil, nil, logptest.NewTestingLogger(t, ""))
		assert.ErrorContains(t, err, WIIIssuerURLEnvVar, "the error must name the missing WII variable")
	})

	t.Run("each Entra token request sends a new WII token", func(t *testing.T) {
		pki := newWIITestPKI(t)
		t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)
		var wiiCalls atomic.Int32
		wii := newWIITestServer(t, pki, func(w http.ResponseWriter, r *http.Request) {
			n := wiiCalls.Add(1)
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the WII request body")
			assert.JSONEq(t, `{"aud":"api://AzureADTokenExchange"}`, string(body), "WII must get the Entra audience")
			_, _ = w.Write(tokenResponse(fmt.Sprintf("eyJ.wii-%d.sig", n), time.Now().Add(wiiRefreshBeforeExpiry/2)))
		})
		t.Setenv(WIIIssuerURLEnvVar, wii.URL)
		t.Setenv(WIISSLCertFileEnvVar, pki.certPath)
		t.Setenv(WIISSLKeyFileEnvVar, pki.keyPath)

		var mu sync.Mutex
		var forms []url.Values
		entra := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.NoError(t, r.ParseForm(), "parsing the Entra token request")
			mu.Lock()
			forms = append(forms, r.PostForm)
			n := len(forms)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			// An expiry inside the oauth2 expiry delta makes each Token call refresh.
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": fmt.Sprintf("entra-%d", n), "token_type": "Bearer", "expires_in": 1})
		}))
		t.Cleanup(entra.Close)

		source, err := AzureNewWIIOAuth2TokenSourceFromEnv(t.Context(), entra.URL, "client-id", []string{"https://management.azure.com/.default"}, url.Values{"extra": {"kept"}}, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err, "the token source must be created when the WII environment is set")

		first, err := source.Token()
		require.NoError(t, err, "the first token request must succeed")
		second, err := source.Token()
		require.NoError(t, err, "the refresh must succeed")

		assert.Equal(t, "entra-1", first.AccessToken, "the first Entra token must be returned")
		assert.Equal(t, "entra-2", second.AccessToken, "the refresh must return a new Entra token")
		require.Len(t, forms, 2, "each Token call must send one Entra request")
		for i, form := range forms {
			assert.Equal(t, fmt.Sprintf("eyJ.wii-%d.sig", i+1), form.Get("client_assertion"), "each Entra request must carry a new WII token")
			assert.Equal(t, azureClientAssertionType, form.Get("client_assertion_type"), "the assertion type must be jwt-bearer")
			assert.Equal(t, "client_credentials", form.Get("grant_type"), "the grant must be client_credentials")
			assert.Equal(t, "client-id", form.Get("client_id"), "the client ID must be in the form")
			assert.Equal(t, "https://management.azure.com/.default", form.Get("scope"), "the scope must be in the form")
			assert.Equal(t, "kept", form.Get("extra"), "extra endpoint params must be kept")
			assert.False(t, form.Has("client_secret"), "no client secret must be sent")
		}
	})
}

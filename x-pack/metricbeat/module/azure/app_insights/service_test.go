// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// mockTokenCredential implements azcore.TokenCredential for testing.
type mockTokenCredential struct {
	token string
	err   error
}

func (m *mockTokenCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if m.err != nil {
		return azcore.AccessToken{}, m.err
	}
	return azcore.AccessToken{Token: m.token}, nil
}

func TestGetAuthorizer(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("returns API key authorizer when only api_key is set", func(t *testing.T) {
		cfg := Config{
			ApplicationId: "app-id",
			ApiKey:        "my-api-key",
		}

		auth, err := getAuthorizer(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, auth)

		_, isTokenAuth := auth.(*tokenCredentialAuthorizer)
		assert.False(t, isTokenAuth, "expected API key authorizer, got tokenCredentialAuthorizer")
	})

	t.Run("returns OAuth2 authorizer when OAuth2 credentials are set", func(t *testing.T) {
		cfg := Config{
			ApplicationId: "app-id",
			TenantId:      "tenant-id",
			ClientId:      "client-id",
			ClientSecret:  "client-secret",
		}

		auth, err := getAuthorizer(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, auth)

		tokenAuth, isTokenAuth := auth.(*tokenCredentialAuthorizer)
		assert.True(t, isTokenAuth, "expected tokenCredentialAuthorizer")
		assert.Equal(t, []string{appInsightsScope}, tokenAuth.scopes)
	})

	t.Run("OAuth2 takes priority when both are set", func(t *testing.T) {
		cfg := Config{
			ApplicationId: "app-id",
			ApiKey:        "my-api-key",
			TenantId:      "tenant-id",
			ClientId:      "client-id",
			ClientSecret:  "client-secret",
		}

		auth, err := getAuthorizer(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, auth)

		_, isTokenAuth := auth.(*tokenCredentialAuthorizer)
		assert.True(t, isTokenAuth, "expected OAuth2 authorizer to take priority")
	})
}

func TestNewOAuth2Authorizer(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("returns tokenCredentialAuthorizer with correct scopes", func(t *testing.T) {
		cfg := Config{
			TenantId:     "tenant-id",
			ClientId:     "client-id",
			ClientSecret: "client-secret",
		}

		auth, err := newOAuth2Authorizer(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, auth)

		tokenAuth, ok := auth.(*tokenCredentialAuthorizer)
		require.True(t, ok, "expected *tokenCredentialAuthorizer")
		assert.Equal(t, []string{appInsightsScope}, tokenAuth.scopes)
		assert.NotNil(t, tokenAuth.credential)
	})

	t.Run("accepts custom active_directory_endpoint", func(t *testing.T) {
		cfg := Config{
			TenantId:                "tenant-id",
			ClientId:                "client-id",
			ClientSecret:            "client-secret",
			ActiveDirectoryEndpoint: "https://login.microsoftonline.us/",
		}

		auth, err := newOAuth2Authorizer(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, auth)

		tokenAuth, ok := auth.(*tokenCredentialAuthorizer)
		require.True(t, ok, "expected *tokenCredentialAuthorizer")
		assert.NotNil(t, tokenAuth.credential)
	})
}

func TestTokenCredentialAuthorizer_WithAuthorization(t *testing.T) {
	t.Run("sets Authorization header with bearer token", func(t *testing.T) {
		auth := &tokenCredentialAuthorizer{
			credential: &mockTokenCredential{token: "test-token-123"},
			scopes:     []string{appInsightsScope},
		}

		req := httptest.NewRequest(http.MethodGet, "https://api.applicationinsights.io/v1/apps/test", nil)

		decorator := auth.WithAuthorization()
		preparer := decorator(autorest.CreatePreparer())
		result, err := preparer.Prepare(req)

		require.NoError(t, err)
		assert.Equal(t, "Bearer test-token-123", result.Header.Get("Authorization"))
	})

	t.Run("propagates error from credential GetToken", func(t *testing.T) {
		tokenErr := errors.New("token acquisition failed")
		auth := &tokenCredentialAuthorizer{
			credential: &mockTokenCredential{err: tokenErr},
			scopes:     []string{appInsightsScope},
		}

		req := httptest.NewRequest(http.MethodGet, "https://api.applicationinsights.io/v1/apps/test", nil)

		decorator := auth.WithAuthorization()
		preparer := decorator(autorest.CreatePreparer())
		_, err := preparer.Prepare(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get token")
		assert.ErrorIs(t, err, tokenErr)
	})

	t.Run("propagates error from previous preparer", func(t *testing.T) {
		auth := &tokenCredentialAuthorizer{
			credential: &mockTokenCredential{token: "test-token"},
			scopes:     []string{appInsightsScope},
		}

		req := httptest.NewRequest(http.MethodGet, "https://api.applicationinsights.io/v1/apps/test", nil)

		prevErr := errors.New("previous preparer failed")
		failingPreparer := autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			return r, prevErr
		})

		decorator := auth.WithAuthorization()
		preparer := decorator(failingPreparer)
		_, err := preparer.Prepare(req)

		require.Error(t, err)
		assert.ErrorIs(t, err, prevErr)
		assert.Empty(t, req.Header.Get("Authorization"), "Authorization header should not be set when previous preparer fails")
	})

	t.Run("preserves existing headers on the request", func(t *testing.T) {
		auth := &tokenCredentialAuthorizer{
			credential: &mockTokenCredential{token: "test-token"},
			scopes:     []string{appInsightsScope},
		}

		req := httptest.NewRequest(http.MethodGet, "https://api.applicationinsights.io/v1/apps/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Custom-Header", "custom-value")

		decorator := auth.WithAuthorization()
		preparer := decorator(autorest.CreatePreparer())
		result, err := preparer.Prepare(req)

		require.NoError(t, err)
		assert.Equal(t, "Bearer test-token", result.Header.Get("Authorization"))
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))
		assert.Equal(t, "custom-value", result.Header.Get("X-Custom-Header"))
	})
}

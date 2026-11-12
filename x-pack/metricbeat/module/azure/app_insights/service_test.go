// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

func TestAPIKeyPolicy_SetsHeader(t *testing.T) {
	var gotAPIKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	pipeline := runtime.NewPipeline(moduleName, moduleVersion, runtime.PipelineOptions{
		PerCall: []policy.Policy{&apiKeyPolicy{apiKey: "key-123"}},
	}, &policy.ClientOptions{})

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, srv.URL)
	require.NoError(t, err, "creating azcore request should not fail")
	resp, err := pipeline.Do(req)
	require.NoError(t, err, "pipeline call should succeed")
	defer resp.Body.Close()

	assert.Equal(t, "key-123", gotAPIKey, "apiKeyPolicy must set the x-api-key header")
}

func TestNewMetricsClient_APIKey(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	cfg := Config{
		ApplicationId: "app-id",
		AuthType:      AuthTypeAPIKey,
		ApiKey:        "test-api-key",
	}

	client, err := newMetricsClient(cfg, logger)
	require.NoError(t, err, "API key client construction should succeed")
	require.NotNil(t, client, "metricsClient must not be nil")
	assert.Equal(t, appInsightsEndpoint, client.endpoint, "default endpoint should be the public App Insights API")
}

func TestNewMetricsClient_ClientSecret(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	cfg := Config{
		ApplicationId: "app-id",
		AuthType:      AuthTypeClientSecret,
		// azidentity validates tenantId is a UUID, so use a syntactically valid one.
		TenantId:     "00000000-0000-0000-0000-000000000000",
		ClientId:     "00000000-0000-0000-0000-000000000001",
		ClientSecret: "client-secret",
	}

	client, err := newMetricsClient(cfg, logger)
	require.NoError(t, err, "client secret client construction should succeed for valid inputs")
	require.NotNil(t, client, "metricsClient must not be nil")

	cfg.TenantId = "invalid tenant"
	_, err = newMetricsClient(cfg, logger)
	require.ErrorContains(t, err, "failed to create client secret credential", "an invalid tenant ID should be reported")
}

func TestNewMetricsClient_CloudConnectors(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	cfg := Config{
		ApplicationId:      "app-id",
		TenantId:           "00000000-0000-0000-0000-000000000000",
		ClientId:           "00000000-0000-0000-0000-000000000001",
		UseCloudConnectors: true,
	}
	require.NoError(t, cfg.Validate(), "cloud connectors config should be valid")
	assert.Equal(t, AuthTypeClientSecret, cfg.AuthType, "cloud connectors should default auth_type to client_secret")

	t.Run("creates the client", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "id-token")
		require.NoError(t, os.WriteFile(p, []byte("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature"), 0o600), "writing the token file should not fail")
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
		t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, p)

		client, err := newMetricsClient(cfg, logger)
		require.NoError(t, err, "identity federation client construction should succeed")
		require.NotNil(t, client, "metricsClient must not be nil")
	})

	t.Run("fails without an identity token source", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
		t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, "")

		_, err := newMetricsClient(cfg, logger)
		require.ErrorContains(t, err, "failed to create identity federation credential", "the identity federation error should be reported")
		require.ErrorContains(t, err, "no identity token source configured", "the cause should be reported")
	})
}

func TestMetricsClient_GetMultiple(t *testing.T) {
	const appID = "00000000-0000-0000-0000-000000000000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "metrics batch must be POSTed")
		assert.Equal(t, "/v1/apps/"+appID+"/metrics", r.URL.Path,
			"application id must appear in the URL path")
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"), "x-api-key header must be propagated by the API key policy")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "reading request body should not fail")
		var sent []map[string]any
		require.NoError(t, json.Unmarshal(body, &sent), "request body must be a JSON array")
		require.Len(t, sent, 1, "test sends one metric in the batch")
		require.Contains(t, sent[0], "parameters", "each batch entry must carry parameters")

		_, _ = w.Write([]byte(`[
			{
				"id": "abc",
				"status": 200,
				"body": {
					"value": {
						"start": "2024-01-01T00:00:00Z",
						"end":   "2024-01-01T00:05:00Z",
						"interval": "PT5M",
						"requests/count": {"sum": 42}
					}
				}
			}
		]`))
	}))
	defer srv.Close()

	pipeline := runtime.NewPipeline(moduleName, moduleVersion, runtime.PipelineOptions{
		PerCall: []policy.Policy{&apiKeyPolicy{apiKey: "test-api-key"}},
	}, &policy.ClientOptions{})

	c := &metricsClient{endpoint: srv.URL, pipeline: pipeline}

	id := "abc"
	metricID := "requests/count"
	timespan := "PT5M"
	body := []MetricsBatchRequestItem{{
		ID: &id,
		Parameters: &MetricsBatchParameters{
			MetricID: metricID,
			Timespan: &timespan,
		},
	}}

	got, err := c.GetMultiple(context.Background(), appID, body)
	require.NoError(t, err, "GetMultiple should succeed against fake server")
	require.NotNil(t, got.Value, "response Value pointer must be set")
	require.Len(t, *got.Value, 1, "fake server returned exactly one item")

	item := (*got.Value)[0]
	require.NotNil(t, item.Body, "response body should be decoded")
	require.NotNil(t, item.Body.Value, "metrics result info should be decoded")
	require.NotNil(t, item.Body.Value.Start, "start time should be decoded")
	require.NotNil(t, item.Body.Value.End, "end time should be decoded")
	assert.Equal(t, "PT5M", *item.Body.Value.Interval, "interval should round-trip")
	assert.Contains(t, item.Body.Value.AdditionalProperties, "requests/count",
		"unknown metric fields must end up in AdditionalProperties")
}

func TestMetricsClient_GetMultiple_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"BadRequest","message":"bad query"}}`))
	}))
	defer srv.Close()

	pipeline := runtime.NewPipeline(moduleName, moduleVersion, runtime.PipelineOptions{
		PerCall: []policy.Policy{&apiKeyPolicy{apiKey: "k"}},
	}, &policy.ClientOptions{})
	c := &metricsClient{endpoint: srv.URL, pipeline: pipeline}

	_, err := c.GetMultiple(context.Background(), "app", nil)
	require.Error(t, err, "non-200 responses must surface as an error")
	// azcore's runtime.NewResponseError embeds the upstream error code into
	// the error message; matching on either the code or the HTTP status keeps
	// the assertion robust against minor format changes across azcore versions.
	errMsg := err.Error()
	assert.True(t,
		strings.Contains(errMsg, "BadRequest") || strings.Contains(errMsg, "400"),
		"error should reference the upstream failure (got: %v)", err)
}

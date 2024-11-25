// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package usage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	apiKey := time.Now().String() // to generate a unique API key everytime
	usagePath := "/usage"
	server := initServer(usagePath, apiKey)
	defer server.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL+"/usage", apiKey))

	events, errs := mbtest.ReportingFetchV2Error(f)
	require.Empty(t, errs, "Expected no errors")
	require.NotEmpty(t, events, "Expected events to be returned")
}

func TestData(t *testing.T) {
	apiKey := time.Now().String() // to generate a unique API key everytime
	usagePath := "/usage"
	server := initServer(usagePath, apiKey)
	defer server.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL+"/usage", apiKey))

	err := mbtest.WriteEventsReporterV2Error(f, t, "")
	require.NoError(t, err, "Writing events should not return an error")
}

func getConfig(url, apiKey string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "openai",
		"metricsets": []string{"usage"},
		"enabled":    true,
		"period":     "1h",
		"api_url":    url,
		"api_keys": []map[string]interface{}{
			{"key": apiKey},
		},
		"rate_limit": map[string]interface{}{
			"limit": 60,
			"burst": 5,
		},
		"collection": map[string]interface{}{
			"lookback_days": 1,
		},
	}
}

func initServer(endpoint string, api_key string) *httptest.Server {
	data := []byte(`{
  "object": "list",
  "data": [
    {
      "organization_id": "org-dummy",
      "organization_name": "Personal",
      "aggregation_timestamp": 1730696460,
      "n_requests": 1,
      "operation": "completion-realtime",
      "snapshot_id": "gpt-4o-realtime-preview-2024-10-01",
      "n_context_tokens_total": 118,
      "n_generated_tokens_total": 35,
      "email": null,
      "api_key_id": null,
      "api_key_name": null,
      "api_key_redacted": null,
      "api_key_type": null,
      "project_id": null,
      "project_name": null,
      "request_type": "",
      "n_cached_context_tokens_total": 0
    },
    {
      "organization_id": "org-dummy",
      "organization_name": "Personal",
      "aggregation_timestamp": 1730696460,
      "n_requests": 1,
      "operation": "completion",
      "snapshot_id": "gpt-4o-2024-08-06",
      "n_context_tokens_total": 31,
      "n_generated_tokens_total": 12,
      "email": null,
      "api_key_id": null,
      "api_key_name": null,
      "api_key_redacted": null,
      "api_key_type": null,
      "project_id": null,
      "project_name": null,
      "request_type": "",
      "n_cached_context_tokens_total": 0
    },
    {
      "organization_id": "org-dummy",
      "organization_name": "Personal",
      "aggregation_timestamp": 1730697540,
      "n_requests": 1,
      "operation": "completion",
      "snapshot_id": "ft:gpt-3.5-turbo-0125:personal:yay-renew:APjjyG8E:ckpt-step-84",
      "n_context_tokens_total": 13,
      "n_generated_tokens_total": 9,
      "email": null,
      "api_key_id": null,
      "api_key_name": null,
      "api_key_redacted": null,
      "api_key_type": null,
      "project_id": null,
      "project_name": null,
      "request_type": "",
      "n_cached_context_tokens_total": 0
    }
  ],
  "ft_data": [],
  "dalle_api_data": [
    {
      "timestamp": 1730696460,
      "num_images": 1,
      "num_requests": 1,
      "image_size": "1024x1024",
      "operation": "generations",
      "user_id": "subham.sarkar@elastic.co",
      "organization_id": "org-FCp10pUDIN4slA4kNZK6UKkX",
      "api_key_id": "key_10xSzP3zPsz8zB5O",
      "api_key_name": "project_key",
      "api_key_redacted": "sk-...zkA",
      "api_key_type": "organization",
      "organization_name": "Personal",
      "model_id": "dall-e-3",
      "project_id": "Default Project",
      "project_name": "Default Project"
    }
  ],
  "whisper_api_data": [
    {
      "timestamp": 1730696460,
      "model_id": "whisper-1",
      "num_seconds": 2,
      "num_requests": 1,
      "user_id": null,
      "organization_id": "org-dummy",
      "api_key_id": null,
      "api_key_name": null,
      "api_key_redacted": null,
      "api_key_type": null,
      "organization_name": "Personal",
      "project_id": null,
      "project_name": null
    }
  ],
  "tts_api_data": [],
  "assistant_code_interpreter_data": [],
  "retrieval_storage_data": []
}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Bearer token
		authHeader := r.Header.Get("Authorization")
		expectedToken := fmt.Sprintf("Bearer %s", api_key)

		if authHeader != expectedToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// If it doesn't match the expected endpoint, return 404
		if r.URL.Path != endpoint {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	return server
}

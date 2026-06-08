// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var (
	config = Config{
		ApplicationId: "",
		ApiKey:        "test-api-key",
		Metrics: []Metric{
			{
				ID: []string{"requests/count"},
			},
		},
	}
)

func TestClient(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	t.Run("return error not valid query", func(t *testing.T) {
		client := NewMockClient(logger)
		client.Config = config
		m := &MockService{}
		m.On("GetMetricValues", mock.Anything, mock.Anything).Return(insights.ListMetricsResultsItem{}, errors.New("invalid query"))
		client.Service = m
		results, err := client.GetMetricValues()
		assert.Error(t, err)
		assert.Nil(t, results.Value)
		m.AssertExpectations(t)
	})
	t.Run("return results", func(t *testing.T) {
		client := NewMockClient(logger)
		client.Config = config
		m := &MockService{}
		metrics := []insights.MetricsResultsItem{{}, {}}
		m.On("GetMetricValues", mock.Anything, mock.Anything).Return(insights.ListMetricsResultsItem{Value: &metrics}, nil)
		client.Service = m
		results, err := client.GetMetricValues()
		assert.NoError(t, err)
		assert.Len(t, *results.Value, 2)
		m.AssertExpectations(t)
	})
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid config with API key (explicit auth_type)",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "api_key",
				ApiKey:        "test-api-key",
			},
			wantErr: "",
		},
		{
			name: "valid config with API key (default auth_type)",
			config: Config{
				ApplicationId: "app-id",
				ApiKey:        "test-api-key",
			},
			wantErr: "",
		},
		{
			name: "valid config with client secret",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "client_secret",
				TenantId:      "tenant-id",
				ClientId:      "client-id",
				ClientSecret:  "client-secret",
			},
			wantErr: "",
		},
		{
			name: "invalid config - api_key auth_type without api_key",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "api_key",
			},
			wantErr: "api_key is required when auth_type is api_key",
		},
		{
			name: "invalid config - default auth_type without api_key",
			config: Config{
				ApplicationId: "app-id",
			},
			wantErr: "api_key is required when auth_type is api_key",
		},
		{
			name: "invalid config - client_secret missing tenant_id",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "client_secret",
				ClientId:      "client-id",
				ClientSecret:  "client-secret",
			},
			wantErr: "tenant_id is required when auth_type is client_secret",
		},
		{
			name: "invalid config - client_secret missing client_id",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "client_secret",
				TenantId:      "tenant-id",
				ClientSecret:  "client-secret",
			},
			wantErr: "client_id is required when auth_type is client_secret",
		},
		{
			name: "invalid config - client_secret missing client_secret",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "client_secret",
				TenantId:      "tenant-id",
				ClientId:      "client-id",
			},
			wantErr: "client_secret is required when auth_type is client_secret",
		},
		{
			name: "invalid config - unknown auth_type",
			config: Config{
				ApplicationId: "app-id",
				AuthType:      "invalid",
			},
			wantErr: "unknown auth_type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

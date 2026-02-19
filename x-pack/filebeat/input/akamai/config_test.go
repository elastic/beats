// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfigValidation(t *testing.T) {
	edgeGridAuth := map[string]interface{}{
		"edgegrid": map[string]interface{}{
			"client_token":  "test-token",
			"client_secret": "test-secret",
			"access_token":  "test-access",
		},
	}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr string
	}{
		{
			name:    "missing resource.url",
			config:  map[string]interface{}{},
			wantErr: "missing required field accessing 'resource'",
		},
		{
			name: "missing config_ids",
			config: map[string]interface{}{
				"resource": map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
			},
			wantErr: "string value is not set accessing 'config_ids'",
		},
		{
			name: "missing auth credentials",
			config: map[string]interface{}{
				"resource":   map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids": "12345",
			},
			wantErr: "at least one auth method must be configured",
		},
		{
			name: "partial auth credentials",
			config: map[string]interface{}{
				"resource":   map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids": "12345",
				"auth": map[string]interface{}{
					"edgegrid": map[string]interface{}{
						"client_token": "test-token",
					},
				},
			},
			wantErr: "auth.edgegrid.client_secret",
		},
		{
			name: "valid minimal config",
			config: map[string]interface{}{
				"resource":   map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids": "12345",
				"auth":       edgeGridAuth,
			},
			wantErr: "",
		},
		{
			name: "invalid event_limit too high",
			config: map[string]interface{}{
				"resource":    map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids":  "12345",
				"auth":        edgeGridAuth,
				"event_limit": 700000,
			},
			wantErr: "event_limit cannot exceed 600000",
		},
		{
			name: "invalid initial_interval too high",
			config: map[string]interface{}{
				"resource":         map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids":       "12345",
				"auth":             edgeGridAuth,
				"initial_interval": "24h",
			},
			wantErr: "initial_interval cannot exceed 12h",
		},
		{
			name: "invalid number_of_workers",
			config: map[string]interface{}{
				"resource":          map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids":        "12345",
				"auth":              edgeGridAuth,
				"number_of_workers": 0,
			},
			wantErr: "number_of_workers must be greater than 0",
		},
		{
			name: "negative offset_ttl",
			config: map[string]interface{}{
				"resource":   map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids": "12345",
				"auth":       edgeGridAuth,
				"offset_ttl": "-1s",
			},
			wantErr: "offset_ttl must be non-negative",
		},
		{
			name: "zero offset_ttl disables proactive check",
			config: map[string]interface{}{
				"resource":   map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids": "12345",
				"auth":       edgeGridAuth,
				"offset_ttl": "0s",
			},
			wantErr: "",
		},
		{
			name: "explicit channel_buffer_size",
			config: map[string]interface{}{
				"resource":            map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids":          "12345",
				"auth":                edgeGridAuth,
				"channel_buffer_size": 256,
			},
			wantErr: "",
		},
		{
			name: "negative channel_buffer_size",
			config: map[string]interface{}{
				"resource":            map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids":          "12345",
				"auth":                edgeGridAuth,
				"channel_buffer_size": -1,
			},
			wantErr: "channel_buffer_size must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(tt.config)
			require.NoError(t, err)

			config := defaultConfig()
			err = cfg.Unpack(&config)
			if err == nil {
				err = config.Validate()
			}

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	assert.Equal(t, defaultInterval, cfg.Interval)
	assert.Equal(t, defaultInitialInterval, cfg.InitialInterval)
	assert.Equal(t, defaultEventLimit, cfg.EventLimit)
	assert.Equal(t, defaultNumberOfWorkers, cfg.NumberOfWorkers)
	assert.Equal(t, defaultInvalidTSRetries, cfg.InvalidTimestampRetries)
	assert.Equal(t, defaultMaxRecoveryAttempts, cfg.MaxRecoveryAttempts)
	assert.Equal(t, defaultOffsetTTL, cfg.OffsetTTL)
	assert.NotNil(t, cfg.Resource)
	assert.NotNil(t, cfg.Resource.Retry.MaxAttempts)
	assert.Equal(t, defaultMaxAttempts, *cfg.Resource.Retry.MaxAttempts)
}

func TestDefaultChannelBufferSize(t *testing.T) {
	cfg := defaultConfig()
	u, _ := url.Parse("https://test.luna.akamaiapis.net")
	cfg.Resource.URL = &urlConfig{URL: u}
	cfg.ConfigIDs = "1"
	cfg.Auth.EdgeGrid = &edgeGridConfig{
		ClientToken: "t", ClientSecret: "s", AccessToken: "a",
	}

	err := cfg.Validate()
	require.NoError(t, err)
	assert.Equal(t, cfg.EventLimit/2, cfg.ChannelBufferSize)
}

func TestEdgeGridSigner(t *testing.T) {
	signer := NewEdgeGridSigner("client-token", "client-secret", "access-token")
	t.Run("createSigningKey", func(t *testing.T) {
		key := signer.createSigningKey("20240101T12:00:00+0000")
		assert.NotEmpty(t, key)
	})

	t.Run("Sign produces valid authorization header", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://test.luna.akamaiapis.net/siem/v1/configs/1?limit=100", nil)
		require.NoError(t, err)

		err = signer.Sign(req)
		require.NoError(t, err)

		authHeader := req.Header.Get("Authorization")
		assert.Contains(t, authHeader, "EG1-HMAC-SHA256")
		assert.Contains(t, authHeader, "client_token=client-token")
		assert.Contains(t, authHeader, "access_token=access-token")
		assert.Contains(t, authHeader, "signature=")
		assert.Contains(t, authHeader, "nonce=")
		assert.Contains(t, authHeader, "timestamp=")
	})
}

func TestMaxRecoveryAttemptsValidation(t *testing.T) {
	edgeGridAuth := map[string]interface{}{
		"edgegrid": map[string]interface{}{
			"client_token":  "test-token",
			"client_secret": "test-secret",
			"access_token":  "test-access",
		},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr string
	}{
		{
			name:    "negative value",
			value:   -1,
			wantErr: "max_recovery_attempts must be non-negative",
		},
		{
			name:    "zero disables cap",
			value:   0,
			wantErr: "",
		},
		{
			name:    "positive value",
			value:   5,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgMap := map[string]interface{}{
				"resource":              map[string]interface{}{"url": "https://test.luna.akamaiapis.net"},
				"config_ids":            "12345",
				"auth":                  edgeGridAuth,
				"max_recovery_attempts": tt.value,
			}
			cfg, err := conf.NewConfigFrom(cfgMap)
			require.NoError(t, err)

			config := defaultConfig()
			err = cfg.Unpack(&config)
			if err == nil {
				err = config.Validate()
			}

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRateLimitConfigValidation(t *testing.T) {
	edgeGridAuth := map[string]interface{}{
		"edgegrid": map[string]interface{}{
			"client_token":  "test-token",
			"client_secret": "test-secret",
			"access_token":  "test-access",
		},
	}

	tests := []struct {
		name      string
		rateLimit map[string]interface{}
		wantErr   string
	}{
		{
			name:      "limit set without burst",
			rateLimit: map[string]interface{}{"limit": 5.0},
			wantErr:   "",
		},
		{
			name:      "both set",
			rateLimit: map[string]interface{}{"limit": 5.0, "burst": 10},
			wantErr:   "",
		},
		{
			name:      "negative limit",
			rateLimit: map[string]interface{}{"limit": -1.0},
			wantErr:   "rate_limit.limit must be greater than zero",
		},
		{
			name:      "negative burst",
			rateLimit: map[string]interface{}{"limit": 5.0, "burst": -1},
			wantErr:   "rate_limit.burst must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgMap := map[string]interface{}{
				"resource": map[string]interface{}{
					"url":        "https://test.luna.akamaiapis.net",
					"rate_limit": tt.rateLimit,
				},
				"config_ids": "12345",
				"auth":       edgeGridAuth,
			}
			cfg, err := conf.NewConfigFrom(cfgMap)
			require.NoError(t, err)

			config := defaultConfig()
			err = cfg.Unpack(&config)
			if err == nil {
				err = config.Validate()
			}

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

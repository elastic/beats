// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
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
	assert.Equal(t, defaultRecoveryInterval, cfg.RecoveryInterval)
	assert.Equal(t, defaultEventLimit, cfg.EventLimit)
	assert.Equal(t, defaultNumberOfWorkers, cfg.NumberOfWorkers)
	assert.Equal(t, defaultInvalidTSRetries, cfg.InvalidTimestampRetries)
	assert.NotNil(t, cfg.Resource)
	assert.NotNil(t, cfg.Resource.Retry.MaxAttempts)
	assert.Equal(t, defaultMaxAttempts, *cfg.Resource.Retry.MaxAttempts)
}

func TestEdgeGridSigner(t *testing.T) {
	signer := NewEdgeGridSigner("client-token", "client-secret", "access-token")
	timestamp := "20240101T12:00:00+0000"
	key := signer.createSigningKey(timestamp)
	assert.NotEmpty(t, key)
}

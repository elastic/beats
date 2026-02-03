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
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestPlugin(t *testing.T) {
	p := Plugin(logp.NewLogger("akamai_test"), nil)
	assert.Equal(t, inputName, p.Name)
	assert.NotNil(t, p.Manager)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr string
	}{
		{
			name:    "missing api_host",
			config:  map[string]interface{}{},
			wantErr: "api_host is required",
		},
		{
			name: "missing config_ids",
			config: map[string]interface{}{
				"api_host": "https://test.luna.akamaiapis.net",
			},
			wantErr: "config_ids is required",
		},
		{
			name: "missing auth credentials",
			config: map[string]interface{}{
				"api_host":   "https://test.luna.akamaiapis.net",
				"config_ids": "12345",
			},
			wantErr: "authentication credentials are required",
		},
		{
			name: "partial auth credentials",
			config: map[string]interface{}{
				"api_host":     "https://test.luna.akamaiapis.net",
				"config_ids":   "12345",
				"client_token": "test-token",
			},
			wantErr: "all of client_token, client_secret, and access_token are required",
		},
		{
			name: "valid minimal config",
			config: map[string]interface{}{
				"api_host":      "https://test.luna.akamaiapis.net",
				"config_ids":    "12345",
				"client_token":  "test-token",
				"client_secret": "test-secret",
				"access_token":  "test-access",
			},
			wantErr: "",
		},
		{
			name: "invalid event_limit too high",
			config: map[string]interface{}{
				"api_host":      "https://test.luna.akamaiapis.net",
				"config_ids":    "12345",
				"client_token":  "test-token",
				"client_secret": "test-secret",
				"access_token":  "test-access",
				"event_limit":   700000,
			},
			wantErr: "event_limit cannot exceed 600000",
		},
		{
			name: "invalid initial_interval too high",
			config: map[string]interface{}{
				"api_host":         "https://test.luna.akamaiapis.net",
				"config_ids":       "12345",
				"client_token":     "test-token",
				"client_secret":    "test-secret",
				"access_token":     "test-access",
				"initial_interval": "24h",
			},
			wantErr: "initial_interval cannot exceed 12h",
		},
		{
			name: "invalid number_of_workers",
			config: map[string]interface{}{
				"api_host":          "https://test.luna.akamaiapis.net",
				"config_ids":        "12345",
				"client_token":      "test-token",
				"client_secret":     "test-secret",
				"access_token":      "test-access",
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

func TestEdgeGridSigner(t *testing.T) {
	signer := NewEdgeGridSigner("client-token", "client-secret", "access-token")

	// Test that signing key is created correctly
	timestamp := "20240101T12:00:00+0000"
	key := signer.createSigningKey(timestamp)
	assert.NotEmpty(t, key)
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	assert.Equal(t, defaultInterval, cfg.Interval)
	assert.Equal(t, defaultInitialInterval, cfg.InitialInterval)
	assert.Equal(t, defaultRecoveryInterval, cfg.RecoveryInterval)
	assert.Equal(t, defaultEventLimit, cfg.EventLimit)
	assert.Equal(t, defaultNumberOfWorkers, cfg.NumberOfWorkers)
	assert.NotNil(t, cfg.Resource)
	assert.NotNil(t, cfg.Resource.Retry.MaxAttempts)
	assert.Equal(t, defaultMaxAttempts, *cfg.Resource.Retry.MaxAttempts)
}

func TestCursor(t *testing.T) {
	t.Run("empty cursor", func(t *testing.T) {
		c := cursor{}
		assert.Empty(t, c.LastOffset)
		assert.False(t, c.RecoveryMode)
	})

	t.Run("cursor with offset", func(t *testing.T) {
		c := cursor{
			LastOffset: "12345",
		}
		assert.Equal(t, "12345", c.LastOffset)
		assert.False(t, c.RecoveryMode)
	})

	t.Run("cursor in recovery mode", func(t *testing.T) {
		c := cursor{
			RecoveryMode: true,
		}
		assert.Empty(t, c.LastOffset)
		assert.True(t, c.RecoveryMode)
	})
}

func TestAPIError(t *testing.T) {
	t.Run("basic error", func(t *testing.T) {
		err := &APIError{
			StatusCode: 400,
			Status:     "Bad Request",
		}
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "Bad Request")
	})

	t.Run("error with detail", func(t *testing.T) {
		err := &APIError{
			StatusCode: 400,
			Status:     "Bad Request",
			Detail:     "Invalid parameter",
		}
		assert.Contains(t, err.Error(), "Invalid parameter")
	})

	t.Run("is invalid timestamp", func(t *testing.T) {
		err := &APIError{
			StatusCode: 400,
			Detail:     "Invalid timestamp provided",
		}
		assert.True(t, err.IsInvalidTimestamp())

		err2 := &APIError{
			StatusCode: 400,
			Detail:     "Some other error",
		}
		assert.False(t, err2.IsInvalidTimestamp())
	})

	t.Run("is offset out of range", func(t *testing.T) {
		err := &APIError{
			StatusCode: 416,
		}
		assert.True(t, err.IsOffsetOutOfRange())

		err2 := &APIError{
			StatusCode: 400,
		}
		assert.False(t, err2.IsOffsetOutOfRange())
	})
}

func TestIsRecoverableError(t *testing.T) {
	t.Run("invalid timestamp is recoverable", func(t *testing.T) {
		err := &APIError{
			StatusCode: 400,
			Detail:     "Invalid timestamp",
		}
		assert.True(t, IsRecoverableError(err))
	})

	t.Run("offset out of range is recoverable", func(t *testing.T) {
		err := &APIError{
			StatusCode: 416,
		}
		assert.True(t, IsRecoverableError(err))
	})

	t.Run("other errors are not recoverable", func(t *testing.T) {
		err := &APIError{
			StatusCode: 500,
			Detail:     "Internal Server Error",
		}
		assert.False(t, IsRecoverableError(err))
	})
}

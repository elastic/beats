// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var configTests = []struct {
	name    string
	config  map[string]interface{}
	wantErr error
}{
	{
		name: "invalid_oauth2_config",
		config: map[string]interface{}{
			"account_name": "beatsblobnew",
			"auth.oauth2": map[string]interface{}{
				"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
				"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
			},
			"max_workers":   2,
			"poll":          true,
			"poll_interval": "10s",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
		},
		wantErr: fmt.Errorf("client_id, client_secret and tenant_id are required for OAuth2 auth accessing config"),
	},
	{
		name: "valid_oauth2_config",
		config: map[string]interface{}{
			"account_name": "beatsblobnew",
			"auth.oauth2": map[string]interface{}{
				"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
				"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
				"tenant_id":     "87654321-abcd-ef90-1234-fedcba098765",
			},
			"max_workers":   2,
			"poll":          true,
			"poll_interval": "10s",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
		},
	},
	{
		name: "valid_retry_config",
		config: map[string]interface{}{
			"account_name":                        "beatsblobnew",
			"auth.shared_credentials.account_key": "someKey",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
			"retry": map[string]interface{}{
				"max_retries":         20,
				"initial_retry_delay": "1s",
				"max_retry_delay":     "30s",
			},
		},
	},
	{
		name: "negative_initial_retry_delay",
		config: map[string]interface{}{
			"account_name":                        "beatsblobnew",
			"auth.shared_credentials.account_key": "someKey",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
			"retry": map[string]interface{}{
				"initial_retry_delay": "-1s",
			},
		},
		wantErr: fmt.Errorf("retry.initial_retry_delay must not be negative, got -1s accessing config"),
	},
	{
		name: "max_retry_delay_below_initial",
		config: map[string]interface{}{
			"account_name":                        "beatsblobnew",
			"auth.shared_credentials.account_key": "someKey",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
			"retry": map[string]interface{}{
				"initial_retry_delay": "30s",
				"max_retry_delay":     "5s",
			},
		},
		wantErr: fmt.Errorf("retry.max_retry_delay (5s) must not be smaller than retry.initial_retry_delay (30s) accessing config"),
	},
}

func TestConfig(t *testing.T) {
	logp.TestingSetup()
	for _, test := range configTests {
		t.Run(test.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(test.config)
			conf := config{}
			err := cfg.Unpack(&conf)

			switch {
			case err == nil && test.wantErr != nil:
				t.Fatalf("expected error unpacking config: %v", test.wantErr)
			case err != nil && test.wantErr == nil:
				t.Fatalf("unexpected error unpacking config: %v", err)
			case err != nil && test.wantErr != nil:
				assert.EqualError(t, err, test.wantErr.Error())
			}
		})
	}
}

// TestRetryConfig checks that the retry block unpacks into the config and maps
// onto the Azure SDK retry options, and that any option left unset keeps the
// SDK-matching default seeded by defaultConfig (including for a partial block).
func TestRetryConfig(t *testing.T) {
	logp.TestingSetup()

	t.Run("explicit", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"account_name":                        "beatsblobnew",
			"auth.shared_credentials.account_key": "someKey",
			"containers": []map[string]interface{}{
				{"name": beatsContainer},
			},
			"retry": map[string]interface{}{
				"max_retries":         20,
				"initial_retry_delay": "1s",
				"max_retry_delay":     "30s",
			},
		})
		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c), "unpacking a valid retry config should succeed")

		assert.Equal(t, 20, c.Retry.MaxRetries, "max_retries should unpack")
		assert.Equal(t, time.Second, c.Retry.InitialRetryDelay, "initial_retry_delay should unpack")
		assert.Equal(t, 30*time.Second, c.Retry.MaxRetryDelay, "max_retry_delay should unpack")

		got := azureRetryOptions(c.Retry)
		assert.Equal(t, int32(20), got.MaxRetries, "MaxRetries should map through")
		assert.Equal(t, time.Second, got.RetryDelay, "RetryDelay should map through")
		assert.Equal(t, 30*time.Second, got.MaxRetryDelay, "MaxRetryDelay should map through")
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"account_name":                        "beatsblobnew",
			"auth.shared_credentials.account_key": "someKey",
			"containers": []map[string]interface{}{
				{"name": beatsContainer},
			},
		})
		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c), "unpacking a config without a retry block should succeed")

		// Omitting the retry block keeps the seeded SDK-matching defaults, so
		// behaviour is identical to not configuring retries at all.
		assert.Equal(t, defaultMaxRetries, c.Retry.MaxRetries, "an unset max_retries must keep the default")
		assert.Equal(t, defaultInitialRetryDelay, c.Retry.InitialRetryDelay, "an unset initial_retry_delay must keep the default")
		assert.Equal(t, defaultMaxRetryDelay, c.Retry.MaxRetryDelay, "an unset max_retry_delay must keep the default")
	})

	t.Run("partial", func(t *testing.T) {
		// A partial retry block overrides only the provided field and keeps the
		// seeded defaults for the rest.
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"account_name":                        "beatsblobnew",
			"auth.shared_credentials.account_key": "someKey",
			"containers": []map[string]interface{}{
				{"name": beatsContainer},
			},
			"retry": map[string]interface{}{
				"max_retries": 7,
			},
		})
		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c), "unpacking a partial retry config should succeed")

		assert.Equal(t, 7, c.Retry.MaxRetries, "an explicit max_retries must be preserved")
		assert.Equal(t, defaultInitialRetryDelay, c.Retry.InitialRetryDelay, "an omitted initial_retry_delay must keep the default")
		assert.Equal(t, defaultMaxRetryDelay, c.Retry.MaxRetryDelay, "an omitted max_retry_delay must keep the default")
	})
}

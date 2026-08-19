// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestConfigInitDefaults(t *testing.T) {
	var cfg Config
	cfg.InitDefaults()

	assert.Equal(t, 24*time.Hour, cfg.BillingUsageLookback, "expected the default billing usage lookback")
	assert.Equal(t, 30*24*time.Hour, cfg.BillingForecastWindow, "expected the default billing forecast window")
	assert.Equal(t, "PT5M", cfg.DefaultTimeGrain, "expected the default resource time grain")
}

func TestConfigUnpackAppliesDefaults(t *testing.T) {
	rawConfig, err := config.NewConfigFrom(validConfig(map[string]any{
		"billing_usage_lookback": "72h",
	}))
	require.NoError(t, err, "expected the test configuration to be created")

	var cfg Config
	require.NoError(t, rawConfig.Unpack(&cfg), "expected the configuration to unpack")

	assert.Equal(t, 72*time.Hour, cfg.BillingUsageLookback, "expected the configured billing usage lookback")
	assert.Equal(t, 30*24*time.Hour, cfg.BillingForecastWindow, "expected the default billing forecast window")
	assert.Equal(t, "PT5M", cfg.DefaultTimeGrain, "expected the default resource time grain")
}

func TestConfigUnpackValidatesBillingDurations(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]any
		wantErr string
	}{
		{
			name: "accepts positive whole-day durations",
			values: map[string]any{
				"billing_usage_lookback":  "72h",
				"billing_forecast_window": "336h",
			},
		},
		{
			name:    "rejects zero usage lookback",
			values:  map[string]any{"billing_usage_lookback": "0s"},
			wantErr: "billing_usage_lookback must be a positive multiple of 24h, got 0s",
		},
		{
			name:    "rejects negative usage lookback",
			values:  map[string]any{"billing_usage_lookback": "-24h"},
			wantErr: "billing_usage_lookback must be a positive multiple of 24h, got -24h0m0s",
		},
		{
			name:    "rejects partial-day usage lookback",
			values:  map[string]any{"billing_usage_lookback": "36h"},
			wantErr: "billing_usage_lookback must be a positive multiple of 24h, got 36h0m0s",
		},
		{
			name:    "rejects zero forecast window",
			values:  map[string]any{"billing_forecast_window": "0s"},
			wantErr: "billing_forecast_window must be a positive multiple of 24h, got 0s",
		},
		{
			name:    "rejects negative forecast window",
			values:  map[string]any{"billing_forecast_window": "-24h"},
			wantErr: "billing_forecast_window must be a positive multiple of 24h, got -24h0m0s",
		},
		{
			name:    "rejects partial-day forecast window",
			values:  map[string]any{"billing_forecast_window": "36h"},
			wantErr: "billing_forecast_window must be a positive multiple of 24h, got 36h0m0s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rawConfig, err := config.NewConfigFrom(validConfig(test.values))
			require.NoError(t, err, "expected the test configuration to be created")

			var cfg Config
			err = rawConfig.Unpack(&cfg)
			if test.wantErr == "" {
				assert.NoError(t, err, "expected the billing durations to be accepted")
				return
			}
			assert.ErrorContains(t, err, test.wantErr, "expected the invalid billing duration to be rejected")
		})
	}
}

func validConfig(values map[string]any) map[string]any {
	cfg := map[string]any{
		"client_id":       "client",
		"client_secret":   "secret",
		"tenant_id":       "tenant",
		"subscription_id": "subscription",
		"period":          "1m",
	}
	for key, value := range values {
		cfg[key] = value
	}
	return cfg
}

func TestGroupMetricsDefinitionsByResourceId(t *testing.T) {

	t.Run("Group metrics definitions by resource ID", func(t *testing.T) {
		metrics := []Metric{
			{
				ResourceId: "resource-1",
				Namespace:  "namespace-1",
				Names:      []string{"metric-1"},
			},
			{
				ResourceId: "resource-1",
				Namespace:  "namespace-1",
				Names:      []string{"metric-2"},
			},
			{
				ResourceId: "resource-1",
				Namespace:  "namespace-1",
				Names:      []string{"metric-3"},
			},
		}

		metricsByResourceId := groupMetricsDefinitionsByResourceId(metrics)

		assert.Equal(t, 1, len(metricsByResourceId))
		assert.Equal(t, 3, len(metricsByResourceId["resource-1"]))
	})
}

func TestCalculateTimespan(t *testing.T) {
	t.Run("Collection period greater than the time grain (PT1M metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period: 5 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T18:51:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT1M metric every 1 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period: 1 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T18:55:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT5M metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT5M"
		cfg := Config{
			Period: 5 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T18:51:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT1H metric every 60 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1H"
		cfg := Config{
			Period: 60 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T17:56:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period is less that time grain (PT1H metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1H"
		cfg := Config{
			Period: 5 * time.Minute,
		}
		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T17:56:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

}

func TestCalculateTimespanWithLatency(t *testing.T) {
	t.Run("Collection period greater than the time grain (PT1M metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period:  5 * time.Minute,
			Latency: 1 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T18:50:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:55:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT1M metric every 1 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period:  1 * time.Minute,
			Latency: 1 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg)

		require.Equal(t, "2024-07-30T18:54:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:55:00Z", endTime.Format(time.RFC3339))
	})
}

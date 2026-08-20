// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistogramAssemblyConfigDefaults(t *testing.T) {
	cfg := config{
		UseTypes: true,
		HistogramAssembly: HistogramAssembly{
			Enabled: true,
		},
	}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, 5*time.Second, cfg.HistogramAssembly.QuietPeriod)
	assert.Equal(t, 30*time.Second, cfg.HistogramAssembly.HardTimeout)
	assert.Equal(t, 10_000, cfg.HistogramAssembly.MaxPendingHistograms)
	assert.Equal(t, 100_000, cfg.HistogramAssembly.MaxPendingBuckets)
}

func TestHistogramAssemblyConfigDisabledByDefault(t *testing.T) {
	cfg := config{UseTypes: true}
	require.NoError(t, cfg.Validate())
	assert.False(t, cfg.HistogramAssembly.Enabled)
	assert.Zero(t, cfg.HistogramAssembly.QuietPeriod)
	assert.Zero(t, cfg.HistogramAssembly.HardTimeout)
	assert.Zero(t, cfg.HistogramAssembly.MaxPendingHistograms)
	assert.Zero(t, cfg.HistogramAssembly.MaxPendingBuckets)
}

func TestHistogramAssemblyConfigValidation(t *testing.T) {
	t.Run("quiet greater than hard", func(t *testing.T) {
		cfg := config{
			UseTypes: true,
			HistogramAssembly: HistogramAssembly{
				Enabled:     true,
				QuietPeriod: 31 * time.Second,
				HardTimeout: 30 * time.Second,
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "quiet_period")
	})

	t.Run("non-positive quiet", func(t *testing.T) {
		cfg := config{
			UseTypes: true,
			HistogramAssembly: HistogramAssembly{
				Enabled:     true,
				QuietPeriod: -1 * time.Second,
				HardTimeout: 30 * time.Second,
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
	})
}

func TestHistogramAssemblyDisabledIgnoresInvalidTuning(t *testing.T) {
	cfg := config{
		UseTypes: true,
		HistogramAssembly: HistogramAssembly{
			Enabled:     false,
			QuietPeriod: -1,
		},
	}
	require.NoError(t, cfg.Validate(), "histogram assembly tuning applies only when histogram_assembly.enabled is true")
}

func TestHistogramAssemblerRequiresUseTypes(t *testing.T) {
	cfg := config{
		HistogramAssembly: HistogramAssembly{
			Enabled: true,
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "histogram_assembly.enabled")
	assert.Contains(t, err.Error(), "use_types")
}

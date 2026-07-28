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
	cfg := config{UseTypes: true, UseHistogramAssembler: true}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, 5*time.Second, cfg.HistogramAssembly.QuietPeriod)
	assert.Equal(t, 30*time.Second, cfg.HistogramAssembly.HardTimeout)
	assert.Equal(t, 10_000, cfg.HistogramAssembly.MaxPendingHistograms)
	assert.Equal(t, 100_000, cfg.HistogramAssembly.MaxPendingBuckets)
	assert.Equal(t, 30*time.Second, cfg.HistogramAssembly.TombstoneTTL)
}

func TestHistogramAssemblyConfigValidation(t *testing.T) {
	t.Run("quiet greater than hard", func(t *testing.T) {
		cfg := config{
			UseTypes:              true,
			UseHistogramAssembler: true,
			HistogramAssembly: HistogramAssembly{
				QuietPeriod:  31 * time.Second,
				HardTimeout:  30 * time.Second,
				TombstoneTTL: time.Second,
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "quiet_period")
	})

	t.Run("non-positive quiet", func(t *testing.T) {
		cfg := config{
			UseTypes:              true,
			UseHistogramAssembler: true,
			HistogramAssembly: HistogramAssembly{
				QuietPeriod:  -1 * time.Second,
				HardTimeout:  30 * time.Second,
				TombstoneTTL: time.Second,
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
	})
}

func TestHistogramAssemblySkippedWhenUseTypesFalse(t *testing.T) {
	cfg := config{UseTypes: false, HistogramAssembly: HistogramAssembly{QuietPeriod: -1}}
	require.NoError(t, cfg.Validate(), "histogram assembly validation applies only when use_types is enabled")
}

func TestHistogramAssemblySkippedWhenAssemblerDisabled(t *testing.T) {
	cfg := config{
		UseTypes:              true,
		UseHistogramAssembler: false,
		HistogramAssembly:     HistogramAssembly{QuietPeriod: -1},
	}
	require.NoError(t, cfg.Validate(), "histogram assembly validation applies only when the assembler is enabled")
}

func TestHistogramAssemblerRequiresUseTypes(t *testing.T) {
	cfg := config{UseHistogramAssembler: true}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use_histogram_assembler")
	assert.Contains(t, err.Error(), "use_types")
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

func TestUsagePeriodFrom(t *testing.T) {
	t.Run("returns the start and end times for the usage period", func(t *testing.T) {
		referenceTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-09 09:41:00")
		assert.NoError(t, err)
		expectedStartTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-08 00:00:00")
		assert.NoError(t, err)
		expectedEndTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-08 23:59:59")
		assert.NoError(t, err)

		actualStartTime, actualEndTime := usageIntervalFrom(referenceTime, 24*time.Hour)

		assert.Equal(t, expectedStartTime, actualStartTime)
		assert.Equal(t, expectedEndTime, actualEndTime)
	})

	t.Run("widens the window when lookback is 72h", func(t *testing.T) {
		referenceTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-09 09:41:00")
		assert.NoError(t, err)
		// 72h lookback from start-of-today (2007-01-09 00:00:00) → 2007-01-06 00:00:00
		// end always = start-of-today minus one second → 2007-01-08 23:59:59
		expectedStartTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-06 00:00:00")
		assert.NoError(t, err)
		expectedEndTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-08 23:59:59")
		assert.NoError(t, err)

		actualStartTime, actualEndTime := usageIntervalFrom(referenceTime, 72*time.Hour)

		assert.Equal(t, expectedStartTime, actualStartTime)
		assert.Equal(t, expectedEndTime, actualEndTime)
	})
}

func TestForecastPeriodFrom(t *testing.T) {
	t.Run("returns the start and end times for the forecast period", func(t *testing.T) {
		referenceTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-09 09:41:00")
		assert.NoError(t, err)

		expectedStartTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-07 00:00:00")
		assert.NoError(t, err)
		expectedEndTime, err := time.Parse("2006-01-02 15:04:05", "2007-02-05 23:59:59")
		assert.NoError(t, err)

		actualStartTime, actualEndTime := forecastIntervalFrom(referenceTime, 30*24*time.Hour)

		assert.Equal(t, expectedStartTime, actualStartTime)
		assert.Equal(t, expectedEndTime, actualEndTime)
	})

	t.Run("uses the configured window duration", func(t *testing.T) {
		referenceTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-09 09:41:00")
		assert.NoError(t, err)

		// forecast always starts at reference - 2 days; end = start + window - 1s
		// with a 7-day window: 2007-01-07 00:00:00 + 7d - 1s = 2007-01-13 23:59:59
		expectedStartTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-07 00:00:00")
		assert.NoError(t, err)
		expectedEndTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-13 23:59:59")
		assert.NoError(t, err)

		actualStartTime, actualEndTime := forecastIntervalFrom(referenceTime, 7*24*time.Hour)

		assert.Equal(t, expectedStartTime, actualStartTime)
		assert.Equal(t, expectedEndTime, actualEndTime)
	})
}

func TestApplyBillingDefaults(t *testing.T) {
	t.Run("sets usage lookback and forecast window when both are unset", func(t *testing.T) {
		cfg := azure.Config{}
		applyBillingDefaults(&cfg)

		assert.Equal(t, defaultUsageLookback, cfg.BillingUsageLookback)
		assert.Equal(t, defaultForecastWindow, cfg.BillingForecastWindow)
	})

	t.Run("does not override an explicitly configured usage lookback", func(t *testing.T) {
		cfg := azure.Config{BillingUsageLookback: 72 * time.Hour}
		applyBillingDefaults(&cfg)

		assert.Equal(t, 72*time.Hour, cfg.BillingUsageLookback)
		assert.Equal(t, defaultForecastWindow, cfg.BillingForecastWindow)
	})

	t.Run("does not override an explicitly configured forecast window", func(t *testing.T) {
		cfg := azure.Config{BillingForecastWindow: 14 * 24 * time.Hour}
		applyBillingDefaults(&cfg)

		assert.Equal(t, defaultUsageLookback, cfg.BillingUsageLookback)
		assert.Equal(t, 14*24*time.Hour, cfg.BillingForecastWindow)
	})
}

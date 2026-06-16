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
)

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

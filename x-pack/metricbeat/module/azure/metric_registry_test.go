// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewMetricRegistry(t *testing.T) {
	logger := logp.NewLogger("test azure monitor")

	t.Run("Collect metrics with a regular 5 minutes period", func(t *testing.T) {
		metricRegistry := NewMetricRegistry(logger)

		// Create a lastCollectionAt parsing the string 2023-12-08T16:37:50.000Z into a time.Time
		lastCollectionAt, _ := time.Parse(time.RFC3339, "2023-12-08T16:37:50.000Z")

		// Create a referenceTime parsing 2023-12-08T16:42:50.000Z into a time.Time
		referenceTime, _ := time.Parse(time.RFC3339, "2023-12-08T16:42:50.000Z")

		metric := Metric{
			ResourceId: "test",
			Namespace:  "test",
		}
		metricCollectionInfo := MetricCollectionInfo{
			timeGrain: "PT5M",
			timestamp: lastCollectionAt,
		}

		metricRegistry.Update(metric, metricCollectionInfo)

		needsUpdate := metricRegistry.NeedsUpdate(referenceTime, metric)

		assert.True(t, needsUpdate, "metric should need update")
	})

	t.Run("Collect metrics using a period 3 seconds longer than previous", func(t *testing.T) {
		metricRegistry := NewMetricRegistry(logger)

		// Create a lastCollectionAt parsing the string 2023-12-08T16:37:50.000Z into a time.Time
		lastCollectionAt, _ := time.Parse(time.RFC3339, "2023-12-08T16:37:50.000Z")

		// Create a referenceTime parsing 2023-12-08T16:42:50.000Z into a time.Time
		referenceTime, _ := time.Parse(time.RFC3339, "2023-12-08T16:42:53.000Z")

		metric := Metric{
			ResourceId: "test",
			Namespace:  "test",
		}
		metricCollectionInfo := MetricCollectionInfo{
			timeGrain: "PT5M",
			timestamp: lastCollectionAt,
		}

		metricRegistry.Update(metric, metricCollectionInfo)

		needsUpdate := metricRegistry.NeedsUpdate(referenceTime, metric)

		assert.True(t, needsUpdate, "metric should need update")
	})

	t.Run("Collect metrics using a period (1 second) shorter than previous", func(t *testing.T) {
		metricRegistry := NewMetricRegistry(logger)

		// Create a referenceTime parsing 2023-12-08T16:42:50.000Z into a time.Time
		referenceTime, _ := time.Parse(time.RFC3339, "2023-12-08T10:58:33.000Z")

		// Create a lastCollectionAt parsing the string 2023-12-08T16:37:50.000Z into a time.Time
		lastCollectionAt, _ := time.Parse(time.RFC3339, "2023-12-08T10:53:34.000Z")

		metric := Metric{
			ResourceId: "test",
			Namespace:  "test",
		}
		metricCollectionInfo := MetricCollectionInfo{
			timeGrain: "PT5M",
			timestamp: lastCollectionAt,
		}

		metricRegistry.Update(metric, metricCollectionInfo)

		needsUpdate := metricRegistry.NeedsUpdate(referenceTime, metric)

		assert.False(t, needsUpdate, "metric should not need update")
	})

	//
	// These tests document the limits of the time.Round function used
	// to round the reference time.
	//

	t.Run("Round outer limits", func(t *testing.T) {
		referenceTime1, _ := time.Parse(time.RFC3339, "2023-12-08T10:58:32.500Z")
		referenceTime2, _ := time.Parse(time.RFC3339, "2023-12-08T10:58:33.499Z")

		assert.Equal(t, referenceTime1.Round(time.Second), referenceTime2.Round(time.Second))
	})

	t.Run("Round inner limits", func(t *testing.T) {
		referenceTime1, _ := time.Parse(time.RFC3339, "2023-12-08T10:58:32.999Z")
		referenceTime2, _ := time.Parse(time.RFC3339, "2023-12-08T10:58:33.001Z")

		assert.Equal(t, referenceTime1.Round(time.Second), referenceTime2.Round(time.Second))
	})
}

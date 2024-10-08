// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetricRegistry instantiates a new metric registry.
func NewMetricRegistry(logger *logp.Logger) *MetricRegistry {
	return &MetricRegistry{
		logger:          logger,
		collectionsInfo: make(map[string]MetricCollectionInfo),
		jitter:          1 * time.Second,
	}
}

// MetricRegistry keeps track of the last time a metric was collected and
// the time grain used.
//
// This is used to avoid collecting the same metric values over and over again
// when the time grain is larger than the collection interval.
type MetricRegistry struct {
	logger          *logp.Logger
	collectionsInfo map[string]MetricCollectionInfo
	// The collection period can be jittered by a second.
	// We introduce a small jitter to avoid skipping collections
	// when the collection period is close (usually < 1s) to the
	// time grain start time.
	jitter time.Duration
}

// Update updates the metric registry with the latest timestamp and
// time grain for the given metric.
func (m *MetricRegistry) Update(metric Metric, info MetricCollectionInfo) {
	m.collectionsInfo[m.buildMetricKey(metric)] = info
}

// NeedsUpdate returns true if the metric needs to be collected again
// for the given `referenceTime`.
func (m *MetricRegistry) NeedsUpdate(referenceTime time.Time, metric Metric) bool {
	// Build a key to store the metric in the registry.
	// The key is a combination of the namespace,
	// resource ID and metric names.
	metricKey := m.buildMetricKey(metric)

	if lastCollection, exists := m.collectionsInfo[metricKey]; exists {
		// Turn the time grain into a duration (for example, PT5M -> 5 minutes).
		timeGrainDuration := asDuration(lastCollection.timeGrain)

		// Adjust the last collection time by adding a small jitter to avoid
		// skipping collections when the collection period is close (usually < 1s).
		timeSinceLastCollection := time.Since(lastCollection.timestamp) + m.jitter

		if timeSinceLastCollection < timeGrainDuration {
			m.logger.Debugw(
				"MetricRegistry: Metric does not need an update",
				"needs_update", false,
				"reference_time", referenceTime,
				"last_collection_time", lastCollection.timestamp,
				"time_since_last_collection_seconds", timeSinceLastCollection.Seconds(),
				"time_grain", lastCollection.timeGrain,
				"time_grain_duration_seconds", timeGrainDuration.Seconds(),
				"resource_id", metric.ResourceId,
				"namespace", metric.Namespace,
				"aggregation", metric.Aggregations,
				"names", strings.Join(metric.Names, ","),
			)

			return false
		}

		// The last collection time is before the start time of the time grain,
		// it means that the metricset needs to collect the metric values again.
		m.logger.Debugw(
			"MetricRegistry: Metric needs an update",
			"needs_update", true,
			"reference_time", referenceTime,
			"last_collection_time", lastCollection.timestamp,
			"time_since_last_collection_seconds", timeSinceLastCollection.Seconds(),
			"time_grain", lastCollection.timeGrain,
			"time_grain_duration_seconds", timeGrainDuration.Seconds(),
			"resource_id", metric.ResourceId,
			"namespace", metric.Namespace,
			"aggregation", metric.Aggregations,
			"names", strings.Join(metric.Names, ","),
		)

		return true
	}

	// If the metric is not in the registry, it means that it has never
	// been collected before.
	//
	// In this case, we need to collect the metric.
	m.logger.Debugw(
		"MetricRegistry: Metric needs an update (no collection info in the metric registry)",
		"needs_update", true,
		"reference_time", referenceTime,
		"resource_id", metric.ResourceId,
		"namespace", metric.Namespace,
		"aggregation", metric.Aggregations,
		"names", strings.Join(metric.Names, ","),
	)

	return true
}

// buildMetricKey builds a key for the metric registry.
//
// The key is a combination of the namespace, resource ID and metric names.
func (m *MetricRegistry) buildMetricKey(metric Metric) string {
	keyComponents := []string{
		metric.Namespace,
		metric.ResourceId,
		metric.Aggregations,
		metric.TimeGrain,
		strings.Join(metric.Names, ","),
	}

	for _, dim := range metric.Dimensions {
		keyComponents = append(keyComponents, fmt.Sprintf("%s=%s", dim.Name, dim.Value))
	}

	return strings.Join(keyComponents, ",")
}

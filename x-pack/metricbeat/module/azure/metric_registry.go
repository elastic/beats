// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
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

	//// Get the now time in UTC, only to be used for logging.
	//// It's interesting to see when the registry evaluate each
	//// metric in relation to the reference time.
	//now := time.Now().UTC()

	if lastCollection, exists := m.collectionsInfo[metricKey]; exists {
		// Turn the time grain into a duration (for example, PT5M -> 5 minutes).
		timeGrainDuration := convertTimeGrainToDuration(lastCollection.timeGrain)

		//// Calculate the start time of the time grain in relation to
		//// the reference time.
		//timeGrainStartTime := referenceTime.Add(-timeGrainDuration)

		//// Only to be used for logging.
		////
		//// The time elapsed since the last collection, and the time
		//// distance between last collection and the start of time
		//// grain.
		//elapsed := referenceTime.Sub(lastCollection.timestamp)
		//distance := lastCollection.timestamp.Sub(timeGrainStartTime)

		// If the last collection time is after the start time of the time grain,
		// it means that we already have a value for the given time grain.
		//
		// In this case, the metricset does not need to collect the metric
		// values again.
		//
		// if time.Since(metricsByGrain.metricsValuesUpdated).Seconds() < float64(timeGrains[compositeKey.timeGrain]) {
		//if lastCollection.timestamp.After(timeGrainStartTime.Add(m.jitter)) {
		lastCollectionSeconds := time.Since(lastCollection.timestamp).Seconds()
		timeGrainSeconds := timeGrainDuration.Seconds()

		if time.Since(lastCollection.timestamp).Seconds() < timeGrainDuration.Seconds() {
			m.logger.Debugw(
				"MetricRegistry: Metric does not need an update",
				"needs_update", false,
				"reference_time", referenceTime,
				//"now", now,
				//"time_grain_start_time", timeGrainStartTime,
				"last_collection_time", lastCollection.timestamp,
				"time_grain", lastCollection.timeGrain,
				"resource_id", metric.ResourceId,
				"namespace", metric.Namespace,
				"aggregation", metric.Aggregations,
				"names", strings.Join(metric.Names, ","),
				//"elapsed", elapsed.String(),
				//"jitter", m.jitter.String(),
				//"distance", distance.String(),
				"last_collection_seconds", lastCollectionSeconds,
				"time_grain_seconds", timeGrainSeconds,
			)

			return false
		}

		// The last collection time is before the start time of the time grain,
		// it means that the metricset needs to collect the metric values again.
		m.logger.Debugw(
			"MetricRegistry: Metric needs an update",
			"needs_update", true,
			"reference_time", referenceTime,
			//"now", now,
			//"time_grain_start_time", timeGrainStartTime,
			"last_collection_time", lastCollection.timestamp,
			"time_grain", lastCollection.timeGrain,
			"resource_id", metric.ResourceId,
			"namespace", metric.Namespace,
			"aggregation", metric.Aggregations,
			"names", strings.Join(metric.Names, ","),
			//"elapsed", elapsed.String(),
			//"jitter", m.jitter.String(),
			//"distance", distance.String(),
			"last_collection_seconds", lastCollectionSeconds,
			"time_grain_seconds", timeGrainSeconds,
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
		//"now", now,
		"time_grain", metric.TimeGrain,
		"resource_id", metric.ResourceId,
		"namespace", metric.Namespace,
		"aggregation", metric.Aggregations,
		"names", strings.Join(metric.Names, ","),
		//"jitter", m.jitter.String(),
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
	}
	keyComponents = append(keyComponents, metric.Names...)

	return strings.Join(keyComponents, ",")
}

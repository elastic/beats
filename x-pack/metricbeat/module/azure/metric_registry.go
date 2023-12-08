package azure

import (
	"github.com/elastic/elastic-agent-libs/logp"
	"strings"
	"time"
)

// NewMetricRegistry instantiates a new metric registry.
func NewMetricRegistry(logger *logp.Logger) *MetricRegistry {
	return &MetricRegistry{
		logger:          logger,
		collectionsInfo: make(map[string]MetricCollectionInfo),
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

	// Get the now time in UTC, only to be used for logging.
	// It's interesting to see when the registry evaluate each
	// metric in relation to the reference time.
	now := time.Now().UTC()

	if collection, exists := m.collectionsInfo[metricKey]; exists {
		// Turn the time grain into a duration (for example, PT5M -> 5 minutes).
		timeGrainDuration := convertTimeGrainToDuration(collection.timeGrain)

		// Calculate the start time of the time grain in relation to
		// the reference time.
		timeGrainStartTime := referenceTime.Add(-timeGrainDuration)

		// The collection period can be jittered by a few seconds.
		// We introduce a small jitter to avoid skipping collections
		// when the collection period is close (1-2 seconds) to the
		// time grain.
		//jitter := 3 * time.Second

		// The time elapsed since the last collection, only to be
		// used for logging.
		elapsed := referenceTime.Sub(collection.timestamp)

		// If the last collection time is after the start time of the time grain,
		// it means that we already have a value for the given time grain.
		//
		// In this case, the metricset does not need to collect the metric
		// values again.
		if collection.timestamp.After(timeGrainStartTime) {
			m.logger.Debugw(
				"MetricRegistry: Metric does not need an update",
				"needs_update", false,
				"reference_time", referenceTime,
				"now", now,
				"time_grain_start_time", timeGrainStartTime,
				"last_collection_at", collection.timestamp,
				"time_grain", metric.TimeGrain,
				"resource_id", metric.ResourceId,
				"namespace", metric.Namespace,
				"names", strings.Join(metric.Names, ","),
				"elapsed", elapsed,
			)

			return false
		}

		// The last collection time is before the start time of the time grain,
		// it means that the metricset needs to collect the metric values again.
		m.logger.Debugw(
			"MetricRegistry: Metric needs an update",
			"needs_update", true,
			"reference_time", referenceTime,
			"now", now,
			"time_grain_start_time", timeGrainStartTime,
			"last_collection_at", collection.timestamp,
			"time_grain", metric.TimeGrain,
			"resource_id", metric.ResourceId,
			"namespace", metric.Namespace,
			"names", strings.Join(metric.Names, ","),
			"elapsed", elapsed,
		)

		return true
	}

	// If the metric is not in the registry, it means that it has never
	// been collected before.
	//
	// In this case, we need to collect the metric.
	m.logger.Debugw(
		"MetricRegistry: Metric needs an update",
		"needs_update", true,
		"reference_time", referenceTime,
		"now", now,
		"time_grain", metric.TimeGrain,
		"resource_id", metric.ResourceId,
		"namespace", metric.Namespace,
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
	}
	keyComponents = append(keyComponents, metric.Names...)

	return strings.Join(keyComponents, ",")
}

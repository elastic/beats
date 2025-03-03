// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// inputMetrics defines metrics for this provider.
type inputMetrics struct {
	unregister func()

	syncTotal            *monitoring.Uint // The total number of full synchronizations.
	syncError            *monitoring.Uint // The number of full synchronizations that failed due to an error.
	syncProcessingTime   metrics.Sample   // Histogram of the elapsed full synchronization times in nanoseconds (time of API contact to items sent to output).
	updateTotal          *monitoring.Uint // The total number of incremental updates.
	updateError          *monitoring.Uint // The number of incremental updates that failed due to an error.
	updateProcessingTime metrics.Sample   // Histogram of the elapsed incremental update times in nanoseconds (time of API contact to items sent to output).
}

// Close removes metrics from the registry.
func (m *inputMetrics) Close() {
	m.unregister()
}

// newMetrics creates a new instance for gathering metrics.
func newMetrics(id string, optionalParent *monitoring.Registry) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(FullName, id, optionalParent)

	out := inputMetrics{
		unregister:           unreg,
		syncTotal:            monitoring.NewUint(reg, "sync_total"),
		syncError:            monitoring.NewUint(reg, "sync_error"),
		syncProcessingTime:   metrics.NewUniformSample(1024),
		updateTotal:          monitoring.NewUint(reg, "update_total"),
		updateError:          monitoring.NewUint(reg, "update_error"),
		updateProcessingTime: metrics.NewUniformSample(1024),
	}

	adapter.NewGoMetrics(reg, "sync_processing_time", adapter.Accept).Register("histogram", metrics.NewHistogram(out.syncProcessingTime))     //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "update_processing_time", adapter.Accept).Register("histogram", metrics.NewHistogram(out.updateProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.

	return &out
}

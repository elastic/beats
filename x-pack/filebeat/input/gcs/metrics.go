// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

var (
	clockValue atomic.Value // Atomic reference to a clock value.
	realClock  = clock{Now: time.Now}
)

type clock struct {
	Now func() time.Time
}

func init() {
	clockValue.Store(realClock)
}

// currentTime returns the current time. This exists to allow unit tests
// simulate the passage of time.
func currentTime() time.Time {
	clock, _ := clockValue.Load().(clock)
	return clock.Now()
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister  func()
	url         *monitoring.String // URL of the input resource
	errorsTotal *monitoring.Uint   // number of errors encountered

	gcsObjectsRequestedTotal *monitoring.Uint // Number of GCS objects downloaded.
	gcsObjectsPublishedTotal *monitoring.Uint // Number of GCS objects processed that were published.
	gcsObjectsListedTotal    *monitoring.Uint // Number of GCS objects returned by list operations.
	gcsObjectsProcessedTotal *monitoring.Uint // Number of GCS objects that matched file_selectors rules.
	gcsBytesProcessedTotal   *monitoring.Uint // Number of GCS bytes processed.
	gcsEventsCreatedTotal    *monitoring.Uint // Number of events created from processing GCS data.
	gcsObjectProcessingTime  metrics.Sample   // Histogram of the elapsed GCS object processing times in nanoseconds (start of download to completion of parsing).
	gcsObjectSizeInBytes     metrics.Sample   // Histogram of processed GCS object size in bytes
	gcsEventsPerObject       metrics.Sample   // Histogram of events in an individual GCS object
}

func newInputMetrics(id string) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, nil)
	out := &inputMetrics{
		unregister:  unreg,
		url:         monitoring.NewString(reg, "url"),
		errorsTotal: monitoring.NewUint(reg, "errors_total"),

		gcsObjectsRequestedTotal: monitoring.NewUint(reg, "gcs_objects_requested_total"),
		gcsObjectsPublishedTotal: monitoring.NewUint(reg, "gcs_objects_published_total"),
		gcsObjectsListedTotal:    monitoring.NewUint(reg, "gcs_objects_listed_total"),
		gcsObjectsProcessedTotal: monitoring.NewUint(reg, "gcs_objects_processed_total"),
		gcsBytesProcessedTotal:   monitoring.NewUint(reg, "gcs_bytes_processed_total"),
		gcsEventsCreatedTotal:    monitoring.NewUint(reg, "gcs_events_created_total"),
		gcsObjectProcessingTime:  metrics.NewUniformSample(1024),
		gcsObjectSizeInBytes:     metrics.NewUniformSample(1024),
		gcsEventsPerObject:       metrics.NewUniformSample(1024),
	}

	adapter.NewGoMetrics(reg, "gcs_object_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsObjectProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "gcs_object_size_in_bytes", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsObjectSizeInBytes)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "gcs_events_per_object", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsEventsPerObject)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}

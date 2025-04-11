// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister        func()
	url               *monitoring.String // URL of the input resource.
	errorsTotal       *monitoring.Uint   // Number of errors encountered.
	decodeErrorsTotal *monitoring.Uint   // Number of decode errors encountered.

	gcsObjectsTracked               *monitoring.Uint // Number of objects currently tracked in the state registry (gauge).
	gcsObjectsRequestedTotal        *monitoring.Uint // Number of GCS objects downloaded.
	gcsObjectsPublishedTotal        *monitoring.Uint // Number of GCS objects processed that were published.
	gcsObjectsListedTotal           *monitoring.Uint // Number of GCS objects returned by list operations.
	gcsBytesProcessedTotal          *monitoring.Uint // Number of GCS bytes processed.
	gcsEventsCreatedTotal           *monitoring.Uint // Number of events created from processing GCS data.
	gcsFailedJobsTotal              *monitoring.Uint // Number of failed jobs.
	gcsExpiredFailedJobsTotal       *monitoring.Uint // Number of expired failed jobs that could not be recovered.
	gcsObjectsInflight              *monitoring.Uint // Number of GCS objects inflight (gauge).
	gcsObjectProcessingTime         metrics.Sample   // Histogram of the elapsed GCS object processing times in nanoseconds (start of download to completion of parsing).
	gcsObjectSizeInBytes            metrics.Sample   // Histogram of processed GCS object size in bytes.
	gcsEventsPerObject              metrics.Sample   // Histogram of event count per GCS object.
	gcsJobsScheduledAfterValidation metrics.Sample   // Histogram of number of jobs scheduled after validation.
	sourceLagTime                   metrics.Sample   // Histogram of the time between the source (Updated) timestamp and the time the object was read.
}

func newInputMetrics(id string, optionalParent *monitoring.Registry) *inputMetrics {
	// TODO: use NewMetricsRegistry instead of inputmon.NewInputRegistry.
	// The id isn't the same as the v2.Context.ID, thus the pipeline metrics
	// won't be in the same registry as the input metrics.
	reg, unreg := inputmon.NewInputRegistry(inputName, id, optionalParent)
	out := &inputMetrics{
		unregister:        unreg,
		url:               monitoring.NewString(reg, "url"),
		errorsTotal:       monitoring.NewUint(reg, "errors_total"),
		decodeErrorsTotal: monitoring.NewUint(reg, "decode_errors_total"),

		gcsObjectsTracked:               monitoring.NewUint(reg, "gcs_objects_tracked_gauge"),
		gcsObjectsRequestedTotal:        monitoring.NewUint(reg, "gcs_objects_requested_total"),
		gcsObjectsPublishedTotal:        monitoring.NewUint(reg, "gcs_objects_published_total"),
		gcsObjectsListedTotal:           monitoring.NewUint(reg, "gcs_objects_listed_total"),
		gcsBytesProcessedTotal:          monitoring.NewUint(reg, "gcs_bytes_processed_total"),
		gcsEventsCreatedTotal:           monitoring.NewUint(reg, "gcs_events_created_total"),
		gcsFailedJobsTotal:              monitoring.NewUint(reg, "gcs_failed_jobs_total"),
		gcsExpiredFailedJobsTotal:       monitoring.NewUint(reg, "gcs_expired_failed_jobs_total"),
		gcsObjectsInflight:              monitoring.NewUint(reg, "gcs_objects_inflight_gauge"),
		gcsObjectProcessingTime:         metrics.NewUniformSample(1024),
		gcsObjectSizeInBytes:            metrics.NewUniformSample(1024),
		gcsEventsPerObject:              metrics.NewUniformSample(1024),
		gcsJobsScheduledAfterValidation: metrics.NewUniformSample(1024),
		sourceLagTime:                   metrics.NewUniformSample(1024),
	}

	adapter.NewGoMetrics(reg, "gcs_object_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsObjectProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "gcs_object_size_in_bytes", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsObjectSizeInBytes)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "gcs_events_per_object", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsEventsPerObject)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "gcs_jobs_scheduled_after_validation", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.gcsJobsScheduledAfterValidation)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "source_lag_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sourceLagTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

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

	absBlobsRequestedTotal          *monitoring.Uint // Number of ABS blobs downloaded.
	absBlobsPublishedTotal          *monitoring.Uint // Number of ABS blobs processed that were published.
	absBlobsListedTotal             *monitoring.Uint // Number of ABS blobs returned by list operations.
	absBytesProcessedTotal          *monitoring.Uint // Number of ABS bytes processed.
	absEventsCreatedTotal           *monitoring.Uint // Number of events created from processing ABS data.
	absBlobsInflight                *monitoring.Uint // Number of ABS blobs inflight (gauge).
	absBlobProcessingTime           metrics.Sample   // Histogram of the elapsed ABS blob processing times in nanoseconds (start of download to completion of parsing).
	absBlobSizeInBytes              metrics.Sample   // Histogram of processed ABS blob size in bytes.
	absEventsPerBlob                metrics.Sample   // Histogram of event count per ABS blob.
	absJobsScheduledAfterValidation metrics.Sample   // Histogram of number of jobs scheduled after validation.
	sourceLagTime                   metrics.Sample   // Histogram of the time between the source (Updated) timestamp and the time the blob was read.
}

func newInputMetrics(id string, optionalParent *monitoring.Registry) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, optionalParent)
	out := &inputMetrics{
		unregister:        unreg,
		url:               monitoring.NewString(reg, "url"),
		errorsTotal:       monitoring.NewUint(reg, "errors_total"),
		decodeErrorsTotal: monitoring.NewUint(reg, "decode_errors_total"),

		absBlobsRequestedTotal:          monitoring.NewUint(reg, "abs_blobs_requested_total"),
		absBlobsPublishedTotal:          monitoring.NewUint(reg, "abs_blobs_published_total"),
		absBlobsListedTotal:             monitoring.NewUint(reg, "abs_blobs_listed_total"),
		absBytesProcessedTotal:          monitoring.NewUint(reg, "abs_bytes_processed_total"),
		absEventsCreatedTotal:           monitoring.NewUint(reg, "abs_events_created_total"),
		absBlobsInflight:                monitoring.NewUint(reg, "abs_blobs_inflight_gauge"),
		absBlobProcessingTime:           metrics.NewUniformSample(1024),
		absBlobSizeInBytes:              metrics.NewUniformSample(1024),
		absEventsPerBlob:                metrics.NewUniformSample(1024),
		absJobsScheduledAfterValidation: metrics.NewUniformSample(1024),
		sourceLagTime:                   metrics.NewUniformSample(1024),
	}

	adapter.NewGoMetrics(reg, "abs_blob_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.absBlobProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "abs_blob_size_in_bytes", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.absBlobSizeInBytes)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "abs_events_per_blob", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.absEventsPerBlob)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "abs_jobs_scheduled_after_validation", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.absJobsScheduledAfterValidation)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "source_lag_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sourceLagTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}

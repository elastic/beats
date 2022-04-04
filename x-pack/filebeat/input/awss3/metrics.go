// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"io"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/monitoring/adapter"
)

type inputMetrics struct {
	id     string               // Input ID.
	parent *monitoring.Registry // Parent registry holding this input's ID as a key.

	sqsMessagesReceivedTotal            *monitoring.Uint // Number of SQS messages received (not necessarily processed fully).
	sqsVisibilityTimeoutExtensionsTotal *monitoring.Uint // Number of SQS visibility timeout extensions.
	sqsMessagesInflight                 *monitoring.Uint // Number of SQS messages inflight (gauge).
	sqsMessagesReturnedTotal            *monitoring.Uint // Number of SQS message returned to queue (happens on errors implicitly after visibility timeout passes).
	sqsMessagesDeletedTotal             *monitoring.Uint // Number of SQS messages deleted.
	sqsMessageProcessingTime            metrics.Sample   // Histogram of the elapsed SQS processing times in nanoseconds (time of receipt to time of delete/return).

	s3ObjectsRequestedTotal *monitoring.Uint // Number of S3 objects downloaded.
	// s3ObjectsAckedTotal is the number of S3 objects processed that were fully ACKed.
	s3ObjectsAckedTotal *monitoring.Uint
	// s3ObjectsListedTotal is the number of S3 objects returned by list operations.
	s3ObjectsListedTotal *monitoring.Uint
	// s3ObjectsProcessedTotal is the number of S3 objects that matched file_selectors rules.
	s3ObjectsProcessedTotal *monitoring.Uint
	s3BytesProcessedTotal   *monitoring.Uint // Number of S3 bytes processed.
	s3EventsCreatedTotal    *monitoring.Uint // Number of events created from processing S3 data.
	s3ObjectsInflight       *monitoring.Uint // Number of S3 objects inflight (gauge).
	s3ObjectProcessingTime  metrics.Sample   // Histogram of the elapsed S3 object processing times in nanoseconds (start of download to completion of parsing).
}

// Close removes the metrics from the registry.
func (m *inputMetrics) Close() {
	m.parent.Remove(m.id)
}

func newInputMetrics(parent *monitoring.Registry, id string) *inputMetrics {
	reg := parent.NewRegistry(id)
	monitoring.NewString(reg, "input").Set(inputName)
	monitoring.NewString(reg, "id").Set(id)
	out := &inputMetrics{
		id:                                  id,
		parent:                              reg,
		sqsMessagesReceivedTotal:            monitoring.NewUint(reg, "sqs_messages_received_total"),
		sqsVisibilityTimeoutExtensionsTotal: monitoring.NewUint(reg, "sqs_visibility_timeout_extensions_total"),
		sqsMessagesInflight:                 monitoring.NewUint(reg, "sqs_messages_inflight_gauge"),
		sqsMessagesReturnedTotal:            monitoring.NewUint(reg, "sqs_messages_returned_total"),
		sqsMessagesDeletedTotal:             monitoring.NewUint(reg, "sqs_messages_deleted_total"),
		sqsMessageProcessingTime:            metrics.NewUniformSample(1024),
		s3ObjectsRequestedTotal:             monitoring.NewUint(reg, "s3_objects_requested_total"),
		s3ObjectsAckedTotal:                 monitoring.NewUint(reg, "s3_objects_acked_total"),
		s3ObjectsListedTotal:                monitoring.NewUint(reg, "s3_objects_listed_total"),
		s3ObjectsProcessedTotal:             monitoring.NewUint(reg, "s3_objects_processed_total"),
		s3BytesProcessedTotal:               monitoring.NewUint(reg, "s3_bytes_processed_total"),
		s3EventsCreatedTotal:                monitoring.NewUint(reg, "s3_events_created_total"),
		s3ObjectsInflight:                   monitoring.NewUint(reg, "s3_objects_inflight_gauge"),
		s3ObjectProcessingTime:              metrics.NewUniformSample(1024),
	}
	adapter.NewGoMetrics(reg, "sqs_message_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sqsMessageProcessingTime))
	adapter.NewGoMetrics(reg, "s3_object_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.s3ObjectProcessingTime))
	return out
}

// monitoredReader implements io.Reader and counts the number of bytes read.
type monitoredReader struct {
	reader         io.Reader
	totalBytesRead *monitoring.Uint
}

func newMonitoredReader(r io.Reader, metric *monitoring.Uint) *monitoredReader {
	return &monitoredReader{reader: r, totalBytesRead: metric}
}

func (m *monitoredReader) Read(p []byte) (int, error) {
	n, err := m.reader.Read(p)
	m.totalBytesRead.Add(uint64(n))
	return n, err
}

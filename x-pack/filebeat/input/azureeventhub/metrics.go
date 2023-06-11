// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// newInputMetrics creates a new `*inputMetrics` to track metrics for this input.
func newInputMetrics(id string, parentRegistry *monitoring.Registry) *inputMetrics {
	reg, unregister := inputmon.NewInputRegistry(inputName, id, parentRegistry)
	inputMetrics := inputMetrics{
		unregister:                  unregister,
		eventsReceived:              monitoring.NewUint(reg, "events_received_total"),
		eventsSanitized:             monitoring.NewUint(reg, "events_sanitized_total"),
		eventsDeserializationFailed: monitoring.NewUint(reg, "events_deserialization_failed_total"),
		eventsProcessed:             monitoring.NewUint(reg, "events_processed_failed_total"),
		eventsProcessingTime:        metrics.NewUniformSample(1024), // TODO: set a reasonable value for the sample size.
		recordsReceived:             monitoring.NewUint(reg, "records_received_total"),
		recordsSerializationFailed:  monitoring.NewUint(reg, "records_serializaion_failed_total"),
		recordsDispatchFailed:       monitoring.NewUint(reg, "records_dispatch_failed_total"),
		recordsProcessed:            monitoring.NewUint(reg, "records_processed_total"),
	}
	_ = adapter.
		NewGoMetrics(reg, "events_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(inputMetrics.eventsProcessingTime))

	return &inputMetrics
}

// inputMetrics tracks metrics for this input.
type inputMetrics struct {
	// unregister is the cancel function to call when the input is
	// stopped.
	unregister func()

	eventsReceived              *monitoring.Uint // eventsReceived tracks the number of eventhub events received.
	eventsSanitized             *monitoring.Uint // eventsSanitized tracks the number of eventhub events that were sanitized.
	eventsDeserializationFailed *monitoring.Uint // eventsDeserializationFailed tracks the number of eventhub events that failed to deserialize.
	eventsProcessed             *monitoring.Uint // eventsProcessed tracks the number of eventhub events that were processed.
	eventsProcessingTime        metrics.Sample   // eventsProcessingTime tracks the time it takes to process an event.
	recordsReceived             *monitoring.Uint // recordsReceived tracks the number of records received (events successfully deserialized into records).
	recordsSerializationFailed  *monitoring.Uint // recordsSerializationFailed tracks the number of records that failed to serialize.
	recordsDispatchFailed       *monitoring.Uint // recordsDispatchFailed tracks the number of records that failed to dispatch.
	recordsProcessed            *monitoring.Uint // recordsProcessed tracks the number of records that were processed successfully.
}

// Close removes the metrics from the registry.
func (m *inputMetrics) Close() {
	m.unregister()
}

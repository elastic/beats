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

// newInputMetrics creates a new `*inputMetrics` to track metrics.
func newInputMetrics(id string, parentRegistry *monitoring.Registry) *inputMetrics {
	reg, unregister := inputmon.NewInputRegistry(inputName, id, parentRegistry)
	inputMetrics := inputMetrics{
		unregister: unregister,

		// Messages
		receivedMessages:  monitoring.NewUint(reg, "received_messages_total"),
		receivedBytes:     monitoring.NewUint(reg, "received_bytes_total"),
		sanitizedMessages: monitoring.NewUint(reg, "sanitized_messages_total"),
		processedMessages: monitoring.NewUint(reg, "processed_messages_total"),

		// Events
		receivedEvents: monitoring.NewUint(reg, "received_events_total"),
		sentEvents:     monitoring.NewUint(reg, "sent_events_total"),

		// General
		processingTime: metrics.NewUniformSample(1024), // TODO: set a reasonable value for the sample size.
		decodeErrors:   monitoring.NewUint(reg, "decode_errors_total"),
	}
	_ = adapter.
		NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(inputMetrics.processingTime))

	return &inputMetrics
}

// inputMetrics tracks metrics for this input.
//
// # Messages vs Events
//
// Messages are the raw data received from the eventhub. Here's an example of a
// message:
//
//	{
//	  "records": [
//	    {
//	      "time": "2019-12-17T13:43:44.4946995Z",
//	      "test": "this is some message"
//	    }
//	  ]
//	}
//
// Events are the objects inside the `records` array. Here's an example of an event
// from the above message:
//
//	{
//	  "time": "2019-12-17T13:43:44.4946995Z",
//	  "test": "this is some message"
//	}
type inputMetrics struct {
	// unregister is the cancel function to call when the input is
	// stopping.
	unregister func()

	// Messages
	receivedMessages  *monitoring.Uint // receivedMessages tracks the number of messages received from eventhub.
	receivedBytes     *monitoring.Uint // receivedBytes tracks the number of bytes received from eventhub.
	sanitizedMessages *monitoring.Uint // sanitizedMessages tracks the number of messages that were sanitized successfully.
	processedMessages *monitoring.Uint // processedMessages tracks the number of messages that were processed successfully.

	// Events
	receivedEvents *monitoring.Uint // receivedEvents tracks the number of events received decoding messages.
	sentEvents     *monitoring.Uint // sentEvents tracks the number of events that were sent successfully.

	// General
	processingTime metrics.Sample   // processingTime tracks the time it takes to process a message.
	decodeErrors   *monitoring.Uint // decodeErrors tracks the number of errors that occurred while decoding a message.
}

// Close unregisters the metrics from the registry.
func (m *inputMetrics) Close() {
	if m.unregister != nil {
		m.unregister()
	}
}

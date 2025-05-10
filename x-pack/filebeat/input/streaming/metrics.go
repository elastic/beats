// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister              func()
	url                     *monitoring.String // URL of the input resource
	celEvalErrors           *monitoring.Uint   // number of errors encountered during cel program evaluation
	batchesReceived         *monitoring.Uint   // number of event arrays received
	errorsTotal             *monitoring.Uint   // number of errors encountered
	receivedBytesTotal      *monitoring.Uint   // number of bytes received
	eventsReceived          *monitoring.Uint   // number of events received
	batchesPublished        *monitoring.Uint   // number of event arrays published
	eventsPublished         *monitoring.Uint   // number of events published
	writeControlErrors      *monitoring.Uint   // number of errors encountered while sending write control messages like ping
	celProcessingTime       metrics.Sample     // histogram of the elapsed successful cel program processing times in nanoseconds
	batchProcessingTime     metrics.Sample     // histogram of the elapsed successful batch processing times in nanoseconds (time of receipt to time of ACK for non-empty batches).
	pingMessageSendTime     metrics.Sample     // histogram of the elapsed successful ping message send times in nanoseconds
	pongMessageReceivedTime metrics.Sample     // histogram of the elapsed successful pong message receive times in nanoseconds
}

func newInputMetrics(id string, optionalParent *monitoring.Registry) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, optionalParent)
	out := &inputMetrics{
		unregister:              unreg,
		url:                     monitoring.NewString(reg, "url"),
		celEvalErrors:           monitoring.NewUint(reg, "cel_eval_errors"),
		batchesReceived:         monitoring.NewUint(reg, "batches_received_total"),
		errorsTotal:             monitoring.NewUint(reg, "errors_total"),
		receivedBytesTotal:      monitoring.NewUint(reg, "received_bytes_total"),
		eventsReceived:          monitoring.NewUint(reg, "events_received_total"),
		batchesPublished:        monitoring.NewUint(reg, "batches_published_total"),
		eventsPublished:         monitoring.NewUint(reg, "events_published_total"),
		writeControlErrors:      monitoring.NewUint(reg, "write_control_errors"),
		celProcessingTime:       metrics.NewUniformSample(1024),
		batchProcessingTime:     metrics.NewUniformSample(1024),
		pingMessageSendTime:     metrics.NewUniformSample(1024),
		pongMessageReceivedTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "cel_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.celProcessingTime))
	_ = adapter.NewGoMetrics(reg, "batch_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime))
	_ = adapter.NewGoMetrics(reg, "ping_message_send_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.pingMessageSendTime))
	_ = adapter.NewGoMetrics(reg, "pong_message_received_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.pongMessageReceivedTime))

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}

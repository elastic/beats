// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"

	"github.com/rcrowley/go-metrics"
)

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister         func()
	resource           *monitoring.String // URL-ish of input resource
	celEvalErrors      *monitoring.Uint   // number of errors encountered during cel program evaluation
	errorsTotal        *monitoring.Uint   // number of errors encountered
	receivedBytesTotal *monitoring.Uint   // number of bytes received
	eventsReceived     *monitoring.Uint   // number of events received
	eventsPublished    *monitoring.Uint   // number of events published
	celProcessingTime  metrics.Sample     // histogram of the elapsed successful cel program processing times in nanoseconds
}

func newInputMetrics(id string) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, nil)
	out := &inputMetrics{
		unregister:         unreg,
		resource:           monitoring.NewString(reg, "resource"),
		celEvalErrors:      monitoring.NewUint(reg, "cel_eval_errors"),
		errorsTotal:        monitoring.NewUint(reg, "errors_total"),
		receivedBytesTotal: monitoring.NewUint(reg, "received_bytes_total"),
		eventsReceived:     monitoring.NewUint(reg, "events_received_total"),
		eventsPublished:    monitoring.NewUint(reg, "events_published_total"),
		celProcessingTime:  metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "cel_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.celProcessingTime))

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}

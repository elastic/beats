// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcppubsub

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	ackedMessageCount       *monitoring.Uint // Number of successfully ACKed messages.
	failedAckedMessageCount *monitoring.Uint // Number of failed ACKed messages.
	nackedMessageCount      *monitoring.Uint // Number of NACKed messages.
	bytesProcessedTotal     *monitoring.Uint // Number of bytes processed.
	processingTime          metrics.Sample   // Histogram of the elapsed time for processing an event in nanoseconds.
}

func newInputMetrics(id string, optionalParent *monitoring.Registry) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, optionalParent)

	out := &inputMetrics{
		unregister:              unreg,
		ackedMessageCount:       monitoring.NewUint(reg, "acked_message_total"),
		failedAckedMessageCount: monitoring.NewUint(reg, "failed_acked_message_total"),
		nackedMessageCount:      monitoring.NewUint(reg, "nacked_message_total"),
		bytesProcessedTotal:     monitoring.NewUint(reg, "bytes_processed_total"),
		processingTime:          metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	return out
}

func (m *inputMetrics) Close() {
	if m == nil {
		return
	}
	m.unregister()
}

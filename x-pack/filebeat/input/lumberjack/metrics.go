// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

type inputMetrics struct {
	unregister func()

	bindAddress           *monitoring.String // Bind address of input.
	batchesReceivedTotal  *monitoring.Uint   // Number of Lumberjack batches received (not necessarily processed fully).
	batchesACKedTotal     *monitoring.Uint   // Number of Lumberjack batches ACKed.
	messagesReceivedTotal *monitoring.Uint   // Number of Lumberjack messages received (not necessarily processed fully).
	batchProcessingTime   metrics.Sample     // Histogram of the elapsed batch processing times in nanoseconds (time of receipt to time of ACK for non-empty batches).
}

// Close removes the metrics from the registry.
func (m *inputMetrics) Close() {
	m.unregister()
}

func newInputMetrics(id string, optionalParent *monitoring.Registry) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, optionalParent)

	out := &inputMetrics{
		unregister:            unreg,
		bindAddress:           monitoring.NewString(reg, "bind_address"),
		batchesReceivedTotal:  monitoring.NewUint(reg, "batches_received_total"),
		batchesACKedTotal:     monitoring.NewUint(reg, "batches_acked_total"),
		messagesReceivedTotal: monitoring.NewUint(reg, "messages_received_total"),
		batchProcessingTime:   metrics.NewUniformSample(1024),
	}
	adapter.NewGoMetrics(reg, "batch_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.

	return out
}

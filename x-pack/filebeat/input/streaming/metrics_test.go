// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestInputMetricsClose asserts that metrics registered by this input are
// removed after Close() is called. This is important because an input with
// the same ID could be re-registered, and that ID cannot exist in the
// monitoring registry.
func TestInputMetricsClose(t *testing.T) {
	reg := monitoring.NewRegistry()

	metrics := newInputMetrics("streaming-8b312b5f-9f99-492c-b035-3dff354a1f01", reg)
	metrics.Close()

	reg.Do(monitoring.Full, func(s string, _ interface{}) {
		t.Errorf("registry should be empty, but found %v", s)
	})
}

// TestNewInputMetricsInstance asserts that all the metrics are initialized
// when a newInputMetrics method is invoked. This avoids nil hit panics when
// a getter is invoked on any uninitialized metric.
func TestNewInputMetricsInstance(t *testing.T) {
	reg := monitoring.NewRegistry()
	metrics := newInputMetrics("streaming-metric-test", reg)

	assert.NotNil(t,
		metrics.errorsTotal,
		metrics.celEvalErrors,
		metrics.batchesReceived,
		metrics.receivedBytesTotal,
		metrics.eventsReceived,
		metrics.batchesPublished,
		metrics.eventsPublished,
		metrics.writeControlErrors,
		metrics.celProcessingTime,
		metrics.batchProcessingTime,
		metrics.pingMessageSendTime,
		metrics.pongMessageReceivedTime,
	)

	assert.Equal(t, uint64(0x0), metrics.errorsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.celEvalErrors.Get())
	assert.Equal(t, uint64(0x0), metrics.batchesReceived.Get())
	assert.Equal(t, uint64(0x0), metrics.receivedBytesTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.eventsReceived.Get())
	assert.Equal(t, uint64(0x0), metrics.batchesPublished.Get())
	assert.Equal(t, uint64(0x0), metrics.eventsPublished.Get())
	assert.Equal(t, uint64(0x0), metrics.writeControlErrors.Get())
	assert.Equal(t, int64(0), metrics.celProcessingTime.Count())
	assert.Equal(t, int64(0), metrics.batchProcessingTime.Count())
	assert.Equal(t, int64(0), metrics.pingMessageSendTime.Count())
	assert.Equal(t, int64(0), metrics.pongMessageReceivedTime.Count())
}

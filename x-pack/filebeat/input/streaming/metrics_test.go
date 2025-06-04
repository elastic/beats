// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestInputMetricsClose asserts that metrics registered by this input are
// removed after Close() is called. This is important because an input with
// the same ID could be re-registered, and that ID cannot exist in the
// monitoring registry.
func TestInputMetricsClose(t *testing.T) {
	t.Skip("with https://github.com/elastic/beats/pull/42618 I don't believe this test is needed anymore. I'm letting it here so it's confirmed if the test is indeed needed or not before merging the PR.")
	reg := inputmon.NewMetricsRegistry(
		"", "", monitoring.NewRegistry(), logp.NewLogger("test"))
	env := v2.Context{
		ID:              "streaming-8b312b5f-9f99-492c-b035-3dff354a1f01",
		MetricsRegistry: monitoring.NewRegistry(),
	}

	// TODO:(AndersonQ): what is actually tested here?
	_ = newInputMetrics(env)

	reg.Do(monitoring.Full, func(s string, _ interface{}) {
		t.Errorf("registry should be empty, but found %v", s)
	})
}

// TestNewInputMetricsInstance asserts that all the metrics are initialized
// when a newInputMetrics method is invoked. This avoids nil hit panics when
// a getter is invoked on any uninitialized metric.
func TestNewInputMetricsInstance(t *testing.T) {
	reg := monitoring.NewRegistry()
	env := v2.Context{
		ID:              "streaming-metric-test",
		MetricsRegistry: reg,
	}
	metrics := newInputMetrics(env)

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

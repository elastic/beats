// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestInputMetricsClose asserts that metrics registered by this input are
// removed after Close() is called. This is important because an input with
// the same ID could be re-registered, and that ID cannot exist in the
// monitoring registry.
func TestInputMetricsClose(t *testing.T) {
	// Replicates how the v2 input creates the metrics registry for a context
	agent := beat.Info{Monitoring: beat.Monitoring{
		Namespace: monitoring.GetNamespace("TestInputMetricsClose")}}
	logger := logp.NewLogger("test")
	reg := inputmon.NewMetricsRegistry(
		"test",
		inputName,
		// It's the parent registry, where the input registry will be created.
		agent.Monitoring.NamespaceRegistry(),
		logger)
	ctx := v2.Context{MetricsRegistry: reg}

	_ = newInputMetrics(ctx, 1)

	// Replicates how the v2 input cancels the metrics registry after the input
	// is done.
	inputmon.CancelMetricsRegistry("test", inputName,
		agent.Monitoring.NamespaceRegistry(), logger)

	// Assert the input registry was removed from the parent registry.
	agent.Monitoring.NamespaceRegistry().
		Do(monitoring.Full, func(s string, _ interface{}) {
			t.Errorf("registry should be empty, but found %v", s)
		})
}

// TestNewInputMetricsInstance asserts that all the metrics are initialized
// when a newInputMetrics method is invoked. This avoids nil hit panics when
// a getter is invoked on any uninitialized metric.
func TestNewInputMetricsInstance(t *testing.T) {
	metrics := newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 1)

	assert.NotNil(t, metrics.sqsMessagesWaiting,
		metrics.sqsMaxMessagesInflight,
		metrics.sqsWorkerStartTimes,
		metrics.sqsWorkerUtilizationLastUpdate,
		metrics.sqsMessagesReceivedTotal,
		metrics.sqsVisibilityTimeoutExtensionsTotal,
		metrics.sqsMessagesInflight,
		metrics.sqsMessagesReturnedTotal,
		metrics.sqsMessagesDeletedTotal,
		metrics.sqsMessagesWaiting,
		metrics.sqsWorkerUtilization,
		metrics.sqsMessageProcessingTime,
		metrics.sqsLagTime,
		metrics.s3ObjectsRequestedTotal,
		metrics.s3ObjectsAckedTotal,
		metrics.s3ObjectsListedTotal,
		metrics.s3ObjectsProcessedTotal,
		metrics.s3BytesProcessedTotal,
		metrics.s3EventsCreatedTotal,
		metrics.s3ObjectsInflight,
		metrics.s3ObjectProcessingTime)

	assert.Equal(t, int64(-1), metrics.sqsMessagesWaiting.Get())
}

func TestInputMetricsSQSWorkerUtilization(t *testing.T) {
	const interval = 5000

	t.Run("worker ends before one interval", func(t *testing.T) {
		fakeTimeMs.Store(0)
		defer useFakeCurrentTimeThenReset()()

		metrics := newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 1)
		metrics.Close()

		id := metrics.beginSQSWorker()
		fakeTimeMs.Add(2500)
		metrics.endSQSWorker(id)

		fakeTimeMs.Store(1 * interval)
		metrics.updateSqsWorkerUtilization()
		assert.Equal(t, 0.5, metrics.sqsWorkerUtilization.Get())
	})
	t.Run("worker ends mid interval", func(t *testing.T) {
		fakeTimeMs.Store(0)
		defer useFakeCurrentTimeThenReset()()

		metrics := newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 1)
		metrics.Close()

		fakeTimeMs.Add(4000)
		id := metrics.beginSQSWorker()

		fakeTimeMs.Store(1 * interval)
		metrics.updateSqsWorkerUtilization()

		fakeTimeMs.Add(1000)
		metrics.endSQSWorker(id)

		fakeTimeMs.Store(2 * interval)
		metrics.updateSqsWorkerUtilization()
		assert.Equal(t, 0.2, metrics.sqsWorkerUtilization.Get())
	})
	t.Run("running worker goes longer than an interval", func(t *testing.T) {
		fakeTimeMs.Store(0)
		defer useFakeCurrentTimeThenReset()()

		metrics := newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 1)
		metrics.Close()

		id := metrics.beginSQSWorker()

		fakeTimeMs.Store(1 * interval)
		metrics.updateSqsWorkerUtilization()
		assert.Equal(t, 1.0, metrics.sqsWorkerUtilization.Get())

		fakeTimeMs.Store(2 * interval)
		metrics.updateSqsWorkerUtilization()
		assert.Equal(t, 1.0, metrics.sqsWorkerUtilization.Get())

		metrics.endSQSWorker(id)
	})
}

var fakeTimeMs = &atomic.Int64{}

func useFakeCurrentTimeThenReset() (reset func()) {
	clockValue.Swap(clock{
		Now: func() time.Time {
			return time.UnixMilli(fakeTimeMs.Load())
		},
	})
	reset = func() {
		clockValue.Swap(realClock)
	}
	return reset
}

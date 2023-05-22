// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestInputMetricsClose asserts that metrics registered by this input are
// removed after Close() is called. This is important because an input with
// the same ID could be re-registered, and that ID cannot exist in the
// monitoring registry.
func TestInputMetricsClose(t *testing.T) {
	reg := monitoring.NewRegistry()

	metrics := newInputMetrics("aws-s3-aws.cloudfront_logs-8b312b5f-9f99-492c-b035-3dff354a1f01", reg, 1)
	metrics.Close()

	reg.Do(monitoring.Full, func(s string, _ interface{}) {
		t.Errorf("registry should be empty, but found %v", s)
	})
}

func TestInputMetricsSQSWorkerUtilization(t *testing.T) {
	const interval = 5000

	t.Run("worker ends before one interval", func(t *testing.T) {
		fakeTimeMs.Store(0)
		defer useFakeCurrentTimeThenReset()()

		reg := monitoring.NewRegistry()
		metrics := newInputMetrics("test", reg, 1)
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

		reg := monitoring.NewRegistry()
		metrics := newInputMetrics("test", reg, 1)
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

		reg := monitoring.NewRegistry()
		metrics := newInputMetrics("test", reg, 1)
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

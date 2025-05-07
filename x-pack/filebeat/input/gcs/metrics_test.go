// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

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

	metrics := newInputMetrics("gcs-cl-bucket.cloudflare_logs-8b312b5f-9f99-492c-b035-3dff354a1f01", reg)
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
	metrics := newInputMetrics("gcs-new-metric-test", reg)

	assert.NotNil(t, metrics.errorsTotal,
		metrics.decodeErrorsTotal,
		metrics.gcsObjectsTracked,
		metrics.gcsObjectsRequestedTotal,
		metrics.gcsObjectsPublishedTotal,
		metrics.gcsObjectsListedTotal,
		metrics.gcsBytesProcessedTotal,
		metrics.gcsEventsCreatedTotal,
		metrics.gcsFailedJobsTotal,
		metrics.gcsExpiredFailedJobsTotal,
		metrics.gcsObjectsInflight,
		metrics.gcsObjectProcessingTime,
		metrics.gcsObjectSizeInBytes,
		metrics.gcsEventsPerObject,
		metrics.gcsJobsScheduledAfterValidation,
		metrics.sourceLagTime,
	)

	assert.Equal(t, uint64(0x0), metrics.errorsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.decodeErrorsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsObjectsTracked.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsObjectsRequestedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsObjectsPublishedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsObjectsListedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsBytesProcessedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsEventsCreatedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsFailedJobsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsExpiredFailedJobsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.gcsObjectsInflight.Get())

}

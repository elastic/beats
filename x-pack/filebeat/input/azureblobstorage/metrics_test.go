// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

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

	metrics := newInputMetrics("abs-cl-bucket.cloudflare_logs-8b312b5f-9f99-492c-b035-3dff354a1f01", reg)
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
	metrics := newInputMetrics("abs-new-metric-test", reg)

	assert.NotNil(t, metrics.errorsTotal,
		metrics.decodeErrorsTotal,
		metrics.absBlobsRequestedTotal,
		metrics.absBlobsPublishedTotal,
		metrics.absBlobsListedTotal,
		metrics.absBytesProcessedTotal,
		metrics.absEventsCreatedTotal,
		metrics.absBlobsInflight,
		metrics.absBlobProcessingTime,
		metrics.absBlobSizeInBytes,
		metrics.absEventsPerBlob,
		metrics.absJobsScheduledAfterValidation,
		metrics.sourceLagTime,
	)

	assert.Equal(t, uint64(0x0), metrics.errorsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.decodeErrorsTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.absBlobsRequestedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.absBlobsPublishedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.absBlobsListedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.absBytesProcessedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.absEventsCreatedTotal.Get())
	assert.Equal(t, uint64(0x0), metrics.absBlobsInflight.Get())
	assert.Equal(t, int64(0), metrics.absBlobProcessingTime.Count())
	assert.Equal(t, int64(0), metrics.absBlobSizeInBytes.Count())
	assert.Equal(t, int64(0), metrics.absEventsPerBlob.Count())
	assert.Equal(t, int64(0), metrics.absJobsScheduledAfterValidation.Count())
	assert.Equal(t, int64(0), metrics.sourceLagTime.Count())
}

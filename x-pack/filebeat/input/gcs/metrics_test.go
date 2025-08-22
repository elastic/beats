// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestNewInputMetricsInstance asserts that all the metrics are initialized
// when a newInputMetrics method is invoked. This avoids nil hit panics when
// a getter is invoked on any uninitialized metric.
func TestNewInputMetricsInstance(t *testing.T) {
	metrics := newInputMetrics(monitoring.NewRegistry())

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

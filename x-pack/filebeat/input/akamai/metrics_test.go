// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestInputMetricsInit(t *testing.T) {
	assert.Nil(t, newInputMetrics(nil, 1, logp.NewNopLogger()))

	m := newInputMetrics(monitoring.NewRegistry(), 2, logp.NewNopLogger())
	require.NotNil(t, m)
	defer m.Close()

	assert.NotNil(t, m.resource)
	assert.NotNil(t, m.requestsTotal)
	assert.NotNil(t, m.requestsSuccess)
	assert.NotNil(t, m.requestsErrors)
	assert.NotNil(t, m.batchesReceived)
	assert.NotNil(t, m.batchesPublished)
	assert.NotNil(t, m.eventsReceived)
	assert.NotNil(t, m.eventsPublished)
	assert.NotNil(t, m.eventsPublishFailed)
	assert.NotNil(t, m.errorsTotal)
	assert.NotNil(t, m.offsetExpired)
	assert.NotNil(t, m.offsetTTLDrops)
	assert.NotNil(t, m.fromClamped)
	assert.NotNil(t, m.hmacRefreshes)
	assert.NotNil(t, m.api400Fatal)
	assert.NotNil(t, m.cursorDrops)
	assert.NotNil(t, m.workersActive)
	assert.NotNil(t, m.workerUtilization)
	assert.NotNil(t, m.requestProcessingTime)
	assert.NotNil(t, m.batchProcessingTime)
	assert.NotNil(t, m.eventsPerBatch)
	assert.NotNil(t, m.failedEventsPerPage)
	assert.NotNil(t, m.responseLatency)

	assert.EqualValues(t, 0, m.requestsTotal.Get())
	assert.EqualValues(t, 0, m.requestsSuccess.Get())
	assert.EqualValues(t, 0, m.requestsErrors.Get())
	assert.EqualValues(t, 0, m.batchesReceived.Get())
	assert.EqualValues(t, 0, m.batchesPublished.Get())
	assert.EqualValues(t, 0, m.eventsReceived.Get())
	assert.EqualValues(t, 0, m.eventsPublished.Get())
	assert.EqualValues(t, 0, m.eventsPublishFailed.Get())
	assert.EqualValues(t, 0, m.errorsTotal.Get())
	assert.EqualValues(t, 0, m.offsetExpired.Get())
	assert.EqualValues(t, 0, m.offsetTTLDrops.Get())
	assert.EqualValues(t, 0, m.fromClamped.Get())
	assert.EqualValues(t, 0, m.hmacRefreshes.Get())
	assert.EqualValues(t, 0, m.api400Fatal.Get())
	assert.EqualValues(t, 0, m.cursorDrops.Get())
	assert.EqualValues(t, 0, m.workersActive.Get())
	assert.EqualValues(t, 0, m.requestProcessingTime.Count())
	assert.EqualValues(t, 0, m.batchProcessingTime.Count())
	assert.EqualValues(t, 0, m.eventsPerBatch.Count())
	assert.EqualValues(t, 0, m.failedEventsPerPage.Count())
	assert.EqualValues(t, 0, m.responseLatency.Count())
}

func TestInputMetricsCountersAndHistograms(t *testing.T) {
	m := newInputMetrics(monitoring.NewRegistry(), 2, logp.NewNopLogger())
	require.NotNil(t, m)
	defer m.Close()

	m.SetResource("https://example.com/siem/v1/configs/1")
	m.AddRequest()
	m.AddRequestSuccess()
	m.AddRequestError()
	m.AddBatchReceived(3)
	m.AddBatchPublished()
	m.AddEventPublished(2)
	m.AddError()
	m.AddOffsetExpired()
	m.AddOffsetTTLDrop()
	m.AddFromClamped()
	m.AddHMACRefresh()
	m.AddAPI400Fatal()
	m.AddPartialPublishFailures(0)
	m.AddPartialPublishFailures(3)
	m.AddCursorDrop()
	m.RecordRequestTime(10 * time.Millisecond)
	m.RecordBatchTime(20 * time.Millisecond)
	m.RecordResponseLatency(5 * time.Millisecond)

	assert.EqualValues(t, 1, m.requestsTotal.Get())
	assert.EqualValues(t, 1, m.requestsSuccess.Get())
	assert.EqualValues(t, 1, m.requestsErrors.Get())
	assert.EqualValues(t, 1, m.batchesReceived.Get())
	assert.EqualValues(t, 1, m.batchesPublished.Get())
	assert.EqualValues(t, 3, m.eventsReceived.Get())
	assert.EqualValues(t, 2, m.eventsPublished.Get())
	assert.EqualValues(t, 2, m.errorsTotal.Get())
	assert.EqualValues(t, 1, m.offsetExpired.Get())
	assert.EqualValues(t, 1, m.offsetTTLDrops.Get())
	assert.EqualValues(t, 1, m.fromClamped.Get())
	assert.EqualValues(t, 1, m.hmacRefreshes.Get())
	assert.EqualValues(t, 1, m.api400Fatal.Get())
	assert.EqualValues(t, 3, m.eventsPublishFailed.Get())
	assert.EqualValues(t, 1, m.failedEventsPerPage.Count())
	assert.EqualValues(t, 1, m.cursorDrops.Get())
	assert.EqualValues(t, 1, m.requestProcessingTime.Count())
	assert.EqualValues(t, 1, m.batchProcessingTime.Count())
	assert.EqualValues(t, 1, m.eventsPerBatch.Count())
	assert.EqualValues(t, 1, m.responseLatency.Count())
}

func TestInputMetricsWorkerTracking(t *testing.T) {
	m := newInputMetrics(monitoring.NewRegistry(), 2, logp.NewNopLogger())
	require.NotNil(t, m)
	defer m.Close()

	id1 := m.BeginWorker()
	id2 := m.BeginWorker()
	assert.NotZero(t, id1)
	assert.NotZero(t, id2)
	assert.EqualValues(t, 2, m.workersActive.Get())

	time.Sleep(5 * time.Millisecond)
	m.EndWorker(id1)
	assert.EqualValues(t, 1, m.workersActive.Get())

	time.Sleep(5 * time.Millisecond)
	m.EndWorker(id2)
	assert.EqualValues(t, 0, m.workersActive.Get())

	m.updateWorkerUtilization()
	util := m.workerUtilization.Get()
	assert.GreaterOrEqual(t, util, float64(0))
	assert.LessOrEqual(t, util, float64(1))
}

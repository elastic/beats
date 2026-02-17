// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/go-concert/timed"
)

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Worker utilization tracking
	maxWorkers             int
	workerUtilizationMutex sync.Mutex
	workerUtilizationLast  time.Time
	workerCurrentPeriod    time.Duration
	workerIDCounter        uint64
	workerStartTimes       map[uint64]time.Time

	// Counters
	resource            *monitoring.String // URL of the input resource
	requestsTotal       *monitoring.Uint   // total number of API requests
	requestsSuccess     *monitoring.Uint   // successful API requests
	requestsErrors      *monitoring.Uint   // failed API requests
	batchesReceived     *monitoring.Uint   // number of event batches received
	batchesPublished    *monitoring.Uint   // number of event batches published
	eventsReceived      *monitoring.Uint   // total events received
	eventsPublished     *monitoring.Uint   // total events published
	eventsPublishFailed *monitoring.Uint   // total failed event publishes
	errorsTotal         *monitoring.Uint   // total errors
	recoveryModeEntries *monitoring.Uint   // times recovery mode was entered
	offsetExpired       *monitoring.Uint   // number of 416 offset expired responses
	hmacRefreshes       *monitoring.Uint   // number of invalid timestamp retries
	api400Fatal         *monitoring.Uint   // number of fatal 400 responses
	cursorDrops         *monitoring.Uint   // number of cursor drops
	workersActive       *monitoring.Uint   // currently active workers (gauge)
	workerUtilization   *monitoring.Float  // worker utilization (0-1)

	// Histograms
	requestProcessingTime metrics.Sample // histogram of request processing times
	batchProcessingTime   metrics.Sample // histogram of batch processing times
	eventsPerBatch        metrics.Sample // histogram of events per batch
	failedEventsPerPage   metrics.Sample // histogram of failed events per page
	responseLatency       metrics.Sample // histogram of API response latencies
}

func newInputMetrics(reg *monitoring.Registry, maxWorkers int, log *logp.Logger) *inputMetrics {
	if reg == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	out := &inputMetrics{
		ctx:                   ctx,
		cancel:                cancel,
		maxWorkers:            maxWorkers,
		workerUtilizationLast: time.Now(),
		workerStartTimes:      make(map[uint64]time.Time),

		// Initialize counters
		resource:            monitoring.NewString(reg, "resource"),
		requestsTotal:       monitoring.NewUint(reg, "akamai_requests_total"),
		requestsSuccess:     monitoring.NewUint(reg, "akamai_requests_success_total"),
		requestsErrors:      monitoring.NewUint(reg, "akamai_requests_errors_total"),
		batchesReceived:     monitoring.NewUint(reg, "batches_received_total"),
		batchesPublished:    monitoring.NewUint(reg, "batches_published_total"),
		eventsReceived:      monitoring.NewUint(reg, "events_received_total"),
		eventsPublished:     monitoring.NewUint(reg, "events_published_total"),
		errorsTotal:         monitoring.NewUint(reg, "errors_total"),
		recoveryModeEntries: monitoring.NewUint(reg, "recovery_mode_entries_total"),
		offsetExpired:       monitoring.NewUint(reg, "offset_expired_total"),
		hmacRefreshes:       monitoring.NewUint(reg, "hmac_refresh_total"),
		api400Fatal:         monitoring.NewUint(reg, "api_400_fatal_total"),
		eventsPublishFailed: monitoring.NewUint(reg, "events_publish_failed_total"),
		cursorDrops:         monitoring.NewUint(reg, "cursor_drops_total"),
		workersActive:       monitoring.NewUint(reg, "workers_active_gauge"),
		workerUtilization:   monitoring.NewFloat(reg, "worker_utilization"),

		// Initialize histograms
		requestProcessingTime: metrics.NewUniformSample(1024),
		batchProcessingTime:   metrics.NewUniformSample(1024),
		eventsPerBatch:        metrics.NewUniformSample(1024),
		failedEventsPerPage:   metrics.NewUniformSample(1024),
		responseLatency:       metrics.NewUniformSample(1024),
	}

	// Register histograms with the monitoring adapter
	_ = adapter.NewGoMetrics(reg, "request_processing_time", log, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.requestProcessingTime))
	_ = adapter.NewGoMetrics(reg, "batch_processing_time", log, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime))
	_ = adapter.NewGoMetrics(reg, "events_per_batch", log, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.eventsPerBatch))
	_ = adapter.NewGoMetrics(reg, "failed_events_per_page", log, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.failedEventsPerPage))
	_ = adapter.NewGoMetrics(reg, "response_latency", log, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.responseLatency))

	// Start periodic worker utilization updates
	if maxWorkers > 0 {
		go timed.Periodic(ctx, 5*time.Second, func() error { //nolint:errcheck // never returns error
			out.updateWorkerUtilization()
			return nil
		})
	}

	return out
}

// Close cancels the context and stops the periodic worker utilization updates.
func (m *inputMetrics) Close() {
	if m == nil {
		return
	}
	m.cancel()
}

// SetResource sets the resource URL metric.
func (m *inputMetrics) SetResource(url string) {
	if m == nil {
		return
	}
	m.resource.Set(url)
}

// AddRequest increments the request counter.
func (m *inputMetrics) AddRequest() {
	if m == nil {
		return
	}
	m.requestsTotal.Inc()
}

// AddRequestSuccess increments the successful request counter.
func (m *inputMetrics) AddRequestSuccess() {
	if m == nil {
		return
	}
	m.requestsSuccess.Inc()
}

// AddRequestError increments the failed request counter.
func (m *inputMetrics) AddRequestError() {
	if m == nil {
		return
	}
	m.requestsErrors.Inc()
	m.errorsTotal.Inc()
}

// AddBatchReceived increments the batches received counter and records events count.
func (m *inputMetrics) AddBatchReceived(eventCount int) {
	if m == nil {
		return
	}
	m.batchesReceived.Inc()
	m.eventsReceived.Add(uint64(eventCount))   //nolint:gosec // eventCount is always positive
	m.eventsPerBatch.Update(int64(eventCount)) //nolint:gosec // eventCount is bounded by event_limit (max 600000)
}

// AddBatchPublished increments the batches published counter.
func (m *inputMetrics) AddBatchPublished() {
	if m == nil {
		return
	}
	m.batchesPublished.Inc()
}

// AddEventPublished increments the events published counter.
func (m *inputMetrics) AddEventPublished(count uint64) {
	if m == nil {
		return
	}
	m.eventsPublished.Add(count)
}

// AddError increments the error counter.
func (m *inputMetrics) AddError() {
	if m == nil {
		return
	}
	m.errorsTotal.Inc()
}

// AddRecoveryModeEntry increments the recovery mode entry counter.
func (m *inputMetrics) AddRecoveryModeEntry() {
	if m == nil {
		return
	}
	m.recoveryModeEntries.Inc()
}

// AddOffsetExpired increments the 416 offset expired counter.
func (m *inputMetrics) AddOffsetExpired() {
	if m == nil {
		return
	}
	m.offsetExpired.Inc()
}

// AddHMACRefresh increments the invalid timestamp retry counter.
func (m *inputMetrics) AddHMACRefresh() {
	if m == nil {
		return
	}
	m.hmacRefreshes.Inc()
}

// AddAPI400Fatal increments the fatal 400 error counter.
func (m *inputMetrics) AddAPI400Fatal() {
	if m == nil {
		return
	}
	m.api400Fatal.Inc()
}

// AddPartialPublishFailures increments the partial publish failures counter.
func (m *inputMetrics) AddPartialPublishFailures(count uint64) {
	if m == nil || count == 0 {
		return
	}
	m.eventsPublishFailed.Add(count)
	m.failedEventsPerPage.Update(int64(count)) //nolint:gosec // count is bounded by event_limit (max 600000), safe for int64
}

// AddCursorDrop increments the cursor drop counter.
func (m *inputMetrics) AddCursorDrop() {
	if m == nil {
		return
	}
	m.cursorDrops.Inc()
}

// RecordRequestTime records the request processing time.
func (m *inputMetrics) RecordRequestTime(d time.Duration) {
	if m == nil {
		return
	}
	m.requestProcessingTime.Update(d.Nanoseconds())
}

// RecordBatchTime records the batch processing time.
func (m *inputMetrics) RecordBatchTime(d time.Duration) {
	if m == nil {
		return
	}
	m.batchProcessingTime.Update(d.Nanoseconds())
}

// RecordResponseLatency records the API response latency.
func (m *inputMetrics) RecordResponseLatency(d time.Duration) {
	if m == nil {
		return
	}
	m.responseLatency.Update(d.Nanoseconds())
}

// BeginWorker tracks the start of a new worker. Returns an ID that must be used
// to call EndWorker when the worker finishes.
func (m *inputMetrics) BeginWorker() uint64 {
	if m == nil {
		return 0
	}
	m.workersActive.Inc()

	m.workerUtilizationMutex.Lock()
	defer m.workerUtilizationMutex.Unlock()
	m.workerIDCounter++
	m.workerStartTimes[m.workerIDCounter] = time.Now()
	return m.workerIDCounter
}

// EndWorker signals that the specified worker has finished.
func (m *inputMetrics) EndWorker(id uint64) {
	if m == nil {
		return
	}
	m.workersActive.Dec()

	m.workerUtilizationMutex.Lock()
	defer m.workerUtilizationMutex.Unlock()

	now := time.Now()
	start, ok := m.workerStartTimes[id]
	if !ok {
		return
	}
	delete(m.workerStartTimes, id)

	if start.Before(m.workerUtilizationLast) {
		m.workerCurrentPeriod += now.Sub(m.workerUtilizationLast)
	} else {
		m.workerCurrentPeriod += now.Sub(start)
	}
}

// updateWorkerUtilization updates the worker utilization metric.
// This is called periodically to compute utilization over time.
func (m *inputMetrics) updateWorkerUtilization() {
	if m == nil {
		return
	}

	m.workerUtilizationMutex.Lock()
	defer m.workerUtilizationMutex.Unlock()

	now := time.Now()
	periodDuration := now.Sub(m.workerUtilizationLast)
	maxUtilization := float64(m.maxWorkers) * periodDuration.Seconds()

	// Add utilization from workers that are still running
	for _, startTime := range m.workerStartTimes {
		if startTime.Before(m.workerUtilizationLast) {
			m.workerCurrentPeriod += periodDuration
		} else {
			m.workerCurrentPeriod += now.Sub(startTime)
		}
	}

	utilization := math.Round(m.workerCurrentPeriod.Seconds()/maxUtilization*1000) / 1000
	if utilization > 1 {
		utilization = 1
	}
	if utilization < 0 || math.IsNaN(utilization) {
		utilization = 0
	}

	m.workerUtilization.Set(utilization)
	m.workerCurrentPeriod = 0
	m.workerUtilizationLast = now
}

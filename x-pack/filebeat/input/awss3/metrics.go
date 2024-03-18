// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/go-concert/timed"
)

var (
	clockValue atomic.Value // Atomic reference to a clock value.
	realClock  = clock{Now: time.Now}
)

type clock struct {
	Now func() time.Time
}

func init() {
	clockValue.Store(realClock)
}

// currentTime returns the current time. This exists to allow unit tests
// simulate the passage of time.
func currentTime() time.Time {
	clock := clockValue.Load().(clock)
	return clock.Now()
}

type inputMetrics struct {
	registry   *monitoring.Registry
	unregister func()
	ctx        context.Context    // ctx signals when to stop the sqs worker utilization goroutine.
	cancel     context.CancelFunc // cancel cancels the ctx context.

	sqsMaxMessagesInflight            int                  // Maximum number of SQS workers allowed.
	sqsWorkerUtilizationMutex         sync.Mutex           // Guards the sqs worker utilization fields.
	sqsWorkerUtilizationLastUpdate    time.Time            // Time of the last SQS worker utilization calculation.
	sqsWorkerUtilizationCurrentPeriod time.Duration        // Elapsed execution duration of any SQS workers that completed during the current period.
	sqsWorkerIDCounter                uint64               // Counter used to assigned unique IDs to SQS workers.
	sqsWorkerStartTimes               map[uint64]time.Time // Map of SQS worker ID to the time at which the worker started.

	sqsMessagesReceivedTotal            *monitoring.Uint  // Number of SQS messages received (not necessarily processed fully).
	sqsMessagesProcessedTotal           *monitoring.Uint  // Number of SQS messages processed fully.
	sqsVisibilityTimeoutExtensionsTotal *monitoring.Uint  // Number of SQS visibility timeout extensions.
	sqsMessagesInflight                 *monitoring.Uint  // Number of SQS messages inflight (gauge).
	sqsMessagesReturnedTotal            *monitoring.Uint  // Number of SQS message returned to queue (happens on errors implicitly after visibility timeout passes).
	sqsMessagesDeletedTotal             *monitoring.Uint  // Number of SQS messages deleted.
	sqsMessagesWaiting                  *monitoring.Int   // Number of SQS messages waiting in the SQS queue (gauge). The value is refreshed every minute via data from GetQueueAttributes.
	sqsWorkerUtilization                *monitoring.Float // Rate of SQS worker utilization over previous 5 seconds. 0 indicates idle, 1 indicates all workers utilized.
	sqsMessageProcessingTime            metrics.Sample    // Histogram of the elapsed SQS processing times in nanoseconds (time of receipt to time of delete/return).
	sqsLagTime                          metrics.Sample    // Histogram of the difference between the SQS SentTimestamp attribute and the time when the SQS message was received expressed in nanoseconds.

	s3ObjectsRequestedTotal *monitoring.Uint // Number of S3 objects downloaded.
	s3ObjectsAckedTotal     *monitoring.Uint // Number of S3 objects processed that were fully ACKed.
	s3ObjectsListedTotal    *monitoring.Uint // Number of S3 objects returned by list operations.
	s3ObjectsProcessedTotal *monitoring.Uint // Number of S3 objects that matched file_selectors rules.
	s3BytesProcessedTotal   *monitoring.Uint // Number of S3 bytes processed.
	s3EventsCreatedTotal    *monitoring.Uint // Number of events created from processing S3 data.
	s3ObjectsInflight       *monitoring.Uint // Number of S3 objects inflight (gauge).
	s3ObjectProcessingTime  metrics.Sample   // Histogram of the elapsed S3 object processing times in nanoseconds (start of download to completion of parsing).
}

// Close cancels the context and removes the metrics from the registry.
func (m *inputMetrics) Close() {
	m.cancel()
	m.unregister()
}

// beginSQSWorker tracks the start of a new SQS worker. The returned ID
// must be used to call endSQSWorker when the worker finishes. It also
// increments the sqsMessagesInflight counter.
func (m *inputMetrics) beginSQSWorker() (id uint64) {
	m.sqsWorkerUtilizationMutex.Lock()
	defer m.sqsWorkerUtilizationMutex.Unlock()
	m.sqsWorkerIDCounter++
	m.sqsWorkerStartTimes[m.sqsWorkerIDCounter] = currentTime()
	return m.sqsWorkerIDCounter
}

// endSQSWorker is used to signal that the specified worker has
// finished. This is used update the SQS worker utilization metric.
// It also decrements the sqsMessagesInflight counter and
// sqsMessageProcessingTime histogram.
func (m *inputMetrics) endSQSWorker(id uint64) {
	m.sqsWorkerUtilizationMutex.Lock()
	defer m.sqsWorkerUtilizationMutex.Unlock()
	now := currentTime()
	start := m.sqsWorkerStartTimes[id]
	delete(m.sqsWorkerStartTimes, id)
	m.sqsMessageProcessingTime.Update(now.Sub(start).Nanoseconds())
	if start.Before(m.sqsWorkerUtilizationLastUpdate) {
		m.sqsWorkerUtilizationCurrentPeriod += now.Sub(m.sqsWorkerUtilizationLastUpdate)
	} else {
		m.sqsWorkerUtilizationCurrentPeriod += now.Sub(start)
	}
}

// updateSqsWorkerUtilization updates the sqsWorkerUtilization metric.
// This is invoked periodically to compute the utilization level
// of the SQS workers. 0 indicates no workers were utilized during
// the period. And 1 indicates that all workers fully utilized
// during the period.
func (m *inputMetrics) updateSqsWorkerUtilization() {
	m.sqsWorkerUtilizationMutex.Lock()
	defer m.sqsWorkerUtilizationMutex.Unlock()

	now := currentTime()
	lastPeriodDuration := now.Sub(m.sqsWorkerUtilizationLastUpdate)
	maxUtilization := float64(m.sqsMaxMessagesInflight) * lastPeriodDuration.Seconds()

	for _, startTime := range m.sqsWorkerStartTimes {
		// If the worker started before the current period then only compute
		// from elapsed time since the last update. Otherwise, it started
		// during the current period so compute time elapsed since it started.
		if startTime.Before(m.sqsWorkerUtilizationLastUpdate) {
			m.sqsWorkerUtilizationCurrentPeriod += lastPeriodDuration
		} else {
			m.sqsWorkerUtilizationCurrentPeriod += now.Sub(startTime)
		}
	}

	utilization := math.Round(m.sqsWorkerUtilizationCurrentPeriod.Seconds()/maxUtilization*1000) / 1000
	if utilization > 1 {
		utilization = 1
	}
	m.sqsWorkerUtilization.Set(utilization)
	m.sqsWorkerUtilizationCurrentPeriod = 0
	m.sqsWorkerUtilizationLastUpdate = now
}

func newInputMetrics(id string, optionalParent *monitoring.Registry, maxWorkers int) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, optionalParent)
	ctx, cancel := context.WithCancel(context.Background())

	out := &inputMetrics{
		registry:                            reg,
		unregister:                          unreg,
		ctx:                                 ctx,
		cancel:                              cancel,
		sqsMaxMessagesInflight:              maxWorkers,
		sqsWorkerStartTimes:                 map[uint64]time.Time{},
		sqsWorkerUtilizationLastUpdate:      currentTime(),
		sqsMessagesReceivedTotal:            monitoring.NewUint(reg, "sqs_messages_received_total"),
		sqsMessagesProcessedTotal:           monitoring.NewUint(reg, "sqs_messages_processed_total"),
		sqsVisibilityTimeoutExtensionsTotal: monitoring.NewUint(reg, "sqs_visibility_timeout_extensions_total"),
		sqsMessagesInflight:                 monitoring.NewUint(reg, "sqs_messages_inflight_gauge"),
		sqsMessagesReturnedTotal:            monitoring.NewUint(reg, "sqs_messages_returned_total"),
		sqsMessagesDeletedTotal:             monitoring.NewUint(reg, "sqs_messages_deleted_total"),
		sqsMessagesWaiting:                  monitoring.NewInt(reg, "sqs_messages_waiting_gauge"),
		sqsWorkerUtilization:                monitoring.NewFloat(reg, "sqs_worker_utilization"),
		sqsMessageProcessingTime:            metrics.NewUniformSample(1024),
		sqsLagTime:                          metrics.NewUniformSample(1024),
		s3ObjectsRequestedTotal:             monitoring.NewUint(reg, "s3_objects_requested_total"),
		s3ObjectsAckedTotal:                 monitoring.NewUint(reg, "s3_objects_acked_total"),
		s3ObjectsListedTotal:                monitoring.NewUint(reg, "s3_objects_listed_total"),
		s3ObjectsProcessedTotal:             monitoring.NewUint(reg, "s3_objects_processed_total"),
		s3BytesProcessedTotal:               monitoring.NewUint(reg, "s3_bytes_processed_total"),
		s3EventsCreatedTotal:                monitoring.NewUint(reg, "s3_events_created_total"),
		s3ObjectsInflight:                   monitoring.NewUint(reg, "s3_objects_inflight_gauge"),
		s3ObjectProcessingTime:              metrics.NewUniformSample(1024),
	}

	// Initializing the sqs_messages_waiting_gauge value to -1 so that we can distinguish between no messages waiting (0) and never collected / error collecting (-1).
	out.sqsMessagesWaiting.Set(int64(-1))
	adapter.NewGoMetrics(reg, "sqs_message_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sqsMessageProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "sqs_lag_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sqsLagTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.
	adapter.NewGoMetrics(reg, "s3_object_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.s3ObjectProcessingTime)) //nolint:errcheck // A unique namespace is used so name collisions are impossible.

	if maxWorkers > 0 {
		// Periodically update the sqs worker utilization metric.
		//nolint:errcheck // This never returns an error.
		go timed.Periodic(ctx, 5*time.Second, func() error {
			out.updateSqsWorkerUtilization()
			return nil
		})
	}

	return out
}

// monitoredReader implements io.Reader and counts the number of bytes read.
type monitoredReader struct {
	reader         io.Reader
	totalBytesRead *monitoring.Uint
}

func newMonitoredReader(r io.Reader, metric *monitoring.Uint) *monitoredReader {
	return &monitoredReader{reader: r, totalBytesRead: metric}
}

func (m *monitoredReader) Read(p []byte) (int, error) {
	n, err := m.reader.Read(p)
	m.totalBytesRead.Add(uint64(n))
	return n, err
}

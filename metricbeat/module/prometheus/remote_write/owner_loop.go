// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package remote_write

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

var intakeNeverClosed = make(chan struct{})

// ErrRemoteWriteCapacityExceeded is returned by optional capacity checkers when a batch must be rejected.
var ErrRemoteWriteCapacityExceeded = errors.New("remote write capacity exceeded")

// RemoteWriteCapacityChecker optionally rejects batches before generator state is mutated.
// Any non-nil error prevents GenerateEvents from running. Use ErrRemoteWriteCapacityExceeded
// for retryable admission control; other errors are treated as internal failures.
type RemoteWriteCapacityChecker interface {
	CheckCapacity(samples model.Samples) error
}

// RemoteWriteOwnerLoopBatchProcessor optionally owns classify → capacity check → commit for
// one owner-loop batch. Capacity rejection must return ErrRemoteWriteCapacityExceeded with no
// generator mutations. Other errors are treated as internal failures.
type RemoteWriteOwnerLoopBatchProcessor interface {
	ProcessOwnerLoopBatch(samples model.Samples) (map[string]mb.Event, error)
}

// RemoteWriteFlushCapable optionally emits buffered events on a timer owned by MetricSet.Run.
type RemoteWriteFlushCapable interface {
	FlushExpired(now time.Time) map[string]mb.Event
	NextFlushInterval() time.Duration
}

// RemoteWriteFlushRetainable allows generators to retain flush events that could not be published.
type RemoteWriteFlushRetainable interface {
	RetainUnpublishedFlushEvents(events map[string]mb.Event)
}

type batchSubmission struct {
	samples model.Samples
	ctx     context.Context
	result  chan batchOutcome
}

// Stable HTTP error bodies for owner-loop outcomes (written via http.Error in handleFunc).
const (
	batchOutcomeMsgCapacityExceeded = "remote write capacity exceeded"
	batchOutcomeMsgPipelineRejected = "remote write pipeline rejected event"
	batchOutcomeMsgPreflightFailed  = "remote write preflight failed"
)

type batchOutcome struct {
	statusCode int
	message    string // non-empty for error responses; 202 leaves message empty
}

type tickSource interface {
	C() <-chan time.Time
	Stop()
}

type stdTicker struct {
	*time.Ticker
}

func (t *stdTicker) C() <-chan time.Time {
	return t.Ticker.C
}

func defaultTickSource(interval time.Duration) tickSource {
	return &stdTicker{time.NewTicker(interval)}
}

func (m *MetricSet) preflightBatch(sub batchSubmission) (batchOutcome, bool) {
	checker, ok := m.promEventsGen.(RemoteWriteCapacityChecker)
	if !ok {
		return batchOutcome{}, true
	}
	err := checker.CheckCapacity(sub.samples)
	if err == nil {
		return batchOutcome{}, true
	}
	if errors.Is(err, ErrRemoteWriteCapacityExceeded) {
		return batchOutcome{
			statusCode: http.StatusServiceUnavailable,
			message:    batchOutcomeMsgCapacityExceeded,
		}, false
	}
	m.Logger().Errorf("remote write preflight failed: %v", err)
	return batchOutcome{
		statusCode: http.StatusInternalServerError,
		message:    batchOutcomeMsgPreflightFailed,
	}, false
}

func (m *MetricSet) processBatch(reporter mb.PushReporterV2, sub batchSubmission) {
	var events map[string]mb.Event
	if processor, ok := m.promEventsGen.(RemoteWriteOwnerLoopBatchProcessor); ok {
		var err error
		events, err = processor.ProcessOwnerLoopBatch(sub.samples)
		if err != nil {
			if errors.Is(err, ErrRemoteWriteCapacityExceeded) {
				m.completeBatch(sub, batchOutcome{
					statusCode: http.StatusServiceUnavailable,
					message:    batchOutcomeMsgCapacityExceeded,
				})
				return
			}
			m.Logger().Errorf("remote write preflight failed: %v", err)
			m.completeBatch(sub, batchOutcome{
				statusCode: http.StatusInternalServerError,
				message:    batchOutcomeMsgPreflightFailed,
			})
			return
		}
	} else {
		if outcome, ok := m.preflightBatch(sub); !ok {
			m.completeBatch(sub, outcome)
			return
		}

		// Request cancellation after preflight admission cannot roll back generator state or
		// events already delivered to the reporter; that matches at-least-once remote_write
		// semantics. Capacity preflight is the only required atomic rejection from the client.
		events = m.promEventsGen.GenerateEvents(sub.samples)
	}

	for _, e := range events {
		select {
		case <-sub.ctx.Done():
			return
		default:
		}
		if !reporter.Event(e) {
			m.Logger().Warn("remote write batch publish stopped: pipeline rejected event")
			m.completeBatch(sub, batchOutcome{
				statusCode: http.StatusServiceUnavailable,
				message:    batchOutcomeMsgPipelineRejected,
			})
			return
		}
	}
	m.completeBatch(sub, batchOutcome{statusCode: http.StatusAccepted})
}

func (m *MetricSet) completeBatch(sub batchSubmission, outcome batchOutcome) {
	select {
	case sub.result <- outcome:
	case <-sub.ctx.Done():
	}
}

func (m *MetricSet) publishFlushEvents(reporter mb.PushReporterV2, events map[string]mb.Event) {
	if len(events) == 0 {
		return
	}
	published := make(map[string]struct{}, len(events))
	for key, e := range events {
		if !reporter.Event(e) {
			m.Logger().Warn("remote write flush publish stopped: pipeline rejected event")
			if retainer, ok := m.promEventsGen.(RemoteWriteFlushRetainable); ok {
				unpublished := make(map[string]mb.Event, len(events)-len(published))
				for k, ev := range events {
					if _, ok := published[k]; !ok {
						unpublished[k] = ev
					}
				}
				retainer.RetainUnpublishedFlushEvents(unpublished)
			}
			return
		}
		published[key] = struct{}{}
	}
}

func (m *MetricSet) processFlush(reporter mb.PushReporterV2, now time.Time) {
	flusher, ok := m.promEventsGen.(RemoteWriteFlushCapable)
	if !ok {
		return
	}
	m.publishFlushEvents(reporter, flusher.FlushExpired(now))
}

func (m *MetricSet) shutdownOwnerLoop(reporter mb.PushReporterV2) {
	for {
		select {
		case sub := <-m.batches:
			m.processBatch(reporter, sub)
		default:
			return
		}
	}
}

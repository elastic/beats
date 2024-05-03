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

package memqueue

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/fifo"
)

// runLoop internal state. These fields could mostly be local variables
// in runLoop.run(), but they're exposed here to facilitate testing. In a
// live queue, only the runLoop goroutine should read or write these fields.
type runLoop struct {
	broker *broker

	// The index of the beginning of the current ring buffer within its backing
	// array. If the queue isn't empty, bufPos points to the oldest remaining
	// event.
	bufPos int

	// The total number of events in the queue.
	eventCount int

	// The total number of bytes in the queue
	byteCount int

	// The number of consumed events waiting for acknowledgment. The next Get
	// request will return events starting at position
	// (bufPos + consumedCount) % len(buf).
	consumedCount int

	// The list of batches that have been consumed and are waiting to be sent
	// to ackLoop for acknowledgment handling. (This list doesn't contain all
	// outstanding batches, only the ones not yet forwarded to ackLoop.)
	consumedBatches batchList

	// pendingPushRequests stores incoming events that can't yet fit in the
	// queue. As space in the queue is freed, these requests will be handled
	// in order.
	pendingPushRequests fifo.FIFO[pushRequest]

	// If there aren't enough events ready to fill an incoming get request,
	// the queue may block based on its flush settings. When this happens,
	// pendingGetRequest stores the request until we're ready to handle it.
	pendingGetRequest *getRequest

	// This timer tracks the configured flush timeout when we will respond
	// to a pending getRequest even if we can't fill the requested event count.
	// It is active if and only if pendingGetRequest is non-nil.
	getTimer *time.Timer
}

func newRunLoop(broker *broker) *runLoop {
	var timer *time.Timer

	// Create the timer we'll use for get requests, but stop it until a
	// get request is active.
	if broker.settings.FlushTimeout > 0 {
		timer = time.NewTimer(broker.settings.FlushTimeout)
		if !timer.Stop() {
			<-timer.C
		}
	}
	return &runLoop{
		broker:   broker,
		getTimer: timer,
	}
}

func (l *runLoop) run() {
	for l.broker.ctx.Err() == nil {
		l.runIteration()
	}
}

func (l *runLoop) isSpaceAvailable() bool {
	maxEvents := l.broker.settings.Events
	maxBytes := l.broker.settings.Bytes

	eventsAvailable := maxEvents <= 0 || l.eventCount < maxEvents
	bytesAvailable := maxBytes <= 0 || l.byteCount < maxBytes

	return eventsAvailable && bytesAvailable
}

func (l *runLoop) canHandlePushRequest(req pushRequest) bool {
	return false
}

// Perform one iteration of the queue's main run loop. Broken out into a
// standalone helper function to allow testing of loop invariants.
func (l *runLoop) runIteration() {
	var getChan chan getRequest
	// Get requests are enabled if the queue has events that weren't yet sent
	// to consumers, and no existing request is active.
	if l.pendingGetRequest == nil && l.eventCount > l.consumedCount {
		getChan = l.broker.getChan
	}

	var consumedChan chan batchList
	// Enable sending to the scheduled ACKs channel if we have
	// something to send.
	if !l.consumedBatches.empty() {
		consumedChan = l.broker.consumedChan
	}

	var timeoutChan <-chan time.Time
	// Enable the timeout channel if a get request is waiting for events
	if l.pendingGetRequest != nil {
		timeoutChan = l.getTimer.C
	}

	select {
	case <-l.broker.ctx.Done():
		return

	case req := <-l.broker.pushChan: // producer pushing new event
		l.handleInsert(req)

	case req := <-l.broker.cancelChan: // producer cancelling active events
		l.handleCancel(&req)

	case req := <-getChan: // consumer asking for next batch
		l.handleGetRequest(&req)

	case consumedChan <- l.consumedBatches:
		// We've sent all the pending batches to the ackLoop for processing,
		// clear the pending list.
		l.consumedBatches = batchList{}

	case count := <-l.broker.deleteChan:
		l.handleDelete(count)

	case req := <-l.broker.metricChan: // asking broker for queue metrics
		l.handleMetricsRequest(&req)

	case <-timeoutChan:
		// The get timer has expired, handle the blocked request
		l.getTimer.Stop()
		l.handleGetReply(l.pendingGetRequest)
		l.pendingGetRequest = nil
	}
}

func (l *runLoop) handleGetRequest(req *getRequest) {
	// Backwards compatibility: if all byte parameters are <= 0, get requests
	// are capped by settings.MaxGetRequest.
	if req.byteCount <= 0 && l.broker.settings.Bytes <= 0 {
		if req.entryCount <= 0 || req.entryCount > l.broker.settings.MaxGetRequest {
			req.entryCount = l.broker.settings.MaxGetRequest
		}
	}
	if l.getRequestShouldBlock(req) {
		l.pendingGetRequest = req
		l.getTimer.Reset(l.broker.settings.FlushTimeout)
		return
	}
	l.handleGetReply(req)
}

func (l *runLoop) getRequestShouldBlock(req *getRequest) bool {
	if l.broker.settings.FlushTimeout <= 0 {
		// Never block if the flush timeout isn't positive
		return false
	}
	eventsAvailable := l.eventCount - l.consumedCount
	// Block if the available events aren't enough to fill the request
	return eventsAvailable < req.entryCount
}

// Respond to the given get request without blocking or waiting for more events
func (l *runLoop) handleGetReply(req *getRequest) {
	eventsAvailable := l.eventCount - l.consumedCount
	batchSize := req.entryCount
	if eventsAvailable < batchSize {
		batchSize = eventsAvailable
	}

	startIndex := l.bufPos + l.consumedCount
	batch := newBatch(l.broker, startIndex, batchSize)

	// Send the batch to the caller and update internal state
	req.responseChan <- batch
	l.consumedBatches.append(batch)
	l.consumedCount += batchSize
}

func (l *runLoop) handleDelete(count int) {
	// Advance position and counters. Event data was already cleared in
	// batch.FreeEntries when the events were vended.
	l.bufPos = (l.bufPos + count) % len(l.broker.buf)
	l.eventCount -= count
	l.consumedCount -= count
}

func (l *runLoop) handleInsert(req pushRequest) {
	if !l.canHandlePushRequest(req) {
		if req.blockIfFull {
			// Add this request to the pending list to be handled when there's space.
			l.pendingPushRequests.Add(req)
		} else {
			l.broker.logger.Debugf("Dropping event, queue is blocked")
			req.resp <- false
		}
		return
	}
	if l.insert(req) {
		// Send back the new event id.
		req.resp <- true

		l.eventCount++

		// See if this gave us enough for a new batch
		l.maybeUnblockGetRequest()
	}
}

// Checks if we can handle pendingGetRequest yet, and handles it if so
func (l *runLoop) maybeUnblockGetRequest() {
	// If a get request is blocked waiting for more events, check if
	// we should unblock it.
	if getRequest := l.pendingGetRequest; getRequest != nil {
		available := l.eventCount - l.consumedCount
		if available >= getRequest.entryCount {
			l.pendingGetRequest = nil
			if !l.getTimer.Stop() {
				<-l.getTimer.C
			}
			l.handleGetReply(getRequest)
		}
	}
}

// Returns true if the event was inserted, false if insertion was cancelled.
func (l *runLoop) insert(req pushRequest) bool {
	if req.producer != nil && req.producer.state.cancelled {
		return false
	}

	index := (l.bufPos + l.eventCount) % len(l.broker.buf)
	l.broker.buf[index] = queueEntry{
		event:      req.event,
		producer:   req.producer,
		producerID: req.producerID,
	}
	return true
}

func (l *runLoop) handleMetricsRequest(req *metricsRequest) {
	req.responseChan <- memQueueMetrics{
		currentQueueSize: l.eventCount,
		occupiedRead:     l.consumedCount,
	}
}

func (l *runLoop) handleCancel(req *producerCancelRequest) {
	var removedCount int

	// Traverse all unconsumed events in the buffer, removing any with
	// the specified producer. As we go we condense all the remaining
	// events to be sequential.
	buf := l.broker.buf
	startIndex := l.bufPos + l.consumedCount
	unconsumedEventCount := l.eventCount - l.consumedCount
	for i := 0; i < unconsumedEventCount; i++ {
		readIndex := (startIndex + i) % len(buf)
		if buf[readIndex].producer == req.producer {
			// The producer matches, skip this event
			removedCount++
		} else {
			// Move the event to its final position after accounting for any
			// earlier indices that were removed.
			// (Count backwards from (startIndex + i), not from readIndex, to avoid
			// sign issues when the buffer wraps.)
			writeIndex := (startIndex + i - removedCount) % len(buf)
			buf[writeIndex] = buf[readIndex]
		}
	}

	// Clear the event pointers at the end of the buffer so we don't keep
	// old events in memory by accident.
	for i := 0; i < removedCount; i++ {
		index := (l.bufPos + l.eventCount - removedCount + i) % len(buf)
		buf[index].event = nil
	}

	// Subtract removed events from the internal event count
	l.eventCount -= removedCount

	// signal cancel request being finished
	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: removedCount}
	}
}

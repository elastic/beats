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

	// The buffer backing the queue. Don't access its internal array directly,
	// use an entryIndex: buf.entry(entryIndex) returns a pointer to the target
	// entry within the buffer.
	// Accessing this way handles the modular arithmetic to convert entry index
	// to buffer index, in a way that's compatible with dynamically growing the
	// underlying array (which is important when the queue has no maximum event
	// count).
	buf circularBuffer

	// The index of the oldest entry in the underlying circular buffer.
	bufferStart entryIndex

	// The current number of events in the queue.
	eventCount int

	// The current number of bytes in the queue.
	byteCount int

	// The number of consumed events waiting for acknowledgment. The next Get
	// request will return events starting at index
	// bufferStart.plus(consumedEventCount).
	consumedEventCount int

	// The number of event bytes in the queue corresponding to consumed events.
	consumedByteCount int

	// The list of batches that have been consumed and are waiting to be sent
	// to ackLoop for acknowledgment handling. (This list doesn't contain all
	// outstanding batches, only the ones not yet forwarded to ackLoop.)
	consumedBatches batchList

	// pendingPushRequests stores incoming events that can't yet fit in the
	// queue. As space in the queue is freed, these requests will be handled
	// in order.
	pendingPushRequests fifo.FIFO[pushRequest]

	// If there aren't enough events ready to fill an incoming get request,
	// the request may block based on the queue flush settings. When this
	// happens, pendingGetRequest stores the request until we can handle it.
	pendingGetRequest *getRequest

	// When a get request is blocked because the queue doesn't have enough
	// events, getTimer stores the flush timer. When it expires, the queue
	// will respond to the request even if the requested number of events
	// and/or bytes is not available.
	// getTimer is active if and only if pendingGetRequest is non-nil.
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

	eventBufSize := broker.settings.Events
	if eventBufSize <= 0 {
		// The queue is using byte limits, start with a buffer of 2^10 and
		// we will expand it as needed.
		eventBufSize = 1 << 10
	}

	return &runLoop{
		broker:   broker,
		getTimer: timer,
		buf:      newCircularBuffer(eventBufSize),
	}
}

func (l *runLoop) run() {
	for l.broker.ctx.Err() == nil {
		l.runIteration()
	}
}

// Perform one iteration of the queue's main run loop. Broken out into a
// standalone helper function to allow testing of loop invariants.
func (l *runLoop) runIteration() {
	var getChan chan getRequest
	// Get requests are enabled if the queue has events that weren't yet sent
	// to consumers, and no existing request is active.
	if l.pendingGetRequest == nil && l.eventCount > l.consumedEventCount {
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
		l.handlePushRequest(req)

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
		if req.entryCount > l.broker.settings.MaxGetRequest {
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
	availableEntries := l.eventCount - l.consumedEventCount
	availableBytes := l.byteCount - l.consumedByteCount

	// The entry/byte limits are satisfied if they are <= 0 (indicating no
	// limit) or if we have at least the requested number available.
	entriesSatisfied := req.entryCount <= 0 || availableEntries >= req.entryCount
	bytesSatisfied := req.byteCount <= 0 || availableBytes >= req.byteCount

	// Block if there are neither enough entries nor enough bytes to fill
	// the request.
	return !entriesSatisfied && !bytesSatisfied
}

// Respond to the given get request without blocking or waiting for more events
func (l *runLoop) handleGetReply(req *getRequest) {
	entriesAvailable := l.eventCount - l.consumedEventCount
	// backwards compatibility: if all byte bounds are <= 0 then batch size
	// can't be more than settings.MaxGetRequest.
	if req.byteCount <= 0 && l.broker.settings.Bytes <= 0 {
		if entriesAvailable > l.broker.settings.MaxGetRequest {
			entriesAvailable = l.broker.settings.MaxGetRequest
		}
	}
	startIndex := l.bufferStart.plus(l.consumedEventCount)
	batchEntryCount := 0
	batchByteCount := 0

	for i := 0; i < entriesAvailable; i++ {
		if req.entryCount > 0 && batchEntryCount+1 > req.entryCount {
			// This would push us over the requested event limit, stop here.
			break
		}
		eventSize := l.buf.entry(startIndex.plus(batchEntryCount)).eventSize
		// Don't apply size checks on the first event: if a single event is
		// larger than the configured batch maximum, we'll still try to send it,
		// we'll just do it in a "batch" of one event.
		if i > 0 && req.byteCount > 0 && batchByteCount+eventSize > req.byteCount {
			// This would push us over the requested byte limit, stop here.
			break
		}
		batchEntryCount++
		batchByteCount += eventSize
	}

	batch := newBatch(l.buf, startIndex, batchEntryCount)

	// Send the batch to the caller and update internal state
	req.responseChan <- batch
	l.consumedBatches.append(batch)
	l.consumedEventCount += batchEntryCount
	l.consumedByteCount += batchByteCount
}

func (l *runLoop) handleDelete(deletedEntryCount int) {
	// Advance position and counters. Event data was already cleared in
	// batch.FreeEntries when the events were vended, so we just need to
	// check the byte total being removed.
	deletedByteCount := 0
	for i := 0; i < deletedEntryCount; i++ {
		entryIndex := l.bufferStart.plus(i)
		deletedByteCount += l.buf.entry(entryIndex).eventSize
	}
	l.bufferStart = l.bufferStart.plus(deletedEntryCount)
	l.eventCount -= deletedEntryCount
	l.byteCount -= deletedByteCount
	l.consumedEventCount -= deletedEntryCount
	l.consumedByteCount -= deletedByteCount

	// We just freed up space in the queue, see if this unblocked any
	// pending inserts.
	l.maybeUnblockPushRequests()
}

func (l *runLoop) handlePushRequest(req pushRequest) {
	// If other inserts are already pending, or we don't have enough room
	// for the new entry, we need to either reject the request or block
	// until we can handle it.
	if !l.pendingPushRequests.Empty() || !l.canFitPushRequest(req) {
		if req.blockIfFull {
			// Add this request to the pending list to be handled when there's space.
			l.pendingPushRequests.Add(req)
		} else {
			// Request doesn't want to block, return failure immediately.
			l.broker.logger.Debugf("queue is full, dropping event")
			req.resp <- false
		}
		return
	}
	// There is space, insert the new event and report the result.
	l.doInsert(req)
}

// Returns true if the given push request can be added to the queue
// without exceeding entry count or byte limits
func (l *runLoop) canFitPushRequest(req pushRequest) bool {
	maxEvents := l.broker.settings.Events
	maxBytes := l.broker.settings.Bytes

	newEventCount := l.eventCount + 1
	newByteCount := l.byteCount + req.eventSize

	eventCountFits := maxEvents <= 0 || newEventCount <= maxEvents
	byteCountFits := maxBytes <= 0 || newByteCount <= maxBytes

	return eventCountFits && byteCountFits
}

// Checks if we can handle pendingGetRequest yet, and handles it if so
func (l *runLoop) maybeUnblockGetRequest() {
	if l.pendingGetRequest != nil {
		if !l.getRequestShouldBlock(l.pendingGetRequest) {
			l.handleGetReply(l.pendingGetRequest)
			l.pendingGetRequest = nil
			if !l.getTimer.Stop() {
				<-l.getTimer.C
			}
		}
	}
}

func (l *runLoop) maybeUnblockPushRequests() {
	req, err := l.pendingPushRequests.First()
	for err == nil {
		if !l.canFitPushRequest(req) {
			break
		}
		l.doInsert(req)
		l.pendingPushRequests.Remove()

		// Fetch the next request
		req, err = l.pendingPushRequests.First()
	}
}

// growEventBuffer is called when there is no limit on the queue event
// count (i.e. the queue size is byte-based) but the queue's event buffer
// (a []queueEntry) is full.
// For this to be possible, queue indices must be stable when the buffer
// size changes. Therefore, entry positions are based on a strictly
// increasing id, so that different events have different positions,
// even when they occupy the same location in the underlying buffer.
// The buffer position is the entry's index modulo the buffer size: for
// a queue with buffer size N, the entries stored in buf[0] will have
// entry indices 0, N, 2*N, 3*N, ...
func (l *runLoop) growEventBuffer() {
	bufSize := l.buf.size()
	newBuffer := newCircularBuffer(bufSize * 2)
	// Copy the elements to the new buffer
	for i := 0; i < bufSize; i++ {
		index := l.bufferStart.plus(i)
		*newBuffer.entry(index) = *l.buf.entry(index)
	}
	l.buf = newBuffer
}

// Insert the given new event without bounds checks, and report the result
// to the caller via the push request's response channel.
func (l *runLoop) doInsert(req pushRequest) {
	// We reject events if their producer was cancelled before they reach
	// the queue.
	if req.producer != nil && req.producer.state.cancelled {
		// Report failure to the caller (this only happens if the producer is
		// closed before we handle the insert request).
		req.resp <- false
		return
	}

	maxEvents := l.broker.settings.Events
	// If there is no event limit, check if we need to grow the current queue
	// buffer to fit the new event.
	if maxEvents <= 0 && l.eventCount >= l.buf.size() {
		l.growEventBuffer()
	}

	entryIndex := l.bufferStart.plus(l.eventCount)
	*l.buf.entry(entryIndex) = queueEntry{
		event:      req.event,
		eventSize:  req.eventSize,
		producer:   req.producer,
		producerID: req.producerID,
	}

	// Report success to the caller
	req.resp <- true

	l.eventCount++
	l.byteCount += req.eventSize

	// See if this gave us enough for a new batch
	l.maybeUnblockGetRequest()
}

func (l *runLoop) handleMetricsRequest(req *metricsRequest) {
	req.responseChan <- memQueueMetrics{
		currentQueueSize: l.eventCount,
		occupiedRead:     l.consumedEventCount,
	}
}

func (l *runLoop) handleCancel(req *producerCancelRequest) {
	var removedCount int

	// Traverse all unconsumed events in the buffer, removing any with
	// the specified producer. As we go we condense all the remaining
	// events to be sequential.
	startIndex := l.bufferStart.plus(l.consumedEventCount)
	unconsumedEventCount := l.eventCount - l.consumedEventCount
	for i := 0; i < unconsumedEventCount; i++ {
		readIndex := startIndex.plus(i)
		entry := *l.buf.entry(readIndex)
		if entry.producer == req.producer {
			// The producer matches, skip this event
			removedCount++
		} else {
			// Move the event to its final position after accounting for any
			// earlier indices that were removed.
			// (Count backwards from (startIndex + i), not from readIndex, to avoid
			// sign issues when the buffer wraps.)
			if removedCount > 0 {
				writeIndex := readIndex.plus(-removedCount)
				*l.buf.entry(writeIndex) = entry
			}
		}
	}

	// Clear the event pointers at the end of the buffer so we don't keep
	// old events in memory by accident.
	for i := l.eventCount - removedCount; i < l.eventCount; i++ {
		entryIndex := l.bufferStart.plus(i)
		l.buf.entry(entryIndex).event = nil
	}

	// Subtract removed events from the internal event count
	l.eventCount -= removedCount

	// signal cancel request being finished
	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: removedCount}
	}
}

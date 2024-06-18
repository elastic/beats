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
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/fifo"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
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

	// observer is a metrics observer used to report internal queue state.
	observer queue.Observer

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

	// closing is set when a close request is received. Once closing is true,
	// the queue will not accept any new events, but will continue responding
	// to Gets and Acks to allow pending events to complete on shutdown.
	closing bool
}

func newRunLoop(broker *broker, observer queue.Observer) *runLoop {
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
	if broker.useByteLimits() {
		// The queue is using byte limits, start with a buffer of 2^10 and
		// we will expand it as needed.
		eventBufSize = 1 << 10
	}

	return &runLoop{
		broker:   broker,
		observer: observer,
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
	var pushChan chan pushRequest
	// Push requests are enabled if the queue isn't closing.
	if !l.closing {
		pushChan = l.broker.pushChan
	}

	var getChan chan getRequest
	// Get requests are enabled if the queue has events that weren't yet sent
	// to consumers, and no existing request is active.
	if l.pendingGetRequest == nil && l.eventCount > l.consumedEventCount {
		getChan = l.broker.getChan
	}

	var consumedChan chan batchList
	// Enable sending to the scheduled ACKs channel if we have
	// something to send.
	if !l.consumedBatches.Empty() {
		consumedChan = l.broker.consumedChan
	}

	var timeoutChan <-chan time.Time
	// Enable the timeout channel if a get request is waiting for events
	if l.pendingGetRequest != nil {
		timeoutChan = l.getTimer.C
	}

	select {
	case <-l.broker.closeChan:
		l.closing = true
		// Get requests are handled immediately during shutdown
		l.maybeUnblockGetRequest()

	case <-l.broker.ctx.Done():
		// The queue is fully shut down, do nothing
		return

	case req := <-pushChan: // producer pushing new event
		l.handlePushRequest(req)

	case req := <-getChan: // consumer asking for next batch
		l.handleGetRequest(&req)

	case consumedChan <- l.consumedBatches:
		// We've sent all the pending batches to the ackLoop for processing,
		// clear the pending list.
		l.consumedBatches = batchList{}

	case count := <-l.broker.deleteChan:
		l.handleDelete(count)

	case <-timeoutChan:
		// The get timer has expired, handle the blocked request
		l.getTimer.Stop()
		l.handleGetReply(l.pendingGetRequest)
		l.pendingGetRequest = nil
	}
}
func (l *runLoop) handleGetRequest(req *getRequest) {
	// When using event-based limits, requests are capped by settings.MaxGetRequest.
	if !l.broker.useByteLimits() && req.entryCount > l.broker.settings.MaxGetRequest {
		req.entryCount = l.broker.settings.MaxGetRequest
	}

	if l.getRequestShouldBlock(req) {
		l.pendingGetRequest = req
		l.getTimer.Reset(l.broker.settings.FlushTimeout)
		return
	}
	l.handleGetReply(req)
}

func (l *runLoop) getRequestShouldBlock(req *getRequest) bool {
	if l.broker.settings.FlushTimeout <= 0 || l.closing {
		// Never block if the flush timeout isn't positive, or during shutdown
		return false
	}

	// The entry/byte limits are satisfied if they are <= 0 (indicating no
	// limit) or if we have at least the requested number available.
	if l.broker.useByteLimits() {
		availableBytes := l.byteCount - l.consumedByteCount
		return req.byteCount <= 0 || availableBytes >= req.byteCount
	}
	availableEntries := l.eventCount - l.consumedEventCount
	fmt.Printf("hi fae, getRequestShouldBlock for %v entries while there are %v available\n", req.entryCount, availableEntries)
	return req.entryCount <= 0 || availableEntries >= req.entryCount
}

// Respond to the given get request without blocking or waiting for more events
func (l *runLoop) handleGetReply(req *getRequest) {
	entriesAvailable := l.eventCount - l.consumedEventCount
	// backwards compatibility: when using event-based limits, batch size
	// can't be more than settings.MaxGetRequest.
	if l.broker.useByteLimits() {
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

	batchBytes := 0
	for i := 0; i < batchEntryCount; i++ {
		batchBytes += batch.entry(i).eventSize
	}

	// Send the batch to the caller and update internal state
	req.responseChan <- batch
	l.consumedBatches.Add(batch)
	l.consumedEventCount += batchEntryCount
	l.consumedByteCount += batchByteCount
	l.observer.ConsumeEvents(batchEntryCount, batchByteCount)
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
	l.observer.RemoveEvents(deletedEntryCount, deletedByteCount)
	if l.closing && l.eventCount == 0 {
		// Our last events were acknowledged during shutdown, signal final shutdown
		l.broker.ctxCancel()
	}

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
// without exceeding the entry count or byte limit.
func (l *runLoop) canFitPushRequest(req pushRequest) bool {
	if l.broker.useByteLimits() {
		newByteCount := l.byteCount + req.eventSize
		return newByteCount <= l.broker.settings.Bytes
	}
	newEventCount := l.eventCount + 1
	return newEventCount <= l.broker.settings.Events
}

func (l *runLoop) maybeUnblockPushRequests() {
	for !l.pendingPushRequests.Empty() {
		req := l.pendingPushRequests.First()
		if !l.canFitPushRequest(req) {
			break
		}
		l.doInsert(req)
		l.pendingPushRequests.Remove()
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
	// If using byte limits (no hard limit on event count), check if we need to
	// grow the current queue buffer to fit the new event.
	if l.broker.useByteLimits() && l.eventCount >= l.buf.size() {
		l.growEventBuffer()
	}

	entryIndex := l.bufferStart.plus(l.eventCount)
	*l.buf.entry(entryIndex) = queueEntry{
		event:      req.event,
		eventSize:  req.eventSize,
		producer:   req.producer,
		producerID: req.producerID,
	}
	l.observer.AddEvent(req.eventSize)

	// Report success to the caller
	req.resp <- true

	l.eventCount++
	l.byteCount += req.eventSize

	// See if this gave us enough for a new batch
	l.maybeUnblockGetRequest()
}

// Checks if we can handle pendingGetRequest yet, and handles it if so
func (l *runLoop) maybeUnblockGetRequest() {
	// If a get request is blocked waiting for more events, check if
	// we should unblock it.
	if getRequest := l.pendingGetRequest; getRequest != nil {
		if !l.getRequestShouldBlock(getRequest) {
			l.pendingGetRequest = nil
			if !l.getTimer.Stop() {
				<-l.getTimer.C
			}
			l.handleGetReply(getRequest)
		}
	}
}

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

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// runLoop internal state. These fields could mostly be local variables
// in runLoop.run(), but they're exposed here to facilitate testing. In a
// live queue, only the runLoop goroutine should read or write these fields.
type runLoop struct {
	broker *broker

	// observer is a metrics observer used to report internal queue state.
	observer queue.Observer

	// The index of the beginning of the current ring buffer within its backing
	// array. If the queue isn't empty, bufPos points to the oldest remaining
	// event.
	bufPos int

	// The total number of events in the queue.
	eventCount int

	// The number of consumed events waiting for acknowledgment. The next Get
	// request will return events starting at position
	// (bufPos + consumedCount) % len(buf).
	consumedCount int

	// The list of batches that have been consumed and are waiting to be sent
	// to ackLoop for acknowledgment handling. (This list doesn't contain all
	// outstanding batches, only the ones not yet forwarded to ackLoop.)
	consumedBatches batchList

	// If there aren't enough events ready to fill an incoming get request,
	// the queue may block based on its flush settings. When this happens,
	// pendingGetRequest stores the request until we're ready to handle it.
	pendingGetRequest *getRequest

	// This timer tracks the configured flush timeout when we will respond
	// to a pending getRequest even if we can't fill the requested event count.
	// It is active if and only if pendingGetRequest is non-nil.
	getTimer *time.Timer

	// closing is set when a close request is received. Once closing is true,
	// the queue will not accept any new events, but will continue responding
	// to Gets and Acks to allow pending events to complete on shutdown.
	closing bool

	// TODO (https://github.com/elastic/beats/issues/37893): entry IDs were a
	// workaround for an external project that no longer exists. At this point
	// they just complicate the API and should be removed.
	nextEntryID queue.EntryID
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
	return &runLoop{
		broker:   broker,
		observer: observer,
		getTimer: timer,
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
	// Push requests are enabled if the queue isn't full or closing.
	if l.eventCount < len(l.broker.buf) && !l.closing {
		pushChan = l.broker.pushChan
	}

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
	case <-l.broker.closeChan:
		l.closing = true
		close(l.broker.closingChan)
		// Get requests are handled immediately during shutdown
		l.maybeUnblockGetRequest()

	case <-l.broker.ctx.Done():
		// The queue is fully shut down, do nothing
		return

	case req := <-pushChan: // producer pushing new event
		l.handleInsert(&req)

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
	if req.entryCount <= 0 || req.entryCount > l.broker.settings.MaxGetRequest {
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

	batchBytes := 0
	for i := 0; i < batchSize; i++ {
		batchBytes += batch.rawEntry(i).eventSize
	}

	// Send the batch to the caller and update internal state
	req.responseChan <- batch
	l.consumedBatches.append(batch)
	l.consumedCount += batchSize
	l.observer.ConsumeEvents(batchSize, batchBytes)
}

func (l *runLoop) handleDelete(count int) {
	byteCount := 0
	for i := 0; i < count; i++ {
		entry := l.broker.buf[(l.bufPos+i)%len(l.broker.buf)]
		byteCount += entry.eventSize
	}
	// Advance position and counters. Event data was already cleared in
	// batch.FreeEntries when the events were vended.
	l.bufPos = (l.bufPos + count) % len(l.broker.buf)
	l.eventCount -= count
	l.consumedCount -= count
	l.observer.RemoveEvents(count, byteCount)
	if l.closing && l.eventCount == 0 {
		// Our last events were acknowledged during shutdown, signal final shutdown
		l.broker.ctxCancel()
	}
}

func (l *runLoop) handleInsert(req *pushRequest) {
	l.insert(req, l.nextEntryID)
	// Send back the new event id.
	req.resp <- l.nextEntryID

	l.nextEntryID++
	l.eventCount++

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

func (l *runLoop) insert(req *pushRequest, id queue.EntryID) {
	index := (l.bufPos + l.eventCount) % len(l.broker.buf)
	l.broker.buf[index] = queueEntry{
		event:      req.event,
		eventSize:  req.eventSize,
		id:         id,
		producer:   req.producer,
		producerID: req.producerID,
	}
	l.observer.AddEvent(req.eventSize)
}

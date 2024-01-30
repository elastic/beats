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
	"github.com/elastic/elastic-agent-libs/logp"
)

// directEventLoop implements the broker main event loop. It buffers events,
// but tries to forward events as early as possible.
type directEventLoop struct {
	broker     *broker
	buf        ringBuffer
	deleteChan chan int

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs batchList

	nextEntryID queue.EntryID
}

// bufferingEventLoop implements the broker main event loop.
// Events in the buffer are forwarded to consumers only if the buffer is full or on flush timeout.
type bufferingEventLoop struct {
	broker     *broker
	deleteChan chan int

	// The current buffer that incoming events are appended to. When it gets
	// full enough, or enough time has passed, it is added to flushList.
	// Events will still be added to buf even after it is in flushList, until
	// either it reaches minEvents or a consumer requests it.
	buf *batchBuffer

	// flushList is the list of buffers that are ready to be sent to consumers.
	flushList flushList

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs batchList

	// The number of events currently waiting in the queue, including
	// those that have not yet been acked.
	eventCount int

	// The next entry ID that will be read by a consumer, and the next
	// entry ID that has been consumed and is waiting for acknowledgment.
	// We need to track these here because bufferingEventLoop discards
	// its event buffers when they are sent to consumers, so we can't
	// look directly at the event itself to get the current id like we
	// do in the unbuffered loop.
	nextConsumedID queue.EntryID
	nextACKedID    queue.EntryID

	minEvents    int
	maxEvents    int
	flushTimeout time.Duration

	// buffer flush timer state
	timer *time.Timer
	idleC <-chan time.Time

	nextEntryID queue.EntryID
}

type flushList struct {
	head  *batchBuffer
	tail  *batchBuffer
	count int
}

func newDirectEventLoop(b *broker, size int) *directEventLoop {
	l := &directEventLoop{
		broker:     b,
		deleteChan: make(chan int),
	}
	l.buf.init(b.logger, size)

	return l
}

func (l *directEventLoop) handleMetricsRequest(req *metricsRequest) {
	// If the queue is empty, we report the "oldest" ID as the next
	// one that will be assigned. Otherwise, we report the ID attached
	// to the oldest queueEntry.
	oldestEntryID := l.nextEntryID
	if oldestEntry := l.buf.OldestEntry(); oldestEntry != nil {
		oldestEntryID = oldestEntry.id
	}

	req.responseChan <- memQueueMetrics{
		currentQueueSize: l.buf.Items(),
		occupiedRead:     l.buf.reserved,
		oldestEntryID:    oldestEntryID,
	}
}

func (l *directEventLoop) insert(req *pushRequest) {
	log := l.broker.logger

	if req.producer != nil && req.producer.state.cancelled {
		reportCancelledState(log, req)
	} else {
		req.resp <- l.nextEntryID
		l.buf.insert(queueEntry{
			event:      req.event,
			id:         l.nextEntryID,
			producer:   req.producer,
			producerID: req.producerID})
		l.nextEntryID++
	}
}

func (l *directEventLoop) handleCancel(req *producerCancelRequest) {
	// log := l.broker.logger
	// log.Debug("handle cancel request")

	var removed int

	if producer := req.producer; producer != nil {
		producer.state.cancelled = true
		removed = l.buf.cancel(producer)
	}

	// signal cancel request being finished
	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: removed}
	}
}

// processACK is called by the ackLoop to process the list of acked batches
func (l *directEventLoop) processACK(lst batchList, N int) {
	log := l.broker.logger
	{
		start := time.Now()
		log.Debug("handle ACKs: ", N)
		defer func() {
			log.Debug("handle ACK took: ", time.Since(start))
		}()
	}

	entries := l.buf.entries

	firstIndex := lst.front().start

	// We want to acknowledge N events starting at position firstIndex
	// in the entries array.
	// We iterate over the events from last to first, so we encounter the
	// highest producer IDs first and can skip subsequent callbacks to the
	// same producer.
	producerCallbacks := []func(){}
	for i := N - 1; i >= 0; i-- {
		// idx is the index in entries of the i-th event after firstIndex, wrapping
		// around the end of the array.
		idx := (firstIndex + i) % len(entries)
		entry := &entries[idx]

		producer := entry.producer

		// Set the producer in the entires array to nil to mark it as visited; a nil
		// producer indicates that an entry requires no more ack processing (either
		// because it has already been ACKed, or because its producer does not listen to ACKs).
		entry.producer = nil
		if producer == nil || entry.producerID <= producer.state.lastACK {
			// This has a lower index than the previous ACK for this producer,
			// so it was covered in the previous call and we can skip it.
			continue
		}
		// This update is safe because lastACK is only used from the event loop.
		count := int(entry.producerID - producer.state.lastACK)
		producer.state.lastACK = entry.producerID

		producerCallbacks = append(producerCallbacks, func() { producer.state.cb(count) })
	}
	l.deleteChan <- N
	for _, f := range producerCallbacks {
		f()
	}
}

func newBufferingEventLoop(b *broker, size int, minEvents int, flushTimeout time.Duration) *bufferingEventLoop {
	l := &bufferingEventLoop{
		broker:       b,
		deleteChan:   make(chan int),
		maxEvents:    size,
		minEvents:    minEvents,
		flushTimeout: flushTimeout,
	}
	l.buf = newBatchBuffer(l.minEvents)

	l.timer = time.NewTimer(flushTimeout)
	if !l.timer.Stop() {
		<-l.timer.C
	}

	return l
}

func (l *bufferingEventLoop) handleMetricsRequest(req *metricsRequest) {
	req.responseChan <- memQueueMetrics{
		currentQueueSize: l.eventCount,
		occupiedRead:     int(l.nextConsumedID - l.nextACKedID),
		oldestEntryID:    l.nextACKedID,
	}
}

func (l *bufferingEventLoop) insert(req *pushRequest, id queue.EntryID) bool {
	if req.producer != nil && req.producer.state.cancelled {
		reportCancelledState(l.broker.logger, req)
		return false
	}

	l.buf.add(queueEntry{
		event:      req.event,
		id:         id,
		producer:   req.producer,
		producerID: req.producerID,
	})
	return true
}

func (l *bufferingEventLoop) handleCancel(req *producerCancelRequest) {
	removed := 0
	if producer := req.producer; producer != nil {
		// remove from actively flushed buffers
		for buf := l.flushList.head; buf != nil; buf = buf.next {
			removed += buf.cancel(producer)
		}
		if !l.buf.flushed {
			removed += l.buf.cancel(producer)
		}

		producer.state.cancelled = true
	}

	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: removed}
	}

	// remove flushed but empty buffers:
	tmpList := flushList{}
	for l.flushList.head != nil {
		b := l.flushList.head
		l.flushList.head = b.next

		if b.length() > 0 {
			tmpList.add(b)
		}
	}
	l.flushList = tmpList
	l.eventCount -= removed
}

func (l *flushList) pop() {
	l.count--
	if l.count > 0 {
		l.head = l.head.next
	} else {
		l.head = nil
		l.tail = nil
	}
}

func (l *flushList) empty() bool {
	return l.head == nil
}

func (l *flushList) add(b *batchBuffer) {
	l.count++
	b.next = nil
	if l.tail == nil {
		l.head = b
		l.tail = b
	} else {
		l.tail.next = b
		l.tail = b
	}
}

func reportCancelledState(log *logp.Logger, req *pushRequest) {
	// do not add waiting events if producer did send cancel signal
	if cb := req.producer.state.dropCB; cb != nil {
		cb(req.event)
	}
}

func (b *broker) runLoop() {
	broker := b

	for {
		var pushChan chan pushRequest
		// Push requests are enabled if the queue isn't yet full.
		if b.eventCount < len(b.buf) {
			pushChan = b.pushChan
		}

		var getChan chan getRequest
		// Get requests are enabled if the queue has events that weren't yet sent
		// to consumers, and no existing request is active.
		if b.pendingGetRequest == nil && b.eventCount > b.consumedCount {
			getChan = b.getChan
		}

		var consumedChan chan batchList
		// Enable sending to the scheduled ACKs channel if we have
		// something to send.
		if !b.consumedBatches.empty() {
			consumedChan = b.consumedChan
		}

		var timeoutChan <-chan time.Time
		// Enable the timeout channel if a get request is waiting for events
		if b.pendingGetRequest != nil {
			timeoutChan = b.getTimer.C
		}

		select {
		case <-broker.done:
			return

		case req := <-pushChan: // producer pushing new event
			b.handleInsert(&req)

		case req := <-b.cancelChan: // producer cancelling active events
			b.handleCancel(&req)

		case req := <-getChan: // consumer asking for next batch
			b.handleGetRequest(&req)

		case consumedChan <- b.consumedBatches:
			b.consumedBatches = batchList{}

		case count := <-b.deleteChan:
			b.handleDelete(count)

		case req := <-b.metricChan: // broker asking for queue metrics
			b.handleMetricsRequest(&req)

		case <-timeoutChan:
			// The get timer has expired, handle the blocked request
			b.getTimer.Stop()
			b.handleGetReply(b.pendingGetRequest)
			b.pendingGetRequest = nil
		}
	}
}

func (b *broker) handleGetRequest(req *getRequest) {
	if req.entryCount <= 0 || req.entryCount > b.settings.MaxGetRequest {
		req.entryCount = b.settings.MaxGetRequest
	}
	if b.getRequestShouldBlock(req) {
		b.pendingGetRequest = req
		b.getTimer.Reset(b.settings.FlushTimeout)
		return
	}
	b.handleGetReply(req)
}

func (b *broker) getRequestShouldBlock(req *getRequest) bool {
	if b.settings.FlushTimeout <= 0 {
		// Never block if the flush timeout isn't positive
		return false
	}
	eventsAvailable := b.eventCount - b.consumedCount
	// Block if the available events aren't enough to fill the request
	return eventsAvailable < req.entryCount
}

// Respond to the given get request without blocking or waiting for more events
func (b *broker) handleGetReply(req *getRequest) {
	eventsAvailable := b.eventCount - b.consumedCount
	batchSize := req.entryCount
	if eventsAvailable < batchSize {
		batchSize = eventsAvailable
	}

	startIndex := b.bufPos + b.consumedCount
	batch := newBatch(b, startIndex, batchSize)

	// Send the batch to the caller and update internal state
	req.responseChan <- batch
	b.consumedBatches.append(batch)
	b.consumedCount += batchSize
}

func (b *broker) handleDelete(count int) {
	// Clear the internal event pointers so they can be garbage collected
	for i := 0; i < count; i++ {
		index := (b.bufPos + i) % len(b.buf)
		b.buf[index].event = nil
	}

	// Advance position and counters
	b.bufPos = (b.bufPos + count) % len(b.buf)
	b.eventCount -= count
	b.consumedCount -= count
}

// Called by ackLoop. This function exists to decouple the work of collecting
// and running producer callbacks from logical deletion of the events, so
// input callbacks can't block the queue by occupying the runLoop goroutine.
func (b *broker) processACK(lst batchList, N int) {
	ackCallbacks := []func(){}
	// First we traverse the entries we're about to remove, collecting any callbacks
	// we need to run.
	lst.reverse()
	for !lst.empty() {
		batch := lst.pop()

		// Traverse entries from last to first, so we can acknowledge the most recent
		// ones first and skip subsequent producer callbacks.
		for i := batch.count - 1; i >= 0; i-- {
			entry := batch.rawEntry(i)
			if entry.producer == nil {
				continue
			}

			if entry.producerID <= entry.producer.state.lastACK {
				// This index was already acknowledged on a previous iteration, skip.
				entry.producer = nil
				continue
			}
			producerState := entry.producer.state
			count := int(entry.producerID - producerState.lastACK)
			ackCallbacks = append(ackCallbacks, func() { producerState.cb(count) })
			entry.producer.state.lastACK = entry.producerID
			entry.producer = nil
		}
	}
	// Signal runLoop to delete the events
	b.deleteChan <- N

	// The events have been removed; notify their listeners.
	for _, f := range ackCallbacks {
		f()
	}
}

func (b *broker) handleInsert(req *pushRequest) {
	if b.insert(req, b.nextEntryID) {
		// Send back the new event id.
		req.resp <- b.nextEntryID

		b.nextEntryID++
		b.eventCount++

		// See if this gave us enough for a new batch
		b.maybeUnblockGetRequest()
	}
}

// Checks if we can handle pendingGetRequest yet, and handles it if so
func (b *broker) maybeUnblockGetRequest() {
	// If a get request is blocked waiting for more events, check if
	// we should unblock it.
	if getRequest := b.pendingGetRequest; getRequest != nil {
		available := b.eventCount - b.consumedCount
		if available >= getRequest.entryCount {
			b.pendingGetRequest = nil
			if b.getTimer.Stop() {
				<-b.getTimer.C
			}
			b.handleGetReply(getRequest)
		}
	}
}

func (b *broker) insert(req *pushRequest, id queue.EntryID) bool {
	if req.producer != nil && req.producer.state.cancelled {
		reportCancelledState(b.logger, req)
		return false
	}

	index := (b.bufPos + b.eventCount) % len(b.buf)
	b.buf[index] = queueEntry{
		event:      req.event,
		id:         id,
		producer:   req.producer,
		producerID: req.producerID,
	}
	return true
}

func (b *broker) handleMetricsRequest(req *metricsRequest) {
	req.responseChan <- memQueueMetrics{
		currentQueueSize: b.eventCount,
		occupiedRead:     b.consumedCount,
	}
}

func (b *broker) handleCancel(req *producerCancelRequest) {
	var removedCount int

	// Traverse all unconsumed events in the buffer, removing any with
	// the specified producer.
	startIndex := b.bufPos + b.consumedCount
	unconsumedEventCount := b.eventCount - b.consumedCount
	for i := 0; i < unconsumedEventCount; i++ {
		readIndex := (startIndex + i) % len(b.buf)
		if b.buf[readIndex].producer == req.producer {
			// The producer matches, skip this event
			removedCount++
		} else {
			// (not readIndex - removedCount since then we'd have sign issues when
			// the buffer wraps.)
			writeIndex := (startIndex + i - removedCount) % len(b.buf)
			b.buf[writeIndex] = b.buf[readIndex]
		}
	}

	// Clear the event pointers at the end of the buffer so we don't keep
	// old events in memory by accident.
	for i := 0; i < removedCount; i++ {
		index := (b.bufPos + b.eventCount - removedCount + i) % len(b.buf)
		b.buf[index].event = nil
	}

	// signal cancel request being finished
	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: removedCount}
	}
}

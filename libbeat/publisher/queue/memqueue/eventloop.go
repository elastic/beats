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
	broker *broker
	buf    ringBuffer

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs chanList

	nextEntryID queue.EntryID
}

// bufferingEventLoop implements the broker main event loop.
// Events in the buffer are forwarded to consumers only if the buffer is full or on flush timeout.
type bufferingEventLoop struct {
	broker *broker

	buf       *batchBuffer
	flushList flushList

	// The number of events currently waiting in the queue, including
	// those that have not yet been acked.
	eventCount int

	// The number of events that have been read by a consumer but not yet acked
	unackedEventCount int

	minEvents    int
	maxEvents    int
	flushTimeout time.Duration

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs chanList

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
		broker: b,
	}
	l.buf.init(b.logger, size)

	return l
}

func (l *directEventLoop) run() {
	var (
		broker = l.broker
		buf    = &l.buf
	)

	for {
		var pushChan chan pushRequest
		// Push requests are enabled if the queue isn't yet full.
		if !l.buf.Full() {
			pushChan = l.broker.pushChan
		}

		var getChan chan getRequest
		// Get requests are enabled if there are events in the queue
		// that haven't yet been sent to a consumer.
		if buf.Avail() > 0 {
			getChan = l.broker.getChan
		}

		var schedACKs chan chanList
		// Sending pending ACKs to the broker's scheduled ACKs
		// channel is enabled if it is nonempty.
		if !l.pendingACKs.empty() {
			schedACKs = l.broker.scheduledACKs
		}

		select {
		case <-broker.done:
			return

		case req := <-pushChan: // producer pushing new event
			l.insert(&req)

		case count := <-l.broker.ackChan:
			// Events have been ACKed, remove them from the internal buffer.
			l.buf.removeEntries(count)

		case req := <-l.broker.cancelChan: // producer cancelling active events
			l.handleCancel(&req)
			// re-enable pushRequest if buffer can take new events

		case req := <-getChan: // consumer asking for next batch
			l.handleGetRequest(&req)

		case req := <-l.broker.metricChan: // broker asking for queue metrics
			l.handleMetricsRequest(&req)

		case schedACKs <- l.pendingACKs:
			// on send complete list of pending batches has been forwarded -> clear list
			l.pendingACKs = chanList{}
		}
	}
}

func (l *directEventLoop) handleMetricsRequest(req *metricsRequest) {
	req.responseChan <- memQueueMetrics{currentQueueSize: l.buf.Items(), occupiedRead: l.buf.reserved}
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

func (l *directEventLoop) handleGetRequest(req *getRequest) {
	// log := l.broker.logger
	// log.Debugf("try reserve %v events", req.sz)

	start, buf := l.buf.reserve(req.entryCount)
	count := len(buf)
	if count == 0 {
		panic("empty batch returned")
	}

	ackCH := newBatchACKState(start, count, l.buf.entries)

	req.responseChan <- getResponse{ackCH.doneChan, buf}
	l.pendingACKs.append(ackCH)
}

// processACK is called by the ackLoop to process the list of acked batches
func (l *directEventLoop) processACK(lst chanList, N int) {
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
	// highest produer IDs first and can skip subsequent callbacks to the
	// same producer.
	for i := N - 1; i >= 0; i-- {
		// idx is the index in entries of the i-th event after firstIndex, wrapping
		// around the end of the array.
		idx := (firstIndex + i) % len(entries)
		entry := &entries[idx]

		producer := entry.producer
		entry.producer = nil
		if producer == nil || entry.producerID <= producer.state.lastACK {
			// This has a lower index than the previous ACK for this producer,
			// so it was covered in the previous call and we can skip it.
			continue
		}
		// This update is safe because lastACK is only used from the event loop.
		count := entry.producerID - producer.state.lastACK
		producer.state.lastACK = entry.producerID

		producer.state.cb(int(count))
	}
}

func newBufferingEventLoop(b *broker, size int, minEvents int, flushTimeout time.Duration) *bufferingEventLoop {
	l := &bufferingEventLoop{
		broker:       b,
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

func (l *bufferingEventLoop) run() {
	broker := l.broker

	for {
		var pushChan chan pushRequest
		// Push requests are enabled if the queue isn't yet full.
		if l.eventCount < l.maxEvents {
			pushChan = l.broker.pushChan
		}

		var getChan chan getRequest
		// Get requests are enabled if the queue has events that
		// weren't yet sent to consumers.
		if !l.flushList.empty() {
			getChan = l.broker.getChan
		}

		var schedACKs chan chanList
		// Enable sending to the scheduled ACKs channel if we have
		// something to send.
		if !l.pendingACKs.empty() {
			schedACKs = l.broker.scheduledACKs
		}

		select {
		case <-broker.done:
			return

		case req := <-pushChan: // producer pushing new event
			l.handleInsert(&req)

		case req := <-l.broker.cancelChan: // producer cancelling active events
			l.handleCancel(&req)

		case req := <-getChan: // consumer asking for next batch
			l.handleGetRequest(&req)

		case schedACKs <- l.pendingACKs:
			l.pendingACKs = chanList{}

		case count := <-l.broker.ackChan:
			l.handleACK(count)

		case req := <-l.broker.metricChan: // broker asking for queue metrics
			l.handleMetricsRequest(&req)

		case <-l.idleC:
			l.idleC = nil
			l.timer.Stop()
			if l.buf.length() > 0 {
				l.flushBuffer()
			}
		}
	}
}

func (l *bufferingEventLoop) handleMetricsRequest(req *metricsRequest) {
	req.responseChan <- memQueueMetrics{currentQueueSize: l.eventCount, occupiedRead: l.unackedEventCount}
}

func (l *bufferingEventLoop) handleInsert(req *pushRequest) {
	if l.insert(req, l.nextEntryID) {
		// Send back the new event id.
		req.resp <- l.nextEntryID

		l.nextEntryID++
		l.eventCount++

		L := l.buf.length()
		if !l.buf.flushed {
			if L < l.minEvents {
				l.startFlushTimer()
			} else {
				l.stopFlushTimer()
				l.flushBuffer()
				l.buf = newBatchBuffer(l.minEvents)
			}
		} else {
			if L >= l.minEvents {
				l.buf = newBatchBuffer(l.minEvents)
			}
		}
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

func (l *bufferingEventLoop) handleGetRequest(req *getRequest) {
	buf := l.flushList.head
	if buf == nil {
		panic("get from non-flushed buffers")
	}

	count := buf.length()
	if count == 0 {
		panic("empty buffer in flush list")
	}

	if sz := req.entryCount; sz > 0 {
		if sz < count {
			count = sz
		}
	}

	if count == 0 {
		panic("empty batch returned")
	}

	entries := buf.entries[:count]
	acker := newBatchACKState(0, count, entries)

	req.responseChan <- getResponse{acker.doneChan, entries}
	l.pendingACKs.append(acker)

	l.unackedEventCount += len(entries)
	buf.entries = buf.entries[count:]
	if buf.length() == 0 {
		l.advanceFlushList()
	}
}

func (l *bufferingEventLoop) handleACK(count int) {
	l.unackedEventCount -= count
	l.eventCount -= count
}

func (l *bufferingEventLoop) startFlushTimer() {
	if l.idleC == nil {
		l.timer.Reset(l.flushTimeout)
		l.idleC = l.timer.C
	}
}

func (l *bufferingEventLoop) stopFlushTimer() {
	if l.idleC != nil {
		l.idleC = nil
		if !l.timer.Stop() {
			<-l.timer.C
		}
	}
}

func (l *bufferingEventLoop) advanceFlushList() {
	l.flushList.pop()
	if l.flushList.count == 0 && l.buf.flushed {
		l.buf = newBatchBuffer(l.minEvents)
	}
}

func (l *bufferingEventLoop) flushBuffer() {
	l.buf.flushed = true

	if l.buf.length() == 0 {
		panic("flushing empty buffer")
	}

	l.flushList.add(l.buf)
}

func (l *bufferingEventLoop) processACK(lst chanList, N int) {
	lst.reverse()
	for !lst.empty() {
		current := lst.pop()
		entries := current.entries

		// Traverse entries from last to first, so we can acknowledge the most recent
		// ones first and skip subsequent producer callbacks.
		for i := len(entries) - 1; i >= 0; i-- {
			entry := &entries[i]
			if entry.producer == nil {
				continue
			}

			if entry.producerID <= entry.producer.state.lastACK {
				// This index was already acknowledged on a previous iteration, skip.
				entry.producer = nil
				continue
			}
			count := entry.producerID - entry.producer.state.lastACK

			entry.producer.state.cb(int(count))
			entry.producer.state.lastACK = entry.producerID
			entry.producer = nil
		}
	}
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

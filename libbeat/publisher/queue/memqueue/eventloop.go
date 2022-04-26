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
	"math"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// directEventLoop implements the broker main event loop. It buffers events,
// but tries to forward events as early as possible.
type directEventLoop struct {
	broker *broker

	buf ringBuffer

	// active broker API channels
	events    chan pushRequest
	get       chan getRequest
	pubCancel chan producerCancelRequest

	// ack handling
	acks        chan int      // ackloop -> eventloop : total number of events ACKed by outputs
	schedACKS   chan chanList // eventloop -> ackloop : active list of batches to be acked
	pendingACKs chanList      // ordered list of active batches to be send to the ackloop
	ackSeq      uint          // ack batch sequence number to validate ordering
}

// bufferingEventLoop implements the broker main event loop.
// Events in the buffer are forwarded to consumers only if the buffer is full or on flush timeout.
type bufferingEventLoop struct {
	broker *broker

	buf        *batchBuffer
	flushList  flushList
	eventCount int

	minEvents    int
	maxEvents    int
	flushTimeout time.Duration

	// active broker API channels
	events    chan pushRequest
	get       chan getRequest
	pubCancel chan producerCancelRequest

	// ack handling
	acks        chan int      // ackloop -> eventloop : total number of events ACKed by outputs
	schedACKS   chan chanList // eventloop -> ackloop : active list of batches to be acked
	pendingACKs chanList      // ordered list of active batches to be send to the ackloop
	ackSeq      uint          // ack batch sequence number to validate ordering

	// buffer flush timer state
	timer *time.Timer
	idleC <-chan time.Time
}

type flushList struct {
	head  *batchBuffer
	tail  *batchBuffer
	count int
}

func newDirectEventLoop(b *broker, size int) *directEventLoop {
	l := &directEventLoop{
		broker:    b,
		events:    b.events,
		get:       nil,
		pubCancel: b.pubCancel,
		acks:      b.acks,
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
		select {
		case <-broker.done:
			//fmt.Printf("broker.done\n")
			return

		case req := <-l.events: // producer pushing new event
			//fmt.Printf("handleInsert\n")
			l.handleInsert(&req)

		case req := <-l.pubCancel: // producer cancelling active events
			//fmt.Printf("handleCancel\n")
			l.handleCancel(&req)

		case req := <-l.get: // consumer asking for next batch
			//fmt.Printf("handleGetRequest\n")
			l.handleGetRequest(&req)

		case l.schedACKS <- l.pendingACKs:
			// on send complete list of pending batches has been forwarded -> clear list and queue
			//fmt.Printf("pendingACKs\n")

			l.schedACKS = nil
			l.pendingACKs = chanList{}

		case count := <-l.acks:
			//fmt.Printf("handleACK(%d)\n", count)

			l.handleACK(count)

		}

		// update get and idle timer after state machine
		l.get = nil
		if buf.Avail() > 0 {
			l.get = broker.requests
		}
	}
}

func (l *directEventLoop) handleInsert(req *pushRequest) {
	// log := l.broker.logger
	// log.Debugf("push event: %v\t%v\t%p\n", req.event, req.seq, req.state)

	if full := l.insert(req); full {
		// log.Debugf("buffer: all regions full")

		// no more space to accept new events -> unset events queue for time being
		l.events = nil
	}
}

// Returns true if the queue is full after handling the insertion request.
func (l *directEventLoop) insert(req *pushRequest) bool {
	log := l.broker.logger

	st := req.state
	if st == nil {
		return l.buf.insert(req.event, clientState{})
	}

	if st.cancelled {
		reportCancelledState(log, req)
		return false
	}

	return l.buf.insert(req.event, clientState{
		seq:   req.seq,
		state: st,
	})
}

func (l *directEventLoop) handleCancel(req *producerCancelRequest) {
	// log := l.broker.logger
	// log.Debug("handle cancel request")

	var (
		removed int
		broker  = l.broker
	)

	if st := req.state; st != nil {
		st.cancelled = true
		removed = l.buf.cancel(st)
	}

	// signal cancel request being finished
	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: removed}
	}

	// re-enable pushRequest if buffer can take new events
	if !l.buf.Full() {
		l.events = broker.events
	}
}

func (l *directEventLoop) handleGetRequest(req *getRequest) {
	// log := l.broker.logger
	// log.Debugf("try reserve %v events", req.sz)

	start, buf := l.buf.reserve(req.sz)
	count := len(buf)
	if count == 0 {
		panic("empty batch returned")
	}

	// log.Debug("newACKChan: ", b.ackSeq, count)
	ackCH := newACKChan(l.ackSeq, start, count, l.buf.entries)
	l.ackSeq++

	req.resp <- getResponse{ackCH, buf}
	l.pendingACKs.append(ackCH)
	l.schedACKS = l.broker.scheduledACKs
}

func (l *directEventLoop) handleACK(count int) {
	// log := l.broker.logger
	// log.Debug("receive buffer ack:", count)

	// Give broker/buffer a chance to clean up most recent ACKs
	// After handling ACKs some buffer has been freed up
	// -> always reenable producers
	l.buf.ack(count)
	l.events = l.broker.events
}

// processACK is used by the ackLoop to process the list of acked batches
func (l *directEventLoop) processACK(lst chanList, N int) {
	log := l.broker.logger
	{
		start := time.Now()
		log.Debug("handle ACKs: ", N)
		defer func() {
			log.Debug("handle ACK took: ", time.Since(start))
		}()
	}

	acks := lst.front()
	start := acks.start
	entries := l.buf.entries

	idx := start + N - 1
	if idx >= len(entries) {
		idx -= len(entries)
	}

	total := 0
	for i := N - 1; i >= 0; i-- {
		if idx < 0 {
			idx = len(entries) - 1
		}

		client := &entries[idx].client
		log.Debugf("try ack index: (idx=%v, i=%v, seq=%v)\n", idx, i, client.seq)

		idx--
		if client.state == nil {
			log.Debug("no state set")
			continue
		}

		count := (client.seq - client.state.lastACK)
		if count == 0 || count > math.MaxUint32/2 {
			// seq number comparison did underflow. This happens only if st.seq has
			// already been acknowledged
			// log.Debug("seq number already acked: ", st.seq)

			client.state = nil
			continue
		}

		log.Debugf("broker ACK events: count=%v, start-seq=%v, end-seq=%v\n",
			count,
			client.state.lastACK+1,
			client.seq,
		)

		total += int(count)
		if total > N {
			panic(fmt.Sprintf("Too many events acked (expected=%v, total=%v)",
				N, total,
			))
		}

		client.state.cb(int(count))
		client.state.lastACK = client.seq
		client.state = nil
	}
}

func newBufferingEventLoop(b *broker, size int, minEvents int, flushTimeout time.Duration) *bufferingEventLoop {
	l := &bufferingEventLoop{
		broker:       b,
		maxEvents:    size,
		minEvents:    minEvents,
		flushTimeout: flushTimeout,

		events:    b.events,
		get:       nil,
		pubCancel: b.pubCancel,
		acks:      b.acks,
	}
	l.buf = newBatchBuffer(l.minEvents)

	l.timer = time.NewTimer(flushTimeout)
	if !l.timer.Stop() {
		<-l.timer.C
	}

	return l
}

func (l *bufferingEventLoop) run() {
	var (
		broker = l.broker
	)
	fmt.Printf("oops someone is running a bufferingEventLoop\n")

	for {
		select {
		case <-broker.done:
			return

		case req := <-l.events: // producer pushing new event
			l.handleInsert(&req)

		case req := <-l.pubCancel: // producer cancelling active events
			l.handleCancel(&req)

		case req := <-l.get: // consumer asking for next batch
			l.handleGetRequest(&req)

		case l.schedACKS <- l.pendingACKs:
			l.schedACKS = nil
			l.pendingACKs = chanList{}

		case count := <-l.acks:
			l.handleACK(count)

		case <-l.idleC:
			l.idleC = nil
			l.timer.Stop()
			if l.buf.length() > 0 {
				l.flushBuffer()
			}
		}
	}
}

func (l *bufferingEventLoop) handleInsert(req *pushRequest) {
	if l.insert(req) {
		l.eventCount++
		if l.eventCount == l.maxEvents {
			l.events = nil // stop inserting events if upper limit is reached
		}

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

func (l *bufferingEventLoop) insert(req *pushRequest) bool {
	if req.state == nil {
		l.buf.add(req.event, clientState{})
		return true
	}

	st := req.state
	if st.cancelled {
		reportCancelledState(l.broker.logger, req)
		return false
	}

	l.buf.add(req.event, clientState{
		seq:   req.seq,
		state: st,
	})
	return true
}

func (l *bufferingEventLoop) handleCancel(req *producerCancelRequest) {
	removed := 0
	if st := req.state; st != nil {
		// remove from actively flushed buffers
		for buf := l.flushList.head; buf != nil; buf = buf.next {
			removed += buf.cancel(st)
		}
		if !l.buf.flushed {
			removed += l.buf.cancel(st)
		}

		st.cancelled = true
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
	if tmpList.empty() {
		l.get = nil
	}

	l.eventCount -= removed
	if l.eventCount < l.maxEvents {
		l.events = l.broker.events
	}
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

	if sz := req.sz; sz > 0 {
		if sz < count {
			count = sz
		}
	}

	if count == 0 {
		panic("empty batch returned")
	}

	entries := buf.entries[:count]
	ackChan := newACKChan(l.ackSeq, 0, count, entries)
	l.ackSeq++

	req.resp <- getResponse{ackChan, entries}
	l.pendingACKs.append(ackChan)
	l.schedACKS = l.broker.scheduledACKs

	buf.entries = buf.entries[count:]
	if buf.length() == 0 {
		l.advanceFlushList()
	}
}

func (l *bufferingEventLoop) handleACK(count int) {
	l.eventCount -= count
	if l.eventCount < l.maxEvents {
		l.events = l.broker.events
	}
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
	if l.flushList.count == 0 {
		// All buffers are empty, disable consumer get
		l.get = nil

		if l.buf.flushed {
			l.buf = newBatchBuffer(l.minEvents)
		}
	}
}

func (l *bufferingEventLoop) flushBuffer() {
	l.buf.flushed = true

	if l.buf.length() == 0 {
		panic("flushing empty buffer")
	}

	l.flushList.add(l.buf)
	l.get = l.broker.requests
}

func (l *bufferingEventLoop) processACK(lst chanList, N int) {
	log := l.broker.logger

	total := 0
	lst.reverse()
	for !lst.empty() {
		current := lst.pop()
		entries := current.entries

		for i := len(entries) - 1; i >= 0; i-- {
			st := &entries[i].client
			if st.state == nil {
				continue
			}

			count := st.seq - st.state.lastACK
			if count == 0 || count > math.MaxUint32/2 {
				// seq number comparison did underflow. This happens only if st.seq has
				// already been acknowledged
				// log.Debug("seq number already acked: ", st.seq)

				st.state = nil
				continue
			}

			log.Debugf("broker ACK events: count=%v, start-seq=%v, end-seq=%v\n",
				count,
				st.state.lastACK+1,
				st.seq,
			)

			total += int(count)
			if total > N {
				panic(fmt.Sprintf("Too many events acked (expected=%v, total=%v)",
					N, total,
				))
			}

			st.state.cb(int(count))
			st.state.lastACK = st.seq
			st.state = nil
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
	log.Debugf("cancelled producer - ignore event: %v\t%v\t%p", req.event, req.seq, req.state)

	// do not add waiting events if producer did send cancel signal

	st := req.state
	if cb := st.dropCB; cb != nil {
		cb(req.event.Content)
	}

}

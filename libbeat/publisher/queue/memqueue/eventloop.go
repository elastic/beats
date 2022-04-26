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
	buf    ringBuffer

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs chanList
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

	minEvents    int
	maxEvents    int
	flushTimeout time.Duration

	// active broker API channels
	pushChan   chan pushRequest
	getChan    chan getRequest
	cancelChan chan producerCancelRequest

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs chanList

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

		case schedACKs <- l.pendingACKs:
			// on send complete list of pending batches has been forwarded -> clear list
			l.pendingACKs = chanList{}
		}
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

	var removed int

	if st := req.state; st != nil {
		st.cancelled = true
		removed = l.buf.cancel(st)
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

	req.responseChan <- getResponse{ackCH.ackChan, buf}
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

		pushChan:   b.pushChan,
		getChan:    nil,
		cancelChan: b.cancelChan,
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
		var schedACKs chan chanList
		if !l.pendingACKs.empty() {
			// Enable sending to the scheduled ACKs channel if we have
			// something to send.
			schedACKs = l.broker.scheduledACKs
		}

		select {
		case <-broker.done:
			return

		case req := <-l.pushChan: // producer pushing new event
			l.handleInsert(&req)

		case req := <-l.cancelChan: // producer cancelling active events
			l.handleCancel(&req)

		case req := <-l.getChan: // consumer asking for next batch
			l.handleGetRequest(&req)

		case schedACKs <- l.pendingACKs:
			l.pendingACKs = chanList{}

		case count := <-l.broker.ackChan:
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
			l.pushChan = nil // stop inserting events if upper limit is reached
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
		l.getChan = nil
	}

	l.eventCount -= removed
	if l.eventCount < l.maxEvents {
		l.pushChan = l.broker.pushChan
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

	req.responseChan <- getResponse{acker.ackChan, entries}
	l.pendingACKs.append(acker)

	buf.entries = buf.entries[count:]
	if buf.length() == 0 {
		l.advanceFlushList()
	}
}

func (l *bufferingEventLoop) handleACK(count int) {
	l.eventCount -= count
	if l.eventCount < l.maxEvents {
		l.pushChan = l.broker.pushChan
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
		l.getChan = nil

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
	l.getChan = l.broker.getChan
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

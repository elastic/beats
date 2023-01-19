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

package proxyqueue

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// eventLoop implements the broker main event loop. It buffers events,
// but tries to forward events as early as possible.
type eventLoop struct {
	broker *broker
	//buf        ringBuffer
	//deleteChan chan int

	// pendingACKs aggregates a list of ACK channels for batches that have been sent
	// to consumers, which is then sent to the broker's scheduledACKs channel.
	pendingACKs chanList

	nextEntryID queue.EntryID
}

func newEventLoop(b *broker, batchSize int) *eventLoop {
	l := &eventLoop{
		broker:     b,
		deleteChan: make(chan int),
	}

	return l
}

func (l *eventLoop) isFull() bool {
	// TODO: add real logic
	return false
}

func (b *broker) newBatch() *ProxiedBatch {
	return &ProxiedBatch{
		queue: b,
		doneChan: make(chan batchDoneMsg)
	}
}

func (b *broker) run() {
	var (
		pendingBatch = b.newBatch()
		pendingACKs pendingACKsList
	)

	for {
		var pushChan chan pushRequest
		// Push requests are enabled if the pending batch isn't yet full.
		if len(pendingBatch.entries) < b.batchSize {
			pushChan = b.pushChan
		}

		var getChan chan getRequest
		// Get requests are enabled if the current pending batch is nonempty.
		if len(pendingBatch.entries) > 0 {
			getChan = l.broker.getChan
		}

		select {
		case <-broker.done:
			return

		case req := <-pushChan: // producer pushing new event
			l.insert(&req)

		case req := <-getChan: // consumer asking for next batch
			l.handleGetRequest(&req)

		case <-pendingACKs.nextDoneChan():
			// TODO: propagate ACKs
		}
	}
}

func (l *eventLoop) insert(req *pushRequest) {
	log := l.broker.logger

	if req.producer != nil && req.producer.state.cancelled {
		reportCancelledState(log, req)
	} else {
		req.responseChan <- l.nextEntryID
		l.buf.insert(queueEntry{
			event:      req.event,
			id:         l.nextEntryID,
			producer:   req.producer,
			producerID: req.producerID})
		l.nextEntryID++
	}
}

func (l *eventLoop) handleCancel(req *producerCancelRequest) {
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

func (l *eventLoop) handleGetRequest(req *getRequest) {
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
func (l *eventLoop) processACK(lst chanList, N int) {
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

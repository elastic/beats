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
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

type forgetfulProducer[T any] struct {
	broker    *broker[T]
	openState openState[T]

	// A forgetful producer has no ACK callback, so ackWait is closed as soon as
	// Close is called — there are no acknowledgments to wait for.
	ackWait chan struct{}
	ackOnce sync.Once
}

type ackProducer[T any] struct {
	broker    *broker[T]
	state     produceState
	openState openState[T]

	// producedCount / ackedCount track this producer's published vs
	// acknowledged events for ackWait accounting. producedCount is written on
	// the publishing goroutine, ackedCount on the broker's ackLoop goroutine,
	// so both are atomic. closed is set by Close. ackWait is closed once the
	// producer is closed and ackedCount has caught up with producedCount (in
	// maybeCloseAckWait), or by the broker on shutdown (the ackLoop stops then,
	// so no further acks would arrive) — see broker.closeProducerAckWaits.
	producedCount atomic.Uint64
	ackedCount    atomic.Uint64
	closed        atomic.Bool
	ackWait       chan struct{}
	ackOnce       sync.Once
}

type openState[T any] struct {
	log          *logp.Logger
	done         chan struct{}
	queueClosing <-chan struct{}
	events       chan pushRequest[T]
	encoder      queue.Encoder[T]

	// resp is used to receive the assigned EntryID after the runLoop
	// processes a push request. It is allocated once per producer and
	// reused across publishes. Publish is synchronous, so only one
	// request is outstanding at a time.
	resp chan queue.EntryID
}

// producerID stores the order of events within a single producer, so multiple
// event acknowledgement callbacks can be coalesced into a single call.
// It is defined as an explicit type to reduce cross-confusion with the id
// of an event in the queue itself, which is a queue.EntryID.
type producerID uint64

type produceState struct {
	cb      ackHandler
	lastACK producerID
}

type ackHandler func(count int)

func newProducer[T any](b *broker[T], cb ackHandler, encoder queue.Encoder[T]) queue.Producer[T] {
	openState := openState[T]{
		log:          b.logger,
		done:         make(chan struct{}),
		queueClosing: b.closingChan,
		events:       b.pushChan,
		encoder:      encoder,
		resp:         make(chan queue.EntryID, 1),
	}

	if cb != nil {
		p := &ackProducer[T]{
			broker:    b,
			openState: openState,
			ackWait:   make(chan struct{}),
		}
		// Wrap the user callback so the producer's ack accounting advances on
		// the ackLoop goroutine alongside delivery, without changing the
		// callback the caller sees.
		p.state.cb = func(n int) {
			p.ackedCount.Add(uint64(n)) //nolint:gosec // G115: n is an ack count, always a small positive value
			cb(n)
			p.maybeCloseAckWait()
		}
		// Register so the broker can close ackWait on shutdown if the producer
		// is closed before its events are acked (the ackLoop stops on shutdown).
		b.registerProducer(p)
		return p
	}
	return &forgetfulProducer[T]{
		broker:    b,
		openState: openState,
		ackWait:   make(chan struct{}),
	}
}

func (p *forgetfulProducer[T]) makePushRequest(event T) pushRequest[T] {
	return pushRequest[T]{
		event: event,
		resp:  p.openState.resp}
}

func (p *forgetfulProducer[T]) Publish(event T) (queue.EntryID, bool) {
	return p.openState.publish(p.makePushRequest(event))
}

func (p *forgetfulProducer[T]) TryPublish(event T) (queue.EntryID, bool) {
	return p.openState.tryPublish(p.makePushRequest(event))
}

func (p *forgetfulProducer[T]) Close() {
	p.ackOnce.Do(func() { close(p.ackWait) })
	p.openState.Close()
}

func (p *forgetfulProducer[T]) ACKWaitChan() <-chan struct{} { return p.ackWait }

func (p *ackProducer[T]) makePushRequest(event T, id producerID) pushRequest[T] {
	return pushRequest[T]{
		event:      event,
		producer:   p,
		producerID: id,
		resp:       p.openState.resp}
}

// Publish adds an event to the queue, blocking until there is room. It returns
// the assigned entry ID and whether the event was accepted (false if the queue
// closed first).
func (p *ackProducer[T]) Publish(event T) (queue.EntryID, bool) {
	// Count the event before enqueuing it, else a concurrent Close could see
	// ackedCount >= producedCount and close ackWait while it's still pending. The
	// count doubles as the 1-based producerID.
	id := producerID(p.producedCount.Add(1)) //nolint:gosec // G115: monotonic per-producer publish count
	entryID, published := p.openState.publish(p.makePushRequest(event, id))
	if !published {
		// Never enqueued: undo the count and re-check so it can't strand ackWait.
		p.producedCount.Add(^uint64(0)) // -1
		p.maybeCloseAckWait()
	}
	return entryID, published
}

// TryPublish adds an event to the queue only if there is room right now, never
// blocking. It returns the assigned entry ID and whether the event was accepted.
func (p *ackProducer[T]) TryPublish(event T) (queue.EntryID, bool) {
	// Count the event before enqueuing it, else a concurrent Close could see
	// ackedCount >= producedCount and close ackWait while it's still pending. The
	// count doubles as the 1-based producerID.
	id := producerID(p.producedCount.Add(1)) //nolint:gosec // G115: monotonic per-producer publish count
	entryID, published := p.openState.tryPublish(p.makePushRequest(event, id))
	if !published {
		p.producedCount.Add(^uint64(0)) // -1
		p.maybeCloseAckWait()
	}
	return entryID, published
}

func (p *ackProducer[T]) Close() {
	p.closed.Store(true)
	// Events published before Close may already be fully acked, in which case
	// no further callback will fire to close ackWait — check now.
	p.maybeCloseAckWait()
	p.openState.Close()
}

func (p *ackProducer[T]) ACKWaitChan() <-chan struct{} { return p.ackWait }

// maybeCloseAckWait closes ackWait exactly once, when the producer has been
// closed and every published event has been acknowledged, and unregisters the
// producer from the broker (it no longer needs the shutdown fan-out).
func (p *ackProducer[T]) maybeCloseAckWait() {
	if p.closed.Load() && p.ackedCount.Load() >= p.producedCount.Load() {
		p.ackOnce.Do(func() {
			close(p.ackWait)
			p.broker.unregisterProducer(p)
		})
	}
}

// forceCloseAckWait closes ackWait unconditionally. Used by the broker on
// shutdown to unblock a waiter even though the producer's events were never
// acknowledged (the ackLoop has stopped). Unregistration is handled by the
// broker's snapshot, so this does not call unregisterProducer.
func (p *ackProducer[T]) forceCloseAckWait() {
	p.ackOnce.Do(func() { close(p.ackWait) })
}

func (st *openState[T]) Close() {
	close(st.done)
}

func (st *openState[T]) publish(req pushRequest[T]) (queue.EntryID, bool) {
	// If we were given an encoder callback for incoming events, apply it before
	// sending the entry to the queue.
	if st.encoder != nil {
		req.event, req.eventSize = st.encoder.EncodeEntry(req.event)
	}
	select {
	case st.events <- req:
		return st.handlePendingResponse(req.resp)
	case <-st.done:
		st.events = nil
		return 0, false
	case <-st.queueClosing:
		st.events = nil
		return 0, false
	}
}

func (st *openState[T]) tryPublish(req pushRequest[T]) (queue.EntryID, bool) {
	// If we were given an encoder callback for incoming events, apply it before
	// sending the entry to the queue.
	if st.encoder != nil {
		req.event, req.eventSize = st.encoder.EncodeEntry(req.event)
	}
	select {
	case st.events <- req:
		return st.handlePendingResponse(req.resp)
	case <-st.done:
		st.events = nil
		return 0, false
	default:
		st.log.Debugf("Dropping event, queue is blocked")
		return 0, false
	}
}

func (st *openState[T]) handlePendingResponse(respChan chan queue.EntryID) (queue.EntryID, bool) {
	// The events channel is buffered, which means we may successfully
	// write to it even if the queue is shutting down. To avoid blocking
	// forever during shutdown, we also have to wait on the queue's
	// shutdown channel.
	select {
	case resp := <-respChan:
		return resp, true
	case <-st.queueClosing:
	}

	// Clear the request channel so we can't write to it again
	st.events = nil

	// Once the queue starts closing, it will not handle any more push
	// requests, however it may have handled ours before the closing
	// channel was triggered (and both may have arrived concurrently
	// at the select statement above). So to know whether our entry was
	// accepted we also need to check if there's a buffered response in
	// our channel.
	select {
	case resp := <-respChan:
		return resp, true
	default:
	}
	return 0, false
}

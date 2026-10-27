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
	"context"
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

var (
	_ queue.ContextProducer[int] = (*forgetfulProducer[int])(nil)
	_ queue.ContextProducer[int] = (*ackProducer[int])(nil)
)

type openState[T any] struct {
	log          *logp.Logger
	done         chan struct{}
	queueClosing <-chan struct{}
	events       chan pushRequest[T]
	encoder      queue.Encoder[T]

	// resp is used to receive the assigned EntryID after the runLoop
	// processes a push request. It is allocated once per producer and
	// reused across publishes. Publish is synchronous, so a producer waits
	// on one request at a time. A request abandoned by PublishContext gets
	// no reply (see pushRequest.claim), so it never leaves a stale EntryID
	// in resp.
	resp chan queue.EntryID

	// claim is handed to the run loop with each PublishContext request (see
	// pushRequest.claim). It is reused until a request is abandoned, then
	// replaced, because the abandoned request still refers to it.
	claim *atomic.Bool
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

// PublishContext is Publish, but stops waiting when ctx is done and returns
// ctx.Err(); the event is then never added. See queue.ContextProducer.
func (p *forgetfulProducer[T]) PublishContext(ctx context.Context, event T) (queue.EntryID, error) {
	return p.openState.publishContext(ctx, p.makePushRequest(event))
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

// PublishContext is Publish, but stops waiting when ctx is done and returns
// ctx.Err(); the event is then never added. See queue.ContextProducer.
func (p *ackProducer[T]) PublishContext(ctx context.Context, event T) (queue.EntryID, error) {
	// Counted before enqueuing for the same reason as in Publish.
	id := producerID(p.producedCount.Add(1))
	entryID, err := p.openState.publishContext(ctx, p.makePushRequest(event, id))
	if err != nil {
		// Never enqueued (publishContext guarantees it): undo the count.
		p.producedCount.Add(^uint64(0)) // -1
		p.maybeCloseAckWait()
	}
	return entryID, err
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

// publishContext is publish, but gives up when ctx is done. A request already
// sent to pushChan cannot be taken back, so on cancel it races the run loop
// for req.claim: if the producer wins, the run loop discards the request and
// the event is never added; if the run loop wins, it has added the event and
// sent the reply, so the publish succeeds despite ctx.
func (st *openState[T]) publishContext(ctx context.Context, req pushRequest[T]) (queue.EntryID, error) {
	if st.encoder != nil {
		req.event, req.eventSize = st.encoder.EncodeEntry(req.event)
	}
	if st.claim == nil {
		st.claim = new(atomic.Bool)
	}
	// The previous request using this claim, if any, was handled by the run
	// loop, so no one else refers to it.
	st.claim.Store(false)
	req.claim = st.claim
	select {
	case st.events <- req:
	case <-st.done:
		st.events = nil
		return 0, queue.ErrClosed
	case <-st.queueClosing:
		st.events = nil
		return 0, queue.ErrClosed
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	select {
	case id := <-req.resp:
		return id, nil
	case <-st.queueClosing:
		id, ok := st.handlePendingResponse(req.resp)
		if !ok {
			return 0, queue.ErrClosed
		}
		return id, nil
	case <-ctx.Done():
	}
	if st.claim.CompareAndSwap(false, true) {
		st.claim = nil
		return 0, ctx.Err()
	}
	// The run loop claimed the request; its reply is sent right after.
	return <-req.resp, nil
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

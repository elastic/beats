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

package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// directProducer implements queue.Producer[publisher.Event] but bypasses the
// queue entirely. Each event is forwarded directly and synchronously to an
// outputs.Client (typically the otelConsumer) inline in the Publish call.
// This matches the OTel receiver contract where Consume* calls are
// synchronous and blocking. All processors run before Publish is called.
type directProducer struct {
	closed atomic.Bool
	nextID atomic.Uint64

	log    *logp.Logger
	client outputs.Client
	ackFn  func(count int)

	// ctx is cancelled by Close to break any in-flight flush retry loops.
	ctx    context.Context
	cancel context.CancelFunc

	// wg tracks all in-flight flush calls so Close can wait for them.
	wg sync.WaitGroup
}

// directBatch implements publisher.Batch for events sent through the
// directProducer. It holds a snapshot of events and signals the flush
// loop via a channel when the output calls Retry, Cancelled, ACK, or Drop.
type directBatch struct {
	events []publisher.Event
	ackFn  func(count int)

	// doneCh is used by the batch signal methods to communicate the outcome
	// back to the flush loop, which blocks until a terminal signal is received.
	doneCh chan batchResult
}

// batchResult communicates the outcome of a batch publish attempt back to
// the flush loop.
type batchResult struct {
	retry bool // true = re-publish, false = done (ACK or Drop)
}

func (b *directBatch) Events() []publisher.Event { return b.events }

func (b *directBatch) ACK() {
	if b.ackFn != nil {
		b.ackFn(len(b.events))
	}
	b.events = nil
	b.doneCh <- batchResult{retry: false}
}

func (b *directBatch) Drop() {
	b.events = nil
	b.doneCh <- batchResult{retry: false}
}

// Retry signals the flush loop to re-publish this batch. In the queue-backed
// path the eventConsumer re-dispatches the batch to a worker. Here we signal
// the iterative flush loop instead, avoiding recursive client.Publish calls
// that would grow the stack unboundedly.
func (b *directBatch) Retry() {
	b.doneCh <- batchResult{retry: true}
}

// RetryEvents replaces the batch contents with the given subset and signals
// a retry. Events not in the subset are implicitly acknowledged.
func (b *directBatch) RetryEvents(events []publisher.Event) {
	b.events = events
	b.Retry()
}

func (b *directBatch) SplitRetry() bool {
	return len(b.events) > 1
}

// Cancelled indicates the send attempt was aborted and should be retried
// without penalty. In the queue-backed path this re-dispatches via the
// eventConsumer without decreasing the TTL counter. In the direct path
// there is no TTL tracking, so this behaves the same as Retry.
func (b *directBatch) Cancelled() {
	b.doneCh <- batchResult{retry: true}
}

func newDirectProducer(
	log *logp.Logger,
	client outputs.Client,
	ackFn func(count int),
) *directProducer {
	ctx, cancel := context.WithCancel(context.Background())
	return &directProducer{
		log:    log,
		client: client,
		ackFn:  ackFn,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Publish forwards a single event to the output client inline. This blocks
// until the event is acknowledged, dropped, or the producer is closed.
func (dp *directProducer) Publish(entry publisher.Event) (queue.EntryID, bool) {
	if dp.closed.Load() {
		return 0, false
	}
	id := queue.EntryID(dp.nextID.Add(1) - 1)
	dp.flush([]publisher.Event{entry})
	return id, true
}

// TryPublish is the same as Publish for the direct producer since there is
// no queue to be full.
func (dp *directProducer) TryPublish(entry publisher.Event) (queue.EntryID, bool) {
	return dp.Publish(entry)
}

// PublishAll sends all entries as a single batch to the output in one
// ConsumeLogs call, rather than one event at a time.
func (dp *directProducer) PublishAll(entries []publisher.Event) (queue.EntryID, bool) {
	if dp.closed.Load() {
		return 0, false
	}
	id := queue.EntryID(dp.nextID.Add(uint64(len(entries))) - uint64(len(entries)))
	dp.flush(entries)
	return id, true
}

// Close stops accepting new events, signals in-flight flush retry loops to
// start their grace period, and waits for all in-flight flushes to complete.
func (dp *directProducer) Close() {
	if !dp.closed.CompareAndSwap(false, true) {
		return
	}
	dp.cancel()
	dp.wg.Wait()
}

const flushShutdownTimeout = 30 * time.Second

// flush sends a batch of events to the output client and blocks until the
// batch is terminally acknowledged (ACK or Drop), or the publish context
// is cancelled. If the output signals Retry or Cancelled, flush
// re-publishes iteratively.
//
// The publish context starts unbounded. When dp.ctx is cancelled (Close),
// it is replaced with a 30-second timeout so remaining retry attempts are
// bounded.
func (dp *directProducer) flush(events []publisher.Event) {
	if len(events) == 0 {
		return
	}

	dp.wg.Add(1)
	defer dp.wg.Done()

	batch := &directBatch{
		events: events,
		ackFn:  dp.ackFn,
		doneCh: make(chan batchResult, 1),
	}

	// publishCtx is passed to the output client. It is NOT dp.ctx because
	// we don't want an already-cancelled context to be handed to the output
	// (that would cause it to reject calls or initialize its backoff with a
	// closed done channel). Instead we start with an unbounded context and
	// switch to a timeout context once dp.ctx is cancelled.
	publishCtx, publishCancel := context.WithCancel(context.Background())
	defer publishCancel()

	// shutting is dp.ctx.Done() initially. Set to nil after the first
	// shutdown signal to avoid re-entering the shutdown branch.
	shutting := dp.ctx.Done()

	for {
		if err := dp.client.Publish(publishCtx, batch); err != nil {
			dp.log.Errorf("direct producer: failed to publish batch: %v", err)
			return
		}

		// client.Publish is synchronous — the output always calls a batch
		// signal method (ACK, Drop, Retry, Cancelled) before returning,
		// so doneCh is guaranteed to have a result.
		result := <-batch.doneCh
		if !result.retry {
			return
		}

		// Before retrying, check if the producer is shutting down.
		select {
		case <-shutting:
			// Producer is closing. Replace the unbounded publishCtx with
			// a timeout so remaining attempts are bounded.
			publishCancel()
			publishCtx, publishCancel = context.WithTimeout(context.Background(), flushShutdownTimeout)
			defer publishCancel()
			shutting = nil // only enter this branch once
		case <-publishCtx.Done():
			// Grace period expired.
			return
		default:
			// Normal retry, continue loop.
		}
	}
}

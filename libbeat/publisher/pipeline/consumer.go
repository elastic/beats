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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// eventConsumer collects and forwards events from the queue to the outputs work queue.
// It accepts retry requests from batches it vends, which will resend them
// to the next available output.
type eventConsumer struct {
	logger *logp.Logger

	// eventConsumer calls the retryObserver methods eventsRetry and eventsDropped.
	retryObserver retryObserver

	// When the output changes, the new target is sent to the worker routine
	// on this channel. Clients should call eventConsumer.setTarget().
	targetChan chan consumerTarget

	// Failed batches are sent to this channel to retry. Clients should call
	// eventConsumer.retry().
	retryChan chan retryRequest

	// Closing this channel signals consumer shutdown. Clients should call
	// eventConsumer.close().
	done chan struct{}

	// queueReader is a helper routine that fetches queue batches in a
	// separate goroutine so we don't block on the control path.
	queueReader queueReader

	// This waitgroup is released when this eventConsumer's worker
	// goroutines return.
	wg sync.WaitGroup
}

// consumerTarget specifies the queue to read from, the parameters needed
// to generate a batch, and the output channel to send batches to.
type consumerTarget struct {
	queue      queue.Queue[publisher.Event]
	ch         chan publisher.Batch
	timeToLive int
	batchSize  int
}

// retryRequest is used by ttlBatch to add itself back to the eventConsumer
// queue for distribution to an output.
type retryRequest struct {
	batch       *ttlBatch
	decreaseTTL bool
}

func newEventConsumer(
	log *logp.Logger,
	observer retryObserver,
) *eventConsumer {
	c := &eventConsumer{
		logger:        log,
		retryObserver: observer,
		queueReader:   makeQueueReader(),

		targetChan: make(chan consumerTarget),
		retryChan:  make(chan retryRequest),
		done:       make(chan struct{}),
	}

	c.wg.Go(func() {
		c.run()
	})

	return c
}

func (c *eventConsumer) run() {
	// The queue type is fixed for the life of a pipeline, but the first
	// setTarget is often an empty pause (nil queue). Wait for a real queue
	// before choosing a loop.
	var target consumerTarget
	for target.queue == nil {
		select {
		case target = <-c.targetChan:
		case <-c.done:
			return
		}
	}

	log := c.logger
	log.Debug("start pipeline event consumer")

	if uq, ok := target.queue.(queue.UnblockingQueue[publisher.Event]); ok {
		c.runUnblocking(log, target, uq)
		return
	}

	// Even though we start a goroutine here, we don't include it in the
	// waitGroup used for shutdown: if the queue itself is not closed yet,
	// then the queueReader may be blocked in a read call to the queue,
	// and waiting on it would deadlock. (This scenario is common; the
	// queue is rarely closed properly on shutdown.) The queueReader itself
	// has no independent state to clean up, and can safely shut down
	// after the eventConsumer is already gone, so nothing is lost by
	// letting it happen asynchronously.
	go c.queueReader.run(log)
	c.runBlocking(log, target)
}

func (c *eventConsumer) runBlocking(log *logp.Logger, target consumerTarget) {
	defer close(c.queueReader.req)

	var (
		pendingRead  bool
		retryBatches []*ttlBatch
		queueBatch   *ttlBatch
	)

outerLoop:
	for {
		// If possible, start reading the next batch in the background.
		// We require a non-nil target channel so we don't queue up a large
		// batch before we know the real requested size for our output.
		if queueBatch == nil && !pendingRead && target.queue != nil && target.ch != nil {
			pendingRead = true
			c.queueReader.req <- queueReaderRequest{
				queue:      target.queue,
				retryer:    c,
				batchSize:  target.batchSize,
				timeToLive: target.timeToLive,
			}
		}

		var active *ttlBatch
		// Choose the active batch: if we have batches to retry, use the first
		// one. Otherwise, use a new batch if we have one.
		if len(retryBatches) > 0 {
			active = retryBatches[0]
		} else if queueBatch != nil {
			active = queueBatch
		}

		// If we have a batch, we'll point the output channel at the target
		// and try to send to it. Otherwise, it will remain nil, and sends
		// to it will always block, so the output case of the select below
		// will be ignored.
		var outputChan chan publisher.Batch
		if active != nil {
			outputChan = target.ch
		}

		// Now we can block until the next state change.
		select {
		case outputChan <- active:
			// Successfully sent a batch to the output workers
			if len(retryBatches) > 0 {
				// This was a retry, advance the retry batch list
				retryBatches = retryBatches[1:]
			} else {
				// This was directly from the queue, clear the value so we can
				// fetch a new one
				queueBatch = nil
			}

		case target = <-c.targetChan:

		case queueBatch = <-c.queueReader.resp:
			pendingRead = false

		case req := <-c.retryChan:
			if b := applyRetry(log, c.retryObserver, req); b != nil {
				retryBatches = append(retryBatches, b)
			}

		case <-c.done:
			releaseHeldBatches(queueBatch, retryBatches)
			break outerLoop
		}
	}
}

func (c *eventConsumer) runUnblocking(
	log *logp.Logger,
	target consumerTarget,
	uq queue.UnblockingQueue[publisher.Event],
) {
	var (
		retryBatches  []*ttlBatch
		queueBatch    *ttlBatch
		debounceTimer *time.Timer
		debounceC     <-chan time.Time
	)
	stopDebounce := func() {
		if debounceTimer == nil {
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
		debounceC = nil
	}
	defer stopDebounce()

outerLoop:
	for {
		// TryGet has no coalescing window of its own. Skip it while a
		// debounce timer is running so ReadyChan can accumulate events
		// the way blocking Get does.
		if queueBatch == nil && uq != nil && target.ch != nil && debounceC == nil {
			batch, err := uq.TryGet(target.batchSize)
			if batch != nil {
				queueBatch = newBatch(c, batch, target.timeToLive)
			} else if err != nil {
				// Queue closed or failed; stop fetching from it.
				uq = nil
			}
		}

		var active *ttlBatch
		if len(retryBatches) > 0 {
			active = retryBatches[0]
		} else if queueBatch != nil {
			active = queueBatch
		}
		var outputChan chan publisher.Batch
		if active != nil {
			outputChan = target.ch
		}

		// Wait on ReadyChan only when we need a new batch and none is
		// available. uq is non-nil here, so ReadyChan is safe to call
		// with no extra local channel.
		if queueBatch == nil && uq != nil && target.ch != nil && active == nil {
			select {
			case <-uq.ReadyChan():
				// First events are available. Wait GetDebounce before
				// TryGet so a trickle does not become one batch per event.
				// Extra ReadyChan signals during the window do not reset
				// it; this matches slabqueue Get.
				if debounceC == nil {
					if d := uq.GetDebounce(); d > 0 {
						if debounceTimer == nil {
							debounceTimer = time.NewTimer(d)
						} else {
							debounceTimer.Reset(d)
						}
						debounceC = debounceTimer.C
					}
				}
			case <-debounceC:
				debounceC = nil
			case target = <-c.targetChan:
				stopDebounce()
				uq, _ = target.queue.(queue.UnblockingQueue[publisher.Event])
			case req := <-c.retryChan:
				if b := applyRetry(log, c.retryObserver, req); b != nil {
					retryBatches = append(retryBatches, b)
				}
			case <-c.done:
				releaseHeldBatches(queueBatch, retryBatches)
				break outerLoop
			}
			continue
		}

		select {
		case outputChan <- active:
			if len(retryBatches) > 0 {
				retryBatches = retryBatches[1:]
			} else {
				queueBatch = nil
			}

		case target = <-c.targetChan:
			stopDebounce()
			uq, _ = target.queue.(queue.UnblockingQueue[publisher.Event])

		case req := <-c.retryChan:
			if b := applyRetry(log, c.retryObserver, req); b != nil {
				retryBatches = append(retryBatches, b)
			}

		case <-c.done:
			releaseHeldBatches(queueBatch, retryBatches)
			break outerLoop
		}
	}
}

func applyRetry(log *logp.Logger, observer retryObserver, req retryRequest) *ttlBatch {
	if !req.decreaseTTL {
		return req.batch
	}
	countFailed := len(req.batch.Events())
	alive := req.batch.reduceTTL()
	countDropped := countFailed - len(req.batch.Events())
	observer.eventsDropped(countDropped)
	observer.eventsRetry(len(req.batch.Events()))
	if !alive {
		log.Info("Drop batch")
		req.batch.Drop()
		return nil
	}
	return req.batch
}

func releaseHeldBatches(queueBatch *ttlBatch, retryBatches []*ttlBatch) {
	// Release any batches we're still holding so the underlying
	// queue can reclaim its backing storage without firing
	// producer ACK callbacks. Release is the abandonment
	// path: slabqueue returns its slot indices to the pool's
	// free list; memqueue advances ackLoop past the batch
	// without invoking input ACK handlers; diskqueue is a
	// no-op (events stay on disk for next-process recovery).
	// We must NOT call Drop here — Drop signals successful
	// delivery and would falsely advance input registries for
	// events the consumer is abandoning.
	if queueBatch != nil {
		queueBatch.Release()
	}
	for _, rb := range retryBatches {
		rb.Release()
	}
}

func (c *eventConsumer) setTarget(target consumerTarget) {
	select {
	case c.targetChan <- target:
	case <-c.done:
	}
}

func (c *eventConsumer) retry(batch *ttlBatch, decreaseTTL bool) {
	select {
	case c.retryChan <- retryRequest{batch: batch, decreaseTTL: decreaseTTL}:
		// The batch is back in eventConsumer's retry queue
	case <-c.done:
		// The consumer has already shut down, drop the batch
		batch.Drop()
	}
}

func (c *eventConsumer) close() {
	close(c.done)
	c.wg.Wait()
}

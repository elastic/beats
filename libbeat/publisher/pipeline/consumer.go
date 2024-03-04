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

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// eventConsumer collects and forwards events from the queue to the outputs work queue.
// It accepts retry requests from batches it vends, which will resend them
// to the next available output.
type eventConsumer struct {
	logger *logp.Logger

	// eventConsumer calls the observer methods eventsRetry and eventsDropped.
	observer outputObserver

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
	queue          queue.Queue
	ch             chan batchRequest
	timeToLive     int
	batchSize      int
	encoderFactory outputs.PreEncoderFactory
}

// retryRequest is used by ttlBatch to add itself back to the eventConsumer
// queue for distribution to an output.
type retryRequest struct {
	batch       *ttlBatch
	decreaseTTL bool
}

func newEventConsumer(
	log *logp.Logger,
	observer outputObserver,
) *eventConsumer {
	c := &eventConsumer{
		logger:      log,
		observer:    observer,
		queueReader: makeQueueReader(),

		targetChan: make(chan consumerTarget),
		retryChan:  make(chan retryRequest),
		done:       make(chan struct{}),
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run()
	}()

	// Even though we start a goroutine here, we don't include it in the
	// waitGroup used for shutdown: if the queue itself is not closed yet,
	// then the queueReader may be blocked in a read call to the queue,
	// and waiting on it would deadlock. (This scenario is common; the
	// queue is rarely closed properly on shutdown.) The queueReader itself
	// has no independent state to clean up, and can safely shut down
	// after the eventConsumer is already gone, so nothing is lost by
	// letting it happen asynchronously.
	go c.queueReader.run(c.logger)

	return c
}

func (c *eventConsumer) run() {
	log := c.logger

	log.Debug("start pipeline event consumer")

	var (
		// Whether there's an outstanding request to queueReader
		pendingRead bool

		// The batches waiting to be retried.
		retryBatches []*ttlBatch

		// The batch read from the queue and waiting to be sent, if any
		queueBatch *ttlBatch

		// The output channel (and associated parameters) that will receive
		// the batches we're loading.
		target consumerTarget
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
		var batchRequestChan chan batchRequest
		if active != nil {
			batchRequestChan = target.ch
		}

		// Now we can block until the next state change.
		select {
		case req := <-batchRequestChan:
			req.responseChan <- active
			// Successfully sent a batch to the output workers
			if len(retryBatches) > 0 {
				// This was a retry, report it to the observer
				c.observer.eventsRetry(len(active.Events()))
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
			if req.decreaseTTL {
				countFailed := len(req.batch.Events())

				alive := req.batch.reduceTTL()

				countDropped := countFailed - len(req.batch.Events())
				c.observer.eventsDropped(countDropped)

				if !alive {
					log.Info("Drop batch")
					req.batch.Drop()
					continue
				}
			}
			retryBatches = append(retryBatches, req.batch)

		case <-c.done:
			break outerLoop
		}
	}

	// Close the queueReader request channel so it knows to shutdown.
	close(c.queueReader.req)
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

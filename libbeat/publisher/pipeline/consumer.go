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

	"github.com/elastic/beats/v7/libbeat/publisher"
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

	// This waitgroup is released when this eventConsumer's worker
	// goroutines return.
	wg sync.WaitGroup

	// The queue the eventConsumer will retrieve batches from.
	queue queue.Queue
}

// consumerTarget specifies the output channel and parameters needed for
// eventConsumer to generate a batch.
type consumerTarget struct {
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
	queue queue.Queue,
	observer outputObserver,
) *eventConsumer {
	c := &eventConsumer{
		logger:   log,
		observer: observer,
		queue:    queue,

		targetChan: make(chan consumerTarget),
		retryChan:  make(chan retryRequest),
		done:       make(chan struct{}),
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run()
	}()
	return c
}

func (c *eventConsumer) run() {
	log := c.logger

	log.Debug("start pipeline event consumer")

	// Create a queueReader to run our queue fetches in the background
	c.wg.Add(1)
	queueReader := makeQueueReader()
	go func() {
		defer c.wg.Done()
		queueReader.run(log)
	}()

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

		// The queue.Consumer we get the raw batches from. Reset whenever
		// the target changes.
		consumer queue.Consumer = c.queue.Consumer()
	)

outerLoop:
	for {
		// If possible, start reading the next batch in the background.
		if queueBatch == nil && !pendingRead {
			pendingRead = true
			queueReader.req <- queueReaderRequest{
				consumer:   consumer,
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
				// This was a retry, report it to the observer
				c.observer.eventsRetry(len(active.Events()))
				retryBatches = retryBatches[1:]
			} else {
				// This was directly from the queue, clear the value so we can
				// fetch a new one
				queueBatch = nil
			}

		case target = <-c.targetChan:

		case queueBatch = <-queueReader.resp:
			pendingRead = false

		case req := <-c.retryChan:
			alive := true
			if req.decreaseTTL {
				countFailed := len(req.batch.Events())

				alive = req.batch.reduceTTL()

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

	// Close the queue.Consumer, otherwise queueReader can get blocked
	// waiting on a read.
	consumer.Close()

	// Close the queueReader request channel so it knows to shutdown.
	close(queueReader.req)

	// If there's an outstanding request, we need to read the response
	// to unblock it, but we won't pass on the value.
	if pendingRead {
		batch := <-queueReader.resp
		if batch != nil {
			// Inform any listeners that we couldn't deliver this batch.
			batch.Drop()
		}
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

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

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// eventConsumer collects and forwards events from the queue to the outputs work queue.
// The eventConsumer is managed by the controller and receives additional pause signals
// from the retryer in case of too many events failing to be send or if retryer
// is receiving cancelled batches from outputs to be closed on output reloading.
type eventConsumer struct {
	logger *logp.Logger
	ctx    batchContext

	// When the output changes, send the new information to targetChan.
	// Set to empty consumerTarget to disable sending until the target is
	// reset.
	targetChan chan consumerTarget

	// Send true / false to pauseChan to pause / unpause the consumer.
	pauseChan chan bool

	// Close the done channel to signal shutdown
	done chan struct{}

	// This waitgroup is released when this eventConsumer's worker
	// goroutines return.
	wg sync.WaitGroup

	// The queue the eventConsumer will retrieve batches from.
	queue queue.Queue

	// The active consumer to read event batches from. Stored here
	// so that eventConsumer can wake up the worker if it's blocked reading
	// the queue consumer during shutdown.
	consumer queue.Consumer
}
type consumerTarget struct {
	ch         chan publisher.Batch
	timeToLive int
	batchSize  int
}

func newEventConsumer(
	log *logp.Logger,
	queue queue.Queue,
	ctx batchContext,
) *eventConsumer {
	c := &eventConsumer{
		logger:     log,
		targetChan: make(chan consumerTarget, 3),
		consumer:   queue.Consumer(),

		queue: queue,
		ctx:   ctx,
	}

	c.wg.Add(1)
	queueReader := makeQueueReader()
	go func() {
		defer c.wg.Done()
		queueReader.run(log)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		//c.loop()
		c.run(queueReader)

	}()
	return c
}

func (c *eventConsumer) close() {
	close(c.done)
	c.wg.Wait()
}

// only called from pipelineController.Set
func (c *eventConsumer) setTarget(target consumerTarget) {
	c.targetChan <- target
}

func (c *eventConsumer) run(queueReader queueReader) { //consumer queue.Consumer) {
	log := c.logger

	log.Debug("start pipeline event consumer")

	var (
		// Whether there's an outstanding request to queueReader
		pendingRead bool

		// The batch waiting to be sent, or nil if we don't yet have one
		batch TTLBatch

		// The output channel (and associated parameters) that will receive
		// the batches we're loading.
		target consumerTarget

		// The most recent value received on pauseChan
		paused bool

		// The queue.Consumer we get the raw batches from. Reset whenever
		// the target changes.
		consumer queue.Consumer = c.queue.Consumer()
	)

outerLoop:
	for {
		// If possible, start reading the next batch in the background.
		if consumer != nil && batch == nil && !pendingRead && !paused {
			pendingRead = true
			queueReader.req <- queueReaderRequest{
				ctx:      c.ctx,
				target:   target,
				consumer: consumer,
			}
		}

		// If we have a batch and are unpaused, we'll set the output
		// channel to the target channel and try to send to it. Otherwise,
		// it will remain nil, and sends to it will always block, so the
		// output case of the select will be ignored.
		var outputChan chan publisher.Batch
		if !paused && batch != nil {
			outputChan = target.ch
		}

		// Now we can block until the next state change.
		select {
		case outputChan <- batch:
			batch = nil

		case target = <-c.targetChan:
			if consumer != nil {
				consumer.Close()
			}
			consumer = c.queue.Consumer()

		case paused = <-c.pauseChan:

		case resp := <-queueReader.resp:
			pendingRead = false
			if resp.err != nil && resp.consumer == consumer {
				// The current consumer returned an error; most likely it has
				// been closed, either for shutdown or because the output is
				// reloading. In either case, stop using this consumer until
				// we get a new one.
				consumer = nil
			} else {
				batch = resp.batch
			}
		case <-c.done:
			break outerLoop
		}
	}

	// Close the queue.Consumer, otherwise queueReader can get blocked
	// waiting on a read.
	c.consumer.Close()

	// Close the queueReader request channel so it knows to shutdown.
	close(queueReader.req)

	// If there's an outstanding request, we need to read the response
	// to unblock it, but we won't pass on the value.
	if pendingRead {
		resp := <-queueReader.resp
		pendingRead = false

		if resp.batch != nil {
			// Inform the
			resp.batch.Drop()
		}
	}

}

type queueReader struct {
	req  chan queueReaderRequest  // "give me a batch for this target"
	resp chan queueReaderResponse // "here is your batch"
}

type queueReaderRequest struct {
	ctx      batchContext
	target   consumerTarget
	consumer queue.Consumer
}

type queueReaderResponse struct {
	consumer queue.Consumer
	batch    TTLBatch
	err      error
}

func makeQueueReader() queueReader {
	qr := queueReader{
		req:  make(chan queueReaderRequest, 1),
		resp: make(chan queueReaderResponse),
	}
	return qr
}

func (qr *queueReader) run(logger *logp.Logger) {
	logger.Debug("pipeline event consumer queue reader: start")
	for {
		req, ok := <-qr.req
		if !ok {
			// The request channel is closed, we're shutting down
			logger.Debug("pipeline event consumer queue reader: stop")
			return
		}
		queueBatch, err := req.consumer.Get(req.target.batchSize)
		var batch TTLBatch
		if queueBatch != nil {
			batch = newBatch(req.ctx, queueBatch, req.target.timeToLive)
		}
		qr.resp <- queueReaderResponse{
			consumer: req.consumer,
			batch:    batch,
			err:      err}
	}
}

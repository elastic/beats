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
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// eventConsumer collects and forwards events from the queue to the outputs work queue.
// The eventConsumer is managed by the controller and receives additional pause signals
// from the retryer in case of too many events failing to be send or if retryer
// is receiving cancelled batches from outputs to be closed on output reloading.

type retryRequest struct {
	batch       *ttlBatch
	decreaseTTL bool
}
type eventConsumer struct {
	logger *logp.Logger
	//ctx    batchContext

	// When the output changes, send the new information to targetChan.
	// Set to empty consumerTarget to disable sending until the target is
	// reset.
	targetChan chan consumerTarget

	// Failed batches are sent to this channel to retry
	retryChan chan retryRequest

	// Close the done channel to signal shutdown
	done chan struct{}

	// This waitgroup is released when this eventConsumer's worker
	// goroutines return.
	wg sync.WaitGroup

	// The queue the eventConsumer will retrieve batches from.
	queue queue.Queue
}
type consumerTarget struct {
	ch         chan publisher.Batch
	timeToLive int
	batchSize  int
}

func newEventConsumer(
	log *logp.Logger,
	queue queue.Queue,
	//ctx batchContext,
) *eventConsumer {
	c := &eventConsumer{
		logger: log,

		targetChan: make(chan consumerTarget),
		retryChan:  make(chan retryRequest),
		done:       make(chan struct{}),

		queue: queue,
		//ctx:   ctx,
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run()
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

		// The batches waiting to be sent.
		buffer []*ttlBatch

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
		fmt.Printf("eventConsumer loop begin\n")
		// If possible, start reading the next batch in the background.
		if len(buffer) == 0 && !pendingRead && !paused {
			fmt.Printf("sending reader request\n")
			pendingRead = true
			queueReader.req <- queueReaderRequest{
				//ctx:      c.ctx,
				target:   target,
				consumer: consumer,
			}
			fmt.Printf("sent\n")
		}
		// If we have a batch and are unpaused, we'll set the output
		// channel to the target channel and try to send to it. Otherwise,
		// it will remain nil, and sends to it will always block, so the
		// output case of the select will be ignored.
		var outputChan chan publisher.Batch
		var batch *ttlBatch
		if !paused && len(buffer) > 0 {
			outputChan = target.ch
			batch = buffer[0]
		}

		// Now we can block until the next state change.
		select {
		case outputChan <- batch:
			buffer = buffer[1:]

		case target = <-c.targetChan:

		case batch = <-queueReader.resp:
			pendingRead = false
			buffer = append(buffer, batch)

		case req := <-c.retryChan:

			buffer = append(buffer, req.batch)

		case <-c.done:
			fmt.Printf("\u001b[31mgot done signal\033[0m\n")
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
		pendingRead = false

		if batch != nil {
			// Inform any listeners that we couldn't deliver this batch.
			batch.Drop()
		}
	}
}

// queueReader is a standalone stateless helper goroutine to dispatch
// reads of the queue without blocking eventConsumer's main loop.
type queueReader struct {
	req  chan queueReaderRequest // "give me a batch for this target"
	resp chan *ttlBatch          // "here is your batch"
}

type queueReaderRequest struct {
	batchFactory func(queue.Batch) *ttlBatch
	target       consumerTarget
	consumer     queue.Consumer
}

func makeQueueReader() queueReader {
	qr := queueReader{
		req:  make(chan queueReaderRequest, 1),
		resp: make(chan *ttlBatch),
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
		queueBatch, _ := req.consumer.Get(req.target.batchSize)
		var batch *ttlBatch
		if queueBatch != nil {
			batch = req.batchFactory(queueBatch) //newBatch(req.ctx.retryer, queueBatch, req.target.timeToLive)
		}
		qr.resp <- batch
	}
}

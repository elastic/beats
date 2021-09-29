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
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
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
	ctx    *batchContext

	// When the output changes, send the new information to targetChan.
	// Set to empty consumerTarget to disable sending until the target is
	// reset.
	targetChan chan consumerTarget

	wait atomic.Bool

	// This waitgroup is released when the worker goroutine
	// eventConsumer.loop returns.
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

type consumerEventTag uint8

const (
	sigConsumerCheck consumerEventTag = iota
	sigConsumerUpdateOutput
	//sigStop
)

var errStopped = errors.New("stopped")

func newEventConsumer(
	log *logp.Logger,
	queue queue.Queue,
	ctx *batchContext,
) *eventConsumer {
	c := &eventConsumer{
		logger:     log,
		targetChan: make(chan consumerTarget, 3),
		consumer:   queue.Consumer(),

		queue: queue,
		ctx:   ctx,
	}

	c.wg.Add(1)
	qr := makeQueueReader()
	go func() {
		defer c.wg.Done()
		qr.run(log)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		//c.loop()
		c.outputWriter(log)
	}()
	return c
}

func (c *eventConsumer) close() {
	c.consumer.Close()
	close(c.targetChan)
	c.wg.Wait()
}

func (c *eventConsumer) sigWait() {
	c.wait.Store(true)
}

func (c *eventConsumer) sigUnWait(ch chan publisher.Batch) {
	c.wait.Store(false)
}

// only called from pipelineController.Set
func (c *eventConsumer) setTarget(target consumerTarget) {
	// The queue consumer needs to be closed in case the eventConsumer loop
	// is currently blocked on a call to queueConsumer.Get. In this case, it
	// would never receive the subsequent signal. Closing the consumer triggers
	// an error return from queueConsumer.Get, ensuring the loop will receive
	// the signal.
	c.consumer.Close()
	c.targetChan <- target
}

type queueReader struct {
	req  chan queueReaderRequest // "give me a batch for this target"
	resp chan publisher.Batch    // "here is your batch"
}

type queueReaderRequest struct {
	ctx      batchContext
	target   consumerTarget
	consumer queue.Consumer
}

type queueReaderResponse struct {
	batch TTLBatch
	err   error
}

func makeQueueReader() queueReader {
	qr := queueReader{
		req:  make(chan queueReaderRequest, 1),
		resp: make(chan publisher.Batch),
	}
	//go qr.run()
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
		//if err != nil {
		// The queue consumer returned an error; most likely it has
		// been closed, either for shutdown or because the output is
		// reloading. In either case, we
		//}
		var batch TTLBatch
		if queueBatch != nil {
			batch = newBatch(req.ctx, queueBatch, req.target.timeToLive)
		}
		qr.resp <- queueReaderResponse{batch: batch, err: err}

	}
}

/*
type outputWriter struct {
}

func (ow *outputWriter) run() {
	readPending := false
	for {

	}
}

	resp <- queueBatch


		if target.ch != nil && batch == nil {
			queueBatch, err := c.consumer.Get(target.batchSize)
			if err != nil {
				// The queue consumer returned an error; most likely it has
				// been closed, either for shutdown or because the output is
				// reloading. In either case, stop writing to this output group
				// until we get a new one.
				target.ch = nil
				continue
			}
			if queueBatch != nil {
				batch = newTTLBatch(c.ctx, queueBatch, target.timeToLive)
			}
		}

	}
}
*/
func (c *eventConsumer) outputWriter() {
	log := c.logger
	log.Debug("start pipeline event consumer output writer")
}

func (c *eventConsumer) loop() { //consumer queue.Consumer) {
	defer fmt.Printf("eventConsumer.loop returning GOODBYE\n")
	log := c.logger
	//consumer := c.queue.Consumer()

	log.Debug("start pipeline event consumer")

	var (
		// The batch waiting to be sent, or nil if we don't yet have one
		batch TTLBatch

		// The output channel (and associated parameters) that will receive
		// the batches we're loading. Set to empty consumerTarget{} to
		// pause queue operation.
		target consumerTarget
	)

	for {
		// If we want a batch but don't yet have one
		if target.ch != nil && batch == nil {
			queueBatch, err := c.consumer.Get(target.batchSize)
			if err != nil {
				// The queue consumer returned an error; most likely it has
				// been closed, either for shutdown or because the output is
				// reloading. In either case, stop writing to this output group
				// until we get a new one.
				target.ch = nil
				continue
			}
			if queueBatch != nil {
				batch = newBatch(c.ctx, queueBatch, target.timeToLive)
			}
		}

		// Start by selecting only on the signal channel, so we don't try
		// sending on the output channel until the signal channel is empty.
		select {
		case newTarget, ok := <-c.targetChan:
			if !ok {
				// signal channel closed, eventConsumer is shutting down
				return
			}
			target = newTarget
			continue
		default:
		}

		// If we have a batch to send and we aren't paused, send it to our
		// target channel. If we have no target, or the target channel was
		// set to nil because of an error in queueConsumer.Get, then the
		// send will block forever, and this select will wait until we
		// either get a new target or our signal channel is closed.
		var outChan chan publisher.Batch
		if batch != nil && !c.wait.Load() {
			outChan = target.ch
		}
		if batch != nil { //&& !c.paused() {
			select {
			case newTarget, ok := <-c.targetChan:
				if !ok {
					// signal channel closed, eventConsumer is shutting down
					return
				}
				target = newTarget
			case outChan <- batch:
				batch = nil
			}
		}
	}
}

/*
func (c *eventConsumer) paused() bool {
	//return c.wait.Load()
}*/

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

	pause atomic.Bool
	wait  atomic.Bool
	sig   chan consumerSignal
	wg    sync.WaitGroup

	queue queue.Queue

	// The active consumer to read event batches from. Stored here
	// so that eventConsumer can wake up the worker if it's blocked reading
	// the queue consumer during shutdown.
	queueConsumer queue.Consumer
}

type consumerSignal struct {
	tag consumerEventTag
	out *outputGroup
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
		logger:        log,
		sig:           make(chan consumerSignal, 3),
		queueConsumer: queue.Consumer(),
		//out:    nil,

		queue: queue,
		ctx:   ctx,
	}

	c.pause.Store(true)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.loop()
	}()
	return c
}

func (c *eventConsumer) close() {
	c.queueConsumer.Close()
	close(c.sig)
	//c.sig <- consumerSignal{tag: sigStop}
	c.wg.Wait()
}

func (c *eventConsumer) sigWait() {
	c.wait.Store(true)
	c.sigHint()
}

func (c *eventConsumer) sigUnWait() {
	c.wait.Store(false)
	c.sigHint()
}

func (c *eventConsumer) sigPause() {
	c.pause.Store(true)
	c.sigHint()
}

func (c *eventConsumer) sigContinue() {
	c.pause.Store(false)
	c.sigHint()
}

func (c *eventConsumer) sigHint() {
	// send signal to unblock a consumer trying to publish events.
	// With flags being set atomically, multiple signals can be compressed into one
	// signal -> drop if queue is not empty
	/*select {
	case c.sig <- consumerSignal{tag: sigConsumerCheck}:
	default:
	}*/
}

// only called from pipelineController.Set
func (c *eventConsumer) updOutput(grp *outputGroup) {
	// The queue consumer needs to be closed in case the eventConsumer loop
	// is currently blocked on a call to queueConsumer.Get. In this case, it
	// would never receive the subsequent signal. Closing the consumer triggers
	// an error return from queueConsumer.Get, ensuring the loop will receive
	// the signal.
	c.queueConsumer.Close()
	// update output
	c.sig <- consumerSignal{
		tag: sigConsumerUpdateOutput,
		out: grp,
	}
}

func (c *eventConsumer) loop() { //consumer queue.Consumer) {
	defer fmt.Printf("eventConsumer.loop returning GOODBYE\n")
	log := c.logger
	//consumer := c.queue.Consumer()

	log.Debug("start pipeline event consumer")

	var (
		// The batch waiting to be sent, or nil if we don't yet have one
		batch Batch

		// The output group that will receive the batches we're loading
		outputGroup *outputGroup
	)

	// handleSignal can update `outputGroup` and `c.queueConsumer`
	handleSignal := func(sig consumerSignal) {
		switch sig.tag {

		case sigConsumerCheck:
			// the only function of this case is to refresh the "paused" flag

		case sigConsumerUpdateOutput:
			outputGroup = sig.out
			c.queueConsumer = c.queue.Consumer()
		}
	}

	for {
		// If we want a batch but don't yet have one
		if outputGroup != nil && batch == nil && !c.paused() {
			queueBatch, err := c.queueConsumer.Get(outputGroup.batchSize)
			if err != nil {
				// There is a problem with the queue consumer; most likely it has
				// been closed, either for shutdown or because the output is
				// reloading. In either case, stop writing to this output group
				// until the next sigConsumerUpdateOutput tells us a new one is ready.
				outputGroup = nil
				continue
			}
			if queueBatch != nil {
				batch = newBatch(c.ctx, queueBatch, outputGroup.timeToLive)
			}
		}

		// Start by selecting only on the signal channel, so we don't try
		// sending on the output channel until the signal channel is empty.
		select {
		case sig, ok := <-c.sig:
			if !ok {
				// signal channel closed, eventConsumer is shutting down
				return
			}
			handleSignal(sig)
			continue
		default:
		}

		// If we have a batch to send and we aren't paused, then point the
		// output channel at the real work queue; otherwise, it is nil, and
		// the select below will block until we get something on the signal
		// channel instead.
		var outputChan chan publisher.Batch
		if outputGroup != nil && batch != nil && !c.paused() {
			outputChan = outputGroup.workQueue
		}
		select {
		case sig, ok := <-c.sig:
			if !ok {
				// signal channel closed, eventConsumer is shutting down
				return
			}
			handleSignal(sig)
		case outputChan <- batch:
			batch = nil
		}
	}
}

func (c *eventConsumer) paused() bool {
	return c.pause.Load() || c.wait.Load()
}

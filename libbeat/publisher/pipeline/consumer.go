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
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// eventConsumer collects and forwards events from the queue to the outputs work queue.
// The eventConsumer is managed by the controller and receives additional pause signals
// from the retryer in case of too many events failing to be send or if retryer
// is receiving cancelled batches from outputs to be closed on output reloading.
type eventConsumer struct {
	logger *logp.Logger
	done   chan struct{}

	ctx *batchContext

	pause atomic.Bool
	wait  atomic.Bool
	sig   chan consumerSignal

	queue    queue.Queue
	consumer queue.Consumer

	out *outputGroup
}

func lf(msg string, v ...interface{}) {
	now := time.Now().Format("15:04:05.00000")
	fmt.Printf(now+" "+msg, v...)
}

func ln(msg string) {
	lf(msg + "\n")
}

type consumerSignal struct {
	tag      consumerEventTag
	consumer queue.Consumer
	out      *outputGroup
}

type consumerEventTag uint8

const (
	sigConsumerCheck consumerEventTag = iota
	sigConsumerUpdateOutput
	sigConsumerUpdateInput
)

func newEventConsumer(
	log *logp.Logger,
	queue queue.Queue,
	ctx *batchContext,
) *eventConsumer {
	c := &eventConsumer{
		logger: log,
		done:   make(chan struct{}),
		sig:    make(chan consumerSignal, 3),
		out:    nil,

		queue:    queue,
		consumer: queue.Consumer(),
		ctx:      ctx,
	}

	c.pause.Store(true)
	go c.loop(c.consumer)
	return c
}

func (c *eventConsumer) close() {
	c.consumer.Close()
	close(c.done)
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
	select {
	case c.sig <- consumerSignal{tag: sigConsumerCheck}:
	default:
	}
}

func (c *eventConsumer) updOutput(grp *outputGroup) {
	lf("in eventConsumer: updOutput: new output group ID = %v\n", grp.id)
	// close consumer to break consumer worker from pipeline
	if err := c.consumer.Close(); err != nil {
		lf("in updGroup: consumer close error: %v\n", err)
	}

	// update output
	c.sig <- consumerSignal{
		tag: sigConsumerUpdateOutput,
		out: grp,
	}

	// update eventConsumer with new queue connection
	c.consumer = c.queue.Consumer()
	c.sig <- consumerSignal{
		tag:      sigConsumerUpdateInput,
		consumer: c.consumer,
	}
}

func (c *eventConsumer) loop(consumer queue.Consumer) {
	log := c.logger

	log.Debug("start pipeline event consumer")

	var (
		oid    uint
		out    workQueue
		batch  *Batch
		paused = true
	)

	handleSignal := func(sig consumerSignal) {
		switch sig.tag {
		case sigConsumerCheck:

		case sigConsumerUpdateOutput:
			lf("in eventConsumer: handling sigConsumerUpdateOutput: new output group ID = %v\n", sig.out.id)
			c.out = sig.out

		case sigConsumerUpdateInput:
			consumer = sig.consumer
		}

		paused = c.paused()
		if !paused && c.out != nil && batch != nil {
			out = c.out.workQueue
			oid = c.out.id
		} else if paused && c.out != nil && batch != nil {
			lf("paused but have batch of %v events\n", len(batch.events))
			//batch.Cancelled()
		} else {
			//lf("over here, batch empty? = %v\n", batch == nil)
			out = nil
		}
	}

	for {
		if !paused && c.out != nil && consumer != nil && batch == nil {
			out = c.out.workQueue
			oid = c.out.id
			queueBatch, err := consumer.Get(c.out.batchSize)
			if err != nil {
				out = nil
				consumer = nil
				continue
			}
			if queueBatch != nil {
				lf("in event consumer: got batch of %v events\n", len(queueBatch.Events()))
				batch = newBatch(c.ctx, queueBatch, c.out.timeToLive)
			}

			paused = c.paused()
			if paused || batch == nil {
				lf("in event consumer: paused: %v, batch size = %v\n", paused, len(batch.events))
				out = nil
			}
		}

		select {
		case sig := <-c.sig:
			lf("in event consumer: 1st select: handling signal %v\n", sig)
			handleSignal(sig)
			continue
		default:
		}

		select {
		case <-c.done:
			log.Debug("stop pipeline event consumer")
			ln("stop pipeline event consumer")
			return
		case sig := <-c.sig:
			lf("in event consumer: 2nd select: handling signal %v\n", sig)
			handleSignal(sig)
		case out <- batch:
			if batch != nil {
				lf("in event consumer: sent batch of %v events to output workQueue (worker ID = %v)\n", len(batch.Events()), oid)
			}
			batch = nil
		}
	}
}

func (c *eventConsumer) paused() bool {
	return c.pause.Load() || c.wait.Load()
}

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
)

// retryer is responsible for accepting and managing failed send attempts. It
// will also accept not yet published events from outputs being dynamically closed
// by the controller. Cancelled batches will be forwarded to the new workQueue,
// without updating the events retry counters.
// If too many batches (number of outputs/3) are stored in the retry buffer,
// will the consumer be paused, until some batches have been processed by some
// outputs.
type retryer struct {
	logger   logger
	observer outputObserver

	done chan struct{}

	consumer interruptor

	sig        chan retryerSignal
	out        workQueue
	in         retryQueue
	doneWaiter sync.WaitGroup
}

type interruptor interface {
	sigWait()
	sigUnWait()
}

type retryQueue chan batchEvent

type retryerSignal struct {
	tag     retryerEventTag
	channel workQueue
}

type batchEvent struct {
	tag   retryerBatchTag
	batch Batch
}

type retryerEventTag uint8

const (
	sigRetryerOutputAdded retryerEventTag = iota
	sigRetryerOutputRemoved
	sigRetryerUpdateOutput
)

type retryerBatchTag uint8

const (
	retryBatch retryerBatchTag = iota
	cancelledBatch
)

func newRetryer(
	log logger,
	observer outputObserver,
	out workQueue,
	c interruptor,
) *retryer {
	r := &retryer{
		logger:     log,
		observer:   observer,
		done:       make(chan struct{}),
		sig:        make(chan retryerSignal, 3),
		in:         retryQueue(make(chan batchEvent, 3)),
		out:        out,
		consumer:   c,
		doneWaiter: sync.WaitGroup{},
	}
	r.doneWaiter.Add(1)
	go r.loop()
	return r
}

func (r *retryer) close() {
	close(r.done)
	// Block until loop() is properly closed
	r.doneWaiter.Wait()
}

func (r *retryer) sigOutputAdded() {
	r.sig <- retryerSignal{tag: sigRetryerOutputAdded}
}

func (r *retryer) sigOutputRemoved() {
	r.sig <- retryerSignal{tag: sigRetryerOutputRemoved}
}

func (r *retryer) retry(b Batch) {
	r.in <- batchEvent{tag: retryBatch, batch: b}
}

func (r *retryer) cancelled(b Batch) {
	r.in <- batchEvent{tag: cancelledBatch, batch: b}
}

func (r *retryer) loop() {
	defer r.doneWaiter.Done()
	var (
		out             workQueue
		consumerBlocked bool

		active     Batch
		activeSize int
		buffer     []Batch
		numOutputs int

		log = r.logger
	)

	for {
		select {
		case <-r.done:
			return
		case evt := <-r.in:
			var (
				countFailed  int
				countDropped int
				batch        = evt.batch
				countRetry   = len(batch.Events())
				alive        = true
			)

			if evt.tag == retryBatch {
				countFailed = len(batch.Events())
				r.observer.eventsFailed(countFailed)

				alive = batch.reduceTTL()

				countRetry = len(batch.Events())
				countDropped = countFailed - countRetry
				r.observer.eventsDropped(countDropped)
			}

			if !alive {
				log.Info("Drop batch")
				batch.Drop()
			} else {
				out = r.out
				buffer = append(buffer, batch)
				out = r.out
				active = buffer[0]
				activeSize = len(active.Events())
				if !consumerBlocked {
					consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
				}
			}

		case out <- active:
			r.observer.eventsRetry(activeSize)

			buffer = buffer[1:]
			active, activeSize = nil, 0

			if len(buffer) == 0 {
				out = nil
			} else {
				active = buffer[0]
				activeSize = len(active.Events())
			}

			if consumerBlocked {
				consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
			}

		case sig := <-r.sig:
			switch sig.tag {
			case sigRetryerOutputAdded:
				numOutputs++
				if consumerBlocked {
					consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
				}
			case sigRetryerOutputRemoved:
				numOutputs--
				if !consumerBlocked {
					consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
				}
			}
		}
	}
}

func (r *retryer) checkConsumerBlock(numOutputs, numBatches int) bool {
	consumerBlocked := blockConsumer(numOutputs, numBatches)
	if r.consumer == nil {
		return consumerBlocked
	}

	if consumerBlocked {
		r.logger.Info("retryer: send wait signal to consumer")
		if r.consumer != nil {
			r.consumer.sigWait()
		}
		r.logger.Info("retryer: send wait signal to consumer: done")
	} else {
		r.logger.Info("retryer: send unwait signal to consumer")
		if r.consumer != nil {
			r.consumer.sigUnWait()
		}
		r.logger.Info("retryer: send unwait signal to consumer: done")
	}

	return consumerBlocked
}

func blockConsumer(numOutputs, numBatches int) bool {
	return numBatches/3 >= numOutputs
}

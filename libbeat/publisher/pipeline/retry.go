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

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// retryer is responsible for accepting and managing failed send attempts. It
// will also accept not yet published events from outputs being dynamically closed
// by outputController. Cancelled batches will be forwarded to the new workQueue,
// without updating the events retry counters.
// If too many batches (number of outputs/3) are stored in the retry buffer,
// the eventConsumer will be paused until more have been processed.
type retryer struct {
	logger logger

	// retryer calls the observer methods eventsRetry, eventsDropped,
	// and eventsFailed.
	observer outputObserver

	// Closing this channel signals to the worker goroutine that it should
	// return.
	done chan struct{}

	// When the retry queue gets too big, this interface can signal the
	// eventConsumer to stop sending to the work queue.
	throttle interruptor

	outputCount chan int
	out         chan publisher.Batch
	in          retryQueue
	doneWaiter  sync.WaitGroup
}

type interruptor interface {
	sigWait()
	sigUnWait()
}

type retryQueue chan batchEvent

/*type retryerSignal struct {
	tag retryerEventTag
}*/

type batchEvent struct {
	tag   retryerBatchTag
	batch TTLBatch
}

/*type retryerEventTag uint8

const (
	sigRetryerOutputAdded retryerEventTag = iota
	sigRetryerOutputRemoved
)*/

type retryerBatchTag uint8

const (
	retryBatch retryerBatchTag = iota
	cancelledBatch
)

func newRetryer(
	log logger,
	observer outputObserver,
	out chan publisher.Batch,
) *retryer {
	r := &retryer{
		logger:      log,
		observer:    observer,
		done:        make(chan struct{}),
		outputCount: make(chan int, 3),
		in:          retryQueue(make(chan batchEvent, 3)),
		out:         out,
		doneWaiter:  sync.WaitGroup{},
	}
	r.doneWaiter.Add(1)
	go r.loop()
	return r
}

func (r *retryer) close() {
	fmt.Printf("retryer.close\n")
	close(r.done)
	//Block until loop() is properly closed
	r.doneWaiter.Wait()
}

/*func (r *retryer) sigOutputAdded() {
	r.sig <- retryerSignal{tag: sigRetryerOutputAdded}
}

func (r *retryer) sigOutputRemoved() {
	r.sig <- retryerSignal{tag: sigRetryerOutputRemoved}
}*/
func (r *retryer) setOutputCount(n int) {
	r.outputCount <- n
}

func (r *retryer) retry(b TTLBatch) {
	r.in <- batchEvent{tag: retryBatch, batch: b}
}

func (r *retryer) cancelled(b TTLBatch) {
	r.in <- batchEvent{tag: cancelledBatch, batch: b}
}

func (r *retryer) loop() {
	defer r.doneWaiter.Done()
	var (
		out             chan publisher.Batch
		consumerBlocked bool

		active     TTLBatch
		buffer     []TTLBatch
		numOutputs int

		log = r.logger
	)

	for {
		fmt.Printf("retryer loop begin:\n")
		select {
		case <-r.done:
			fmt.Printf("retryer.loop got done signal!\n")
			return
		case evt := <-r.in:
			fmt.Printf("retryer.in: evt %v\n", evt)
			var (
				batch = evt.batch
				alive = true
			)

			if evt.tag == retryBatch {
				countFailed := len(batch.Events())
				r.observer.eventsFailed(countFailed)

				alive = batch.reduceTTL()

				countDropped := countFailed - len(batch.Events())
				r.observer.eventsDropped(countDropped)
			}

			if !alive {
				log.Info("Drop batch")
				batch.Drop()
			} else {
				buffer = append(buffer, batch)
				out = r.out
				active = buffer[0]
				if !consumerBlocked {
					consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
				}
			}

		case out <- active:
			fmt.Printf("retryer wrote to the work queue\n")
			r.observer.eventsRetry(len(active.Events()))

			buffer = buffer[1:]
			active = nil

			if len(buffer) == 0 {
				out = nil
			} else {
				active = buffer[0]
			}

			if consumerBlocked {
				consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
			}

		case numOutputs = <-r.outputCount:
			consumerBlocked = r.checkConsumerBlock(numOutputs, len(buffer))
		}
	}
}

func (r *retryer) checkConsumerBlock(numOutputs, numBatches int) bool {
	consumerBlocked := shouldBlockConsumer(numOutputs, numBatches)
	fmt.Printf("checkConsumerBlock: %v\n", consumerBlocked)
	if r.throttle != nil {
		if consumerBlocked {
			r.logger.Info("retryer: send wait signal to consumer")
			r.throttle.sigWait()
		} else {
			fmt.Printf("sending unwait?\n")
			r.logger.Info("retryer: send unwait signal to consumer")
			r.throttle.sigUnWait()
			fmt.Printf("sent\n")
		}
	} else {
		fmt.Printf("we have no more throttle\n")
	}

	return consumerBlocked
}

func shouldBlockConsumer(numOutputs, numBatches int) bool {
	return numBatches/3 >= numOutputs
}

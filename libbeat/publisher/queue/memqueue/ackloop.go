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

package memqueue

// ackLoop implements the brokers asynchronous ACK worker.
// Multiple concurrent ACKs from consecutive published batches will be batched up by the
// worker, to reduce the number of signals to return to the producer and the
// broker event loop.
// Producer ACKs are run in the ackLoop go-routine.
type ackLoop struct {
	broker *broker

	// A list of batches given to queue consumers,
	// used to maintain sequencing of event acknowledgements.
	pendingBatches batchList
}

func newACKLoop(broker *broker) *ackLoop {
	return &ackLoop{broker: broker}
}

func (l *ackLoop) run() {
	b := l.broker
	for {
		var nextBatchChan chan batchDoneMsg
		if !l.pendingBatches.Empty() {
			nextBatchChan = l.pendingBatches.First().doneChan
		}

		select {
		case <-b.ctx.Done():
			// The queue is shutting down.
			return

		case chanList := <-b.consumedChan:
			// New batches have been generated, add them to the pending list
			l.pendingBatches.Concat(chanList)

		case <-nextBatchChan:
			// The oldest outstanding batch has been acknowledged, advance our
			// position as much as we can.
			l.handleBatchSig()
		}
	}
}

// handleBatchSig collects and handles a batch ACK/Cancel signal. handleBatchSig
// is run by the ackLoop.
func (l *ackLoop) handleBatchSig() {
	ackedBatches := l.collectAcked()

	if !ackedBatches.Empty() {
		// report acks to waiting clients
		l.processACK(ackedBatches)
	}
}

func (l *ackLoop) collectAcked() batchList {
	ackedBatches := batchList{}

	// The first batch is always included, since that's what triggered the call
	// to collectAcked.
	nextBatch := l.pendingBatches.ConsumeFirst()
	ackedBatches.Add(nextBatch)

	done := false
	for !l.pendingBatches.Empty() && !done {
		nextBatch = l.pendingBatches.First()
		select {
		case <-nextBatch.doneChan:
			ackedBatches.Add(nextBatch)
			l.pendingBatches.Remove()

		default:
			done = true
		}
	}

	return ackedBatches
}

// Called by ackLoop. This function exists to decouple the work of collecting
// and running producer callbacks from logical deletion of the events, so
// input callbacks can't block the queue by occupying the runLoop goroutine.
func (l *ackLoop) processACK(lst batchList) {
	ackCallbacks := []func(){}
	batches := []batch{}
	for !lst.Empty() {
		batches = append(batches, lst.First())
		lst.Remove()
	}
	// First we traverse the entries we're about to remove, collecting any callbacks
	// we need to run.
	// Traverse entries from last to first, so we can acknowledge the most recent
	// ones first and skip repeated producer callbacks.
	eventCount := 0
	for batchIndex := len(batches) - 1; batchIndex >= 0; batchIndex-- {
		batch := batches[batchIndex]
		eventCount += batch.count

		for i := batch.count - 1; i >= 0; i-- {
			entry := batch.entry(i)
			if entry.producer == nil {
				continue
			}

			if entry.producerID <= entry.producer.state.lastACK {
				// This index was already acknowledged on a previous iteration, skip.
				entry.producer = nil
				continue
			}
			producerState := entry.producer.state
			count := int(entry.producerID - producerState.lastACK)
			ackCallbacks = append(ackCallbacks, func() { producerState.cb(count) })
			entry.producer.state.lastACK = entry.producerID
			entry.producer = nil
		}
	}
	// Signal runLoop to delete the events
	l.broker.deleteChan <- eventCount
	l.broker.logger.Debug("ackloop: return ack to broker loop:", eventCount)

	// The events have been removed; notify their listeners.
	for _, f := range ackCallbacks {
		f()
	}
}

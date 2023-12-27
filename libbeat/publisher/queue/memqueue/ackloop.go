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

	// A list of ACK channels given to queue consumers,
	// used to maintain sequencing of event acknowledgements.
	ackChans chanList

	processACK func(chanList, int)
}

func (l *ackLoop) run() {
	for {
		nextBatchChan := l.ackChans.nextBatchChannel()

		select {
		case <-l.broker.done:
			// The queue is shutting down.
			return

		case chanList := <-l.broker.scheduledACKs:
			// A new batch has been generated, add its ACK channel to the end of
			// the pending list.
			l.ackChans.concat(&chanList)

		case <-nextBatchChan:
			// The oldest outstanding batch has been acknowledged, advance our
			// position as much as we can.
			l.handleBatchSig()
		}
	}
}

// handleBatchSig collects and handles a batch ACK/Cancel signal. handleBatchSig
// is run by the ackLoop.
func (l *ackLoop) handleBatchSig() int {
	lst := l.collectAcked()

	count := 0
	for current := lst.front(); current != nil; current = current.next {
		count += current.count
	}

	if count > 0 {
		if callback := l.broker.ackCallback; callback != nil {
			callback(count)
		}

		// report acks to waiting clients
		l.processACK(lst, count)
	}

	for !lst.empty() {
		releaseACKChan(lst.pop())
	}

	// return final ACK to EventLoop, in order to clean up internal buffer
	l.broker.logger.Debug("ackloop: return ack to broker loop:", count)

	l.broker.logger.Debug("ackloop:  done send ack")
	return count
}

func (l *ackLoop) collectAcked() chanList {
	lst := chanList{}

	acks := l.ackChans.pop()
	lst.append(acks)

	done := false
	for !l.ackChans.empty() && !done {
		acks := l.ackChans.front()
		select {
		case <-acks.doneChan:
			lst.append(l.ackChans.pop())

		default:
			done = true
		}
	}

	return lst
}

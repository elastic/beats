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
	sig    chan batchAckMsg
	lst    chanList

	totalACK uint64

	processACK func(chanList, int)
}

func newACKLoop(b *broker, processACK func(chanList, int)) *ackLoop {
	l := &ackLoop{broker: b}
	l.processACK = processACK
	return l
}

func (l *ackLoop) run() {
	var (
		// log = l.broker.logger

		// Buffer up acked event counter in acked. If acked > 0, acks will be set to
		// the broker.acks channel for sending the ACKs while potentially receiving
		// new batches from the broker event loop.
		// This concurrent bidirectionally communication pattern requiring 'select'
		// ensures we can not have any deadlock between the event loop and the ack
		// loop, as the ack loop will not block on any channel
		acked int
		acks  chan int
	)

	for {
		select {
		case <-l.broker.done:
			return

		case acks <- acked:
			acks, acked = nil, 0

		case lst := <-l.broker.scheduledACKs:
			l.lst.concat(&lst)

		case <-l.sig:
			acked += l.handleBatchSig()
			if acked > 0 {
				acks = l.broker.acks
			}
		}

		// log.Debug("ackloop INFO")
		// log.Debug("ackloop:   total events scheduled = ", l.totalSched)
		// log.Debug("ackloop:   total events ack = ", l.totalACK)
		// log.Debug("ackloop:   total batches scheduled = ", l.batchesSched)
		// log.Debug("ackloop:   total batches ack = ", l.batchesACKed)

		l.sig = l.lst.channel()
		// if l.sig == nil {
		// 	log.Debug("ackloop: no ack scheduled")
		// } else {
		// 	log.Debug("ackloop: schedule ack: ", l.lst.head.seq)
		// }
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
		if listener := l.broker.ackListener; listener != nil {
			listener.OnACK(count)
		}

		// report acks to waiting clients
		l.processACK(lst, count)
	}

	for !lst.empty() {
		releaseACKChan(lst.pop())
	}

	// return final ACK to EventLoop, in order to clean up internal buffer
	l.broker.logger.Debug("ackloop: return ack to broker loop:", count)

	l.totalACK += uint64(count)
	l.broker.logger.Debug("ackloop:  done send ack")
	return count
}

func (l *ackLoop) collectAcked() chanList {
	lst := chanList{}

	acks := l.lst.pop()
	l.broker.logger.Debugf("ackloop: receive ack [%v: %v, %v]", acks.seq, acks.start, acks.count)
	lst.append(acks)

	done := false
	for !l.lst.empty() && !done {
		acks := l.lst.front()
		select {
		case <-acks.ch:
			l.broker.logger.Debugf("ackloop: receive ack [%v: %v, %v]", acks.seq, acks.start, acks.count)
			lst.append(l.lst.pop())

		default:
			done = true
		}
	}

	return lst
}

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

package proxyqueue

// producerACKData tracks the number of events that need to be acknowledged
// from a single batch targeting a single producer.
type producerACKData struct {
	producer *producer
	count    int
}

// batchACKState stores the metadata associated with a batch of events sent to
// a consumer. When the consumer ACKs that batch, its doneChan is closed.
// The run loop for the broker checks the doneChan for the next sequential
// outstanding batch (to ensure ACKs are delivered in order) and calls the
// producer's ackHandler when appropriate.
type batchACKState struct {
	next     *batchACKState
	doneChan chan struct{}
	acks     []producerACKData
}

type pendingACKsList struct {
	head *batchACKState
	tail *batchACKState
}

func (l *pendingACKsList) append(ackState *batchACKState) {
	if l.head == nil {
		l.head = ackState
	} else {
		l.tail.next = ackState
	}
	l.tail = ackState
}

func (l *pendingACKsList) nextDoneChan() chan struct{} {
	if l.head != nil {
		return l.head.doneChan
	}
	return nil
}

func (l *pendingACKsList) pop() *batchACKState {
	ch := l.head
	if ch != nil {
		l.head = ch.next
		if l.head == nil {
			l.tail = nil
		}
		ch.next = nil
	}
	return ch
}

func acksForBatch(b *batch) []producerACKData {
	results := []producerACKData{}
	// We traverse the list back to front, so we can coalesce multiple events
	// into a single entry in the ACK data.
	for i := len(b.entries) - 1; i >= 0; i-- {
		entry := b.entries[i]
		if producer := entry.producer; producer != nil {
			if producer.producedCount > producer.consumedCount {
				results = append(results, producerACKData{
					producer: producer,
					count:    int(producer.producedCount - producer.consumedCount),
				})
				producer.consumedCount = producer.producedCount
			}
		}
	}
	return results
}

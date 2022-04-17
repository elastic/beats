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

	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue"
)

type retryer interface {
	retry(batch *ttlBatch, decreaseTTL bool)
}

type ttlBatch struct {
	original queue.Batch

	// The internal hook back to the eventConsumer, used to implement the
	// publisher.Batch retry interface.
	retryer retryer

	// How many retries until we drop this batch. -1 means it can't be dropped.
	ttl int

	// The cached events returned from original.Events(). If some but not
	// all of the events are ACKed, those ones are removed from the list.
	events []publisher.Event
}

var batchPool = sync.Pool{
	New: func() interface{} {
		return &ttlBatch{}
	},
}

func newBatch(retryer retryer, original queue.Batch, ttl int) *ttlBatch {
	if original == nil {
		panic("empty batch")
	}

	b := batchPool.Get().(*ttlBatch)
	*b = ttlBatch{
		original: original,
		retryer:  retryer,
		ttl:      ttl,
		events:   original.Events(),
	}
	return b
}

func releaseBatch(b *ttlBatch) {
	*b = ttlBatch{} // clear batch
	batchPool.Put(b)
}

func (b *ttlBatch) Events() []publisher.Event {
	return b.events
}

func (b *ttlBatch) ACK() {
	b.original.ACK()
	releaseBatch(b)
}

func (b *ttlBatch) Drop() {
	b.original.ACK()
	releaseBatch(b)
}

func (b *ttlBatch) Retry() {
	b.retryer.retry(b, true)
}

func (b *ttlBatch) Cancelled() {
	b.retryer.retry(b, false)
}

func (b *ttlBatch) RetryEvents(events []publisher.Event) {
	b.events = events
	b.Retry()
}

// reduceTTL reduces the time to live for all events that have no 'guaranteed'
// sending requirements.  reduceTTL returns true if the batch is still alive.
func (b *ttlBatch) reduceTTL() bool {
	if b.ttl <= 0 {
		return true
	}

	b.ttl--
	if b.ttl > 0 {
		return true
	}

	// filter for evens with guaranteed send flags
	events := b.events[:0]
	for _, event := range b.events {
		if event.Guaranteed() {
			events = append(events, event)
		}
	}
	b.events = events

	if len(b.events) > 0 {
		b.ttl = -1 // we need infinite retry for all events left in this batch
		return true
	}

	// all events have been dropped:
	return false
}

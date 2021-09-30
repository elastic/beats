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

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type TTLBatch interface {
	publisher.Batch

	reduceTTL() bool
}

type ttlBatch struct {
	original queue.Batch
	consumer *eventConsumer
	ttl      int
	events   []publisher.Event
}

type batchContext struct {
	retryer *retryer
}

var batchPool = sync.Pool{
	New: func() interface{} {
		return &ttlBatch{}
	},
}

func newBatch(consumer *eventConsumer, original queue.Batch, ttl int) *ttlBatch {
	if original == nil {
		panic("empty batch")
	}

	b := batchPool.Get().(*ttlBatch)
	*b = ttlBatch{
		original: original,
		consumer: consumer,
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
	//b.ctx.retryer.retry(b)
	select {
	case b.consumer.retryChan <- retryRequest{batch: b, decreaseTTL: true}:
	case <-b.consumer.done:
		// The consumer has already shut down
		b.Drop()
	}
}

func (b *ttlBatch) Cancelled() {
	//b.ctx.retryer.cancelled(b)
	select {
	// TODO: have retryChan include a cancel vs retry param
	case b.consumer.retryChan <- retryRequest{batch: b, decreaseTTL: false}:
	case <-b.consumer.done:
		// The consumer has already shut down
		b.Drop()
	}
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

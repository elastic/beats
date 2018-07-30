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

	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

type Batch struct {
	original queue.Batch
	ctx      *batchContext
	ttl      int
	events   []publisher.Event
}

type batchContext struct {
	observer outputObserver
	retryer  *retryer
}

var batchPool = sync.Pool{
	New: func() interface{} {
		return &Batch{}
	},
}

func newBatch(ctx *batchContext, original queue.Batch, ttl int) *Batch {
	if original == nil {
		panic("empty batch")
	}

	b := batchPool.Get().(*Batch)
	*b = Batch{
		original: original,
		ctx:      ctx,
		ttl:      ttl,
		events:   original.Events(),
	}
	return b
}

func releaseBatch(b *Batch) {
	*b = Batch{} // clear batch
	batchPool.Put(b)
}

func (b *Batch) Events() []publisher.Event {
	return b.events
}

func (b *Batch) ACK() {
	b.ctx.observer.outBatchACKed(len(b.events))
	b.original.ACK()
	releaseBatch(b)
}

func (b *Batch) Drop() {
	b.original.ACK()
	releaseBatch(b)
}

func (b *Batch) Retry() {
	b.ctx.retryer.retry(b)
}

func (b *Batch) Cancelled() {
	b.ctx.retryer.cancelled(b)
}

func (b *Batch) RetryEvents(events []publisher.Event) {
	b.updEvents(events)
	b.Retry()
}

func (b *Batch) CancelledEvents(events []publisher.Event) {
	b.updEvents(events)
	b.Cancelled()
}

func (b *Batch) updEvents(events []publisher.Event) {
	l1 := len(b.events)
	l2 := len(events)
	if l1 > l2 {
		// report subset of events not to be retried as ACKed
		b.ctx.observer.outBatchACKed(l1 - l2)
	}

	b.events = events
}

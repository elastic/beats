package pipeline

import (
	"sync"

	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/broker"
)

type Batch struct {
	original broker.Batch
	ctx      *batchContext
	ttl      int
	events   []publisher.Event
}

type batchContext struct {
	retryer *retryer
}

var batchPool = sync.Pool{
	New: func() interface{} {
		return &Batch{}
	},
}

func newBatch(ctx *batchContext, original broker.Batch, ttl int) *Batch {
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
	b.events = events
	b.Retry()
}

func (b *Batch) CancelledEvents(events []publisher.Event) {
	b.events = events
	b.Cancelled()
}

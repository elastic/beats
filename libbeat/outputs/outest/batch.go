package outest

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/publisher"
)

type Batch struct {
	events   []publisher.Event
	Signals  []BatchSignal
	OnSignal func(sig BatchSignal)
}

type BatchSignal struct {
	Tag    BatchSignalTag
	Events []publisher.Event
}

type BatchSignalTag uint8

const (
	BatchACK BatchSignalTag = iota
	BatchDrop
	BatchRetry
	BatchRetryEvents
	BatchCancelled
	BatchCancelledEvents
)

func NewBatch(in ...beat.Event) *Batch {
	events := make([]publisher.Event, len(in))
	for i, c := range in {
		events[i] = publisher.Event{Content: c}
	}
	return &Batch{events: events}
}

func (b *Batch) Events() []publisher.Event {
	return b.events
}

func (b *Batch) ACK() {
	b.doSignal(BatchSignal{Tag: BatchACK})
}

func (b *Batch) Drop() {
	b.doSignal(BatchSignal{Tag: BatchDrop})
}

func (b *Batch) Retry() {
	b.doSignal(BatchSignal{Tag: BatchRetry})
}

func (b *Batch) RetryEvents(events []publisher.Event) {
	b.doSignal(BatchSignal{Tag: BatchRetryEvents, Events: events})
}

func (b *Batch) Cancelled() {
	b.doSignal(BatchSignal{Tag: BatchCancelled})
}

func (b *Batch) CancelledEvents(events []publisher.Event) {
	b.doSignal(BatchSignal{Tag: BatchCancelledEvents, Events: events})
}

func (b *Batch) doSignal(sig BatchSignal) {
	b.Signals = append(b.Signals, sig)
	if b.OnSignal != nil {
		b.OnSignal(sig)
	}
}

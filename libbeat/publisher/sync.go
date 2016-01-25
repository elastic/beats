package publisher

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
)

type syncPublisher struct {
	pub *PublisherType
}

type syncClient func(message) bool

func newSyncPublisher(pub *PublisherType, hwm, bulkHWM int) *syncPublisher {
	return &syncPublisher{pub: pub}
}

func (p *syncPublisher) client() eventPublisher {
	return p
}

func (p *syncPublisher) PublishEvent(ctx Context, event common.MapStr) bool {
	msg := message{context: ctx, event: event}
	return p.send(msg)
}

func (p *syncPublisher) PublishEvents(ctx Context, events []common.MapStr) bool {
	msg := message{context: ctx, events: events}
	return p.send(msg)
}

func (p *syncPublisher) send(m message) bool {
	if p.pub.disabled {
		debug("publisher disabled")
		outputs.SignalCompleted(m.context.Signal)
		return true
	}

	signal := m.context.Signal
	sync := outputs.NewSyncSignal()
	if len(p.pub.Output) > 1 {
		m.context.Signal = outputs.NewSplitSignaler(sync, len(p.pub.Output))
	} else {
		m.context.Signal = sync
	}

	for _, o := range p.pub.Output {
		o.send(m)
	}

	ok := sync.Wait()
	if ok {
		outputs.SignalCompleted(signal)
	} else if signal != nil {
		signal.Failed()
	}
	return ok
}

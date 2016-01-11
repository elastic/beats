package publisher

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
)

type syncPublisher struct {
	messageWorker
	pub *PublisherType
}

type syncClient func(message) bool

func newSyncPublisher(pub *PublisherType) *syncPublisher {
	s := &syncPublisher{pub: pub}
	s.messageWorker.init(&pub.wsPublisher, defaultChanSize, newPreprocessor(pub, s))
	return s
}

func (p *syncPublisher) client() eventPublisher {
	return syncClient(p.forward)
}

func (p *syncPublisher) onStop() {}

func (p *syncPublisher) onMessage(m message) {
	signal := outputs.NewSplitSignaler(m.context.signal, len(p.pub.Output))
	m.context.signal = signal
	for _, o := range p.pub.Output {
		o.send(m)
	}
}

func (c syncClient) PublishEvent(ctx context, event common.MapStr) bool {
	return c(message{context: ctx, event: event})
}

func (c syncClient) PublishEvents(ctx context, events []common.MapStr) bool {
	return c(message{context: ctx, events: events})
}

func (p *syncPublisher) forward(m message) bool {
	sync := outputs.NewSyncSignal()
	signal := m.context.signal
	m.context.signal = sync
	p.send(m)
	if sync.Wait() {
		outputs.SignalCompleted(signal)
		return true
	}
	if signal != nil {
		signal.Failed()
	}
	return false
}

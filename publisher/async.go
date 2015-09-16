package publisher

import (
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type asyncPublisher struct {
	messageWorker
	pub *PublisherType
}

type asyncClient func(message)

func newAsyncPublisher(pub *PublisherType) *asyncPublisher {
	p := &asyncPublisher{pub: pub}
	p.messageWorker.init(&pub.wsPublisher, 1000, newPreprocessor(pub, p))
	return p
}

func (p *asyncPublisher) onMessage(m message) {
	// m.signal is not set yet. But a async client type supporting signals might
	// be implemented in the furute.
	// If m.signal is nil, NewSplitSignaler will return nil -> signaler will
	// only set if client did send one
	m.signal = outputs.NewSplitSignaler(m.signal, len(p.pub.Output))
	for _, o := range p.pub.Output {
		o.publish(m)
	}
}

func (p *asyncPublisher) client() EventPublisher {
	return asyncClient(p.send)
}

func (c asyncClient) PublishEvent(event common.MapStr) bool {
	c(message{event: event})
	return true
}

func (c asyncClient) PublishEvents(events []common.MapStr) bool {
	c(message{events: events})
	return true
}

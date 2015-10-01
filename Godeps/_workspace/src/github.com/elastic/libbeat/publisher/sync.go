package publisher

import (
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type syncPublisher struct {
	messageWorker
	pub *PublisherType
}

type syncClient func(message) bool

func newSyncPublisher(pub *PublisherType) *syncPublisher {
	s := &syncPublisher{pub: pub}
	s.messageWorker.init(&pub.wsPublisher, 1000, newPreprocessor(pub, s))
	return s
}

func (p *syncPublisher) client(confirmOnly bool) eventPublisher {
	if confirmOnly {
		return syncClient(p.forward)
	}
	return syncClient(p.forceForward)
}

func (p *syncPublisher) onStop() {}

func (p *syncPublisher) onMessage(m message) {
	signal := outputs.NewSplitSignaler(m.signal, len(p.pub.Output))
	m.signal = signal
	for _, o := range p.pub.Output {
		o.send(m)
	}
}

func (c syncClient) PublishEvent(event common.MapStr) bool {
	return c(message{event: event})
}

func (c syncClient) PublishEvents(events []common.MapStr) bool {
	return c(message{events: events})
}

func (p *syncPublisher) forward(m message) bool {
	sync := outputs.NewSyncSignal()
	m.signal = sync
	p.send(m)
	return sync.Wait()
}

func (p *syncPublisher) forceForward(m message) bool {
	for {
		if ok := p.forward(m); ok {
			return true
		}
	}
}

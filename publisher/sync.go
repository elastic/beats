package publisher

import (
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type syncPublisher struct {
	messageWorker
	pub *PublisherType
}

type syncClient func(message)

type syncSignal struct {
	ch chan bool
}

func newSyncPublisher(pub *PublisherType) *syncPublisher {
	s := &syncPublisher{pub: pub}
	s.messageWorker.init(&pub.wsPublisher, 1000, newPreprocessor(pub, s))
	return s
}

func (p *syncPublisher) client() EventPublisher {
	return syncClient(p.send)
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
	return c.send(message{event: event})
}

func (c syncClient) PublishEvents(events []common.MapStr) bool {
	return c.send(message{events: events})
}

func (c syncClient) send(m message) bool {
	sync := newSyncSignal()
	m.signal = sync
	c(m)
	return sync.wait()
}

func newSyncSignal() *syncSignal { return &syncSignal{make(chan bool)} }
func (s *syncSignal) wait() bool { return <-s.ch }
func (s *syncSignal) Completed() { s.ch <- true }
func (s *syncSignal) Failed()    { s.ch <- false }

package publisher

import "github.com/elastic/beats/libbeat/outputs"

type syncPipeline struct {
	pub *Publisher
}

type syncSignal struct {
	ch   chan bool
	done chan struct{}
}

func newSyncPipeline(pub *Publisher, hwm, bulkHWM int) *syncPipeline {
	return &syncPipeline{pub: pub}
}

func (p *syncPipeline) publish(m message) bool {
	if p.pub.disabled {
		debug("publisher disabled")
		outputs.SignalCompleted(m.context.Signal)
		return true
	}

	signal := m.context.Signal
	sync := &syncSignal{done: m.client.done, ch: make(chan bool, 1)}
	if len(p.pub.Output) > 1 {
		m.context.Signal = outputs.NewSplitSignaler(sync, len(p.pub.Output))
	} else {
		m.context.Signal = sync
	}

	for _, o := range p.pub.Output {
		o.send(m)
	}

	// Await completion signal from output plugin. If client has been disconnected
	// ignore any signal and drop events no matter if send or not.
	select {
	case <-sync.done:
		// do not signal on 'drop' when closing connection
		return false
	case ok := <-sync.ch:
		if ok {
			outputs.SignalCompleted(signal)
		} else if signal != nil {
			signal.Failed()
		}
		return ok
	}
}

func (s *syncSignal) Completed() {
	s.ch <- true
}

func (s *syncSignal) Failed() {
	s.ch <- false
}

package publisher

import "github.com/elastic/beats/libbeat/common/op"

type syncPipeline struct {
	pub *BeatPublisher
}

func newSyncPipeline(pub *BeatPublisher, hwm, bulkHWM int) *syncPipeline {
	return &syncPipeline{pub: pub}
}

func (p *syncPipeline) publish(m message) bool {
	if p.pub.disabled {
		debug("publisher disabled")
		op.SigCompleted(m.context.Signal)
		return true
	}

	client := m.client
	signal := m.context.Signal
	sync := op.NewSignalChannel()
	if len(p.pub.Output) > 1 {
		m.context.Signal = op.SplitSignaler(sync, len(p.pub.Output))
	} else {
		m.context.Signal = sync
	}

	for _, o := range p.pub.Output {
		o.send(m)
	}

	// Await completion signal from output plugin. If client has been disconnected
	// ignore any signal and drop events no matter if send or not.
	select {
	case <-client.canceler.Done():
		return false // return false, indicating events potentially not being send
	case sig := <-sync.C:
		sig.Apply(signal)
		return sig == op.SignalCompleted
	}
}

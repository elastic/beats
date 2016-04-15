package publisher

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type asyncPipeline struct {
	outputs []worker
	pub     *Publisher
}

type asyncSignal struct {
	cancel *canceling
	signal outputs.Signaler
}

const (
	defaultBulkSize = 2048
)

func newAsyncPipeline(
	pub *Publisher,
	hwm, bulkHWM int,
	ws *common.WorkerSignal,
) *asyncPipeline {
	p := &asyncPipeline{pub: pub}

	var outputs []worker
	for _, out := range pub.Output {
		outputs = append(outputs, makeAsyncOutput(ws, hwm, bulkHWM, out))
	}

	p.outputs = outputs
	return p
}

func (p *asyncPipeline) publish(m message) bool {
	if p.pub.disabled {
		debug("publisher disabled")
		outputs.SignalCompleted(m.context.Signal)
		return true
	}

	if m.context.Signal != nil {
		var s outputs.Signaler = &asyncSignal{
			cancel: m.client.canceling,
			signal: m.context.Signal,
		}
		if len(p.outputs) > 1 {
			s = outputs.NewSplitSignaler(s, len(p.outputs))
		}
		m.context.Signal = s
	}

	for _, o := range p.outputs {
		o.send(m)
	}
	return true
}

func makeAsyncOutput(
	ws *common.WorkerSignal,
	hwm, bulkHWM int,
	worker *outputWorker,
) worker {
	config := worker.config

	flushInterval := config.FlushInterval
	maxBulkSize := config.BulkMaxSize
	logp.Info("Flush Interval set to: %v", flushInterval)
	logp.Info("Max Bulk Size set to: %v", maxBulkSize)

	// batching disabled
	if flushInterval <= 0 || maxBulkSize <= 0 {
		return worker
	}

	debug("create bulk processing worker (interval=%v, bulk size=%v)",
		flushInterval, maxBulkSize)
	return newBulkWorker(ws, hwm, bulkHWM, worker, flushInterval, maxBulkSize)
}

func (s *asyncSignal) Completed() {
	s.cancel.lock.RLock()
	defer s.cancel.lock.RUnlock()

	if s.cancel.active {
		s.signal.Completed()
	}
}

func (s *asyncSignal) Failed() {
	s.cancel.lock.RLock()
	defer s.cancel.lock.RUnlock()

	if s.cancel.active {
		s.signal.Failed()
	}
}

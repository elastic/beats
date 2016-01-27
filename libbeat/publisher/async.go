package publisher

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type asyncPublisher struct {
	outputs []worker
	pub     *PublisherType
	ws      workerSignal
}

const (
	defaultFlushInterval = 1000 * time.Millisecond // 1s
	defaultBulkSize      = 2048
)

func newAsyncPublisher(pub *PublisherType, hwm, bulkHWM int) *asyncPublisher {
	p := &asyncPublisher{pub: pub}
	p.ws.Init()

	var outputs []worker
	for _, out := range pub.Output {
		outputs = append(outputs, asyncOutputer(&p.ws, hwm, bulkHWM, out))
	}

	p.outputs = outputs
	return p
}

// onStop will send stop signal to message batching workers
func (p *asyncPublisher) onStop() { p.ws.stop() }

func (p *asyncPublisher) client() eventPublisher {
	return p
}

func (p *asyncPublisher) PublishEvent(ctx Context, event common.MapStr) bool {
	p.send(message{context: ctx, event: event})
	return true
}

func (p *asyncPublisher) PublishEvents(ctx Context, events []common.MapStr) bool {
	p.send(message{context: ctx, events: events})
	return true
}

func (p *asyncPublisher) send(m message) {
	if p.pub.disabled {
		debug("publisher disabled")
		outputs.SignalCompleted(m.context.Signal)
		return
	}

	// m.signal is not set yet. But a async client type supporting signals might
	// be implemented in the future.
	// If m.Signal is nil, NewSplitSignaler will return nil -> signaler will
	// only set if client did send one
	if m.context.Signal != nil && len(p.outputs) > 1 {
		m.context.Signal = outputs.NewSplitSignaler(m.context.Signal, len(p.outputs))
	}
	for _, o := range p.outputs {
		o.send(m)
	}
}

func asyncOutputer(ws *workerSignal, hwm, bulkHWM int, worker *outputWorker) worker {
	config := worker.config

	flushInterval := defaultFlushInterval
	if config.FlushInterval != nil {
		flushInterval = time.Duration(*config.FlushInterval) * time.Millisecond
	}
	logp.Info("Flush Interval set to: %v", flushInterval)

	maxBulkSize := defaultBulkSize
	if config.BulkMaxSize != nil {
		maxBulkSize = *config.BulkMaxSize
	}
	logp.Info("Max Bulk Size set to: %v", maxBulkSize)

	// batching disabled
	if flushInterval <= 0 || maxBulkSize <= 0 {
		return worker
	}

	debug("create bulk processing worker (interval=%v, bulk size=%v)",
		flushInterval, maxBulkSize)
	return newBulkWorker(ws, hwm, bulkHWM, worker, flushInterval, maxBulkSize)
}

package publisher

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type asyncPublisher struct {
	messageWorker
	outputs []worker
	pub     *PublisherType
	ws      workerSignal
}

type asyncClient struct {
	publisher *asyncPublisher
}

const (
	defaultFlushInterval = 1000 * time.Millisecond // 1s
	defaultBulkSize      = 10000
)

func newAsyncPublisher(pub *PublisherType) *asyncPublisher {

	p := &asyncPublisher{pub: pub}
	p.ws.Init()

	var outputs []worker
	for _, out := range pub.Output {
		outputs = append(outputs, asyncOutputer(&p.ws, out))
	}

	p.outputs = outputs
	p.messageWorker.init(&pub.wsPublisher, 1000, newPreprocessor(pub, p))
	return p
}

// onStop will send stop signal to message batching workers
func (p *asyncPublisher) onStop() { p.ws.stop() }

func (p *asyncPublisher) onMessage(m message) {
	debug("async forward to outputers (%v)", len(p.outputs))

	// m.signal is not set yet. But a async client type supporting signals might
	// be implemented in the furute.
	// If m.signal is nil, NewSplitSignaler will return nil -> signaler will
	// only set if client did send one
	if m.context.signal != nil && len(p.outputs) > 1 {
		m.context.signal = outputs.NewSplitSignaler(m.context.signal, len(p.outputs))
	}
	for _, o := range p.outputs {
		o.send(m)
	}
}

func (p *asyncPublisher) client() eventPublisher {
	return asyncClient{p}
}

func (c asyncClient) PublishEvent(ctx context, event common.MapStr) bool {
	c.publisher.send(message{context: ctx, event: event})
	return true
}

func (c asyncClient) PublishEvents(ctx context, events []common.MapStr) bool {
	c.publisher.send(message{context: ctx, events: events})
	return true
}

func asyncOutputer(ws *workerSignal, worker *outputWorker) worker {
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
	return newBulkWorker(ws, 1000, worker, flushInterval, maxBulkSize)
}

package publisher

import (
	"time"

	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
)

type bulkWorker struct {
	output worker
	ws     *workerSignal

	queue       chan message
	bulkQueue   chan message
	guaranteed  bool
	flushTicker *time.Ticker

	maxBatchSize int
	data         []outputs.Data // batched events
	pending      []op.Signaler  // pending signalers for batched events
}

func newBulkWorker(
	ws *workerSignal, hwm int, bulkHWM int,
	output worker,
	flushInterval time.Duration,
	maxBatchSize int,
) *bulkWorker {
	b := &bulkWorker{
		output:       output,
		ws:           ws,
		queue:        make(chan message, hwm),
		bulkQueue:    make(chan message, bulkHWM),
		flushTicker:  time.NewTicker(flushInterval),
		maxBatchSize: maxBatchSize,
		data:         make([]outputs.Data, 0, maxBatchSize),
		pending:      nil,
	}

	b.ws.wg.Add(1)
	go b.run()
	return b
}

func (b *bulkWorker) send(m message) {
	send(b.queue, b.bulkQueue, m)
}

func (b *bulkWorker) run() {
	defer b.shutdown()

	for {
		select {
		case <-b.ws.done:
			return
		case m := <-b.queue:
			b.onEvent(&m.context, m.datum)
		case m := <-b.bulkQueue:
			b.onEvents(&m.context, m.data)
		case <-b.flushTicker.C:
			b.flush()
		}
	}
}

func (b *bulkWorker) flush() {
	if len(b.data) > 0 {
		b.publish()
	}
}

func (b *bulkWorker) onEvent(ctx *Context, data outputs.Data) {
	b.data = append(b.data, data)
	b.guaranteed = b.guaranteed || ctx.Guaranteed

	signal := ctx.Signal
	if signal != nil {
		b.pending = append(b.pending, signal)
	}

	if len(b.data) == cap(b.data) {
		b.publish()
	}
}

func (b *bulkWorker) onEvents(ctx *Context, data []outputs.Data) {
	for len(data) > 0 {
		// split up bulk to match required bulk sizes.
		// If input events have been split up bufferFull will be set and
		// bulk request will be published.
		spaceLeft := cap(b.data) - len(b.data)
		consume := len(data)
		bufferFull := spaceLeft <= consume
		signal := ctx.Signal
		b.guaranteed = b.guaranteed || ctx.Guaranteed
		if spaceLeft < consume {
			consume = spaceLeft
			if signal != nil {
				// creating cascading signaler chain for
				// subset of events being send
				signal = op.SplitSignaler(signal, 2)
			}
		}

		// buffer events
		b.data = append(b.data, data[:consume]...)
		data = data[consume:]
		if signal != nil {
			b.pending = append(b.pending, signal)
		}

		if bufferFull {
			b.publish()
		}
	}
}

func (b *bulkWorker) publish() {
	b.output.send(message{
		context: Context{
			publishOptions: publishOptions{Guaranteed: b.guaranteed},
			Signal:         op.CombineSignalers(b.pending...),
		},
		data: b.data,
	})

	b.pending = nil
	b.guaranteed = false
	b.data = make([]outputs.Data, 0, b.maxBatchSize)
}

func (b *bulkWorker) shutdown() {
	b.flushTicker.Stop()
	stopQueue(b.queue)
	stopQueue(b.bulkQueue)
	b.ws.wg.Done()
}

package publisher

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type bulkWorker struct {
	output worker
	ws     *workerSignal

	queue       chan message
	flushTicker *time.Ticker

	maxBatchSize int
	events       []common.MapStr    // batched events
	pending      []outputs.Signaler // pending signalers for batched events
}

func newBulkWorker(
	ws *workerSignal, hwm int, output worker,
	flushInterval time.Duration,
	maxBatchSize int,
) *bulkWorker {
	b := &bulkWorker{
		output:       output,
		ws:           ws,
		queue:        make(chan message, hwm),
		flushTicker:  time.NewTicker(flushInterval),
		maxBatchSize: maxBatchSize,
		events:       make([]common.MapStr, 0, maxBatchSize),
		pending:      nil,
	}

	ws.wg.Add(1)
	go b.run()
	return b
}

func (b *bulkWorker) send(m message) {
	b.queue <- m
}

func (b *bulkWorker) run() {
	defer b.shutdown()

	for {
		select {
		case <-b.ws.done:
			return
		case m := <-b.queue:
			if m.event != nil { // single event
				b.onEvent(m.context.signal, m.event)
			} else { // batch of events
				b.onEvents(m.context.signal, m.events)
			}

			// buffer full?
			if len(b.events) == cap(b.events) {
				b.publish()
			}
		case <-b.flushTicker.C:
			if len(b.events) > 0 {
				b.publish()
			}
		}
	}
}

func (b *bulkWorker) onEvent(signal outputs.Signaler, event common.MapStr) {
	b.events = append(b.events, event)
	if signal != nil {
		b.pending = append(b.pending, signal)
	}
}

func (b *bulkWorker) onEvents(signal outputs.Signaler, events []common.MapStr) {
	for len(events) > 0 {
		// split up bulk to match required bulk sizes.
		// If input events have been split up bufferFull will be set and
		// bulk request will be published.
		bufferFull := false
		spaceLeft := cap(b.events) - len(b.events)
		consume := len(events)
		if spaceLeft < consume {
			bufferFull = true
			consume = spaceLeft
			if signal != nil {
				// creating cascading signaler chain for
				// subset of events being send
				signal = outputs.NewSplitSignaler(signal, 2)
			}
		}

		// buffer events
		b.events = append(b.events, events[:consume]...)
		events = events[consume:]
		if signal != nil {
			b.pending = append(b.pending, signal)
		}

		if bufferFull {
			b.publish()
		}
	}
}

func (b *bulkWorker) publish() {
	// TODO: remember/merge and forward context options to output worker
	b.output.send(message{
		context: context{
			signal: outputs.NewCompositeSignaler(b.pending...),
		},
		events: b.events,
	})

	b.pending = nil
	b.events = make([]common.MapStr, 0, b.maxBatchSize)
}

func (b *bulkWorker) shutdown() {
	b.flushTicker.Stop()
	stopQueue(b.queue)
	b.ws.wg.Done()
}

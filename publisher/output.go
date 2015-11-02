package publisher

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

type outputWorker struct {
	messageWorker
	out         outputs.BulkOutputer
	config      outputs.MothershipConfig
	maxBulkSize int
}

func newOutputWorker(
	config outputs.MothershipConfig,
	out outputs.Outputer,
	ws *workerSignal,
	hwm int,
) *outputWorker {
	maxBulkSize := defaultBulkSize
	if config.Bulk_size != nil {
		maxBulkSize = *config.Bulk_size
	}

	o := &outputWorker{
		out:         outputs.CastBulkOutputer(out),
		config:      config,
		maxBulkSize: maxBulkSize,
	}
	o.messageWorker.init(ws, hwm, o)
	return o
}

func (o *outputWorker) onStop() {}

func (o *outputWorker) onMessage(m message) {

	if m.event != nil {
		o.onEvent(m.signal, m.event)
	} else {
		o.onBulk(m.signal, m.events)
	}
}

func (o *outputWorker) onEvent(s outputs.Signaler, event common.MapStr) {
	debug("output worker: publish single event")
	ts := time.Time(event["@timestamp"].(common.Time)).UTC()
	_ = o.out.PublishEvent(s, ts, event)
}

func (o *outputWorker) onBulk(signal outputs.Signaler, events []common.MapStr) {
	if len(events) == 0 {
		debug("output worker: no events to publish")
		outputs.SignalCompleted(signal)
		return
	}

	if o.maxBulkSize < 0 || len(events) <= o.maxBulkSize {
		o.sendBulk(signal, events)
		return
	}

	// start splitting bulk request
	splits := (len(events) + (o.maxBulkSize - 1)) / o.maxBulkSize
	signal = outputs.NewSplitSignaler(signal, splits)
	for len(events) > 0 {
		o.sendBulk(signal, events[:o.maxBulkSize])
		events = events[o.maxBulkSize:]
	}
}

func (o *outputWorker) sendBulk(signal outputs.Signaler, events []common.MapStr) {
	debug("output worker: publish %v events", len(events))
	ts := time.Time(events[0]["@timestamp"].(common.Time)).UTC()

	err := o.out.BulkPublish(signal, ts, events)
	if err != nil {
		logp.Info("Error bulk publishing events: %s", err)
	}
}

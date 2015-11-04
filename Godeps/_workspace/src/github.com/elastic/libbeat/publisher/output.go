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
	if config.BulkMaxSize != nil {
		maxBulkSize = *config.BulkMaxSize
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
		o.onEvent(&m.context, m.event)
	} else {
		o.onBulk(&m.context, m.events)
	}
}

func (o *outputWorker) onEvent(ctx *context, event common.MapStr) {
	debug("output worker: publish single event")
	ts := time.Time(event["@timestamp"].(common.Time)).UTC()

	if !ctx.sync {
		_ = o.out.PublishEvent(ctx.signal, ts, event)
		return
	}

	signal := outputs.NewSyncSignal()
	for {
		o.out.PublishEvent(signal, ts, event)
		if signal.Wait() {
			outputs.SignalCompleted(ctx.signal)
			break
		}
	}
}

func (o *outputWorker) onBulk(ctx *context, events []common.MapStr) {
	if len(events) == 0 {
		debug("output worker: no events to publish")
		outputs.SignalCompleted(ctx.signal)
		return
	}

	var sync *outputs.SyncSignal
	if ctx.sync {
		sync = outputs.NewSyncSignal()
	}

	if o.maxBulkSize < 0 || len(events) <= o.maxBulkSize {
		o.sendBulk(sync, ctx, events)
		return
	}

	// start splitting bulk request
	splits := (len(events) + (o.maxBulkSize - 1)) / o.maxBulkSize
	ctx.signal = outputs.NewSplitSignaler(ctx.signal, splits)
	for len(events) > 0 {
		sz := o.maxBulkSize
		if sz > len(events) {
			sz = len(events)
		}
		o.sendBulk(sync, ctx, events[:sz])
		events = events[sz:]
	}
}

func (o *outputWorker) sendBulk(
	sync *outputs.SyncSignal,
	ctx *context,
	events []common.MapStr,
) {
	debug("output worker: publish %v events", len(events))
	ts := time.Time(events[0]["@timestamp"].(common.Time)).UTC()

	if sync == nil {
		err := o.out.BulkPublish(ctx.signal, ts, events)
		if err != nil {
			logp.Info("Error bulk publishing events: %s", err)
		}
		return
	}

	for done := false; !done; done = sync.Wait() {
		o.out.BulkPublish(sync, ts, events)
	}
	outputs.SignalCompleted(ctx.signal)
}

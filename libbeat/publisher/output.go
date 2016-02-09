package publisher

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type outputWorker struct {
	messageWorker
	out         outputs.BulkOutputer
	config      outputs.MothershipConfig
	maxBulkSize int
}

var (
	errSendFailed = errors.New("failed send attempt")
)

func newOutputWorker(
	config outputs.MothershipConfig,
	out outputs.Outputer,
	ws *workerSignal,
	hwm int,
	bulkHWM int,
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
	o.messageWorker.init(ws, hwm, bulkHWM, o)
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

func (o *outputWorker) onEvent(ctx *Context, event common.MapStr) {
	debug("output worker: publish single event")
	o.out.PublishEvent(ctx.Signal, outputs.Options{ctx.Guaranteed}, event)
}

func (o *outputWorker) onBulk(ctx *Context, events []common.MapStr) {
	if len(events) == 0 {
		debug("output worker: no events to publish")
		outputs.SignalCompleted(ctx.Signal)
		return
	}

	if o.maxBulkSize < 0 || len(events) <= o.maxBulkSize {
		o.sendBulk(ctx, events)
		return
	}

	// start splitting bulk request
	splits := (len(events) + (o.maxBulkSize - 1)) / o.maxBulkSize
	ctx.Signal = outputs.NewSplitSignaler(ctx.Signal, splits)
	for len(events) > 0 {
		sz := o.maxBulkSize
		if sz > len(events) {
			sz = len(events)
		}
		o.sendBulk(ctx, events[:sz])
		events = events[sz:]
	}
}

func (o *outputWorker) sendBulk(
	ctx *Context,
	events []common.MapStr,
) {
	debug("output worker: publish %v events", len(events))

	opts := outputs.Options{ctx.Guaranteed}
	err := o.out.BulkPublish(ctx.Signal, opts, events)
	if err != nil {
		logp.Info("Error bulk publishing events: %s", err)
	}
}

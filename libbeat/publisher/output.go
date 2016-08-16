package publisher

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type outputWorker struct {
	messageWorker
	out         outputs.BulkOutputer
	config      outputConfig
	maxBulkSize int
}

type outputConfig struct {
	BulkMaxSize   int           `config:"bulk_max_size"`
	FlushInterval time.Duration `config:"flush_interval"`
}

var (
	defaultConfig = outputConfig{
		FlushInterval: 1 * time.Second,
		BulkMaxSize:   2048,
	}
)

var (
	errSendFailed = errors.New("failed send attempt")
)

func newOutputWorker(
	cfg *common.Config,
	out outputs.Outputer,
	ws *workerSignal,
	hwm int,
	bulkHWM int,
) *outputWorker {
	config := defaultConfig
	err := cfg.Unpack(&config)
	if err != nil {
		logp.Err("Failed to read output worker config: %v", err)
		return nil
	}

	o := &outputWorker{
		out:         outputs.CastBulkOutputer(out),
		config:      config,
		maxBulkSize: config.BulkMaxSize,
	}
	o.messageWorker.init(ws, hwm, bulkHWM, o)
	return o
}

func (o *outputWorker) onStop() {
	err := o.out.Close()
	if err != nil {
		logp.Info("Failed to close outputer: %s", err)
	}
}

func (o *outputWorker) onMessage(m message) {
	if m.datum.Event != nil {
		o.onEvent(&m.context, m.datum)
	} else {
		o.onBulk(&m.context, m.data)
	}
}

func (o *outputWorker) onEvent(ctx *Context, data outputs.Data) {
	debug("output worker: publish single event")
	opts := outputs.Options{Guaranteed: ctx.Guaranteed}
	o.out.PublishEvent(ctx.Signal, opts, data)
}

func (o *outputWorker) onBulk(ctx *Context, data []outputs.Data) {
	if len(data) == 0 {
		debug("output worker: no events to publish")
		op.SigCompleted(ctx.Signal)
		return
	}

	if o.maxBulkSize < 0 || len(data) <= o.maxBulkSize {
		o.sendBulk(ctx, data)
		return
	}

	// start splitting bulk request
	splits := (len(data) + (o.maxBulkSize - 1)) / o.maxBulkSize
	ctx.Signal = op.SplitSignaler(ctx.Signal, splits)
	for len(data) > 0 {
		sz := o.maxBulkSize
		if sz > len(data) {
			sz = len(data)
		}
		o.sendBulk(ctx, data[:sz])
		data = data[sz:]
	}
}

func (o *outputWorker) sendBulk(
	ctx *Context,
	data []outputs.Data,
) {
	debug("output worker: publish %v events", len(data))

	opts := outputs.Options{Guaranteed: ctx.Guaranteed}
	err := o.out.BulkPublish(ctx.Signal, opts, data)
	if err != nil {
		logp.Info("Error bulk publishing events: %s", err)
	}
}

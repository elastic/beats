package publisher

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type outputWorker struct {
	messageWorker
	out    outputs.BulkOutputer
	config outputs.MothershipConfig
}

func newOutputWorker(
	config outputs.MothershipConfig,
	out outputs.Outputer,
	ws *workerSignal,
	hwm int,
) *outputWorker {
	o := &outputWorker{out: outputs.CastBulkOutputer(out), config: config}
	o.messageWorker.init(ws, hwm, o)
	return o
}

func (o *outputWorker) onStop() {}

func (o *outputWorker) onMessage(m message) {

	if m.event != nil {
		debug("output worker: publish single event")
		ts := time.Time(m.event["timestamp"].(common.Time)).UTC()
		_ = o.out.PublishEvent(m.signal, ts, m.event)
	} else {
		if len(m.events) == 0 {
			debug("output worker: no events to publish")
			outputs.SignalCompleted(m.signal)
			return
		}

		debug("output worker: publish %v events", len(m.events))
		ts := time.Time(m.events[0]["timestamp"].(common.Time)).UTC()
		_ = o.out.BulkPublish(m.signal, ts, m.events)
	}
}

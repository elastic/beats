package publisher

import (
	"time"

	"github.com/elastic/libbeat/outputs"
)

type outputWorker struct {
	out    outputs.BulkOutputer
	config outputs.MothershipConfig
	worker *messageWorker
}

func newOutputWorker(
	config outputs.MothershipConfig,
	out outputs.Outputer,
	ws *workerSignal,
	hwm int,
) *outputWorker {
	o := &outputWorker{out: outputs.CastBulkOutputer(out)}
	o.worker = newMessageWorker(ws, hwm, o)
	return o
}

func (o *outputWorker) publish(m message) {
	o.worker.queue <- m
}

func (o *outputWorker) onMessage(m message) {
	if m.event != nil {
		o.out.PublishEvent(m.signal, m.event["ts"].(time.Time), m.event)
	} else {
		if len(m.events) == 0 {
			outputs.SignalCompleted(m.signal)
			return
		}

		ts := m.events[0]["ts"].(time.Time)
		o.out.BulkPublish(m.signal, ts, m.events)
	}
}

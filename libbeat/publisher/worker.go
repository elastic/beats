package publisher

import (
	"expvar"

	"github.com/elastic/beats/libbeat/common"
)

// Metrics that can retrieved through the expvar web interface.
var (
	messagesInWorkerQueues = expvar.NewInt("libbeatMessagesInWorkerQueues")
)

type worker interface {
	send(m message)
}

type messageWorker struct {
	queue     chan message
	bulkQueue chan message
	ws        *common.WorkerSignal
	handler   messageHandler
}

type message struct {
	context Context
	event   common.MapStr
	events  []common.MapStr
}

type messageHandler interface {
	onMessage(m message)
	onStop()
}

func newMessageWorker(ws *common.WorkerSignal, hwm, bulkHWM int, h messageHandler) *messageWorker {
	p := &messageWorker{}
	p.init(ws, hwm, bulkHWM, h)
	return p
}

func (p *messageWorker) init(ws *common.WorkerSignal, hwm, bulkHWM int, h messageHandler) {
	p.queue = make(chan message, hwm)
	p.bulkQueue = make(chan message, bulkHWM)
	p.ws = ws
	p.handler = h
	defer p.ws.WorkerStart()
	go p.run()
}

func (p *messageWorker) run() {
	defer p.shutdown()
	for {
		select {
		case <-p.ws.Done:
			return
		case m := <-p.queue:
			p.onEvent(m)
		case m := <-p.bulkQueue:
			p.onEvent(m)
		}
	}
}

func (p *messageWorker) shutdown() {
	p.handler.onStop()
	close(p.queue)
	close(p.bulkQueue)
	p.ws.WorkerFinished()
}

func (p *messageWorker) onEvent(m message) {
	messagesInWorkerQueues.Add(-1)
	p.handler.onMessage(m)
	p.ws.DoneEvent()
}

func (p *messageWorker) send(m message) {
	p.ws.AddEvent(1)
	if m.event != nil {
		p.queue <- m
	} else {
		p.bulkQueue <- m
	}
	messagesInWorkerQueues.Add(1)
}

package publisher

import (
	"expvar"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
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
	ws        *workerSignal
	handler   messageHandler
}

type message struct {
	context Context
	event   common.MapStr
	events  []common.MapStr
}

type workerSignal struct {
	done chan struct{}
	wg   sync.WaitGroup
}

type messageHandler interface {
	onMessage(m message)
	onStop()
}

func newMessageWorker(ws *workerSignal, hwm, bulkHWM int, h messageHandler) *messageWorker {
	p := &messageWorker{}
	p.init(ws, hwm, bulkHWM, h)
	return p
}

func (p *messageWorker) init(ws *workerSignal, hwm, bulkHWM int, h messageHandler) {
	p.queue = make(chan message, hwm)
	p.bulkQueue = make(chan message, bulkHWM)
	p.ws = ws
	p.handler = h
	ws.wg.Add(1)
	go p.run()
}

func (p *messageWorker) run() {
	defer p.shutdown()
	for {
		select {
		case <-p.ws.done:
			return
		case m := <-p.queue:
			messagesInWorkerQueues.Add(-1)
			p.handler.onMessage(m)
		case m := <-p.bulkQueue:
			messagesInWorkerQueues.Add(-1)
			p.handler.onMessage(m)
		}
	}
}

func (p *messageWorker) shutdown() {
	p.handler.onStop()
	stopQueue(p.queue)
	stopQueue(p.bulkQueue)
	p.ws.wg.Done()
}

func (p *messageWorker) send(m message) {
	if m.event != nil {
		p.queue <- m
	} else {
		p.bulkQueue <- m
	}
	messagesInWorkerQueues.Add(1)
}

func (ws *workerSignal) stop() {
	close(ws.done)
	ws.wg.Wait()
}

func newWorkerSignal() *workerSignal {
	w := &workerSignal{}
	w.Init()
	return w
}

func (ws *workerSignal) Init() {
	ws.done = make(chan struct{})
}

func stopQueue(qu chan message) {
	close(qu)
	for msg := range qu { // clear queue and send fail signal
		outputs.SignalFailed(msg.context.Signal, nil)
	}
}

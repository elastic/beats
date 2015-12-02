package publisher

import (
	"sync"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

type worker interface {
	send(m message)
}

type messageWorker struct {
	queue   chan message
	ws      *workerSignal
	handler messageHandler
}

type message struct {
	context context
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

func newMessageWorker(ws *workerSignal, hwm int, h messageHandler) *messageWorker {
	p := &messageWorker{}
	p.init(ws, hwm, h)
	return p
}

func (p *messageWorker) init(ws *workerSignal, hwm int, h messageHandler) {
	p.queue = make(chan message, hwm)
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
			p.handler.onMessage(m)
		}
	}
}

func (p *messageWorker) shutdown() {
	p.handler.onStop()
	stopQueue(p.queue)
	p.ws.wg.Done()
}

func (p *messageWorker) send(m message) {
	p.queue <- m
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
		outputs.SignalFailed(msg.context.signal, nil)
	}
}

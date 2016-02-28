package publisher

import (
	"expvar"
	"sync"

	"github.com/elastic/beats/libbeat/common"
)

// Metrics that can retrieved through the expvar web interface.
var (
	messagesInWorkerQueues = expvar.NewInt("libbeatMessagesInWorkerQueues")
)

type worker interface {
	send(m message)
	shutdown()
}

type messageWorker struct {
	queue     chan message
	bulkQueue chan message
	wg        *sync.WaitGroup
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

func newMessageWorker(wg *sync.WaitGroup, hwm, bulkHWM int, h messageHandler) *messageWorker {
	p := &messageWorker{}
	p.init(wg, hwm, bulkHWM, h)
	return p
}

func (p *messageWorker) init(wg *sync.WaitGroup, hwm, bulkHWM int, h messageHandler) {
	p.queue = make(chan message, hwm)
	p.bulkQueue = make(chan message, bulkHWM)
	p.wg = wg
	p.handler = h
	wg.Add(1)
	go p.run()
}

func (p *messageWorker) run() {
	defer func() {
		p.wg.Done()
	}()

	var queueClosed bool
	var bulkQueueClosed bool

	for {
		select {
		case m, ok := <-p.queue:
			if !ok {
				queueClosed = true
			} else {
				messagesInWorkerQueues.Add(-1)
				p.handler.onMessage(m)
				p.wg.Done()
			}
		case m, ok := <-p.bulkQueue:
			if !ok {
				bulkQueueClosed = true
			} else {
				messagesInWorkerQueues.Add(-1)
				p.handler.onMessage(m)
				p.wg.Done()
			}
		}
		if queueClosed && bulkQueueClosed {
			return
		}
	}
}

func (p *messageWorker) shutdown() {
	p.handler.onStop()
	close(p.queue)
	close(p.bulkQueue)
	p.wg.Wait()
}

func (p *messageWorker) send(m message) {
	p.wg.Add(1)
	if m.event != nil {
		p.queue <- m
	} else {
		p.bulkQueue <- m
	}
	messagesInWorkerQueues.Add(1)
}

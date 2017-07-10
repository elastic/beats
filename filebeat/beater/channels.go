package beater

import (
	"sync"

	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/filebeat/util"
)

type registrarLogger struct {
	done chan struct{}
	ch   chan<- []*util.Data
}

type finishedLogger struct {
	wg *sync.WaitGroup
}

func newRegistrarLogger(reg *registrar.Registrar) *registrarLogger {
	return &registrarLogger{
		done: make(chan struct{}),
		ch:   reg.Channel,
	}
}

func (l *registrarLogger) Close() { close(l.done) }
func (l *registrarLogger) Published(events []*util.Data) bool {
	select {
	case <-l.done:
		// set ch to nil, so no more events will be send after channel close signal
		// has been processed the first time.
		// Note: nil channels will block, so only done channel will be actively
		//       report 'closed'.
		l.ch = nil
		return false
	case l.ch <- events:
		return true
	}
}

func newFinishedLogger(wg *sync.WaitGroup) *finishedLogger {
	return &finishedLogger{wg}
}

func (l *finishedLogger) Published(events []*util.Data) bool {
	for range events {
		l.wg.Done()
	}

	return true
}

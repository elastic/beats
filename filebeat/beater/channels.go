package beater

import (
	"sync"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/monitoring"
)

type registrarLogger struct {
	done chan struct{}
	ch   chan<- []file.State
}

type finishedLogger struct {
	wg *eventCounter
}

type eventCounter struct {
	added *monitoring.Uint
	done  *monitoring.Uint
	count *monitoring.Int
	wg    sync.WaitGroup
}

func newRegistrarLogger(reg *registrar.Registrar) *registrarLogger {
	return &registrarLogger{
		done: make(chan struct{}),
		ch:   reg.Channel,
	}
}

func (l *registrarLogger) Close() { close(l.done) }
func (l *registrarLogger) Published(states []file.State) {
	select {
	case <-l.done:
		// set ch to nil, so no more events will be send after channel close signal
		// has been processed the first time.
		// Note: nil channels will block, so only done channel will be actively
		//       report 'closed'.
		l.ch = nil
	case l.ch <- states:
	}
}

func newFinishedLogger(wg *eventCounter) *finishedLogger {
	return &finishedLogger{wg}
}

func (l *finishedLogger) Published(n int) bool {
	for i := 0; i < n; i++ {
		l.wg.Done()
	}
	return true
}

func (c *eventCounter) Add(delta int) {
	c.count.Add(int64(delta))
	c.added.Add(uint64(delta))
	c.wg.Add(delta)
}

func (c *eventCounter) Done() {
	c.count.Dec()
	c.done.Inc()
	c.wg.Done()
}

func (c *eventCounter) Wait() {
	c.wg.Wait()
}

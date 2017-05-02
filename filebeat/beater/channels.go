package beater

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/filebeat/spooler"
	"github.com/elastic/beats/filebeat/util"
)

type spoolerOutlet struct {
	wg      *sync.WaitGroup
	done    <-chan struct{}
	spooler *spooler.Spooler

	isOpen int32 // atomic indicator
}

type publisherChannel struct {
	done chan struct{}
	ch   chan []*util.Data
}

type registrarLogger struct {
	done chan struct{}
	ch   chan<- []*util.Data
}

type finishedLogger struct {
	wg *sync.WaitGroup
}

func newSpoolerOutlet(
	done <-chan struct{},
	s *spooler.Spooler,
	wg *sync.WaitGroup,
) *spoolerOutlet {
	return &spoolerOutlet{
		done:    done,
		spooler: s,
		wg:      wg,
		isOpen:  1,
	}
}

func (o *spoolerOutlet) OnEvent(data *util.Data) bool {
	open := atomic.LoadInt32(&o.isOpen) == 1
	if !open {
		return false
	}

	if o.wg != nil {
		o.wg.Add(1)
	}

	select {
	case <-o.done:
		if o.wg != nil {
			o.wg.Done()
		}
		atomic.StoreInt32(&o.isOpen, 0)
		return false
	case o.spooler.Channel <- data:
		return true
	}
}

func newPublisherChannel() *publisherChannel {
	return &publisherChannel{
		done: make(chan struct{}),
		ch:   make(chan []*util.Data, 1),
	}
}

func (c *publisherChannel) Close() { close(c.done) }
func (c *publisherChannel) Send(events []*util.Data) bool {
	select {
	case <-c.done:
		// set ch to nil, so no more events will be send after channel close signal
		// has been processed the first time.
		// Note: nil channels will block, so only done channel will be actively
		//       report 'closed'.
		c.ch = nil
		return false
	case c.ch <- events:
		return true
	}
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

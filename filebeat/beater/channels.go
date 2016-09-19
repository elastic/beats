package beater

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/filebeat/input"
)

type prospectorOutput struct {
	wg     *sync.WaitGroup
	done   <-chan struct{}
	input  chan *input.Event
	isOpen int32 // atomic indicator
}

type spoolerOutput struct {
	done chan struct{}
	ch   chan []*input.Event
}

type publisherOutput struct {
	done chan struct{}
	ch   chan<- []*input.Event
}

type logger struct {
	wg *sync.WaitGroup
}

func newProspectorOutput(
	done <-chan struct{},
	input chan *input.Event,
	wg *sync.WaitGroup,
) *prospectorOutput {
	return &prospectorOutput{
		done:   done,
		input:  input,
		wg:     wg,
		isOpen: 1,
	}
}

func (o *prospectorOutput) Send(event *input.Event) bool {
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
	case o.input <- event:
		return true
	}
}

func newSpoolerOutput() *spoolerOutput {
	return &spoolerOutput{
		done: make(chan struct{}),
		ch:   make(chan []*input.Event, 1),
	}
}

func (c *spoolerOutput) Close() { close(c.done) }
func (c *spoolerOutput) Send(events []*input.Event) bool {
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

func newPublisherOutput(ch chan []*input.Event) *publisherOutput {
	return &publisherOutput{
		done: make(chan struct{}),
		ch:   ch,
	}
}

func (l *publisherOutput) Close() { close(l.done) }
func (l *publisherOutput) Send(events []*input.Event) bool {
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

func newLogger(wg *sync.WaitGroup) *logger {
	return &logger{wg}
}

func (l *logger) Log(events []*input.Event) bool {
	for range events {
		l.wg.Done()
	}

	return true
}

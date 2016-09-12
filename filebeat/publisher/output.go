package publisher

import "github.com/elastic/beats/filebeat/input"

type Output struct {
	done chan struct{}
	ch   chan []*input.Event
}

// Creates a Channel struct which can be used to send events to the publisher
// The channel itself must be passed to the publisher as publisherChan.GetChannel()
func NewOutput() *Output {
	return &Output{
		done: make(chan struct{}),
		ch:   make(chan []*input.Event, 1),
	}
}

func (c *Output) Close() { close(c.done) }
func (c *Output) Send(events []*input.Event) bool {
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

func (c *Output) GetChannel() chan []*input.Event {
	return c.ch
}

package scheduling

import "github.com/elastic/beats/libbeat/common/atomic"

type Context interface {
	Done() <-chan struct{}
}

type context struct {
	done   chan struct{}
	active atomic.Bool
}

func newContext() *context {
	return &context{
		done:   make(chan struct{}),
		active: atomic.MakeBool(true),
	}
}

func (c *context) Close() {
	if c.active.CAS(true, false) {
		close(c.done)
	}
}

func (c *context) Done() <-chan struct{} {
	return c.done
}

func (c *context) Active() bool {
	return c.active.Load()
}

package publisher

import "sync"

type canceling struct {
	lock   sync.RWMutex
	done   chan struct{} // signaler channel when client gets disconnected
	active bool
}

func newCanceller() *canceling {
	return &canceling{
		done:   make(chan struct{}),
		active: true,
	}
}

func (c *canceling) cancel() {
	c.lock.Lock()
	c.active = false
	c.lock.Unlock()

	// signal client being closed
	close(c.done)
}

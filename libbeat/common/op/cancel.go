package op

import "sync"

type Canceler struct {
	lock   sync.RWMutex
	done   chan struct{}
	active bool
}

func NewCanceler() *Canceler {
	return &Canceler{
		done:   make(chan struct{}),
		active: true,
	}
}

func (c *Canceler) Cancel() {
	c.lock.Lock()
	c.active = false
	c.lock.Unlock()

	close(c.done)
}

func (c *Canceler) Done() <-chan struct{} {
	return c.done
}

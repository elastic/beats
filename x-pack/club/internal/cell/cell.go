package cell

import (
	"github.com/elastic/go-concert/unison"
)

// Cell stores some state of type interface{}. A cell must have only a single owner
// that reads the state, or waits for state updates, but is allowed to have multiple setters.
// In this sense Cell is a multi producer, single consumer channel.
// Intermittent updates are lost, in case the cell is updated faster than the
// consumer tries to read for state updates. Updates are immediate, there will be no backpressure applied to producers.
//
// A typical use-case for cell is to generate asynchronous configuration updates (no deltas).
type Cell struct {
	mu unison.Mutex

	writeID uint // logical config state update counter
	readID  uint // local read state update counter. readID always follows writeID. We are using the most recent config if readID == waitID
	state   interface{}

	waiter chan struct{}
}

type waiter struct {
	ch chan struct{}
}

// NewCell creates a new call instance with its initial state. Subsequent reads will return this state, if there have been no updates.
func NewCell(st interface{}) *Cell {
	return &Cell{
		mu:    unison.MakeMutex(),
		state: st,
	}
}

// Get returns the current state.
func (c *Cell) Get() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.read()
}

// Wait blocks until it an update since the last call to Get or Wait has been found.
// The cancel context can be used to interrupt the call to Wait early. The
// error value will be set to the value returned by cancel.Err() in case Wait
// was interrupted. Wait does not produce any errors that need to be handled by itself.
func (c *Cell) Wait(cancel unison.Canceler) (interface{}, error) {
	c.mu.Lock()

	if c.readID < c.writeID {
		defer c.mu.Unlock()
		return c.read(), nil
	}

	waiter := make(chan struct{})
	c.waiter = waiter
	c.mu.Unlock()

	select {
	case <-cancel.Done():
		// we don't bother to check the waiter channel again. Cancellation if
		// detected has priority.
		c.mu.Lock()
		defer c.mu.Unlock()
		c.waiter = nil
		return nil, cancel.Err()
	case <-waiter:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.read(), nil
	}
}

// Set updates the state of the Cell and unblocks a waiting consumer.
// Set does not block.
func (c *Cell) Set(st interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.writeID++
	c.state = st
	c.notify()
}

func (c *Cell) read() interface{} {
	c.readID = c.writeID
	return c.state
}

func (c *Cell) notify() {
	if c.waiter != nil {
		close(c.waiter)
		c.waiter = nil
	}
}

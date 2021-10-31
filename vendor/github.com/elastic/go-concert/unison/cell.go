// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package unison

import "sync"

// Cell stores some state of type interface{}.
// Intermittent updates are lost, in case the Cell is updated faster than the
// consumer tries to read for state updates. Updates are immediate, there will
// be no backpressure applied to producers.
//
// In case the Cell is used to transmit updates without backpressure, the
// absolute state must be computed by the producer beforehand.
//
// A typical use-case for cell is to generate asynchronous configuration updates (no deltas).
//
// The zero value of Cell is valid, but a value of type Cell can not be copied.
type Cell struct {
	// All writes/reads to any of the internal fields must be guarded by mu.
	mu sync.Mutex

	// Current cell state to be Set and Get
	state interface{}

	// logical config state update counters.
	// The readID always follows writeID. We are using the most recent state
	// update if readID == waitID.
	writeID uint64
	readID  uint64

	// waiter is not nil if we have at least on go-routine blocked in `Wait`. The
	// `waiter` channel will be closed on `Set`.
	waiter chan struct{}

	// mini-object pool. If a wait gets cancelled we move the active 'waiter' to
	// 'waiterBuf'. The next call to Wait will reuse the already allocated
	// resource.
	waiterBuf chan struct{}

	// number of go-routines that share the current waiter. `numWaiter` must be 0 if `waiter == nil`.
	// Invariant: The `waiter` must not be nil if `numWaiter > 0`
	numWaiter int

	// current `waiter` instance ID in order to track potential races between multiple go-routines using Wait.
	// We use fine grained locking. If `waiterSessionID` is increased since our last lock attempt, then our
	// current wait session is 'outdated' (numWaiter, waiter must not be modified).
	waiterSessionID uint
}

// NewCell creates a new call instance with its initial state. Subsequent reads
// will return this state, if there have been no updates.
func NewCell(st interface{}) *Cell {
	return &Cell{state: st}
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
func (c *Cell) Wait(cancel Canceler) (interface{}, error) {
	c.mu.Lock()

	if c.readID != c.writeID {
		defer c.mu.Unlock()
		return c.read(), nil
	}

	var waiter chan struct{}
	var waiterSession uint

	if c.waiter == nil {
		// no active waiter: start new session
		if c.waiterBuf != nil {
			waiter = c.waiterBuf
			c.waiterBuf = nil
		} else {
			waiter = make(chan struct{})
		}

		c.waiter = waiter
		c.waiterSessionID++
	} else {
		// some other go-routine is already waiting. Let's join the current session.
		waiter = c.waiter
	}
	waiterSession = c.waiterSessionID
	c.numWaiter++
	c.mu.Unlock()

	select {
	case <-cancel.Done():
		// we don't bother to check the waiter channel again. Cancellation if
		// detected has priority.
		c.mu.Lock()
		defer c.mu.Unlock()

		// if waiterID and c.waiterID do not match we have had a race with `Set`
		// cleaning up the waiter state and another go-routine already calling wait
		// before we managed to lock the mutex.  In that case our waiterSession has
		// already been expired and we must not attempt to clean up the current
		// waiter state.
		if c.waiterSessionID == waiterSession {
			c.numWaiter--
			if c.numWaiter < 0 {
				// Race between Set and context cancellation. Set did already clean up the overall waiter state.
				// We must not attempt to clean up the state again -> repair state by undoing the local cleanup
				c.numWaiter++
			} else if c.numWaiter == 0 {
				// No more go-routine waiting for a state update and Set did not trigger yet. Let's clean up.
				c.waiterBuf = c.waiter
				c.waiter = nil
			}
		}
		return nil, cancel.Err()
	case <-waiter:
		c.mu.Lock()
		defer c.mu.Unlock()

		// waiter resource has been cleaned up by `Set`. Just read and return the
		// current known state.
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

	if c.waiter != nil {
		close(c.waiter)
		c.waiter = nil
		c.numWaiter = 0
	}
}

// read returns the current state and ensures that the next wait operation will
// only block correctly if there was no Set since the last read.
//
// IMPORTANT: c.mu MUST be locked while calling read.
func (c *Cell) read() interface{} {
	c.readID = c.writeID
	return c.state
}

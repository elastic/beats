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

package throttler

import (
	"math"
	"sync"

	"github.com/elastic/beats/libbeat/common/atomic"
)

const (
	stateUnstarted = iota
	stateStarted
	stateStopped
)

// Throttler is useful for managing access to some resource that can handle a certain amount of concurrency only.
// You could also do this with a Pool, but this uses a constant amount of memory, and doesn't need to have token
// objects passed around which is cleaner.
type Throttler struct {
	limit   uint
	buf     chan struct{}
	done    chan struct{}
	state   atomic.Int
	startWg sync.WaitGroup
}

// NewThrottler returns a new *Throttler that is not yet started. You must invoke Start for it to do anything.
func NewThrottler(limit uint) *Throttler {
	if limit < 1 { // assume unlimited
		limit = math.MaxUint32
	}

	t := &Throttler{
		limit:   limit,
		buf:     make(chan struct{}, limit),
		done:    make(chan struct{}),
		state:   atomic.MakeInt(stateUnstarted),
		startWg: sync.WaitGroup{},
	}

	t.startWg.Add(1)

	return t
}

// Start starts the internal thread and unblocks callers of AcquireSlot() which were invoked before this was called.
// Returns true if moving from unstarted to started. A stopped throttler cannot restart.
func (t *Throttler) Start() bool {
	if t.state.CAS(stateUnstarted, stateStarted) {
		t.startWg.Done()
		return true
	}

	return false
}

// Stop halts the internal goroutine. Once invoked this throttler will no longer be able to perform work.
// Returns true if moving from started to stopped. False in all other conditions.
func (t *Throttler) Stop() bool {
	if t.state.CAS(stateStarted, stateStopped) {
		close(t.done)
		return true
	}

	return false
}

// AcquireSlot attempts to acquire a resource. It returns whether acquisition was successful.
// If acquisition was successful releaseSlotFn must be invoked, otherwise it may be ignored.
func (t *Throttler) AcquireSlot() (acquired bool, releaseSlotFn func()) {
	// If we haven't started yet wait for it
	state := t.state.Load()
	if state == stateUnstarted {
		t.startWg.Wait()
	} else if state == stateStopped {
		// The select blocks below checking for a closed done aren't quite enough,
		// we need the stateStopped check here because select will pick a block at random, and we *always* want
		// AcquireSlot to return false once Stop is invoked.
		return false, func() {}
	}

	select {
	// Block until a resource is available
	case t.buf <- struct{}{}:
		released := atomic.NewBool(false)
		return true, func() {
			// Only release once even if this is invoked multiple times
			if released.CAS(false, true) {
				select {
				case <-t.buf:
					// release the acquired resource
				case <-t.done:
					// Release if we're shutdown
				}
			}
		}
	// If we're shutting down exit early
	case <-t.done:
		return false, func() {}
	}
}

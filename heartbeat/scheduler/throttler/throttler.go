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

	"github.com/elastic/beats/libbeat/common/atomic"
)

// Throttler is useful for managing access to some resource that can handle a certain amount of concurrency only.
// You could also do this with a Pool, but this uses a constant amount of memory, and doesn't need to have token
// objects passed around which is cleaner.
type Throttler struct {
	limit          uint
	availableSlots uint
	active         atomic.Int
	starts         chan struct{}
	stops          chan struct{}
	done           chan struct{}
	isDone         atomic.Bool
}

// NewThrottler returns a new *Throttler that is not yet started. You must invoke Start for it to do anything.
func NewThrottler(limit uint) *Throttler {
	if limit < 1 { // assume unlimited
		limit = math.MaxUint32
	}

	t := &Throttler{
		limit:          limit,
		availableSlots: limit,
		active:         atomic.Int{},
		starts:         make(chan struct{}),
		stops:          make(chan struct{}),
		done:           make(chan struct{}),
		isDone:         atomic.MakeBool(false),
	}

	return t
}

// Start starts the internal thread and unblocks callers of AcquireSlot() which were invoked before this was called.
func (t *Throttler) Start() {
	go func() {
		for {
			// If no slots are available, we just wait for jobs to stop, in which case
			// we can increase the number of slots for next time through the loop
			if t.availableSlots < 1 {
				select {
				case <-t.stops:
					t.availableSlots++
				case <-t.done:
					return
				}
			} else {
				select {
				case <-t.stops:
					t.availableSlots++
				case <-t.starts:
					t.availableSlots--
				case <-t.done:
					return
				}
			}
		}
	}()
}

// Stop halts the internal goroutine. Once invoked this throttler will no longer be able to perform work.
func (t *Throttler) Stop() {
	close(t.done)
}

// AcquireSlot attempts to acquire a resource. It returns whether acquisition was successful.
// If acquisition was successful releaseSlotFn must be invoked, otherwise it may be ignored.
func (t *Throttler) AcquireSlot() (acquired bool, releaseSlotFn func()) {
	t.active.Inc()
	select {
	// Block until a resource is available
	case t.starts <- struct{}{}:
		return true, func() {
			t.active.Dec()
			t.stops <- struct{}{}
		}
	// If we're shutting down exit early
	case <-t.done:
		return false, func() {}
	}
}

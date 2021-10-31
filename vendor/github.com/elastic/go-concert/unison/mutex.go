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

import (
	"time"
)

// Mutex provides a mutex based on go channels. The lock operations support
// timeout or cancellation by a context. Moreover one can try to lock the mutex
// from within a select statement when using Await.
//
// The zero value of Mutex will not be able to Lock the mutex ever. The Lock
// method will never return.  Calling Unlock will panic.
type Mutex struct {
	ch chan struct{}
}

// doneContext is a subset of context.Context, to allow more restrained
// cancellation types as well.
type doneContext interface {
	Done() <-chan struct{}
	Err() error
}

// MakeMutex creates a mutex.
func MakeMutex() Mutex {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return Mutex{ch: ch}
}

// Lock blocks until the mutex has been acquired.
// The zero value of Mutex will block forever.
func (c Mutex) Lock() {
	<-c.ch
}

// LockTimeout will try to lock the mutex. A failed lock attempt
// returns false, once the amount of configured duration has been passed.
//
// If duration is 0, then the call behaves like TryLock.
// If duration is <0, then the call behaves like Lock if the Mutex has been
// initialized, otherwise fails.
//
// The zero value of Mutex will never succeed.
func (c Mutex) LockTimeout(duration time.Duration) bool {
	switch {
	case duration == 0:
		return c.TryLock()
	case duration < 0:
		if c.ch == nil {
			return false
		}
		c.Lock()
		return true
	}

	timer := time.NewTimer(duration)
	select {
	case <-c.ch:
		timer.Stop()
		return true
	case <-timer.C:
		select {
		case <-c.ch: // still lock, if timer and lock occured at the same time
			return true
		default:
			return false
		}
	}
}

// LockContext tries to lock the mutex. The Log operation can be cancelled by
// the context.  LockContext returns nil on success, otherwise the error value
// returned by context.Err, which MUST NOT return nil after cancellation.
func (c Mutex) LockContext(context doneContext) error {
	select {
	case <-context.Done():
		return context.Err()
	default:
	}

	select {
	case <-c.ch:
		return nil
	case <-context.Done():
		return context.Err()
	}
}

// TryLock attempts to lock the mutex. If the mutex has been already locked
// false is returned.
func (c Mutex) TryLock() bool {
	select {
	case <-c.ch:
		return true
	default:
		return false
	}
}

// Await returns a channel that will be triggered if the lock attempt did succeed.
// One can use the channel with select-case. The mutex is assumed to be locked if
// the branch waiting on the mutex has been triggered.
func (c Mutex) Await() <-chan struct{} {
	return c.ch
}

// Unlock unlocks the mutex.
//
// The zero value of Mutex will panic.
func (c Mutex) Unlock() {
	select {
	case c.ch <- struct{}{}:
	default:
		panic("unlock on unlocked mutex")
	}
}

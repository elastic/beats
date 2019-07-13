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

package memlog

import "sync"

// Note: lock is copy from github.com/elastic/go-txfile/lock.go

type lock struct {
	mu sync.Mutex

	// conditions + mutexes
	shared    *sync.Cond
	exclusive *sync.Cond
	reserved  sync.Mutex

	// state
	sharedCount uint
	pendingSet  bool
}

type sharedLock lock
type reservedLock lock
type pendingLock lock
type exclusiveLock lock

func (l *lock) init() {
	l.shared = sync.NewCond(&l.mu)
	l.exclusive = sync.NewCond(&l.mu)
}

// Shared returns the files shared locker.
func (l *lock) Shared() *sharedLock { return (*sharedLock)(l) }

// Reserved returns the files reserved locker.
func (l *lock) Reserved() *reservedLock { return (*reservedLock)(l) }

// Pending returns the files pending locker.
func (l *lock) Pending() *pendingLock { return (*pendingLock)(l) }

// Pending returns the files exclusive locker.
func (l *lock) Exclusive() *exclusiveLock { return (*exclusiveLock)(l) }

func (l *sharedLock) Lock()       { waitCond(l.shared, l.check, l.inc) }
func (l *sharedLock) Unlock()     { withLocker(&l.mu, l.dec) }
func (l *sharedLock) check() bool { return !l.pendingSet }
func (l *sharedLock) inc()        { l.sharedCount++ }
func (l *sharedLock) dec() {
	l.sharedCount--
	if l.sharedCount == 0 {
		l.exclusive.Signal()
	}
}

func (l *reservedLock) Lock()   { l.reserved.Lock() }
func (l *reservedLock) Unlock() { l.reserved.Unlock() }

func (l *pendingLock) Lock() {
	l.mu.Lock()
	l.pendingSet = true
	l.mu.Unlock()
}
func (l *pendingLock) Unlock() {
	l.mu.Lock()
	l.pendingSet = false
	l.mu.Unlock()
	l.shared.Broadcast()
}

func (l *exclusiveLock) Lock()       { waitCond(l.exclusive, l.check, func() {}) }
func (l *exclusiveLock) Unlock()     {}
func (l *exclusiveLock) check() bool { return l.sharedCount == 0 }

func waitCond(c *sync.Cond, check func() bool, upd func()) {
	withLocker(c.L, func() {
		for !check() {
			c.Wait()
		}
		upd()
	})
}

func withLocker(l sync.Locker, fn func()) {
	l.Lock()
	defer l.Unlock()
	fn()
}

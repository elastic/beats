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

// TxLock provides locking support for transactional updates with multiple concurrent readers
// and one writer. Unlike sync.RWLock, the writer and readers can coexist.
// Users of TxLock must ensure proper isolation, between writers/readers.
// Changes by the writer must not be accessible by readers yet. A writer should
// hold the exclusive lock before making the changes available to others.
//
// Lock types:
//   - Shared: Shared locks are used by readonly loads. Multiple readers
//             can co-exist with one active writer.
//   - Reserved: Writer use the reserved lock. Once locked no other
//               writer can acquire the Reserved lock. The shared lock can
//               still be locked by concurrent readers.
//   - Pending: The pending lock is used by the writer to signal
//              a write is about to be committed.  After acquiring the pending
//              lock no new reader is allowed to acquire the shared lock.
//              Readers will have to wait until the pending lock is released.
//              Existing readers can still coexist, but no new reader is
//              allowed.
//   - Exclusive: Once the exclusive lock is acquired by a write transaction,
//                No other active transactions/locks exist anymore.
//                Locking the exclusive lock blocks until the shared lock has been
//                released by all readers.
//
// Each Locktype can be accessed using `(*lock).<Type>()`. Each lock type
// implements a `Lock` and `Unlock` method.
//
// The zero value of TxLock must not be used.
type TxLock struct {
	mu sync.Mutex

	// conditions + mutexes
	shared    *sync.Cond
	exclusive *sync.Cond
	reserved  sync.Mutex

	// state
	sharedCount uint
	reservedSet bool
	pendingSet  bool
}

// TxSharedLock is used by readers to lock the shared lock on a TxLock.
type TxSharedLock TxLock

// TxReservedLock is used by writers to lock the reserved lock on a TxLock.
// Only one go-routine is allowed to acquire the reserved lock at a time.
type TxReservedLock TxLock

// TxPendingLock is used by writers to signal the TxLock that the shared lock
// can not be acquired anymore. Readers will unblock once Unlock on the pending
// lock is called.
type TxPendingLock TxLock

// TxExclusiveLock is used by writers to wait for exclusive access to all resources.
// The writer should make changes visible to future readers only after acquiring the
// exclusive lock.
type TxExclusiveLock TxLock

// NewTxLock creates a new initialized TxLock instance.
func NewTxLock() *TxLock {
	l := &TxLock{}
	l.shared = sync.NewCond(&l.mu)
	l.exclusive = sync.NewCond(&l.mu)
	return l
}

// Get returns the standard Locker for the given transaction type.
func (l *TxLock) Get(readonly bool) sync.Locker {
	if readonly {
		return l.Shared()
	}
	return l.Reserved()
}

// Shared returns the files shared locker.
func (l *TxLock) Shared() *TxSharedLock { return (*TxSharedLock)(l) }

// Reserved returns the files reserved locker.
func (l *TxLock) Reserved() *TxReservedLock { return (*TxReservedLock)(l) }

// Pending returns the files pending locker.
func (l *TxLock) Pending() *TxPendingLock { return (*TxPendingLock)(l) }

// Exclusive returns the files exclusive locker.
func (l *TxLock) Exclusive() *TxExclusiveLock { return (*TxExclusiveLock)(l) }

// Lock locks the shared lock. It blocks as long as the pending lock
// is in use.
func (l *TxSharedLock) Lock() {
	waitCond(l.shared, l.check, l.inc)
}

// Unlock unlocks the shared lock. Unlocking potentially unblocks a waiting
// exclusive lock.
func (l *TxSharedLock) Unlock() {
	withLocker(&l.mu, l.dec)
}

func (l *TxSharedLock) check() bool { return !l.pendingSet }
func (l *TxSharedLock) inc()        { l.sharedCount++ }
func (l *TxSharedLock) dec() {
	l.sharedCount--
	if l.sharedCount == 0 {
		l.exclusive.Signal()
	}
}

// Lock acquires the reserved lock. Only one go-routine can hold the reserved
// lock at a time.  The reserved lock should only be acquire by writers.
// Writers must not acquire the shared lock.
func (l *TxReservedLock) Lock() {
	l.reserved.Lock()
	l.reservedSet = true
}

// Unlock releases the reserved lock.
func (l *TxReservedLock) Unlock() {
	l.reservedSet = false
	l.reserved.Unlock()
}

// Lock acquires the pending lock. The reserved lock must be acquired before.
func (l *TxPendingLock) Lock() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.reservedSet {
		panic("reserved lock must be set when acquiring the pending lock")
	}

	l.pendingSet = true
}

// Unlock releases the pending lock, potentially unblocking waiting readers.
func (l *TxPendingLock) Unlock() {
	l.mu.Lock()
	l.pendingSet = false
	l.mu.Unlock()
	l.shared.Broadcast()
}

// Lock acquires the exclusive lock. Once acquired it is guaranteed that no
// other reader or writer go-routine exists.
func (l *TxExclusiveLock) Lock() {
	if !l.pendingSet {
		panic("the pending lock must be set when acquiring the exclusive lock")
	}
	waitCond(l.exclusive, l.check, func() {})
}

// Unlock is a noop. It guarantees that TxExclusiveLock is compatible to sync.Locker.
func (l *TxExclusiveLock) Unlock() {}

func (l *TxExclusiveLock) check() bool {
	return l.sharedCount == 0
}

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

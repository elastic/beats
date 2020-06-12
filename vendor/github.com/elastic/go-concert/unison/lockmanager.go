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
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/elastic/go-concert"
	"github.com/elastic/go-concert/atomic"
)

// LockManager gives access to a set of Locks by name.  The lock manager can
// forcefully unlock a lock. Routines using a managed lock can use the
// LockSession to list for special Lock events.
//
// The zero value of LockManager is directly usable, but a LockManager must no
// be copied by value.
type LockManager struct {
	initOnce sync.Once
	mu       sync.Mutex
	table    map[string]*lockEntry
}

// ManagedLock is a mutex like structure that is managed by the LockManager.
// A managed lock can loose it's lock lease at any time. The LockX methods
// return a LockSession that can be used to listen for the current Locks state
// changes.
//
// The lock will automatically be released in case the ManagedLock is garbage
// collected. One should not rely on this behavior, but releaseing a zombie
// lock guarantees that other routines might eventually be able to make
// progress in case of fatal errors.
type ManagedLock struct {
	key     string
	manager *LockManager
	session *LockSession
	entry   *lockEntry
}

// LockOption is used to pass additonal settings to all LockX methods of the ManagedLock.
type LockOption interface {
	apply(l *ManagedLock)
}

// LockSession provides signal with the current lock state. Lock sessions must
// not be reused, as each Lock operation returns a new Session object.
type LockSession struct {
	isLocked             atomic.Bool
	done, unlocked, lost *concert.OnceSignaler
}

// WithSignalCallbacks is a LockOption that configures additional callbacks to
// be executed on lock session state changes.
// A LockSession is valid until the lock has been Unlocked. The callbacks are
// registered with the lock session, and will not be called anymore once the LockSession is finalized.
type WithSignalCallbacks struct {
	// Done is executed after the Unlocked or Lost event has been emitted. Done will never be executed twice,
	// even if both events get emitted.
	Done func()

	// Unlocked is called when the lock has been explicitely unlocked.
	Unlocked func()

	// Lost is called when the Lock was force unlocked by the LockManager.
	Lost func()
}

// lockEntry is the shared lock instances that all ManagedLocks refer too.
// The lockEntry is held in the LockManagers table for as long as at least one
// ManagedLock holds the lock or is attempting to acquire the lock.
// The lockEntry is supposed to be created lazily and shall be deleted from the
// LockManager as early as possible.
type lockEntry struct {
	session    *LockSession
	muInternal sync.Mutex // internal mutex

	// shared user mutex
	Mutex

	// book keeping, so we can remove the entry from the lock manager if there
	// are not more references to this entry.
	key string
	ref concert.RefCount
}

// GC Finalizer for the ManagedLock. This variable is used for testing.
var managedLockFinalizer = (*ManagedLock).finalize

// NewLockManager creates a new LockManager instance.
func NewLockManager() *LockManager {
	m := &LockManager{}
	m.init()
	return m
}

func (m *LockManager) init() {
	m.initOnce.Do(func() {
		m.table = map[string]*lockEntry{}
	})
}

// Access gives access to a ManagedLock. The ManagedLock MUST NOT be used by
// more than one go-routine. If 2 go-routines need to coordinate on a lock
// managed by the same LockManager, then 2 individual ManagedLock instances for
// the same key must be created.
func (m *LockManager) Access(key string) *ManagedLock {
	m.init()
	return newManagedLock(m, key)
}

// ForceUnlock unlocks the ManagedLock that is currently holding the Lock.
// It is advised to listen on the LockSession.LockLost or LockSession.Done events.
func (m *LockManager) ForceUnlock(key string) {
	m.init()

	m.mu.Lock()
	entry := m.findEntry(key)
	m.mu.Unlock()

	entry.muInternal.Lock()
	session := entry.session
	if session != nil {
		entry.session = nil
		if session.isLocked.Load() {
			entry.Mutex.Unlock()
		}

	}
	entry.muInternal.Unlock()

	session.forceUnlock()
	m.releaseEntry(entry)
}

// ForceUnlockAll force unlocks all current locks that are managed by this lock manager.
func (m *LockManager) ForceUnlockAll() {
	m.init()

	m.mu.Lock()
	for _, entry := range m.table {
		entry.muInternal.Lock()
		session := entry.session
		if session != nil && session.isLocked.Load() {
			entry.session = nil
			entry.Mutex.Unlock()
		}
		entry.muInternal.Unlock()

		session.forceUnlock()
	}
	m.mu.Unlock()
}

func (m *LockManager) createEntry(key string) *lockEntry {
	entry := &lockEntry{
		Mutex: MakeMutex(),
		key:   key,
	}
	m.table[key] = entry
	return entry
}

func (m *LockManager) findEntry(key string) *lockEntry {
	entry := m.table[key]
	if entry != nil {
		entry.ref.Retain()
	}
	return entry
}

func (m *LockManager) findOrCreate(key string, create bool) (entry *lockEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry = m.findEntry(key); entry == nil && create {
		entry = m.createEntry(key)
	}
	return entry
}

func (m *LockManager) releaseEntry(entry *lockEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ref.Release() {
		delete(m.table, entry.key)
	}
}

func newManagedLock(mngr *LockManager, key string) *ManagedLock {
	return &ManagedLock{key: key, manager: mngr}
}

func (ml *ManagedLock) finalize() {
	defer ml.unlink()
	if ml.session != nil {
		ml.doUnlock()
	}
}

// Key reports the key the lock will lock/unlock.
func (ml *ManagedLock) Key() string {
	return ml.key
}

// Lock the key. It blocks until the lock becomes
// available.
// Lock returns a LockSession, which is valid until after Unlock is called.
//
// Note: After loosing a lock, the ManagedLock must still call 'Unlock' in
//       order to be reusable.
func (ml *ManagedLock) Lock(opts ...LockOption) *LockSession {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	ml.entry.Lock()
	return ml.markLocked(opts)
}

// TryLock attempts to acquire the lock. If the lock is already held by another
// shared lock, then TryLock will return false.
//
// On success a LockSession will be returned as well. The Lock session is valid
// until Unlock has been called.
func (ml *ManagedLock) TryLock(opts ...LockOption) (*LockSession, bool) {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	if !ml.entry.TryLock() {
		ml.unlink()
		return nil, false
	}

	return ml.markLocked(opts), true
}

// LockTimeout will try to acquire lock. A failed lock attempt
// returns false, once the amount of configured duration has been passed.
//
// If duration is 0, then the call behaves like TryLock.
// If duration is <0, then the call behaves like Lock
//
// On success a LockSession will be returned as well. The Lock session is valid
// until Unlock has been called.
func (ml *ManagedLock) LockTimeout(duration time.Duration, opts ...LockOption) (*LockSession, bool) {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	if !ml.entry.LockTimeout(duration) {
		ml.unlink()
		return nil, false
	}
	return ml.markLocked(opts), ml.IsLocked()
}

// LockContext tries to acquire the lock. The Log operation can be cancelled by
// the context.  LockContext returns nil on success, otherwise the error value
// returned by context.Err, which MUST NOT return nil after cancellation.
//
// On success a LockSession will be returned as well. The Lock session is valid
// until Unlock has been called.
func (ml *ManagedLock) LockContext(context doneContext, opts ...LockOption) (*LockSession, error) {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	err := ml.entry.LockContext(context)
	if err != nil {
		ml.unlink()
		return nil, err
	}

	return ml.markLocked(opts), nil
}

// Unlock releases a resource.
func (ml *ManagedLock) Unlock() {
	checkActiveLockSession(ml.session)
	ml.doUnlock()
}

func (ml *ManagedLock) doUnlock() {
	session, entry := ml.session, ml.entry

	// The lock can be forcefully and asynchronously unreleased by the
	// LockManager. We can only unlock the entry, iff our mutex session is
	// still locked and the entries lock session still matches our lock session.
	// If none of these is the case, then the session was already closed.
	entry.muInternal.Lock()
	if session == entry.session {
		entry.session = nil
		if session.isLocked.Load() {
			entry.Unlock()
		}
	}
	entry.muInternal.Unlock()

	// always signal unlock, independent of the current state of the registry.
	// This will trigger the 'Unlocked' signal, indicating to session listeners that
	// the routine holding the lock deliberately unlocked the ManagedLock.
	// Note: a managed lock must always be explicitely unlocked, no matter of the
	// session state.
	session.unlock()

	ml.unlink()
	ml.markUnlocked()
}

// IsLocked checks if the resource currently holds the lock for the key
func (ml *ManagedLock) IsLocked() bool {
	return ml.session != nil && ml.session.isLocked.Load()
}

// link ensures that the managed lock is 'linked' to the shared lockEntry in
// the LockManagers table.
func (ml *ManagedLock) link(create bool) {
	if ml.entry == nil {
		ml.entry = ml.manager.findOrCreate(ml.key, create)
	}
}

// unlink removes the references to the table entry.
func (ml *ManagedLock) unlink() {
	if ml.entry == nil {
		return
	}

	entry := ml.entry
	ml.entry = nil
	ml.manager.releaseEntry(entry)
}

func (ml *ManagedLock) markLocked(opts []LockOption) *LockSession {
	session := newLockSession()
	ml.session = session

	for i := range opts {
		opts[i].apply(ml)
	}

	ml.entry.muInternal.Lock()
	ml.entry.session = session
	ml.entry.muInternal.Unlock()

	// in case we miss an unlock operation (programmer error or panic that has
	// been caught) we set a finalizer to eventually free the resource.
	// The Unlock operation will unsert the finalizer.
	runtime.SetFinalizer(ml, managedLockFinalizer)
	return session
}

func (ml *ManagedLock) markUnlocked() {
	runtime.SetFinalizer(ml, nil)
}

func newLockSession() *LockSession {
	return &LockSession{
		isLocked: atomic.MakeBool(true),
		done:     concert.NewOnceSignaler(),
		unlocked: concert.NewOnceSignaler(),
		lost:     concert.NewOnceSignaler(),
	}
}

// Done returns a channel to wait for a final signal. The signal will become available
// if the session has been finished due to an Unlock or Forced Unlock.
func (s *LockSession) Done() <-chan struct{} {
	if s == nil {
		return concert.ClosedChan()
	}
	return s.done.Done()
}

// Unlocked returns a channel, that will signal that the ManagedLock was
// unlocked.  A ManagedLock can still be unlocked (which will trigger the
// signal), even after loosing the actual Lock.
func (s *LockSession) Unlocked() <-chan struct{} {
	if s == nil {
		return concert.ClosedChan()
	}
	return s.unlocked.Done()
}

// LockLost return a channel, that will signal that the ManagedLock has lost
// its lock status.  When receiving this signal, ongoing operations should be
// cancelled, or results should be ignored, as other MangedLocks might be able
// to acquire the lock the moment the current session has lost the lock.
func (s *LockSession) LockLost() <-chan struct{} {
	if s == nil {
		return concert.ClosedChan()
	}
	return s.lost.Done()
}

func (s *LockSession) unlock()      { s.doUnlock(s.unlocked) }
func (s *LockSession) forceUnlock() { s.doUnlock(s.lost) }
func (s *LockSession) doUnlock(kind *concert.OnceSignaler) {
	s.isLocked.Store(false)
	kind.Trigger()
	s.done.Trigger()
}

func (opt WithSignalCallbacks) apply(lock *ManagedLock) {
	lock.session.done.OnSignal(opt.Done)
	lock.session.unlocked.OnSignal(opt.Unlocked)
	lock.session.lost.OnSignal(opt.Lost)
}

func checkNoActiveLockSession(s *LockSession) {
	invariant(s == nil, "lock still has an active lock session, missing call to Unlock to finish the session")
}

func checkActiveLockSession(s *LockSession) {
	invariant(s != nil, "no active lock session")
}

func invariant(b bool, message string) {
	if !b {
		panic(errors.New(message))
	}
}

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

package statestore

import (
	"sync"

	"github.com/elastic/go-concert"
	"github.com/elastic/go-concert/atomic"

	"github.com/elastic/beats/v7/libbeat/registry"
)

// Store provides some coordinates access to a registry.Store.
// All update and read operations require users to acquire a resource first.
// A Resource must be locked before it can be modified. This ensures that at most
// one go-routine has access to a resource. Lock/TryLock/Unlock can be used to
// coordinate resource access even between independent components.
type Store struct {
	active atomic.Bool

	session *storeSession
	shared  *sharedStore
}

// sharedStore is the shared store instance as is tracked by the Connector.
// Any two go-routines accessing a Store will reference the same underlying store.
// If one of the go-routines closes the store, we ensure that the shared resources
// are kept alive until all go-routines have given up access.
type sharedStore struct {
	refCount concert.RefCount

	name string

	persistentStore *registry.Store
	resourcesMux    sync.Mutex
	resources       table
}

// storeSession keeps track of the lifetime of a Store instance.
// In flight resource update operations do extend the lifetime of
// a Store, even if the store has been closed by a go-routine.
//
// A session will shutdown when the Store is closed and all pending
// update operations have been persisted.
type storeSession struct {
	refCount concert.RefCount
	store    *sharedStore

	// keep track of owner, so we can remove close the shared store once the last
	// session goes away.
	connector *Connector
}

func newSession(connector *Connector, store *sharedStore) *storeSession {
	session := &storeSession{connector: connector, store: store}
	session.refCount.Action = func(_ error) { session.Close() }
	return session
}

func (s *storeSession) Close() {
	s.connector.releaseStore(s.store)
	s.store = nil
	s.connector = nil
}

func (s *storeSession) Retain()       { s.refCount.Retain() }
func (s *storeSession) Release() bool { return s.refCount.Release() }

// newStore creates a new state Store with an active session.
func newStore(session *storeSession) *Store {
	invariant(session != nil, "missing a persistent store")

	return &Store{
		active:  atomic.MakeBool(true),
		shared:  session.store,
		session: session,
	}
}

// Close deactivates the store and waits for all resources to be released.
// Resources can not be accessed anymore, but in progress resource updates are
// still active, until they are eventually ACKed.  The underlying persistent
// store will be finally closed once all pending updates have been written to
// the persistent store.
func (s *Store) Close() {
	s.active.Store(false)
	s.session.Release()
}

// Access an unlocked resource. This creates a handle to a resource that may
// not yet exist in the persistent registry.
func (s *Store) Access(key ResourceKey) *Resource {
	return newResource(s, key)
}

func (s *Store) findOrCreate(key ResourceKey, lm lockMode) (res *resourceEntry) {
	if !s.active.Load() {
		return nil
	}

	withLockMode(&s.shared.resourcesMux, lm, func() error {
		if res = s.shared.resources.Find(key); res == nil {
			res = s.shared.resources.Create(key)
		}
		return nil
	})
	return res
}

// find returns the in memory resource entry, if the key is known to the store.
// find returns nil if the resource is unknown so far.
func (s *Store) find(key ResourceKey, lm lockMode) (res *resourceEntry) {
	if !s.active.Load() {
		return nil
	}
	return s.shared.find(key, lm)
}

// create adds a new entry to the in-memory table and returns the pointer to the entry.
// It fails if the store has been deactivated and returns 'null'.
func (s *Store) create(key ResourceKey, lm lockMode) (res *resourceEntry) {
	if !s.active.Load() {
		return nil
	}
	return s.shared.create(key, lm)
}

func (s *sharedStore) close() error {
	err := s.persistentStore.Close()
	s.persistentStore = nil
	return err
}

func (s *sharedStore) releaseEntry(entry *resourceEntry) {
	s.resourcesMux.Lock()
	defer s.resourcesMux.Unlock()
	if entry.refCount.Release() {
		s.remove(entry.key, lockAlreadyTaken)
	}
}

func (s *sharedStore) find(key ResourceKey, lm lockMode) (res *resourceEntry) {
	withLockMode(&s.resourcesMux, lm, func() error {
		res = s.resources.Find(key)
		return nil
	})
	return res
}

func (s *sharedStore) create(key ResourceKey, lm lockMode) (res *resourceEntry) {
	withLockMode(&s.resourcesMux, lm, func() error {
		res = s.resources.Create(key)
		return nil
	})
	return res
}

func (s *sharedStore) remove(key ResourceKey, lm lockMode) {
	withLockMode(&s.resourcesMux, lm, func() error {
		s.resources.Remove(key)
		return nil
	})
}

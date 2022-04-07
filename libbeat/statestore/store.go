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
	"github.com/elastic/beats/v8/libbeat/statestore/backend"
	"github.com/elastic/go-concert/atomic"
	"github.com/elastic/go-concert/unison"
)

type sharedStore struct {
	reg      *Registry
	refCount atomic.Int

	name    string
	backend backend.Store
}

// Store instance. The backend is shared between multiple instances of this store.
// The backend will be closed only after the last instance have been closed.
// No transaction can be created once the store instance has been closed.
// A Store is not thread-safe. Each go-routine accessing a store should create
// an instance using `Registry.Get`.
type Store struct {
	shared *sharedStore
	// wait group to ensure active operations can finish, but not started anymore after the store has been closed.
	active unison.SafeWaitGroup
}

func newSharedStore(reg *Registry, name string, backend backend.Store) *sharedStore {
	return &sharedStore{
		reg:      reg,
		refCount: atomic.MakeInt(1),
		name:     name,
		backend:  backend,
	}
}

func newStore(shared *sharedStore) *Store {
	shared.Retain()
	return &Store{
		shared: shared,
	}
}

// Close deactivates the current store. No new transacation can be generated.
// Already active transaction will continue to function until Closed.
// The backing store will be closed once all stores and active transactions have been closed.
func (s *Store) Close() error {
	if err := s.active.Add(1); err != nil {
		return &ErrorClosed{operation: "store/close", name: s.shared.name}
	}
	s.active.Close()
	s.active.Done()

	s.active.Wait()
	return s.shared.Release()
}

// Has checks if the given key exists.
// Has returns an error if the store has already been closed or the storage backend returns an error.
func (s *Store) Has(key string) (bool, error) {
	const operation = "store/has"
	if err := s.active.Add(1); err != nil {
		return false, &ErrorClosed{operation: operation, name: s.shared.name}
	}
	defer s.active.Done()

	has, err := s.shared.backend.Has((key))
	if err != nil {
		return false, &ErrorOperation{name: s.shared.name, operation: operation, cause: err}
	}
	return has, nil
}

// Get unpacks the value for a given key into "into".
// Get returns an error if the store has already been closed, the key does not
// exist, or the storage backend returns an error.
func (s *Store) Get(key string, into interface{}) error {
	const operation = "store/get"
	if err := s.active.Add(1); err != nil {
		return &ErrorClosed{operation: operation, name: s.shared.name}
	}
	defer s.active.Done()

	err := s.shared.backend.Get(key, into)
	if err != nil {
		return &ErrorOperation{name: s.shared.name, operation: operation, cause: err}
	}
	return nil
}

// Set inserts or overwrite a key value pair.
// Set returns an error if the store has been closed, the value can not be
// encoded by the store, or the storage backend did failed.
func (s *Store) Set(key string, from interface{}) error {
	const operation = "store/get"
	if err := s.active.Add(1); err != nil {
		return &ErrorClosed{operation: operation, name: s.shared.name}
	}
	defer s.active.Done()

	if err := s.shared.backend.Set((key), from); err != nil {
		return &ErrorOperation{name: s.shared.name, operation: operation, cause: err}
	}
	return nil
}

// Remove removes a key value pair from the store. Remove does not error if the
// key is unknown to the store.
// An error is returned if the store has already been closed or the operation
// itself fails in the storage backend.
func (s *Store) Remove(key string) error {
	const operation = "store/remove"
	if err := s.active.Add(1); err != nil {
		return &ErrorClosed{operation: operation, name: s.shared.name}
	}
	defer s.active.Done()

	if err := s.shared.backend.Remove((key)); err != nil {
		return &ErrorOperation{name: s.shared.name, operation: operation, cause: err}
	}
	return nil
}

// Each iterates over all key-value pairs in the store.
// The iteration stops if fn returns false or an error value != nil.
// If the store has been closed already an error is returned.
func (s *Store) Each(fn func(string, ValueDecoder) (bool, error)) error {
	if err := s.active.Add(1); err != nil {
		return &ErrorClosed{operation: "store/each", name: s.shared.name}
	}
	defer s.active.Done()

	return s.shared.backend.Each(fn)
}

func (s *sharedStore) Retain() {
	s.refCount.Inc()
}

func (s *sharedStore) Release() error {
	if s.refCount.Dec() == 0 && s.tryUnregister() {
		return s.backend.Close()
	}
	return nil
}

// tryUnregister removed the store from the registry. tryUnregister returns false
// if the store has been retained in the meantime. True is returned if the store
// can be closed for sure.
func (s *sharedStore) tryUnregister() bool {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	if s.refCount.Load() > 0 {
		return false
	}

	s.reg.unregisterStore(s)
	return true
}

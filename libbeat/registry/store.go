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

package registry

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/registry/backend"
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
	shared   *sharedStore
	active   bool
	activeTx sync.WaitGroup // active transaction
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
		active: true,
	}
}

// Close deactivates the current store. No new transacation can be generated.
// Already active transaction will continue to function until Closed.
// The backing store will be closed once all stores and active transactions have been closed.
func (s *Store) Close() error {
	if !s.active {
		return errStoreClosed
	}

	s.active = false
	s.activeTx.Wait()

	s.shared.Release()
	return nil
}

// Begin starts a new transaction within the current store.
// The store ref-counter is increased, such that the final close on a store
// will only happen if all transaction have been closed as well.
// A transaction started with `Begin` must be closed, rolled back or comitted.
// For 'local' transaction and/or guarantees that `Close`, `Rollback`, or `Commit`
// is called correctly use the `Update` or `View` methods.
func (s *Store) Begin(readonly bool) (*Tx, error) {
	if !s.active {
		return nil, errStoreClosed
	}

	tx, err := s.shared.backend.Begin(readonly)
	if err != nil {
		return nil, err
	}

	s.activeTx.Add(1)
	return newTx(s, tx, readonly), nil
}

// Update runs fn within a writeable transaction. The transaction will be
// rolled back if fn panics or returns an error.
// The transaction will be comitted if fn returns without error.
func (s *Store) Update(fn func(tx *Tx) error) error {
	tx, err := s.Begin(false)
	if err != nil {
		return err
	}

	defer tx.Close()
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// View executes a readonly transaction. An error is return if the readonly
// transaction can not be generated or fn returns an error.
func (s *Store) View(fn func(tx *Tx) error) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}

	defer tx.Close()
	return fn(tx)
}

func (s *Store) finishTx(tx *Tx) {
	s.activeTx.Done()
}

func (s *sharedStore) Retain() {
	s.refCount.Inc()
}

func (s *sharedStore) Release() {
	if s.refCount.Dec() == 0 && s.tryUnregister() {
		s.backend.Close()
	}
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

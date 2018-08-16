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

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/registry/backend"
)

type sharedStore struct {
	reg      *Registry
	refCount atomic.Int

	name    string
	backend backend.Store
}

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

	s.shared.close()
	return nil
}

// Begin starts a new transaction within the current store.
// The store ref-counter is increased, such that the final close on a store
// will only happen if all transaction have been closed as well.
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

func (s *sharedStore) close() {
	if s.refCount.Dec() > 0 {
		return
	}

	reg := s.reg
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if s.refCount.Load() > 0 {
		return
	}

	reg.unregisterStore(s)
	s.backend.Close()
}

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

// Package storetest provides helpers for testing functionality that requires a statestore.
package storetest

import (
	"errors"
	"sync"

	"github.com/menderesk/beats/v7/libbeat/common/transform/typeconv"
	"github.com/menderesk/beats/v7/libbeat/statestore/backend"
)

// MemoryStore provides a dummy backend store that holds all access stores and
// data in memory. The Stores field is accessible for introspection or custom
// initialization. Stores should not be modified while a test is active.
// For validation one can use the statestore API or introspect the tables directly.
//
// The zero value is MemoryStore is a valid store instance. The Stores field
// will be initialized lazily if it has not been setup upfront.
//
// Example: Create store for testing:
//    store := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
type MemoryStore struct {
	Stores map[string]*MapStore
	mu     sync.Mutex
}

// MapStore implements a single in memory storage. The MapStore holds all
// key-value pairs in a map[string]interface{}.
type MapStore struct {
	mu     sync.RWMutex
	closed bool
	Table  map[string]interface{}
}

type valueUnpacker struct {
	from interface{}
}

// CreateValueDecoder creates a backend.ValueDecoder that can be used to unpack
// an value into a custom go type.
func CreateValueDecoder(v interface{}) backend.ValueDecoder {
	return valueUnpacker{v}
}

var errMapStoreClosed = errors.New("store closed")
var errUnknownKey = errors.New("unknown key")

// NewMemoryStoreBackend creates a new backend.Registry instance that can be
// used with the statestore.
func NewMemoryStoreBackend() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) init() {
	if m.Stores == nil {
		m.Stores = map[string]*MapStore{}
	}
}

// Access returns a MapStore that for the given name. A new store is created
// and registered in the Stores table, if the store name is new to MemoryStore.
func (m *MemoryStore) Access(name string) (backend.Store, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()

	store, exists := m.Stores[name]
	if !exists {
		store = &MapStore{}
		m.Stores[name] = store
	} else {
		store.Reopen()
	}
	return store, nil
}

// Close closes the store.
func (m *MemoryStore) Close() error { return nil }

func (s *MapStore) init() {
	if s.Table == nil {
		s.Table = map[string]interface{}{}
	}
}

// Reopen marks the MapStore as open in case it has been closed already.  All
// key-value pairs and store operations are accessible after reopening the
// store.
func (s *MapStore) Reopen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = false
}

// Close marks the store as closed. The Store API calls like Has, Get, Set, and
// Remove will fail until the store is reopenned.
func (s *MapStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// IsClosed returns true if the store is marked as closed.
func (s *MapStore) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Has checks if the key value pair is known to the store.
// It returns an error if the store is marked as closed.
func (s *MapStore) Has(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return false, errMapStoreClosed
	}

	s.init()
	_, exists := s.Table[key]
	return exists, nil
}

// Get returns a key value pair from the store. An error is returned if the
// store has been closed, the key is unknown, or an decoding error occured.
func (s *MapStore) Get(key string, into interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return errMapStoreClosed
	}

	s.init()
	val, exists := s.Table[key]
	if !exists {
		return errUnknownKey
	}
	return typeconv.Convert(into, val)
}

// Set inserts or overwrites a key-value pair.
// An error is returned if the store is marked as closed or the value being
// passed in can not be encoded.
func (s *MapStore) Set(key string, from interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errMapStoreClosed
	}

	s.init()
	var tmp interface{}
	if err := typeconv.Convert(&tmp, from); err != nil {
		return err
	}
	s.Table[key] = tmp
	return nil
}

// Remove removes a key value pair from the store.
// An error is returned if the store is marked as closed.
func (s *MapStore) Remove(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errMapStoreClosed
	}

	s.init()
	delete(s.Table, key)
	return nil
}

// Each iterates all key value pairs in the store calling fn.
// The iteration stops if fn returns false or an error.
// Each returns an error if the store is closed, or fn returns an error.
func (s *MapStore) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return errMapStoreClosed
	}

	s.init()
	for k, v := range s.Table {
		cont, err := fn(k, CreateValueDecoder(v))
		if !cont || err != nil {
			return err
		}
	}
	return nil
}

func (d valueUnpacker) Decode(to interface{}) error {
	return typeconv.Convert(to, d.from)
}

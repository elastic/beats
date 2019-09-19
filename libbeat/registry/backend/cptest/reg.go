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

package cptest

import (
	"testing"

	"github.com/elastic/beats/libbeat/registry/backend"
)

// Registry wraps a backend.Registry and adds testing.T. The methods on
// Registry check and report errors to the testing instance, while the actual
// error return type - normally returned by the registry frontend - is removed
// from the Registry methods.
type Registry struct {
	T *testing.T
	backend.Registry
}

// Store is returned when accessing a store from the testing Registry.
// The store methods check and report unexpected backend errors to the
// underlying testing instance.
type Store struct {
	backend.Store

	Registry *Registry
	name     string
}

// Tx is returned by the testing store. Tx wraps a backend transaction, adding
// a many convenience methods.
type Tx struct {
	Store *Store
	backend.Tx
}

// Access creates a new test Store, by accessing and wrapping the backend
// store.
func (r *Registry) Access(name string) *Store {
	t := r.T
	s, err := r.Registry.Access(name)
	if err != nil {
		t.Fatal(err)
	}
	return &Store{Registry: r, name: name, Store: s}
}

// ReopenIf reopens the store if b is true.
func (s *Store) ReopenIf(b bool) {
	if b {
		s.Reopen()
	}
}

// Reopen reopens the store by closing the backend store and using the registry
// backend to access the same store again.
func (s *Store) Reopen() {
	t := s.Registry.T

	s.Close()
	if t.Failed() {
		t.Fatal("Test already failed")
	}

	store, err := s.Registry.Registry.Access(s.name)
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}

	s.Store = store
}

// Close closes the testing store.
func (s *Store) Close() {
	t := s.Registry.T

	if err := s.Store.Close(); err != nil {
		t.Errorf("error closing store %q: %v", s.name, err)
	}
}

// Begin creates a new transaction. The test will fail immediately (Fatal), if
// the backend returns an unexpected error.
func (s *Store) Begin(readonly bool) *Tx {
	t := s.Registry.T

	tx, err := s.Store.Begin(readonly)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	return &Tx{Store: s, Tx: tx}
}

// MustUpdate create a Write transaction that must succeed. MustUpdate fails if
// the final commit fails.
func (s *Store) MustUpdate(fn func(tx *Tx)) {
	s.Update(func(tx *Tx) bool {
		fn(tx)
		return true
	})
}

// Update create a new write transaction. The transaction is only committed if
// fn return true.
// The test fails if the transaction is supposed to succeed, but the final commit fails.
func (s *Store) Update(fn func(tx *Tx) bool) {
	tx := s.Begin(false)
	defer tx.Close()

	commit := fn(tx)

	t := s.Registry.T
	if commit && !t.Failed() {
		must(t, tx.Commit(), "commit failed")
	}
}

// View runs fn with a temporary readonly transaction.
func (s *Store) View(fn func(tx *Tx)) {
	tx := s.Begin(true)
	defer tx.Close()
	fn(tx)
}

// Set sets a key-value using a temporary transaction.
func (s *Store) Set(key backend.Key, val interface{}) {
	s.MustUpdate(func(tx *Tx) { tx.MustSet(key, val) })
}

// UpdValue updates a key-value pair using a temporary write transaction.
func (s *Store) UpdValue(key backend.Key, val interface{}) {
	s.MustUpdate(func(tx *Tx) {
		t := s.Registry.T
		must(t, tx.Update(key, val), "update failed")
	})
}

// Remove removes a key-value pair using a temporary write transaction.
func (s *Store) Remove(key backend.Key) {
	s.MustUpdate(func(tx *Tx) {
		t := s.Registry.T
		must(t, tx.Remove(key), "unexpected error on remove")
	})
}

// Has checks if a key exists in the store, using a temporary readonly transaction.
func (s *Store) Has(key backend.Key) (found bool) {
	s.View(func(tx *Tx) { found = tx.Has(key) })
	return found
}

// GetValue decodes a key-value pair into to, using a temporary readonly transaction.
func (s *Store) GetValue(k backend.Key, to interface{}) {
	s.View(func(tx *Tx) {
		tx.MustGetValue(k, to)
	})
}

// Has checks if a key exists in the store or within the current transaction.
// The test fails if the backend reports an error.
func (tx *Tx) Has(k backend.Key) bool {
	t := tx.Store.Registry.T
	found, err := tx.Tx.Has(k)
	must(t, err, "error testing for key presence")
	return found
}

// MustGet returns the value decoder for a key-value pair, or nil if the key is unknown.
// The test fails if the backend reports an error.
func (tx *Tx) MustGet(k backend.Key) backend.ValueDecoder {
	dec, err := tx.Get(k)
	must(tx.Store.Registry.T, err, "unknown key")
	return dec
}

// GetValue decodes a key-value pair into to, only if the key exists.
// Error returned by the backend will be returned.
func (tx *Tx) GetValue(k backend.Key, to interface{}) error {
	dec, err := tx.Get(k)
	if err == nil && dec != nil {
		err = dec.Decode(to)
	}
	return err
}

// MustGetValue decodes a key-value pair into to, only if the key exists.
// The test fails if the backend reports an error.
func (tx *Tx) MustGetValue(k backend.Key, to interface{}) {
	t := tx.Store.Registry.T

	err := tx.GetValue(k, to)
	if err != nil {
		t.Fatalf("Failed to read key: %q", k)
	}
}

// MustSet sets a key value pair in the current transaction.
// The test fails if the backend reports an error.
func (tx *Tx) MustSet(k backend.Key, v interface{}) {
	t := tx.Store.Registry.T
	must(t, tx.Tx.Set(k, v), "must set value")
}

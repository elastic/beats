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

type Registry struct {
	T *testing.T
	backend.Registry
}

type Store struct {
	backend.Store

	Registry *Registry
	name     string
}

type Tx struct {
	Store *Store
	backend.Tx
}

func (r *Registry) Access(name string) *Store {
	t := r.T
	s, err := r.Registry.Access(name)
	if err != nil {
		t.Fatal(err)
	}
	return &Store{Registry: r, name: name, Store: s}
}

func (s *Store) ReopenIf(b bool) {
	if b {
		s.Reopen()
	}
}

func (s *Store) Reopen() {
	t := s.Registry.T

	s.Close()
	if t.Failed() {
		t.Fatal("Test already failed")
	}

	store, err := s.Registry.Registry.Access(s.name)
	if err != nil {
		t.Fatalf("Repoen failed: %v", err)
	}

	s.Store = store
}

func (s *Store) Close() {
	t := s.Registry.T

	if err := s.Store.Close(); err != nil {
		t.Errorf("error closing store %q: %v", s.name, err)
	}
}

func (s *Store) Begin(readonly bool) *Tx {
	t := s.Registry.T

	tx, err := s.Store.Begin(readonly)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	return &Tx{Store: s, Tx: tx}
}

func (s *Store) MustUpdate(fn func(tx *Tx)) {
	s.Update(func(tx *Tx) bool {
		fn(tx)
		return true
	})
}

func (s *Store) Update(fn func(tx *Tx) bool) {
	tx := s.Begin(false)
	defer tx.Close()

	commit := fn(tx)

	t := s.Registry.T
	if commit && !t.Failed() {
		must(t, tx.Commit(), "commit failed")
	}
}

func (s *Store) View(fn func(tx *Tx)) {
	tx := s.Begin(true)
	defer tx.Close()
	fn(tx)
}

func (s *Store) Set(key []byte, val interface{}) {
	s.MustUpdate(func(tx *Tx) { tx.MustSet(key, val) })
}

func (s *Store) UpdValue(key []byte, val interface{}) {
	s.MustUpdate(func(tx *Tx) {
		t := s.Registry.T
		must(t, tx.Update(key, val), "update failed")
	})
}

func (s *Store) Remove(key []byte) {
	s.MustUpdate(func(tx *Tx) {
		t := s.Registry.T
		must(t, tx.Remove(key), "unexpected error on remove")
	})
}

func (s *Store) Has(key []byte) (found bool) {
	s.View(func(tx *Tx) { found = tx.Has(key) })
	return found
}

func (s *Store) GetValue(k []byte, to interface{}) {
	s.View(func(tx *Tx) {
		tx.MustGetValue(k, to)
	})
}

func (tx *Tx) Has(k []byte) bool {
	t := tx.Store.Registry.T
	found, err := tx.Tx.Has(k)
	must(t, err, "error testing for key presence")
	return found
}

func (tx *Tx) MustGet(k []byte) backend.ValueDecoder {
	dec, err := tx.Get(k)
	must(tx.Store.Registry.T, err, "unknown key")
	return dec
}

func (tx *Tx) GetValue(k []byte, to interface{}) error {
	dec, err := tx.Get(k)
	if err == nil && dec != nil {
		err = dec.Decode(to)
	}
	return err
}

func (tx *Tx) MustGetValue(k []byte, to interface{}) {
	t := tx.Store.Registry.T

	err := tx.GetValue(k, to)
	if err != nil {
		t.Fatalf("Failed to read key: %q", k)
	}
}

func (tx *Tx) MustSet(k []byte, v interface{}) {
	t := tx.Store.Registry.T
	must(t, tx.Tx.Set(k, v), "must set value")
}

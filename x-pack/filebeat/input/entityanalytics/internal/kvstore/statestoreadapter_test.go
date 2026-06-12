// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/entcollect"
)

func TestStateStoreAdapter_GetSet(t *testing.T) {
	store := newTestStore(t)
	a := NewStateStoreAdapter(store)

	if err := a.Set("k1", "hello"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got string
	if err := a.Get("k1", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "hello" {
		t.Errorf("Get = %q; want %q", got, "hello")
	}
}

func TestStateStoreAdapter_GetMissing(t *testing.T) {
	store := newTestStore(t)
	a := NewStateStoreAdapter(store)

	var got string
	err := a.Get("nonexistent", &got)
	if err == nil {
		t.Fatal("Get on missing key should return error")
	}
	if !errors.Is(err, entcollect.ErrKeyNotFound) {
		t.Fatalf("Get error = %v; want ErrKeyNotFound", err)
	}
}

func TestStateStoreAdapter_Delete(t *testing.T) {
	store := newTestStore(t)
	a := NewStateStoreAdapter(store)

	if err := a.Set("k1", "val"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := a.Delete("k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var got string
	err := a.Get("k1", &got)
	if !errors.Is(err, entcollect.ErrKeyNotFound) {
		t.Fatalf("Get after Delete = %v; want ErrKeyNotFound", err)
	}
}

func TestStateStoreAdapter_DeleteAbsent(t *testing.T) {
	store := newTestStore(t)
	a := NewStateStoreAdapter(store)

	if err := a.Delete("nonexistent"); err != nil {
		t.Fatalf("Delete of absent key should return nil; got %v", err)
	}
}

func TestStateStoreAdapter_Each(t *testing.T) {
	store := newTestStore(t)
	a := NewStateStoreAdapter(store)

	if err := a.Set("a", 1); err != nil {
		t.Fatalf("Set a: %v", err)
	}
	if err := a.Set("b", 2); err != nil {
		t.Fatalf("Set b: %v", err)
	}

	seen := map[string]int{}
	err := a.Each(func(key string, decode func(any) error) (bool, error) {
		var v int
		if err := decode(&v); err != nil {
			return false, err
		}
		seen[key] = v
		return true, nil
	})
	if err != nil {
		t.Fatalf("Each: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("Each visited %d keys; want 2", len(seen))
	}
	if seen["a"] != 1 || seen["b"] != 2 {
		t.Errorf("Each values = %v; want {a:1, b:2}", seen)
	}
}

func TestStateStoreAdapter_EachStopEarly(t *testing.T) {
	store := newTestStore(t)
	a := NewStateStoreAdapter(store)

	if err := a.Set("a", 1); err != nil {
		t.Fatalf("Set a: %v", err)
	}
	if err := a.Set("b", 2); err != nil {
		t.Fatalf("Set b: %v", err)
	}

	count := 0
	err := a.Each(func(key string, decode func(any) error) (bool, error) {
		count++
		return false, nil
	})
	if err != nil {
		t.Fatalf("Each: %v", err)
	}
	if count != 1 {
		t.Errorf("Each visited %d keys; want 1 (early stop)", count)
	}
}

// newTestStore creates a *statestore.Store backed by an in-memory map.
func newTestStore(t *testing.T) *statestore.Store {
	t.Helper()
	reg := statestore.NewRegistry(&testRegistry{stores: map[string]*testBackendStore{}})
	t.Cleanup(func() { reg.Close() })
	s, err := reg.Get("test")
	if err != nil {
		t.Fatalf("registry Get: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// testRegistry implements backend.Registry with in-memory stores.
type testRegistry struct {
	mu     sync.Mutex
	stores map[string]*testBackendStore
}

func (r *testRegistry) Access(name string) (backend.Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.stores[name]; ok {
		return s, nil
	}
	s := &testBackendStore{data: map[string][]byte{}}
	r.stores[name] = s
	return s, nil
}

func (r *testRegistry) Close() error { return nil }

// testBackendStore is an in-memory backend.Store.
type testBackendStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func (s *testBackendStore) Close() error { return nil }

func (s *testBackendStore) Has(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok, nil
}

func (s *testBackendStore) Get(key string, value interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw, ok := s.data[key]
	if !ok {
		return errors.New("key unknown")
	}
	return json.Unmarshal(raw, value)
}

func (s *testBackendStore) Set(key string, value interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = raw
	return nil
}

func (s *testBackendStore) Remove(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *testBackendStore) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.data {
		dec := &jsonDecoder{raw: v}
		cont, err := fn(k, dec)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}
	return nil
}

func (s *testBackendStore) SetID(_ string) {}

type jsonDecoder struct {
	raw []byte
}

func (d *jsonDecoder) Decode(to interface{}) error {
	return json.Unmarshal(d.raw, to)
}

// testBackendStoreStrict is like testBackendStore but returns an error
// from Remove when the key doesn't exist, mimicking the real ES
// backend's behaviour (HTTP 404 → error instead of silent success).
type testBackendStoreStrict struct {
	testBackendStore
}

func (s *testBackendStoreStrict) Remove(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return fmt.Errorf("404 Not Found: {\"found\":false,\"_id\":\"%s\"}", key)
	}
	delete(s.data, key)
	return nil
}

// newTestStoreStrict creates a *statestore.Store backed by a strict
// in-memory backend that errors on Remove for missing keys.
func newTestStoreStrict(t *testing.T) *statestore.Store {
	t.Helper()
	strict := &testBackendStoreStrict{testBackendStore{data: map[string][]byte{}}}
	reg := statestore.NewRegistry(&strictRegistry{store: strict})
	t.Cleanup(func() { reg.Close() })
	s, err := reg.Get("test")
	if err != nil {
		t.Fatalf("registry Get: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

type strictRegistry struct {
	store *testBackendStoreStrict
}

func (r *strictRegistry) Access(_ string) (backend.Store, error) {
	return r.store, nil
}

func (r *strictRegistry) Close() error { return nil }

func TestStateStoreAdapter_DeleteAbsent_StrictBackend(t *testing.T) {
	store := newTestStoreStrict(t)
	a := NewStateStoreAdapter(store)

	if err := a.Delete("nonexistent"); err != nil {
		t.Fatalf("Delete should swallow missing-key error from strict backend; got %v", err)
	}
}

func TestStateStoreAdapter_DeletePresent_StrictBackend(t *testing.T) {
	store := newTestStoreStrict(t)
	a := NewStateStoreAdapter(store)

	if err := a.Set("k1", "val"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := a.Delete("k1"); err != nil {
		t.Fatalf("Delete of present key should succeed; got %v", err)
	}
	var got string
	err := a.Get("k1", &got)
	if !errors.Is(err, entcollect.ErrKeyNotFound) {
		t.Fatalf("Get after Delete = %v; want ErrKeyNotFound", err)
	}
}

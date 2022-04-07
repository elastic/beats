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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/statestore/storetest"
)

func TestStore_Close(t *testing.T) {
	t.Run("close succeeds", func(t *testing.T) {
		makeClosedTestStore(t)
	})
	t.Run("fails if store has been closed", func(t *testing.T) {
		assert.Error(t, makeClosedTestStore(t).Close())
	})
}

func TestStore_Has(t *testing.T) {
	t.Run("fails if store has been closed", func(t *testing.T) {
		store := makeClosedTestStore(t)
		_, err := store.Has("test")
		assertClosed(t, err)
	})
	t.Run("error is passed through", func(t *testing.T) {
		ms := newMockStore()
		ms.OnHas("test").Return(false, errors.New("oops"))
		defer ms.AssertExpectations(t)

		store := makeTestMockedStore(t, ms)
		defer store.Close()

		_, err := store.Has("test")
		assert.Error(t, err)
	})
	t.Run("return result from backend", func(t *testing.T) {
		data := map[string]interface{}{"known_key": "test"}
		store := makeTestStore(t, data)
		defer store.Close()

		got, err := store.Has("known_key")
		assert.NoError(t, err)
		assert.True(t, got)

		got, err = store.Has("unknown_key")
		assert.NoError(t, err)
		assert.False(t, got)
	})
}

func TestStore_Get(t *testing.T) {
	t.Run("fails if store has been closed", func(t *testing.T) {
		store := makeClosedTestStore(t)
		var tmp interface{}
		assertClosed(t, store.Get("test", &tmp))
	})
	t.Run("error is passed through", func(t *testing.T) {
		ms := newMockStore()
		defer ms.AssertExpectations(t)

		store := makeTestMockedStore(t, ms)
		defer store.Close()

		ms.OnGet("test").Return(errors.New("oops"))
		var tmp interface{}
		err := store.Get("test", &tmp)
		assert.Error(t, err)
	})
	t.Run("return result from backend", func(t *testing.T) {
		data := map[string]interface{}{"known_key": "test"}
		store := makeTestStore(t, data)
		defer store.Close()

		var got interface{}
		err := store.Get("known_key", &got)
		assert.NoError(t, err)
		assert.Equal(t, "test", got)
	})
}

func TestStore_Set(t *testing.T) {
	t.Run("fails if store has been closed", func(t *testing.T) {
		store := makeClosedTestStore(t)
		var tmp interface{}
		assertClosed(t, store.Set("test", &tmp))
	})
	t.Run("error is passed through", func(t *testing.T) {
		ms := newMockStore()
		defer ms.AssertExpectations(t)

		store := makeTestMockedStore(t, ms)
		defer store.Close()

		ms.OnSet("test").Return(errors.New("oops"))
		err := store.Set("test", nil)
		assert.Error(t, err)
	})
	t.Run("set key in backend", func(t *testing.T) {
		data := map[string]interface{}{}
		store := makeTestStore(t, data)
		defer store.Close()

		err := store.Set("key", "value")
		assert.NoError(t, err)
		assert.Equal(t, "value", data["key"])
	})
}

func TestStore_Remove(t *testing.T) {
	t.Run("fails if store has been closed", func(t *testing.T) {
		store := makeClosedTestStore(t)
		assertClosed(t, store.Remove("test"))
	})
	t.Run("error is passed through", func(t *testing.T) {
		ms := newMockStore()
		ms.OnRemove("test").Return(errors.New("oops"))
		defer ms.AssertExpectations(t)

		store := makeTestMockedStore(t, ms)
		defer store.Close()

		assert.Error(t, store.Remove("test"))
	})
	t.Run("remove key from backend", func(t *testing.T) {
		data := map[string]interface{}{"key": "test"}
		store := makeTestStore(t, data)

		err := store.Remove("key")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(data))
	})
}

func TestStore_Each(t *testing.T) {
	t.Run("fails if store has been closed", func(t *testing.T) {
		store := makeClosedTestStore(t)
		assertClosed(t, store.Each(func(string, ValueDecoder) (bool, error) {
			return true, nil
		}))
	})
	t.Run("correctly iterate pairs", func(t *testing.T) {
		data := map[string]interface{}{
			"a": map[string]interface{}{"field": "hello"},
			"b": map[string]interface{}{"field": "test"},
		}
		store := makeTestStore(t, data)
		defer store.Close()

		got := map[string]interface{}{}
		err := store.Each(func(key string, dec ValueDecoder) (bool, error) {
			var tmp interface{}
			if err := dec.Decode(&tmp); err != nil {
				t.Fatalf("failed to read value from store: %v", err)
			}
			got[key] = tmp
			return true, nil
		})

		assert.NoError(t, err)
		assert.Equal(t, data, got)
	})
}

func makeTestStore(t *testing.T, data map[string]interface{}) *Store {
	memstore := &storetest.MapStore{Table: data}
	reg := NewRegistry(&storetest.MemoryStore{
		Stores: map[string]*storetest.MapStore{
			"test": memstore,
		},
	})
	store, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	return store
}

func makeTestMockedStore(t *testing.T, ms *mockStore) *Store {
	mr := newMockRegistry()
	mr.OnAccess("test").Once().Return(ms, nil)

	reg := NewRegistry(mr)
	s, err := reg.Get("test")
	require.NoError(t, err)

	ms.OnClose().Return(nil)
	return s
}

func makeClosedTestStore(t *testing.T) *Store {
	s := makeTestMockedStore(t, newMockStore())
	require.NoError(t, s.Close())
	return s
}

func assertClosed(t *testing.T, err error) {
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsClosed(err) {
		t.Fatalf("The error does not seem to indicate a failure because of a closed store. Error: %v", err)
	}
}

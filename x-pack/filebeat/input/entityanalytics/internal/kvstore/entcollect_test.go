// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"errors"
	"testing"

	"github.com/elastic/entcollect"
)

const testEntcollectBucket = "entcollect.test"

func TestEntcollectStore_SetGet(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_SetGet.db")
	t.Cleanup(func() { testCleanupStore(store) })

	type obj struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	err := store.RunTransaction(true, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)

		if err := es.Set("key1", &obj{Name: "alice", N: 42}); err != nil {
			return err
		}

		var got obj
		if err := es.Get("key1", &got); err != nil {
			return err
		}
		want := obj{Name: "alice", N: 42}
		if got != want {
			t.Errorf("Get(key1) = %+v; want %+v", got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunTransaction: %v", err)
	}
}

func TestEntcollectStore_GetMissing(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_GetMissing.db")
	t.Cleanup(func() { testCleanupStore(store) })

	err := store.RunTransaction(false, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		var got string
		return es.Get("no-such-key", &got)
	})
	if !errors.Is(err, entcollect.ErrKeyNotFound) {
		t.Fatalf("Get(missing) = %v; want ErrKeyNotFound", err)
	}
}

func TestEntcollectStore_Delete(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_Delete.db")
	t.Cleanup(func() { testCleanupStore(store) })

	err := store.RunTransaction(true, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)

		if err := es.Set("key1", "value1"); err != nil {
			return err
		}
		if err := es.Delete("key1"); err != nil {
			return err
		}
		var got string
		err := es.Get("key1", &got)
		if !errors.Is(err, entcollect.ErrKeyNotFound) {
			t.Errorf("Get after Delete = %v; want ErrKeyNotFound", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunTransaction: %v", err)
	}
}

func TestEntcollectStore_DeleteMissing(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_DeleteMissing.db")
	t.Cleanup(func() { testCleanupStore(store) })

	err := store.RunTransaction(true, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		return es.Delete("no-such-key")
	})
	if err != nil {
		t.Fatalf("Delete(missing) = %v; want nil", err)
	}
}

func TestEntcollectStore_Each(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_Each.db")
	t.Cleanup(func() { testCleanupStore(store) })

	err := store.RunTransaction(true, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		for _, k := range []string{"a", "b", "c"} {
			if err := es.Set(k, k+"-val"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	err = store.RunTransaction(false, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		got := map[string]string{}
		return es.Each(func(key string, decode func(any) error) (bool, error) {
			var val string
			if err := decode(&val); err != nil {
				return false, err
			}
			got[key] = val
			return true, nil
		})
	})
	if err != nil {
		t.Fatalf("Each: %v", err)
	}
}

func TestEntcollectStore_EachEmpty(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_EachEmpty.db")
	t.Cleanup(func() { testCleanupStore(store) })

	err := store.RunTransaction(false, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		called := false
		err := es.Each(func(key string, decode func(any) error) (bool, error) {
			called = true
			return true, nil
		})
		if called {
			t.Error("Each callback was called on empty bucket")
		}
		return err
	})
	if err != nil {
		t.Fatalf("Each(empty): %v", err)
	}
}

func TestEntcollectStore_EachStopEarly(t *testing.T) {
	t.Parallel()

	store := testSetupStore(t, "TestEntcollectStore_EachStopEarly.db")
	t.Cleanup(func() { testCleanupStore(store) })

	err := store.RunTransaction(true, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		for _, k := range []string{"a", "b", "c"} {
			if err := es.Set(k, k); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	err = store.RunTransaction(false, func(tx *Transaction) error {
		es := NewEntcollectStore(tx, testEntcollectBucket)
		count := 0
		return es.Each(func(key string, decode func(any) error) (bool, error) {
			count++
			return count < 2, nil
		})
	})
	if err != nil {
		t.Fatalf("Each(stop early): %v", err)
	}
}

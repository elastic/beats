// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	testBucket = []byte("test-bucket")
	testKey    = []byte("test-key")
	testValue  = []byte("test-value")
)

func testSetupStore(t *testing.T, filename string) *Store {
	store, err := NewStore(logp.L(), filename, 0644)
	require.NoError(t, err)

	return store
}

func testCleanupStore(store *Store) {
	filename := store.db.Path()
	_ = store.Close()
	_ = os.Remove(filename)
}

func testAssertValueEquals(t *testing.T, store *Store, bucket, key, value []byte) {
	var gotValue []byte

	err := store.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return ErrBucketNotFound
		}

		gotValue = b.Get(key)
		if gotValue == nil {
			return ErrKeyNotFound
		}

		return nil
	})
	require.NoError(t, err)
	require.Equal(t, value, gotValue)
}

func testAssertJSONValueEquals(t *testing.T, store *Store, bucket, key []byte, value any) {
	valueData, err := json.Marshal(&value)
	require.NoError(t, err)

	testAssertValueEquals(t, store, bucket, key, valueData)
}

func testAssertValueNil(t *testing.T, store *Store, bucket, key []byte) {
	var gotValue []byte

	err := store.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}

		gotValue = b.Get(key)

		return nil
	})

	require.NoError(t, err)
	require.Nil(t, gotValue)
}

func testStoreSetBucket(t *testing.T, store *Store, bucket []byte) {
	err := store.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})

	require.NoError(t, err)
}

func testStoreSetValue(t *testing.T, store *Store, bucket, key, value []byte) {
	err := store.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}

		return b.Put(key, value)
	})

	require.NoError(t, err)
}

func testStoreSetJSONValue(t *testing.T, store *Store, bucket, key []byte, value any) {
	valueData, err := json.Marshal(&value)
	require.NoError(t, err)

	testStoreSetValue(t, store, bucket, key, valueData)
}

func TestStore_RunTransaction(t *testing.T) {
	t.Run("run-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore(t, "TestStore_RunTransaction_run-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.SetBytes(testBucket, testKey, testValue)
		})
		require.NoError(t, err)

		testAssertValueEquals(t, store, testBucket, testKey, testValue)
	})

	t.Run("run-err", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore(t, "TestStore_RunTransaction_run-err.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			err := tx.SetBytes(testBucket, testKey, testValue)
			require.NoError(t, err)

			return errors.New("test error")
		})

		require.ErrorContains(t, err, "test error")
		testAssertValueNil(t, store, testBucket, testKey)
	})

	t.Run("run-panic", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore(t, "TestStore_RunTransaction_run-panic.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		require.Panics(t, func() {
			_ = store.RunTransaction(true, func(tx *Transaction) error {
				err := tx.SetBytes(testBucket, testKey, testValue)
				require.NoError(t, err)

				panic("test panic")
			})
		})
	})

	t.Run("run-panic-err", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore(t, "TestStore_RunTransaction_run-panic-err.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		require.Panics(t, func() {
			_ = store.RunTransaction(true, func(tx *Transaction) error {
				err := tx.SetBytes(testBucket, testKey, testValue)
				require.NoError(t, err)

				panic(errors.New("test panic-err"))
			})
		})
	})
}

func TestStore_BeginTx(t *testing.T) {
	t.Run("begin-writable", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore(t, "TestStore_BeginTx_begin-writable.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		tx, err := store.BeginTx(true)
		require.NoError(t, err)

		err = tx.SetBytes(testBucket, testKey, testValue)
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		testAssertValueEquals(t, store, testBucket, testKey, testValue)
	})

	t.Run("begin-readonly", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore(t, "TestStore_BeginTx_begin-readonly.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		tx, err := store.BeginTx(false)
		require.NoError(t, err)

		err = tx.SetBytes(testBucket, testKey, testValue)
		require.ErrorIs(t, err, bbolt.ErrTxNotWritable)

		err = tx.Rollback()
		require.NoError(t, err)

		testAssertValueNil(t, store, testBucket, testKey)
	})
}

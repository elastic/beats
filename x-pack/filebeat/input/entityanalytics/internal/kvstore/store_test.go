// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	testBucket = []byte("test-bucket")
	testKey    = []byte("test-key")
	testValue  = []byte("test-value")
)

func testSetupStore(filename string) *Store {
	store, err := NewStore(logp.L(), filename, 0644)
	if err != nil {
		panic(err)
	}

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
	assert.NoError(t, err)
	assert.Equal(t, value, gotValue)
}

func testAssertJSONValueEquals(t *testing.T, store *Store, bucket, key []byte, value any) {
	valueData, err := json.Marshal(&value)
	assert.NoError(t, err)

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

	assert.NoError(t, err)
	assert.Nil(t, gotValue)
}

func testStoreSetBucket(t *testing.T, store *Store, bucket []byte) {
	err := store.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})

	assert.NoError(t, err)
}

func testStoreSetValue(t *testing.T, store *Store, bucket, key, value []byte) {
	err := store.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}

		return b.Put(key, value)
	})

	assert.NoError(t, err)
}

func testStoreSetJSONValue(t *testing.T, store *Store, bucket, key []byte, value any) {
	valueData, err := json.Marshal(&value)
	assert.NoError(t, err)

	testStoreSetValue(t, store, bucket, key, valueData)
}

func TestStore_RunTransaction(t *testing.T) {
	t.Run("run-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestStore_RunTransaction_run-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.SetBytes(testBucket, testKey, testValue)
		})
		assert.NoError(t, err)

		testAssertValueEquals(t, store, testBucket, testKey, testValue)
	})

	t.Run("run-err", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestStore_RunTransaction_run-err.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			err := tx.SetBytes(testBucket, testKey, testValue)
			assert.NoError(t, err)

			return errors.New("test error")
		})

		assert.ErrorContains(t, err, "test error")
		testAssertValueNil(t, store, testBucket, testKey)
	})

	t.Run("run-panic", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestStore_RunTransaction_run-panic.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		assert.Panics(t, func() {
			_ = store.RunTransaction(true, func(tx *Transaction) error {
				err := tx.SetBytes(testBucket, testKey, testValue)
				assert.NoError(t, err)

				panic("test panic")
			})
		})
	})

	t.Run("run-panic-err", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestStore_RunTransaction_run-panic-err.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		assert.Panics(t, func() {
			_ = store.RunTransaction(true, func(tx *Transaction) error {
				err := tx.SetBytes(testBucket, testKey, testValue)
				assert.NoError(t, err)

				panic(errors.New("test panic-err"))
			})
		})
	})
}

func TestStore_BeginTx(t *testing.T) {
	t.Run("begin-writable", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestStore_BeginTx_begin-writable.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		tx, err := store.BeginTx(true)
		assert.NoError(t, err)

		err = tx.SetBytes(testBucket, testKey, testValue)
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		testAssertValueEquals(t, store, testBucket, testKey, testValue)
	})

	t.Run("begin-readonly", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestStore_BeginTx_begin-readonly.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		tx, err := store.BeginTx(false)
		assert.NoError(t, err)

		err = tx.SetBytes(testBucket, testKey, testValue)
		assert.ErrorIs(t, err, bbolt.ErrTxNotWritable)

		err = tx.Rollback()
		assert.NoError(t, err)

		testAssertValueNil(t, store, testBucket, testKey)
	})
}

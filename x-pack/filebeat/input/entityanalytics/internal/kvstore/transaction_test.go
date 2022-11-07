// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"testing"

	"go.etcd.io/bbolt"

	"github.com/stretchr/testify/assert"
)

func TestTransaction_GetBytes(t *testing.T) {
	t.Run("get-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_GetBytes-get-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testStoreSetValue(t, store, testBucket, testKey, testValue)

		err := store.RunTransaction(false, func(tx *Transaction) error {
			gotValue, err := tx.GetBytes(testBucket, testKey)
			if err != nil {
				return err
			}
			assert.Equal(t, testValue, gotValue)

			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("get-err-bucket", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_GetBytes-get-err-bucket.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(false, func(tx *Transaction) error {
			gotValue, err := tx.GetBytes(testBucket, testKey)
			assert.Nil(t, gotValue)

			return err
		})

		assert.ErrorIs(t, err, ErrBucketNotFound)
	})

	t.Run("get-err-key", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_GetBytes-get-err-key.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testStoreSetBucket(t, store, testBucket)

		err := store.RunTransaction(false, func(tx *Transaction) error {
			gotValue, err := tx.GetBytes(testBucket, testKey)
			assert.Nil(t, gotValue)

			return err
		})

		assert.ErrorIs(t, err, ErrKeyNotFound)
	})
}

func TestTransaction_Get(t *testing.T) {
	type testObject struct {
		A int `json:"a"`
	}

	t.Run("get-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Get-get-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testObjectValue := testObject{A: 1234}
		testStoreSetJSONValue(t, store, testBucket, testKey, &testObjectValue)

		var gotObject testObject
		err := store.RunTransaction(false, func(tx *Transaction) error {
			return tx.Get(testBucket, testKey, &gotObject)
		})

		assert.NoError(t, err)
		assert.Equal(t, testObjectValue, gotObject)
	})

	t.Run("get-err-key", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_GetBytes-get-err-key.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testStoreSetBucket(t, store, testBucket)

		var gotObject testObject
		err := store.RunTransaction(false, func(tx *Transaction) error {
			return tx.Get(testBucket, testKey, &gotObject)
		})

		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("get-err-json", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Get-get-err-json.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testStoreSetValue(t, store, testBucket, testKey, testValue)

		var gotObject testObject
		err := store.RunTransaction(false, func(tx *Transaction) error {
			return tx.Get(testBucket, testKey, &gotObject)
		})

		assert.ErrorContains(t, err, "kvstore: json unmarshal error:")
	})
}

func TestTransaction_ForEach(t *testing.T) {
	t.Run("foreach-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_ForEach_foreach-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		wantKeys := [][]byte{[]byte("A"), []byte("B"), []byte("C")}
		wantValues := [][]byte{testValue, testValue, testValue}
		for i := range wantKeys {
			testStoreSetValue(t, store, testBucket, wantKeys[i], wantValues[i])
		}

		var gotKeys [][]byte
		var gotValues [][]byte
		err := store.RunTransaction(false, func(tx *Transaction) error {
			err := tx.ForEach(testBucket, func(key, value []byte) error {
				gotKeys = append(gotKeys, key)
				gotValues = append(gotValues, value)

				return nil
			})

			return err
		})

		assert.NoError(t, err)
		assert.Equal(t, wantKeys, gotKeys)
		assert.Equal(t, wantValues, gotValues)
	})

	t.Run("foreach-err-bucket", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_ForEach_foreach-err-bucket.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		var gotKeys [][]byte
		var gotValues [][]byte
		err := store.RunTransaction(false, func(tx *Transaction) error {
			err := tx.ForEach(testBucket, func(key, value []byte) error {
				gotKeys = append(gotKeys, key)
				gotValues = append(gotValues, value)

				return nil
			})

			return err
		})

		assert.ErrorIs(t, err, ErrBucketNotFound)
		assert.Nil(t, gotKeys)
		assert.Nil(t, gotValues)
	})

}

func TestTransaction_SetBytes(t *testing.T) {
	t.Run("set-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_SetBytes-set-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.SetBytes(testBucket, testKey, testValue)
		})
		assert.NoError(t, err)

		testAssertValueEquals(t, store, testBucket, testKey, testValue)
	})

	t.Run("set-err-empty-bucket", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_SetBytes-set-err-empty-bucket.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.SetBytes([]byte(""), testKey, testValue)
		})
		assert.ErrorIs(t, err, bbolt.ErrBucketNameRequired)
	})

	t.Run("set-err-empty-key", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_SetBytes-set-err-empty-key.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.SetBytes(testBucket, []byte(""), testValue)
		})
		assert.ErrorIs(t, err, bbolt.ErrKeyRequired)
	})
}

func TestTransaction_Set(t *testing.T) {
	type testObject struct {
		A int `json:"a"`
	}

	t.Run("set-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Set-set-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testObjectValue := testObject{A: 1234}
		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.Set(testBucket, testKey, &testObjectValue)
		})
		assert.NoError(t, err)

		testAssertJSONValueEquals(t, store, testBucket, testKey, &testObjectValue)
	})
}

func TestTransaction_Delete(t *testing.T) {
	t.Run("delete-ok", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Delete-delete-ok.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testStoreSetValue(t, store, testBucket, testKey, testValue)

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.Delete(testBucket, testKey)
		})
		assert.NoError(t, err)

		testAssertValueNil(t, store, testBucket, testKey)
	})

	t.Run("delete-no-bucket", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Delete-delete-no-bucket.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		err := store.RunTransaction(true, func(tx *Transaction) error {
			return tx.Delete(testBucket, testKey)
		})
		assert.NoError(t, err)
	})

	t.Run("delete-err", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Delete-delete-err.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		testStoreSetValue(t, store, testBucket, testKey, testValue)

		err := store.RunTransaction(false, func(tx *Transaction) error {
			return tx.Delete(testBucket, testKey)
		})

		assert.ErrorIs(t, err, bbolt.ErrTxNotWritable)
		testAssertValueEquals(t, store, testBucket, testKey, testValue)

	})
}

func TestTransaction_Commit(t *testing.T) {
	t.Run("commit-writable", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Commit-commit-writable.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		tx, err := store.BeginTx(true)
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.True(t, tx.closed.Load())

		// Verify that multiple calls to Rollback don't fail.
		err = tx.Commit()
		assert.NoError(t, err)
	})

	t.Run("commit-readonly", func(t *testing.T) {
		t.Parallel()

		store := testSetupStore("TestTransaction_Rollback-commit-readonly.db")
		t.Cleanup(func() {
			testCleanupStore(store)
		})

		tx, err := store.BeginTx(false)
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.True(t, tx.closed.Load())

		// Verify that multiple calls to Rollback don't fail.
		err = tx.Commit()
		assert.NoError(t, err)
	})
}

func TestTransaction_Rollback(t *testing.T) {
	store := testSetupStore("TestTransaction_Rollback.db")
	t.Cleanup(func() {
		testCleanupStore(store)
	})

	// Verify that multiple calls to Rollback don't fail.
	tx, err := store.BeginTx(false)
	assert.NoError(t, err)

	err = tx.Rollback()
	assert.NoError(t, err)

	err = tx.Rollback()
	assert.NoError(t, err)
}

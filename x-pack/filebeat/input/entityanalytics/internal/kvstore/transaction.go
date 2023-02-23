// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

var (
	// ErrBucketNotFound is an error indicating a bucket was not found.
	ErrBucketNotFound = errors.New("kvstore: bucket not found")
	// ErrKeyNotFound is an error indicating a key was not found.
	ErrKeyNotFound = errors.New("kvstore: key not found")
)

// Transaction represents a transaction on the key/value store. In the case of
// a write transaction, any changes made are held in memory. Only on a commit
// will changes be written to disk. There can only be one write transaction open
// at any given time. Subsequent transactions are blocked until the current
// transaction is closed. There can be any number of read-only transactions open
// at any given time.
type Transaction struct {
	tx        *bbolt.Tx
	closed    atomic.Bool
	writeable bool
}

// GetBytes returns the bytes found at key in bucket. If the bucket or key are
// not present in the database, then an error is returned.
func (t *Transaction) GetBytes(bucket, key []byte) ([]byte, error) {
	var b *bbolt.Bucket
	var value []byte

	if b = t.tx.Bucket(bucket); b == nil {
		return nil, ErrBucketNotFound
	}
	if value = b.Get(key); value == nil {
		return nil, ErrKeyNotFound
	}

	return value, nil
}

// Get will get the data at key in bucket and will attempt to decode the data
// into value. The decoding method is JSON.
func (t *Transaction) Get(bucket, key []byte, value any) error {
	data, err := t.GetBytes(bucket, key)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("kvstore: json unmarshal error: %w", err)
	}

	return err
}

// ForEach executes a function for each key/value pair in a bucket.
// If the provided function returns an error then the iteration is stopped and
// the error is returned to the caller. The provided function must not modify
// the bucket; this will result in undefined behavior.
func (t *Transaction) ForEach(bucket []byte, fn func(key, value []byte) error) error {
	var b *bbolt.Bucket

	if b = t.tx.Bucket(bucket); b == nil {
		return ErrBucketNotFound
	}

	return b.ForEach(func(k, v []byte) error {
		return fn(k, v)
	})
}

// SetBytes will set the value of key in bucket. If the key or bucket do not
// exist, they will be automatically created.
func (t *Transaction) SetBytes(bucket, key, value []byte) error {
	var b *bbolt.Bucket
	var err error

	if b, err = t.tx.CreateBucketIfNotExists(bucket); err != nil {
		return fmt.Errorf("kvstore: create/get bucket: %w", err)
	}
	if err = b.Put(key, value); err != nil {
		return fmt.Errorf("kvstore: put value: %w", err)
	}

	return nil
}

// Set will set the data at key in bucket using the encoded representation of
// value. The encoding method is JSON.
func (t *Transaction) Set(bucket, key []byte, value any) error {
	data, err := json.Marshal(&value)
	if err != nil {
		return fmt.Errorf("kvstore: json marshal error: %w", err)
	}

	return t.SetBytes(bucket, key, data)
}

// Delete will delete the value at key in bucket. If the bucket or key do not
// exist, this function will be a no-op.
func (t *Transaction) Delete(bucket, key []byte) error {
	var b *bbolt.Bucket
	var err error

	if b = t.tx.Bucket(bucket); b == nil {
		return nil
	}
	if err = b.Delete(key); err != nil {
		return fmt.Errorf("kvstore: delete key: %w", err)
	}

	return nil
}

// Commit will write any changes to disk. For read-only transactions, calling
// Commit will route to Rollback.
func (t *Transaction) Commit() error {
	if !t.closed.CAS(false, true) {
		return nil
	}
	if !t.writeable {
		return t.tx.Rollback()
	}
	return t.tx.Commit()
}

// Rollback closes the transaction. For write transactions, all updates made
// within the transaction will be discarded.
func (t *Transaction) Rollback() error {
	if !t.closed.CAS(false, true) {
		return nil
	}
	return t.tx.Rollback()
}

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

package registry

import (
	"github.com/elastic/beats/libbeat/registry/backend"
)

// Tx provides transactional access to a store. Tx objects provide support for
// syncing and isolating reads and writes to a store. A Tx object itself is not
// thread-safe and should not be used concurrently from multiple go-routines.
// A Tx object should not be kept alive for too long, so to guarantee other
// transaction will not be blocked for too long. The backend implements the
// actual locking strategies.
type Tx struct {
	store    *Store
	active   bool
	readonly bool
	backend  tx

	gen *idGen
}

type tx interface {
	backend.Tx
}

func newTx(store *Store, backend backend.Tx, readonly bool) *Tx {
	return &Tx{store: store, active: true, readonly: readonly, backend: backend}
}

func (tx *Tx) close() error {
	if tx.active {
		if tx.gen != nil {
			tx.gen.close()
		}

		err := tx.backend.Close()
		tx.store.finishTx(tx)

		tx.active = false

		return err
	}
	return nil
}

// Close closes the transaction. The transaction object can not be used afterwards.
// A writeable transaction not rolled back or committed yet will be automatically rolled back on Close.
// Close can be called multiple times. All extra close operations will have no effect.
func (tx *Tx) Close() (err error) {
	defer func() {
		closeErr := tx.close()
		if err == nil {
			err = closeErr
		}
	}()

	if tx.active && !tx.readonly {
		err = tx.backend.Rollback()
	}
	return
}

// Rollback undoes all changes and closes the current transaction.
func (tx *Tx) Rollback() error {
	if !tx.active {
		return errTxClosed
	}
	return tx.Close()
}

// Commit applies all local changes and closes the current transaction.
func (tx *Tx) Commit() error {
	defer tx.close()

	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Commit()
}

// Has returns true if the key is present in the store.
func (tx *Tx) Has(key Key) (bool, error) {
	if !tx.active {
		return false, errTxClosed
	}
	return tx.backend.Has(backend.Key(key))
}

// Get returns a value decoder if the key is present in the store.
// The value decoder can be used to unpack the value into a custom go structure
// or map. The value decoder is alive and can be used multiple times, as long
// as the owning transaction has not been closed yet.
func (tx *Tx) Get(key Key) (ValueDecoder, error) {
	if !tx.active {
		return nil, errTxClosed
	}
	return tx.backend.Get(backend.Key(key))
}

// Insert inserts a new key-value pair into the store. The key will be
// generated and returned on success.
// The value should be a map or a go struct. During serialization all fields
// found in val will be added to the inserted document.
func (tx *Tx) Insert(val interface{}) (Key, error) {
	if tx.gen == nil {
		tx.gen = newIDGen()
	}

	var key Key
	for {
		key = tx.gen.Make()
		exists, err := tx.backend.Has(backend.Key(key))
		if err != nil {
			return nil, err
		}

		if !exists {
			break
		}
	}

	err := tx.backend.Set(backend.Key(key), val)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Remove removes a key-value pair from the store.
func (tx *Tx) Remove(key Key) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Remove(backend.Key(key))
}

// Set inserts a known key-value pair into the store.
// The value should be a map or a go struct. During serialization all fields
// found in val will be added to the inserted document.
func (tx *Tx) Set(key Key, val interface{}) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Set(backend.Key(key), val)
}

// Update updates/adds the given fields in a key-value pair.
// If the key is not known to a store, a new empty document is before updating.
// The value should be a map or a go struct. During serialization all fields
// will be added to the document.
func (tx *Tx) Update(key Key, fields interface{}) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Update(backend.Key(key), fields)
}

// EachKey iterates all entries in a store, calling fn for each found key.
func (tx *Tx) EachKey(fn func(Key) (bool, error)) error {
	if !tx.active {
		return errTxClosed
	}

	return tx.backend.EachKey(false, func(k backend.Key) (bool, error) {
		return fn(Key(k))
	})
}

// Each iterates all entries in a store, calling fn for each found key-value pair.
func (tx *Tx) Each(fn func(Key, ValueDecoder) (bool, error)) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Each(false, func(k backend.Key, v backend.ValueDecoder) (bool, error) {
		return fn(Key(k), v)
	})
}

func (tx *Tx) internalEach(fn func(Key, ValueDecoder) (bool, error)) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Each(true, func(k backend.Key, v backend.ValueDecoder) (bool, error) {
		return fn(Key(k), v)
	})
}

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

func (tx *Tx) Rollback() error {
	if !tx.active {
		return errTxClosed
	}
	return tx.Close()
}

func (tx *Tx) Commit() error {
	defer tx.close()

	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Commit()
}

func (tx *Tx) Has(key Key) (bool, error) {
	if !tx.active {
		return false, errTxClosed
	}
	return tx.backend.Has(backend.Key(key))
}

func (tx *Tx) Get(key Key) (ValueDecoder, error) {
	if !tx.active {
		return nil, errTxClosed
	}
	return tx.backend.Get(backend.Key(key))
}

func (tx *Tx) Insert(val interface{}) (Key, error) {
	if tx.gen == nil {
		tx.gen = newIDGen()
	}

	key := tx.gen.Make()
	err := tx.backend.Set(backend.Key(key), val)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (tx *Tx) Remove(key Key) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Remove(backend.Key(key))
}

func (tx *Tx) Set(key Key, val interface{}) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Set(backend.Key(key), val)
}

func (tx *Tx) Update(key Key, fields interface{}) error {
	if !tx.active {
		return errTxClosed
	}
	return tx.backend.Update(backend.Key(key), fields)
}

func (tx *Tx) EachKey(fn func(Key) (bool, error)) error {
	if !tx.active {
		return errTxClosed
	}

	return tx.backend.EachKey(false, func(k backend.Key) (bool, error) {
		return fn(Key(k))
	})
}

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

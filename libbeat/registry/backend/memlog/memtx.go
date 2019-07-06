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

package memlog

import "github.com/elastic/beats/libbeat/common"

type memTx struct {
	parent memTxParent

	store  *memStore
	hashFn hashFn
	state  *txState
}

type memTxParent interface {
	IsReadonly() bool
	checkRead() error
	checkWrite() error
	getTypeConv() *typeConv
}

func (tx *memTx) init(parent memTxParent, store *memStore, state *txState) {
	*tx = memTx{
		parent: parent,
		store:  store,
		state:  state,
		hashFn: newHashFn(),
	}
}

func (tx *memTx) Rollback() {}

func (tx *memTx) Commit() {
	tx.CommitTo(&tx.store.tbl, false)
}

// CommitTo applies all changes to the given hashtable. If cpy is set, update
// operations that must modify the current value, will copy the original entry
// in the hashtable first.
func (tx *memTx) CommitTo(tbl *hashtable, cpy bool) {
	for hash, line := range tx.state.bins {
		bin := tbl.bins[hash]
		modified := false

		for i := range line {
			st := &line[i]
			if !st.modified {
				continue
			}

			modified = true
			switch {
			// apply remove ops
			case !st.exists:
				if idx := bin.index(st.key); idx >= 0 {
					bin.remove(idx)
				}

			// apply insert/set ops
			case st.value == nil && st.updates != nil:
				if idx := bin.index(st.key); idx >= 0 {
					bin[idx].value = st.updates
				} else {
					bin = append(bin, entry{
						key:   st.key,
						value: st.updates,
					})
				}

			// apply update ops
			case st.value != nil && st.updates != nil:
				value := st.value
				if cpy {
					value = value.Clone()
				}
				value.DeepUpdate(st.updates)

				if idx := bin.index(st.key); idx >= 0 {
					bin[idx].value = value
				} else {
					bin = append(bin, entry{
						key:   st.key,
						value: value,
					})
				}
			}
		}

		if modified {
			if len(bin) == 0 {
				delete(tbl.bins, hash)
			} else {
				tbl.bins[hash] = bin
			}
		}
	}
}

func (tx *memTx) Has(k []byte) bool {
	key := tx.makeKeyPair(k)
	if ref := tx.state.find(key); !ref.IsNil() {
		return ref.Access().exists
	}
	return tx.store.has(key)
}

func (tx *memTx) Get(k []byte) *txCacheEntry {
	key := tx.makeKeyPair(k)

	// try to read value from tx cache
	txRef := tx.state.find(key)
	if !txRef.IsNil() {
		entry := txRef.Access()
		if !entry.exists {
			return nil
		}
		return entry
	}

	// Not cached, try to find value in storage.
	ref := tx.store.find(key)
	if ref.IsNil() {
		return nil // key is unknown
	}

	// append new cache entry to tx cache
	txRef = tx.cacheEntry(txRef, ref)
	return txRef.Access()
}

func (tx *memTx) GetReadonly(k []byte) *valueDecoder {
	ref := tx.store.find(tx.makeKeyPair(k))
	if ref.IsNil() {
		return nil // key is unknown
	}
	return newValueDecoder(tx.parent, ref.Access().value)
}

func (tx *memTx) Set(k []byte, v interface{}) error {
	opValue, err := tx.decodeOpValue(v)
	if err != nil {
		return err
	}

	key := tx.makeKeyPair(k)
	txRef := tx.state.find(key)
	tx.insertKV(txRef, k, opValue)
	return nil
}

func (tx *memTx) Update(k []byte, v interface{}) error {
	opValue, err := tx.decodeOpValue(v)
	if err != nil {
		return err
	}

	key := tx.makeKeyPair(k)
	txRef := tx.state.find(key)
	if txRef.IsNil() {
		ref := tx.store.find(key)
		if ref.IsNil() {
			// unknown key -> insert op
			tx.insertKV(txRef, k, opValue)
			return nil
		}

		// key is in store, but not in the tx-cache. Move into cache, so we can
		// record the update
		txRef = tx.cacheEntry(txRef, ref)
	}

	entry := txRef.Access()
	entry.exists = true
	entry.recordUpdate(opValue)
	return nil
}

func (tx *memTx) insertKV(txRef cacheEntryRef, k []byte, value common.MapStr) {
	var rawKey []byte
	if txRef.IsNil() {
		rawKey = make([]byte, len(k))
		copy(rawKey, k)
	} else {
		rawKey = txRef.Access().key
	}
	tx.state.ops++

	entry := txCacheEntry{tx: tx.parent, key: rawKey}
	entry.recordInsert(value)
	tx.state.setEntry(txRef, entry)
}

func (tx *memTx) Remove(k []byte) {
	key := tx.makeKeyPair(k)
	txRef := tx.state.find(key)

	var rawKey []byte
	if txRef.IsNil() {
		ref := tx.store.find(key)
		if ref.IsNil() {
			return // key does not exist -> done
		}

		rawKey = ref.Access().key
	} else {
		rawKey = txRef.Access().key
	}

	entry := txCacheEntry{tx: tx.parent, key: rawKey}
	entry.recordRemove()
	tx.state.setEntry(txRef, entry)
}

func (tx *memTx) cacheEntry(at cacheEntryRef, ref valueRef) cacheEntryRef {
	entry := ref.Access()
	return tx.state.addEntry(at, txCacheEntry{
		tx:       tx.parent,
		key:      entry.key,
		value:    entry.value,
		exists:   true,
		modified: false,
	})
}

func (tx *memTx) makeKeyPair(k []byte) keyPair {
	return keyPair{
		hash: tx.hash(k),
		key:  k,
	}
}

func (tx *memTx) hash(k []byte) uint64 { return tx.hashFn(k) }

func (tx *memTx) decodeOpValue(in interface{}) (common.MapStr, error) {
	var opVal map[string]interface{}
	tc := tx.parent.getTypeConv()
	return common.MapStr(opVal), tc.Convert(&opVal, in)
}

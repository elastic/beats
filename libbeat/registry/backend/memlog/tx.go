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

import (
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/registry/backend"
)

type transaction struct {
	store *store
	state txState
	mem   memTx

	typeConv *typeconv.Converter

	active   bool
	readonly bool
}

func newTransaction(store *store, readonly bool) *transaction {
	tx := &transaction{
		store:    store,
		readonly: readonly,
		active:   true,
	}

	tx.state.init()
	tx.mem.init(tx, tx.store.mem, &tx.state)
	return tx
}

func (tx *transaction) close() {
	if tx.active {
		lock := chooseTxLock(&tx.store.lock, tx.readonly)
		lock.Unlock()
		tx.active = false

		if tx.typeConv != nil {
			tx.typeConv = nil
		}
	}
}

func (tx *transaction) Close() error {
	if tx.active {
		tx.execRollback()
	}
	return nil
}

func (tx *transaction) Rollback() error {
	if !tx.active {
		return errTxInactive
	}

	tx.execRollback()
	return nil
}

func (tx *transaction) execRollback() {
	defer tx.close()
	tx.mem.Rollback()

}

func (tx *transaction) needsCheckpoint() bool {
	tbl := tx.store.mem.tbl
	diskStore := tx.store.disk
	if diskStore.mustCheckpoint() {
		return true
	}

	pairs := uint(tbl.Len())
	logs := diskStore.numLogs()
	totalLogs := logs + tx.state.ops
	return tx.store.predCheckpoint(pairs, totalLogs)
}

func (tx *transaction) Commit() error {
	if !tx.active {
		return errTxInactive
	}
	if tx.readonly {
		return errTxReadonly
	}

	// Close rolls back all changes if the transaction did not complete without
	// error.
	defer tx.Close()

	tbl := tx.store.mem.tbl
	diskStore := tx.store.disk
	checkpoint := tx.needsCheckpoint()

	pending, exclusive := tx.store.lock.Pending(), tx.store.lock.Exclusive()

	// If disk store needs to create a snapshot, we apply the transaction into a
	// copy of the in memory state only. On success, replace the in memory state with
	// the in memory snapshot.
	if checkpoint {
		tbl = tbl.copySpline()
		tx.mem.CommitTo(&tbl, true)
		err := diskStore.commitCheckpoint(&tbl)
		if err != nil {
			return err
		}

		// Acquire exclusive lock -> no readers/writers
		pending.Lock()
		defer pending.Unlock()
		exclusive.Lock()
		defer exclusive.Unlock()

		// replace hashtable with new snapshot table
		tx.store.mem.tbl = tbl
		return nil
	}

	// Transaction log only mode.
	// Write transaction state to the transaction log file.
	// On success we commit all state updates to the in-memory state.

	// block any new readonly transactions being generated
	pending.Lock()
	defer pending.Unlock()

	// Append transaction state to the log file + commit to in memory state.
	err := diskStore.commitOps(&tx.state)
	if err != nil {
		return err
	}

	// wait for all active transaction to be finished before copying the new
	// state into the shared in memory k/v tables.
	exclusive.Lock()
	defer exclusive.Unlock()

	tx.mem.Commit()
	return nil
}

func (tx *transaction) Has(k backend.Key) (has bool, err error) {
	if err = tx.checkRead(); err == nil {
		has = tx.mem.Has(k)
	}
	return
}

func (tx *transaction) Get(k backend.Key) (v backend.ValueDecoder, err error) {
	if err = tx.checkRead(); err != nil {
		return nil, err
	}

	if tx.IsReadonly() {
		if vd := tx.mem.GetReadonly(k); vd != nil {
			return vd, nil
		}
		return nil, nil
	}

	if entry := tx.mem.Get(k); entry != nil {
		return entry, nil
	}
	return nil, nil
}

func (tx *transaction) Set(k backend.Key, from interface{}) (err error) {
	if err = tx.checkWrite(); err == nil {
		err = tx.mem.Set(k, from)
	}
	return
}

func (tx *transaction) Remove(k backend.Key) (err error) {
	if err = tx.checkWrite(); err == nil {
		tx.mem.Remove(k)
	}
	return
}

func (tx *transaction) Update(k backend.Key, fields interface{}) (err error) {
	if err = tx.checkWrite(); err == nil {
		err = tx.mem.Update(k, fields)
	}
	return err
}

func (tx *transaction) EachKey(
	internal bool,
	fn func(backend.Key) (bool, error),
) error {
	if err := tx.checkRead(); err != nil {
		return err
	}

	return reportLoopErr(tx.onEach(
		func(ref cacheEntryRef) error {
			return eachLoopErr(fn(ref.Access().key))
		},
		func(ref valueRef) error {
			return eachLoopErr(fn(ref.Access().key))
		},
	))
}

func (tx *transaction) Each(
	internal bool,
	fn func(backend.Key, backend.ValueDecoder) (bool, error),
) error {
	if err := tx.checkActive(); err != nil {
		return err
	}

	return reportLoopErr(tx.onEach(
		func(ref cacheEntryRef) error {
			entry := ref.Access()
			return eachLoopErr(fn(entry.key, entry))
		},
		func(ref valueRef) error {
			entry := ref.Access()
			vd := newValueDecoder(tx, entry.value)
			return eachLoopErr(fn(entry.key, vd))
		},
	))
}

func eachLoopErr(cont bool, err error) error {
	if err != nil {
		return err
	}
	if !cont {
		return errSigStopEach
	}
	return nil
}

func reportLoopErr(err error) error {
	if err == errSigStopEach {
		return nil
	}
	return err
}

func (tx *transaction) onEach(
	onCached func(cacheEntryRef) error,
	onStored func(valueRef) error,
) error {
	if tx.IsReadonly() || len(tx.state.bins) == 0 {
		return tx.eachStored(onStored)
	}
	return tx.onEachTx(onCached, onStored)
}

func (tx *transaction) eachStored(fn func(valueRef) error) error {
	for _, bin := range tx.store.mem.tbl.bins {
		for idx := range bin {
			ref := valueRef{bin, idx}
			if err := fn(ref); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tx *transaction) onEachTx(
	onCached func(cacheEntryRef) error,
	onStored func(valueRef) error,
) error {
	for hash, bin := range tx.store.mem.tbl.bins {
		line := tx.state.bins[hash]
		err := iterActiveHashLine(onCached, onStored, hash, bin, line)
		if err != nil {
			return err
		}
	}

	// report cached lines not in the store:
	for hash, line := range tx.state.bins {
		if len(tx.store.mem.tbl.bins[hash]) > 0 {
			continue // all entries for 'hash' already reported
		}

		err := iterCacheLine(onCached, hash, line)
		if err != nil {
			return err
		}
	}

	return nil
}

func iterActiveHashLine(
	onCached func(cacheEntryRef) error,
	onStored func(valueRef) error,
	hash uint64,
	bin bin,
	line txCacheLine,
) error {
	matching := 0

	// Iterate and report all keys known to the store, and not marked as deleted yet.
	for i := range bin {
		ref := valueRef{bin, i}
		entry := ref.Access()
		idx := line.index(entry.key)

		if idx >= 0 {
			matching++
			txRef := cacheEntryRef{hash: hash, line: line, idx: idx}
			entry := txRef.Access()
			if entry.exists {
				if err := onCached(txRef); err != nil {
					return err
				}
			}
		} else {
			if err := onStored(ref); err != nil {
				return err
			}
		}
	}

	// done. We did report all keys for the current hash
	if matching == len(line) {
		return nil
	}

	// This loop is active in case of hash collissions. Report entries in the
	// transaction not being in the store yet.
	for i := range line {
		txRef := cacheEntryRef{hash: hash, line: line, idx: i}
		entry := txRef.Access()
		if !entry.exists { // key/value pair has been delete
			continue
		}
		if idx := bin.index(entry.key); idx >= 0 { // value in store -> already reported
			continue
		}

		// new entry, not in store -> report
		if err := onCached(txRef); err != nil {
			return err
		}
	}

	return nil
}

func iterCacheLine(
	onCached func(cacheEntryRef) error,
	hash uint64,
	line txCacheLine,
) error {
	for i := range line {
		txRef := cacheEntryRef{hash: hash, line: line, idx: i}
		entry := txRef.Access()
		if !entry.exists {
			continue // key/value pair has been deleted
		}
		if err := onCached(txRef); err != nil {
			return err
		}
	}

	return nil
}

func (tx *transaction) IsReadonly() bool {
	return tx.readonly
}

func (tx *transaction) checkActive() error {
	if !tx.active {
		return errTxInactive
	}
	return nil
}

func (tx *transaction) checkWrite() error {
	if !tx.active {
		return errTxInactive
	}
	if tx.readonly {
		return errTxReadonly
	}

	return nil
}

func (tx *transaction) checkRead() error {
	return tx.checkActive()
}

func (tx *transaction) getTypeConv() *typeconv.Converter {
	tc := tx.typeConv
	if tc == nil {
		tc = typeconv.NewConverter()
		tx.typeConv = tc
	}
	return tc
}

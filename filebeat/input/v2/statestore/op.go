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
	"runtime"

	"github.com/elastic/beats/libbeat/registry"
)

// ResourceUpdateOp defers a state update to be written to the persistent store.
// The operation can be applied to the registry using ApplyWith. Calling Close
// will mark the operation as done.
type ResourceUpdateOp struct {
	store   *Store
	key     ResourceKey
	entry   *resourceEntry
	updates interface{}
}

func newUpdateOp(store *Store, key ResourceKey, entry *resourceEntry, updates interface{}) *ResourceUpdateOp {
	op := &ResourceUpdateOp{
		store:   store,
		key:     key,
		entry:   entry,
		updates: updates,
	}
	return op
}

// ApplyWith applies the operation using the withTx helper function. The helper
// function is responsible for creating and maintaining a write transaction for
// the provided store.  If possible the helper should keep the transaction open
// if an array of operations are applied.
func (op *ResourceUpdateOp) ApplyWith(withTx func(*registry.Store, func(*registry.Tx) error) error) error {
	return withTx(op.store.persistentStore, func(tx *registry.Tx) error {
		return tx.Update(registry.Key(op.key), op.updates)
	})
}

// Close marks the operation as done. ApplyWith can not be run anymore
// afterwards.  If all pending operations have been closed, the persistent
// store is assumed to be in sync with the in memory state.
func (op *ResourceUpdateOp) Close() {
	op.closePending()
	op.unlink()
	runtime.SetFinalizer(op, nil)
}

func (op *ResourceUpdateOp) closePending() {
	entry := op.entry

	entry.value.mux.Lock()
	defer entry.value.mux.Unlock()

	invariant(entry.value.pending > 0, "there should be pending updates")
	entry.value.pending--
	if entry.value.pending == 0 {
		// we are in sync now, let's remove duplicate data from main memory.
		entry.value.cached = nil
	}
}

func (op *ResourceUpdateOp) unlink() {
	store, entry := op.store, op.entry

	store.resourcesMux.Lock()
	defer store.resourcesMux.Unlock()
	if entry.Release() {
		store.resources.Remove(op.key)
	}
}

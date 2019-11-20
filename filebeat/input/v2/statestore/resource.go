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
	"errors"
	"runtime"

	"github.com/elastic/beats/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/libbeat/registry"
)

// Resource is used to lock and modify a resource its registry contents in a
// store.
// A resource must not be used anymore once the store has been closed.
type Resource struct {
	store    *Store
	isLocked bool

	key   ResourceKey
	entry *resourceEntry
}

func newResource(store *Store, key ResourceKey) *Resource {
	res := &Resource{
		store:    store,
		isLocked: false,
		key:      key,
	}

	// in case we miss an unlock operation (programmer error or panic that hash
	// been caught) we set a finalizer to eventually free the resource.
	// The Unlock operation will unsert the finalizer.
	runtime.SetFinalizer(res, (*Resource).finalize)

	store.session.Retain()
	return res
}

func (r *Resource) close() {
	runtime.SetFinalizer(r, nil)
	r.finalize()
}

func (r *Resource) finalize() {
	defer r.unlink()

	if r.IsLocked() {
		r.doUnlock()
	}

	r.store.session.Release()
}

func (r *Resource) link(create bool) {
	if r.entry != nil {
		return
	}

	if create {
		r.entry = r.store.findOrCreate(r.key, lockRequired)
	} else {
		r.entry = r.store.find(r.key, lockRequired)
	}
}

// unlink removes the pointer to the memory backed resource.
// If the in memory resource entry is not used by any other Resource,
// then we will remove it from the table.
func (r *Resource) unlink() {
	if r.entry == nil {
		return
	}

	entry := r.entry
	r.entry = nil

	r.store.shared.releaseEntry(entry)
}

// Lock locks a resource held by the store. It blocks until the lock becomes
// available.
func (r *Resource) Lock() {
	checkNotLocked(r.IsLocked())

	r.link(true)
	r.entry.Lock()
	r.isLocked = true
}

// TryLock attempts to lock the resource. It returns true if the lock has been
// acquired successfully.
func (r *Resource) TryLock() bool {
	checkNotLocked(r.IsLocked())

	r.link(true)
	r.isLocked = r.entry.TryLock()
	if !r.isLocked {
		r.unlink()
	}
	return r.isLocked
}

// Unlock releases a resource.
func (r *Resource) Unlock() {
	checkLocked(r.IsLocked())

	r.doUnlock()
	r.close()
}

func (r *Resource) doUnlock() {
	r.entry.Unlock()
	r.markUnlocked()
}

// IsLocked checks if the resource currently holds the lock to the shared
// registry entry.
func (r *Resource) IsLocked() bool {
	return r.isLocked
}

func (r *Resource) markLocked() {
	r.isLocked = true
}

func (r *Resource) markUnlocked() {
	r.isLocked = false
}

// Has checks if resource is already in registry.
// Has does not require the lock to be taken.
func (r *Resource) Has() (bool, error) {
	const op = "resource/has"

	if !r.store.active.Load() {
		return false, raiseClosed(op)
	}

	has := false

	err := r.store.shared.persistentStore.View(func(tx *registry.Tx) error {
		found, err := tx.Has(registry.Key(r.key))
		if err == nil {
			has = found
		}
		return err
	})

	if err != nil {
		err = &Error{
			op:      op,
			message: "failed to lookup resource",
			cause:   err,
		}
	}
	return has, err
}

// Update the registry state with fields in val.
// If the resource key is unknown, then a new document is inserted into the
// registry.
// Update requires the resource to be locked.
//
// It is recommended to use Update only for resource meta-data updates,
// that allow us to track and identify a resource. Read state updates
// should be handled by UpdateOp.
//
// The update call is thread-safe, as the update operation itself is protected.
// But data races are still possible, if any two go-routines update
// an overlapping set of fields.
func (r *Resource) Update(val interface{}) error {
	const op = "resource/update"

	if !r.store.active.Load() {
		return raiseClosed(op)
	}

	checkLocked(r.IsLocked())

	entry := r.entry
	invariant(entry != nil, "in memory state is not linked as expected")

	// update cached state if in-memory and persistent state are not in sync.
	entry.value.mux.Lock()
	defer entry.value.mux.Unlock()
	if entry.value.pending != 0 {
		if err := typeconv.Convert(&entry.value.cached, val); err != nil {
			return &Error{op: op, message: "failed to update in memory state", cause: err}
		}
	}

	err := r.store.shared.persistentStore.Update(func(tx *registry.Tx) error {
		return tx.Update(registry.Key(r.key), val)
	})
	if err != nil {
		return &Error{op: op, message: "failed to update persistent store", cause: err}
	}
	return nil
}

// Read current state of resource. If there are pending operations, this is
// the last in-memory state. If there are no operations, or all pending
// operations have been acked, we read directly from the registry.
// Read does not require the resource to be locked. But if the lock is
// not taken, you are subject to data races as the go-routine holding a lock on the resource
// can update its contents.
func (r *Resource) Read(to interface{}) error {
	const op = "resource/read"

	if !r.store.active.Load() {
		return raiseClosed(op)
	}

	r.link(false)
	entry := r.entry

	if entry != nil {
		entry.value.mux.Lock()

		// If in-memory and persistent store are not in sync, we require
		// the in-memory store to be the most authorative.
		if entry.value.pending != 0 {
			defer entry.value.mux.Unlock()
			if err := typeconv.Convert(to, entry.value.cached); err != nil {
				return &Error{op: op, message: "failed to read in-memory state", cause: err}
			}
			return nil
		}

		entry.value.mux.Unlock()
	}

	err := r.store.shared.persistentStore.View(func(tx *registry.Tx) error {
		vd, err := tx.Get(registry.Key(r.key))
		if err != nil || vd == nil {
			return err
		}
		return vd.Decode(to)
	})
	if err != nil {
		return &Error{op: op, message: "failed to read state from persistent store", cause: err}
	}
	return nil
}

// UpdateOp creates a resource update operation.
// The in memory state of the resource is updated right away, but the
// persistent registry state is not updated yet.
// Executing the returned operation updates the persistent state and
// invalidates the operation.
// As long as there are active operations, the in-memory state and the
// persistent state are not in sync yet.
// If in-memory state and active operations are not in sync, then
// read operations will use the in-memory state only.
//
// It is recommended to use UpdateOp for read state updates only. Resource
// metadata should be updated using Update instead.
func (r *Resource) UpdateOp(val interface{}) (*ResourceUpdateOp, error) {
	const op = "resource/create-update-op"

	if !r.store.active.Load() {
		return nil, raiseClosed(op)
	}

	checkLocked(r.IsLocked())

	entry := r.entry
	invariant(entry != nil, "in memory state is not linked to the resource")

	entry.value.mux.Lock()
	defer entry.value.mux.Unlock()

	// load current state from persistent store if there is no cached entry in
	// the resource.
	if entry.value.pending == 0 {
		err := r.store.shared.persistentStore.View(func(tx *registry.Tx) error {
			vd, err := tx.Get(registry.Key(r.key))
			if err != nil || vd == nil {
				return err
			}
			return vd.Decode(&entry.value.cached)
		})
		if err != nil {
			return nil, &Error{op: op, message: "failed to load state from persistent store", cause: err}
		}
	}

	if err := typeconv.Convert(&entry.value.cached, val); err != nil {
		return nil, &Error{op: op, message: "failed to apply the update to in-memory state", cause: err}
	}

	entry.value.pending++
	return newUpdateOp(r.store.session, r.key, entry, val), nil
}

func checkLocked(b bool) {
	invariant(!b, "try to access unlocked resource")
}

func checkNotLocked(b bool) {
	invariant(b, "invalid operation on locked resource")
}

func invariant(b bool, message string) {
	if !b {
		panic(errors.New(message))
	}
}

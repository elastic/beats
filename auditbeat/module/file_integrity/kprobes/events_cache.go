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

//go:build linux

package kprobes

import (
	"path/filepath"
)

type (
	dEntriesIndex     map[dKey]*dEntry
	dEntriesMoveIndex map[uint64]*dEntry
)

// dEntryCache is a cache of directory entries (dEntries) that exposes appropriate methods to add, get, remove and
// handle move operations. Note that dEntryCache is designed to be utilised by a single goroutine at a time and thus
// is not thread safe.
type dEntryCache struct {
	index     dEntriesIndex
	moveCache dEntriesMoveIndex
}

func newDirEntryCache() *dEntryCache {
	return &dEntryCache{
		index:     make(map[dKey]*dEntry),
		moveCache: make(map[uint64]*dEntry),
	}
}

// Get returns the dEntry associated with the given key.
func (d *dEntryCache) Get(key dKey) *dEntry {
	entry, exists := d.index[key]
	if !exists {
		return nil
	}

	return entry
}

// removeRecursively removes the given entry and all its children from the dEntryCache. Note that it is
// the responsibility of the caller to release the resources associated with the entry by calling Release.
func removeRecursively(d *dEntryCache, entry *dEntry) {
	for _, child := range entry.Children {
		removeRecursively(d, child)
	}

	delete(d.index, dKey{
		Ino:      entry.Ino,
		DevMajor: entry.DevMajor,
		DevMinor: entry.DevMinor,
	})
}

// Remove removes the given entry and all its children from the dEntryCache. Note that it is
// the responsibility of the caller to release the resources associated with the entry by calling
// Release on the dEntry.
func (d *dEntryCache) Remove(entry *dEntry) *dEntry {
	if entry == nil {
		return nil
	}

	entry.Parent.RemoveChild(entry.Name)
	entry.Parent = nil

	removeRecursively(d, entry)
	return entry
}

// Add adds the given dEntry to the dEntryCache.
func (d *dEntryCache) Add(entry *dEntry, parent *dEntry) {
	if entry == nil {
		return
	}

	_ = addRecursive(d, entry, parent, parent.Path(), nil)
}

// addRecursive recursively adds entries to the dEntryCache and calls a function on each entry's path (if specified).
// addRecursive satisfies the needs of Add and MoveTo. For the latter the caller would like to traverse all new dEntries
// added to the dEntryCache and this is done efficiently by providing a callback function.
func addRecursive(d *dEntryCache, entry *dEntry, parent *dEntry, rootPath string, cb func(path string) error) error {
	var path string
	if cb != nil {
		path = filepath.Join(rootPath, entry.Name)
		if err := cb(path); err != nil {
			return err
		}
	}

	parent.AddChild(entry)

	d.index[dKey{
		Ino:      entry.Ino,
		DevMajor: entry.DevMajor,
		DevMinor: entry.DevMinor,
	}] = entry

	for _, child := range entry.Children {
		if err := addRecursive(d, child, entry, path, cb); err != nil {
			return err
		}
	}

	return nil
}

// MoveFrom removes the given entry from the dEntryCache, adds it in the intermediate moveCache associating it
// with the caller process TID and returns it. It returns nil if the entry was not found in the dEntryCache.
// Note, that such as association between the entry and the caller process TID is mandatory as Move{To,From} events
// for older Linux kernel provide only the Filename of the moved file and only parent info is available.
func (d *dEntryCache) MoveFrom(tid uint64, entry *dEntry) {
	if entry == nil {
		return
	}

	d.Remove(entry)

	d.moveCache[tid] = entry
}

// MoveTo gets the entry associated with the given TID from the moveCache and moves it to the under the new parent
// entry. Also, supplying a callback function allows the caller to traverse all new dEntries added to the dEntryCache.
// It returns true if the entry was found in the moveCache and false otherwise.
func (d *dEntryCache) MoveTo(tid uint64, newParent *dEntry, newFileName string, cb func(path string) error) (bool, error) {
	entry, exists := d.moveCache[tid]
	if !exists {
		return false, nil
	}

	delete(d.moveCache, tid)
	entry.Name = newFileName

	return true, addRecursive(d, entry, newParent, newParent.Path(), cb)
}

// MoveClear removes the entry associated with the given TID from the moveCache.
func (d *dEntryCache) MoveClear(tid uint64) {
	entry, exists := d.moveCache[tid]
	if !exists {
		return
	}

	delete(d.moveCache, tid)
	entry.Release()
}

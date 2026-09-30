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

package filestream

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/file"
)

// fileStateTable records the identity of each open harvester file.
// Each input has its own table.
type fileStateTable struct {
	mu      sync.Mutex
	entries map[string]*openFileState
}

// openFileState records one open harvester file.
type openFileState struct {
	table *fileStateTable

	name string

	// os is the open file identity from fstat, guarded by table.mu.
	// A zero value means that the identity is unknown.
	os file.StateOS
}

func newFileStateTable() *fileStateTable {
	return &fileStateTable{
		entries: make(map[string]*openFileState),
	}
}

// Register records a newly opened harvester and returns its handle. A newer
// handle replaces an older entry for the same name.
func (t *fileStateTable) Register(name string) *openFileState {
	if t == nil {
		return nil
	}

	h := &openFileState{
		table: t,
		name:  name,
	}
	t.mu.Lock()
	t.entries[name] = h
	t.mu.Unlock()

	return h
}

// Deregister removes h only if it still owns its current key.
func (t *fileStateTable) Deregister(h *openFileState) {
	if t == nil || h == nil {
		return
	}

	t.mu.Lock()
	if t.entries[h.name] == h {
		delete(t.entries, h.name)
	}
	t.mu.Unlock()
}

// LookupOSState returns the open file identity for name.
// A missing entry or zero StateOS returns ok=false. For example, Windows can
// fail to read the identity of a locked file or a file pending deletion.
func (t *fileStateTable) LookupOSState(name string) (file.StateOS, bool) {
	if t == nil {
		return file.StateOS{}, false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	h, ok := t.entries[name]
	if !ok || h.os == (file.StateOS{}) {
		return file.StateOS{}, false
	}
	return h.os, true
}

// PinOSState records the open file identity from fstat.
func (h *openFileState) PinOSState(st file.StateOS) {
	if h == nil {
		return
	}

	h.table.mu.Lock()
	h.os = st
	h.table.mu.Unlock()
}

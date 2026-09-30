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

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
)

// fileStateTable shares file state between an input's scanner and harvesters.
type fileStateTable struct {
	mu      sync.Mutex
	entries map[string]*openFileState
	byPath  map[string]*openFileState
}

// openFileState shares one open file's state with the scanner.
type openFileState struct {
	table *fileStateTable

	// name is the entry key. "" until Publish.
	name string

	// os is the open file identity from fstat. Zero means unknown.
	os file.StateOS

	// desc is the latest scanner descriptor.
	desc loginp.FileDescriptor
}

func newFileStateTable() *fileStateTable {
	return &fileStateTable{
		entries: make(map[string]*openFileState),
		byPath:  make(map[string]*openFileState),
	}
}

// NewHandle returns a handle for a newly opened harvester file.
func (t *fileStateTable) NewHandle() *openFileState {
	if t == nil {
		return nil
	}
	return &openFileState{table: t}
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
	if t.byPath[h.desc.Filename] == h {
		delete(t.byPath, h.desc.Filename)
	}
	t.mu.Unlock()
}

// UpdateDescriptor updates an open harvester's scanner descriptor. It does not
// create entries for files without an open harvester.
func (t *fileStateTable) UpdateDescriptor(name string, desc loginp.FileDescriptor) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if h, ok := t.entries[name]; ok {
		t.setDesc(h, desc)
	}
}

// setDesc stores desc on h and moves h's path entry if the file name
// changed. The caller holds t.mu.
func (t *fileStateTable) setDesc(h *openFileState, desc loginp.FileDescriptor) {
	if old := h.desc.Filename; old != desc.Filename {
		if t.byPath[old] == h {
			delete(t.byPath, old)
		}
		if desc.Filename != "" {
			t.byPath[desc.Filename] = h
		}
	}
	h.desc = desc
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

// PinnedDescriptor returns the descriptor and file identity for an open harvester.
// It returns ok=false if the path has no pinned harvester.
func (t *fileStateTable) PinnedDescriptor(path string) (desc loginp.FileDescriptor, pin file.StateOS, ok bool) {
	if t == nil {
		return loginp.FileDescriptor{}, file.StateOS{}, false
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	h, ok := t.byPath[path]
	if !ok || h.os == (file.StateOS{}) {
		return loginp.FileDescriptor{}, file.StateOS{}, false
	}
	return h.desc, h.os, true
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

// Publish lists h under name with that identity's descriptor and removes h's
// previous entry if h still owns it.
func (h *openFileState) Publish(name string, desc loginp.FileDescriptor) {
	if h == nil {
		return
	}

	t := h.table
	t.mu.Lock()
	defer t.mu.Unlock()

	t.setDesc(h, desc)
	if t.entries[h.name] == h {
		delete(t.entries, h.name)
	}
	h.name = name
	t.entries[name] = h
}

// FingerprintSum returns the completed SHA-256 fingerprint or "".
func (h *openFileState) FingerprintSum() string {
	if h == nil {
		return ""
	}

	h.table.mu.Lock()
	defer h.table.mu.Unlock()

	if h.desc.Fingerprint.Complete() {
		return h.desc.Fingerprint.Sum
	}
	return ""
}

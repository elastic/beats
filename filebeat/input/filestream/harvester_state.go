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
		h.desc = desc
	}
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

	h.desc = desc
	if t.entries[h.name] == h {
		delete(t.entries, h.name)
	}
	h.name = name
	t.entries[name] = h
}

// LongerReplacement returns the path in the latest scanner descriptor if it is
// a different file from the open one and larger than offset, or "".
func (h *openFileState) LongerReplacement(offset int64) string {
	if h == nil {
		return ""
	}

	h.table.mu.Lock()
	defer h.table.mu.Unlock()

	// GZIP sizes are compressed, so they cannot be compared to offset.
	if h.desc.Info == nil || h.desc.GZIP || h.os == (file.StateOS{}) {
		return ""
	}
	// A zero StateOS means the scanner could not read the identity.
	scanned := h.desc.Info.GetOSState()
	if scanned == (file.StateOS{}) || h.os.IsSame(scanned) || h.desc.Info.Size() <= offset {
		return ""
	}
	return h.desc.Filename
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

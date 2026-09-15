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

package mapstr

import (
	"sync/atomic"
	"unsafe"
)

// Pointer stores a pointer to atomically get/set a mapstr.M object
// This should give faster access for use cases with lots of reads and a few
// changes.
// It's important to note that modifying the map is not thread safe, only fully
// replacing it.
type Pointer struct {
	p *unsafe.Pointer
}

// NewMPointer initializes and returns a pointer to the given M
func NewPointer(m M) Pointer {
	pointer := unsafe.Pointer(&m)
	return Pointer{p: &pointer}
}

// Get returns the M stored under this pointer
func (m Pointer) Get() M {
	if m.p == nil {
		return nil
	}
	return *(*M)(atomic.LoadPointer(m.p))
}

// Set stores a pointer the given M, replacing any previous one
func (m *Pointer) Set(p M) {
	atomic.StorePointer(m.p, unsafe.Pointer(&p))
}

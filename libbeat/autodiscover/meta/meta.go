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

package meta

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
)

// Map stores a map of id -> MapStrPointer
type Map struct {
	mutex sync.RWMutex
	meta  map[uint64]mapstr.MPointer
}

// NewMap instantiates and returns a new meta.Map
func NewMap() *Map {
	return &Map{
		meta: make(map[uint64]mapstr.MPointer),
	}
}

// Store inserts or updates given meta under the given id. Then it returns a MapStrPointer to it
func (m *Map) Store(id uint64, meta mapstr.M) mapstr.MPointer {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if meta == nil {
		return mapstr.MPointer{}
	}

	p, ok := m.meta[id]
	if !ok {
		// create
		p = common.NewMapStrPointer(meta)
		m.meta[id] = p
	} else {
		// update
		p.Set(meta)
	}

	return p
}

// Remove meta stored under the given id
func (m *Map) Remove(id uint64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.meta, id)
}

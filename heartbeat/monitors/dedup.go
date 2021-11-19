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

package monitors

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func newDedup() dedup {
	return dedup{
		byId: map[string]*Monitor{},
		mtx:  &sync.Mutex{},
	}
}

type dedup struct {
	byId map[string]*Monitor
	mtx  *sync.Mutex
}

func (um dedup) register(m *Monitor) {
	um.mtx.Lock()
	defer um.mtx.Unlock()

	closed := um.stopUnsafe(m)
	if closed {
		logp.Warn("monitor ID %s is configured for multiple monitors! IDs should be unique values, last seen config will win", m.stdFields.ID)
	}

	um.byId[m.stdFields.ID] = m
}

func (um dedup) unregister(m *Monitor) {
	um.mtx.Lock()
	defer um.mtx.Unlock()

	um.stopUnsafe(m)

	delete(um.byId, m.stdFields.ID)
}

func (um dedup) stopUnsafe(m *Monitor) bool {
	if existing, ok := um.byId[m.stdFields.ID]; ok {
		existing.Stop()
		return ok
	}
	return false
}

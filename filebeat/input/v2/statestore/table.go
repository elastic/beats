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
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-concert/atomic"
)

// In memory registry state table. Updates are written directly to this table.
// As long as there are pending operations, we read the state from this table.
// If there is no entry with cached value present, we assume the registry to be
// in sync with all updates applied.
// Entries are reference counted allowing us to free space in the table if there is
// no more go-routine potentially accessing a resource.
type table map[ResourceKey]*resourceEntry

// resourceEntry keeps track of actual resource locks and pending updates.
type resourceEntry struct {
	key      ResourceKey
	refCount atomic.Uint
	lock     chan struct{}
	value    valueState
}

// valueState keeps track of pending updates to a value.
// As long as there are pending updates, cached holds the last known correct value
// and pending will be > 0.
// If `pending` is 0, then the state store and the persistent registry are in sync.
// In this case `cached` will be nil and the registry is used for reading a value.
type valueState struct {
	mux     sync.Mutex
	pending uint          // pending updates until value is in sync
	cached  common.MapStr // current value if state == valueOutOfSync
}

func (t table) Create(k ResourceKey) *resourceEntry {
	lock := make(chan struct{}, 1)
	lock <- struct{}{}
	r := &resourceEntry{
		key:      k,
		lock:     lock,
		refCount: atomic.MakeUint(1),
	}
	t[k] = r
	return r
}

func (t table) Find(k ResourceKey) *resourceEntry {
	r := t[k]
	if r != nil {
		r.Retain()
	}
	return r
}

func (t table) Remove(k ResourceKey) {
	delete(t, k)
}

func (r *resourceEntry) Retain() {
	r.refCount.Inc()
}

func (r *resourceEntry) Release() bool {
	return r.refCount.Dec() == 0
}

func (r *resourceEntry) Lock() {
	<-r.lock
}

func (r *resourceEntry) TryLock() bool {
	select {
	case <-r.lock:
		return true
	default:
		return false
	}
}

func (r *resourceEntry) Unlock() {
	r.lock <- struct{}{}
}

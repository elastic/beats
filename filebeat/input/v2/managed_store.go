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

package v2

import (
	"sync"

	"go.uber.org/atomic"

	"github.com/elastic/go-concert"
)

// managedStoreAccessor keeps track of active stores in use.
// It can be used to shutdown all active stores ones an input has finished.
// With the help of the managedStoreAccessor most inputs that use one store
// only during its lifetime don't need to release stores.
// Due to the reference counting, a cascade of managedStoreAccessor can be used
// to keep track of global active stores. This ensures that resource locks
// are a truly globally shared resource, while shared persistent stores are only
// create if there is a need to do so.
type managedStoreAccessor struct {
	accessor StoreAccessor

	mu           sync.Mutex
	activeStores map[string]*managedStore
}

type managedStore struct {
	active atomic.Bool

	manager *managedStoreAccessor
	name    string
	ref     concert.RefCount
	store   Store
}

func (a *managedStoreAccessor) OpenStore(name string) (Store, error) {
	a.mux.Lock()
	defer a.mux.Unlock()

	if ms := a.activeStores[name]; ms != nil {
		ms.ref.Retain()
		return ms, nil
	}

	store, err := a.accessor.OpenStore(name)
	if err != nil {
		return nil, err
	}

	ms := &managedStore{
		manager: a,
		name:    name,
		store:   store,
		active:  atomic.MakeBool(true),
	}
	a.activeStores[name] = ms
	return ms, nil
}

func (a *managedStoreAccessor) shutdown() {
	a.mux.Lock()
	defer a.mux.Unlock()
	for _, ms := range a.activeStores {
		ms.store.Deactivate()
	}

	for name := range a.activeStores {
		delete(a.activeStores, name)
	}
}

func (ms *managedStore) Deactivate() {
	ms.manager.mux.Lock()
	defer ms.manager.mux.Unlock()

	if ms.ref.Release() {
		ms.store.Deactivate()
		delete(ms.manager.activeStores, ms.name)
	}
}

func (ms *managedStore) Access(key string) Resource {
	if !ms.active.Load() {
		return nil
	}
	return ms.store.Access(key)
}

func (ms *managedStore) Lock(key string) Resource {
	if !ms.active.Load() {
		return nil
	}
	return ms.store.Lock(key)
}

func (ms *managedStore) TryLock(key string) (Resource, bool) {
	if !ms.active.Load() {
		return nil, false
	}
	return ms.store.TryLock(key)
}

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

	"github.com/menderesk/beats/v7/libbeat/statestore/backend"
)

// Registry manages multiple key-value stores.
// When working with a registry, one must access a store. Depending on backend
// a store can be an index, a table, or a directory. All access to a store is
// handled by transaction.
type Registry struct {
	backend backend.Registry

	mu     sync.Mutex
	active map[string]*sharedStore // active/open stores
	wg     sync.WaitGroup
}

// ValueDecoder is used to decode retrieved from an actual store.  A
// ValueDecoder instance is valid for the lifetime of the transaction only.
type ValueDecoder = backend.ValueDecoder

// NewRegistry creates a new Registry with a configured backend.
func NewRegistry(backend backend.Registry) *Registry {
	return &Registry{
		backend: backend,
		active:  map[string]*sharedStore{},
	}
}

// Close closes the backend storage. Close blocks until all stores in use are closed.
func (r *Registry) Close() error {
	r.wg.Wait() // wait for all stores being closed
	return r.backend.Close()
}

// Get opens a shared store. A store is closed and released only after all it's
// users have closed the store.
func (r *Registry) Get(name string) (*Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	shared := r.active[name]
	if shared == nil {
		backend, err := r.backend.Access(name)
		if err != nil {
			return nil, &ErrorAccess{name: name, cause: err}
		}

		shared = newSharedStore(r, name, backend)
		defer shared.Release()

		r.active[name] = shared
		r.wg.Add(1)
	}

	return newStore(shared), nil
}

func (r *Registry) unregisterStore(s *sharedStore) {
	_, exists := r.active[s.name]
	if !exists {
		panic("removing an unknown store")
	}

	delete(r.active, s.name)
	r.wg.Done()
}

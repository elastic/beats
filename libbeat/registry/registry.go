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

package registry

import (
	"sync"

	"github.com/elastic/beats/libbeat/registry/backend"
)

type Registry struct {
	backend backend.Registry

	mu     sync.Mutex
	active map[string]*sharedStore // active/open stores
	wg     sync.WaitGroup
}

type Key []byte

type ValueDecoder interface {
	Decode(to interface{}) error
}

func NewRegistry(backend backend.Registry) *Registry {
	return &Registry{
		backend: backend,
		active:  map[string]*sharedStore{},
	}
}

func (r *Registry) Close() error {
	r.wg.Wait() // wait for all stores being closed
	return r.backend.Close()
}

// Get opens a shared store.
func (r *Registry) Get(name string) (*Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	shared := r.active[name]
	if shared == nil {
		backend, err := r.backend.Access(name)
		if err != nil {
			return nil, err
		}

		shared = newSharedStore(r, name, backend)
		r.active[name] = shared
		r.wg.Add(1)
	} else {
		shared.refCount.Inc()
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

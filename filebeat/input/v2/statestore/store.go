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

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/registry"
)

// Store provides some coordinates access to a registry.Store.
// All update and read operations require users to acquire an resource first.
// A Resource must be locked before it can be modified. This ensures that at most
// one go-routine has access to a resource. Lock/TryLock/Unlack can be used to
// coordinate resource access even between independent components.
type Store struct {
	log *logp.Logger

	persistentStore *registry.Store

	resourcesMux sync.Mutex
	resources    table
}

// NewStore creates a new state Store that is backed by a persistent store.
func NewStore(log *logp.Logger, store *registry.Store) *Store {
	invariant(log != nil, "missing a logger")
	invariant(store != nil, "missing a persistent store")

	return &Store{
		log:             log,
		persistentStore: store,
		resources:       table{},
	}
}

// Access an unlocked resource. This creates a handle to a resource that may
// not yet exist in the persistent registry.
func (s *Store) Access(key ResourceKey) *Resource {
	return newResource(s, key)
}

// Lock locks and returns the resource for a given key.
func (s *Store) Lock(key ResourceKey) *Resource {
	res := s.Access(key)
	res.Lock()
	return res
}

// TryLock locks and returns the resource for a given key.
// The locked return value is set to false if TryLock failed, but the resource
// itself is always returned.
func (s *Store) TryLock(key ResourceKey) (res *Resource, locked bool) {
	res = s.Access(key)
	return res, res.TryLock()
}

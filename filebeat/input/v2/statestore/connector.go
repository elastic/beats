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

// Connector is used to connect to a store backed by a registry.
type Connector struct {
	log      *logp.Logger
	registry *registry.Registry

	mux sync.Mutex

	// set of stores currently with at least one active session.
	stores map[string]*sharedStore
}

// NewConnector creates a new store connector for accessing a resource Store.
func NewConnector(log *logp.Logger, reg *registry.Registry) *Connector {
	invariant(log != nil, "missing logger")
	invariant(reg != nil, "missing registry")

	return &Connector{
		log:      log,
		registry: reg,
		stores:   map[string]*sharedStore{},
	}
}

// Open creates a connection to a named store.
func (c *Connector) Open(name string) (*Store, error) {
	ok := false

	c.mux.Lock()
	defer c.mux.Unlock()

	persistentStore, err := c.registry.Get(name)
	if err != nil {
		return nil, err
	}
	defer ifNotOK(&ok, func() {
		persistentStore.Close()
	})

	shared := c.stores[name]
	if shared == nil {
		shared = &sharedStore{
			name:            name,
			persistentStore: persistentStore,
			resources:       table{},
		}
		c.stores[name] = shared
	} else {
		shared.refCount.Retain()
	}

	ok = true
	return newStore(newSession(c, shared)), nil
}

func (c *Connector) releaseStore(store *sharedStore) {
	c.mux.Lock()
	released := store.refCount.Release()
	if released {
		delete(c.stores, store.name)
	}
	c.mux.Unlock()

	if released {
		store.close()
	}
}

func ifNotOK(b *bool, fn func()) {
	if !(*b) {
		fn()
	}
}

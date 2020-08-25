// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
)

// Registry maintains a collection of named shared stores that can be used by persistent caches.
type Registry struct {
	mutex sync.Mutex
	path  string

	stores map[string]*sharedStore
}

// NewCache creates a new cache with one of the stores in a registry. If the store doesn't exist,
// it is created.
func (r *Registry) NewCache(name string, opts Options) (*PersistentCache, error) {
	logger := logp.NewLogger("persistentcache")

	store, err := r.OpenStore(logger, name)
	if err != nil {
		return nil, err
	}

	return &PersistentCache{
		log:      logger,
		store:    store,
		registry: r,

		refreshOnAccess: opts.RefreshOnAccess,
		timeout:         opts.Timeout,
	}, nil
}

// OpenStore opens a store in the registry. If a store with the same name already exists, it is
// returned. If not, a new store is created.
func (r *Registry) OpenStore(logger *logp.Logger, name string) (*Store, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.stores == nil {
		r.stores = make(map[string]*sharedStore)
	}

	if _, ok := r.stores[name]; !ok {
		rootPath := r.path
		if rootPath == "" {
			rootPath = paths.Resolve(paths.Data, cacheFile)
		}

		store, err := newStore(logger, rootPath, name)
		if err != nil {
			return nil, err
		}
		r.stores[name] = &sharedStore{Store: store, useCount: 0}
	}

	r.stores[name].useCount++
	return r.stores[name].Store, nil
}

// ReleaseStore announces the registry that a store is not being used anymore by one of its
// consumers. When all consumers have released the store, it is closed.
func (r *Registry) ReleaseStore(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	store, ok := r.stores[name]
	if !ok {
		panic(fmt.Sprintf("store '%s' not managed by this registry", name))
	}

	store.useCount--
	if store.useCount == 0 {
		delete(r.stores, name)
		return store.Close()
	}
	return nil
}

type sharedStore struct {
	*Store
	useCount uint32
}

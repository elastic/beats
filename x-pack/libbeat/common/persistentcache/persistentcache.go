// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
)

const (
	cacheFile     = "cache"
	cacheFileMode = os.FileMode(0600)
)

// Persistent cache is a persistent map of keys to values. Elements added to the
// cache are stored until they are explicitly deleted or are expired due to time-based
// eviction based on last access or add time.
//
// Expired elements are not visible through classes methods, but they do remain
// stored in the cache until CleanUp() is invoked. Therefore CleanUp() must be
// invoked periodically to prevent the cache from becoming a memory leak. If
// you want to start a goroutine to perform periodic clean-up then see
// StartJanitor().
//
// Cache does not support storing nil values. Any attempt to put nil into
// the cache will cause a panic.
type PersistentCache struct {
	log *logp.Logger

	store           *statestore.Store
	refreshOnAdd    bool
	removalListener common.RemovalListener

	clock func() time.Time
}

// PersistentCacheOptions are the options that can be used to custimize
type PersistentCacheOptions struct {
	// If set to true, expiration time of an entry is only updated
	// when the object is added to the cache, and not when the
	// cache is accessed.
	RefreshOnAdd bool

	// RemovalListener is called every time a key is removed.
	RemovalListener common.RemovalListener
}

// NewPersistentCache creates and returns a new persistent cache. d is the length of time after last
// access that cache elements expire. Cache returned by this method must be closed with Close() when
// not needed anymore.
func NewPersistentCache(name string, d time.Duration, opts PersistentCacheOptions) (*PersistentCache, error) {
	return newPersistentCache(&registry, name, d, opts)
}

func newPersistentCache(registry *persistentCacheRegistry, name string, d time.Duration, opts PersistentCacheOptions) (*PersistentCache, error) {
	logger := logp.NewLogger("persistentcache")

	store, err := registry.openStore(logger, name)
	if err != nil {
		return nil, err
	}

	return &PersistentCache{
		log:   logger,
		store: store,

		refreshOnAdd:    opts.RefreshOnAdd,
		removalListener: opts.RemovalListener,
	}, nil
}

// Put writes the given key and value to the map replacing any
// existing value if it exists.
func (c *PersistentCache) Put(k string, v common.Value) error {
	return c.PutWithTimeout(k, v, 0)
}

// PutWithTimeout writes the given key and value to the map replacing any
// existing value if it exists.
// The cache expiration time will be overwritten by timeout of the key being
// inserted.
func (c *PersistentCache) PutWithTimeout(k string, v common.Value, timeout time.Duration) error {
	return c.store.Set(k, v)
}

// Get the current value associated with a key or nil if the key is not
// present. The last access time of the element is updated.
func (c *PersistentCache) Get(k string, v common.Value) error {
	return c.store.Get(k, v)
}

// CleanUp performs maintenance on the cache by removing expired elements from
// the cache. If a RemoveListener is registered it will be invoked for each
// element that is removed during this clean up operation. The RemovalListener
// is invoked on the caller's goroutine.
func (c *PersistentCache) CleanUp() int {
	return 0
}

// StartJanitor starts a goroutine that will periodically invoke the cache's
// CleanUp() method.
func (c *PersistentCache) StartJanitor(interval time.Duration) {
}

// StopJanitor stops the goroutine created by StartJanitor.
func (c *PersistentCache) StopJanitor() {
}

// Close releases all resources associated with this cache.
func (c *PersistentCache) Close() error {
	// TODO: Close the global registry when all stores have been closed
	return c.store.Close()
}

func (c *PersistentCache) now() time.Time {
	if c.clock != nil {
		return c.clock()
	}
	return time.Now()
}

type persistentCacheRegistry struct {
	sync.Mutex

	path     string
	registry *statestore.Registry
}

func (r *persistentCacheRegistry) openStore(logger *logp.Logger, name string) (*statestore.Store, error) {
	r.Lock()
	defer r.Unlock()

	if r.registry == nil {
		rootPath := r.path
		if rootPath == "" {
			rootPath = paths.Resolve(paths.Data, cacheFile)
		}
		backend, err := memlog.New(logger, memlog.Settings{
			Root:     rootPath,
			FileMode: cacheFileMode,
		})
		if err != nil {
			return nil, errors.Wrap(err, "opening store for persistent cache")
		}
		r.registry = statestore.NewRegistry(backend)
	}

	return r.registry.Get(name)
}

var registry persistentCacheRegistry

func openStore(logger *logp.Logger, name string) (*statestore.Store, error) {
	return registry.openStore(logger, name)
}

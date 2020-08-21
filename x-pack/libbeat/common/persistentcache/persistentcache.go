// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"encoding/json"
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

// registry is the global persistent caches registry
var registry PersistentCacheRegistry

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
	removalListener RemovalListener

	clock func() time.Time
}

// RemovalListener is a function called when a entry is removed from cache
type RemovalListener func(k string, v common.Value)

// PersistentCacheOptions are the options that can be used to custimize
type PersistentCacheOptions struct {
	// If set to true, expiration time of an entry is only updated
	// when the object is added to the cache, and not when the
	// cache is accessed.
	RefreshOnAdd bool

	// RemovalListener is called every time a key is removed.
	RemovalListener RemovalListener
}

// NewPersistentCache creates and returns a new persistent cache. d is the length of time after last
// access that cache elements expire. Cache returned by this method must be closed with Close() when
// not needed anymore.
func NewPersistentCache(name string, d time.Duration, opts PersistentCacheOptions) (*PersistentCache, error) {
	return newPersistentCache(&registry, name, d, opts)
}

func newPersistentCache(registry *PersistentCacheRegistry, name string, d time.Duration, opts PersistentCacheOptions) (*PersistentCache, error) {
	logger := logp.NewLogger("persistentcache")

	store, err := registry.OpenStore(logger, name)
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

type persistentCacheEntry struct {
	Expiry time.Time
	Item   []byte
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
	var err error
	var entry persistentCacheEntry
	entry.Item, err = json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "encoding item to store in cache")
	}
	if timeout > 0 {
		entry.Expiry = c.now().Add(timeout)
	}
	return c.store.Set(k, entry)
}

// Get the current value associated with a key or nil if the key is not
// present. The last access time of the element is updated.
func (c *PersistentCache) Get(k string, v common.Value) error {
	var entry persistentCacheEntry
	err := c.store.Get(k, &entry)
	if err != nil {
		return err
	}
	err = json.Unmarshal(entry.Item, v)
	if err != nil {
		return errors.Wrap(err, "decoding item stored in cache")
	}
	return nil
}

// CleanUp performs maintenance on the cache by removing expired elements from
// the cache. If a RemoveListener is registered it will be invoked for each
// element that is removed during this clean up operation. The RemovalListener
// is invoked on the caller's goroutine.
func (c *PersistentCache) CleanUp() int {
	var expired []string
	var entry persistentCacheEntry
	c.store.Each(func(key string, decoder statestore.ValueDecoder) (bool, error) {
		decoder.Decode(&entry)
		if c.expired(&entry) {
			expired = append(expired, key)
		}
		return true, nil
	})
	for _, key := range expired {
		c.store.Remove(key)
	}
	return len(expired)
}

func (c *PersistentCache) expired(entry *persistentCacheEntry) bool {
	return !entry.Expiry.IsZero() && c.now().After(entry.Expiry)
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
	return c.store.Close()
}

func (c *PersistentCache) now() time.Time {
	if c.clock != nil {
		return c.clock()
	}
	return time.Now()
}

type PersistentCacheRegistry struct {
	mutex    sync.Mutex
	path     string
	registry *statestore.Registry
}

func (r *PersistentCacheRegistry) OpenStore(logger *logp.Logger, name string) (*statestore.Store, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

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

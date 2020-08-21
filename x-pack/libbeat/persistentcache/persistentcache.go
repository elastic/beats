// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

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
var registry Registry

var expiredError = errors.New("key expired")

// PersistentCache is a persistent map of keys to values. Elements added to the
// cache are stored until they are explicitly deleted or are expired due to time-based
// eviction based on last access or add time.
//
// Expired elements are not visible through classes methods, but they do remain
// stored in the cache until CleanUp() is invoked. Therefore CleanUp() must be
// invoked periodically to prevent the cache from becoming a memory leak. If
// you want to start a goroutine to perform periodic clean-up then see
// StartJanitor().
type PersistentCache struct {
	log *logp.Logger

	store           *statestore.Store
	refreshOnAccess bool
	timeout         time.Duration
	janitorQuit     chan struct{}

	clock func() time.Time
}

// Options are the options that can be used to custimize
type Options struct {
	// Lenght of time before cache elements expire
	Timeout time.Duration

	// If set to true, expiration time of an entry is updated
	// when the object is accessed.
	RefreshOnAccess bool
}

// New creates and returns a new persistent cache. d is the length of time after last
// access that cache elements expire. Cache returned by this method must be closed with Close() when
// not needed anymore.
func New(name string, opts Options) (*PersistentCache, error) {
	return newCache(&registry, name, opts)
}

func newCache(registry *Registry, name string, opts Options) (*PersistentCache, error) {
	logger := logp.NewLogger("persistentcache")

	store, err := registry.OpenStore(logger, name)
	if err != nil {
		return nil, err
	}

	return &PersistentCache{
		log:   logger,
		store: store,

		refreshOnAccess: opts.RefreshOnAccess,
		timeout:         opts.Timeout,
	}, nil
}

type cacheEntry struct {
	Expiration int64  `json:"e,omitempty"`
	Item       string `json:"i,omitempty"`
}

func (e *cacheEntry) refresh(now time.Time, initialTimeout, defaultTimeout time.Duration) bool {
	timeout := initialTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	if timeout > 0 {
		e.Expiration = now.Add(timeout).Unix()
		return true
	}
	return false
}

// Put writes the given key and value to the map replacing any
// existing value if it exists.
func (c *PersistentCache) Put(k string, v interface{}) error {
	return c.PutWithTimeout(k, v, 0)
}

// PutWithTimeout writes the given key and value to the map replacing any
// existing value if it exists.
// The cache expiration time will be overwritten by timeout of the key being
// inserted.
func (c *PersistentCache) PutWithTimeout(k string, v interface{}, timeout time.Duration) error {
	var entry cacheEntry
	d, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encoding item to store in cache: %w", err)
	}
	entry.Item = base64.StdEncoding.EncodeToString(d)
	if err != nil {
		return fmt.Errorf("encoding item to store in cache: %w", err)
	}
	c.refresh(&entry, timeout)
	return c.store.Set(k, entry)
}

// Get the current value associated with a key or nil if the key is not
// present. The last access time of the element is updated.
func (c *PersistentCache) Get(k string, v interface{}) error {
	var entry cacheEntry
	err := c.store.Get(k, &entry)
	if err != nil {
		return err
	}
	if c.expired(&entry) {
		return expiredError
	}
	if c.refreshOnAccess && c.refresh(&entry, 0) {
		c.store.Set(k, entry)
	}
	d, err := base64.StdEncoding.DecodeString(entry.Item)
	if err != nil {
		return fmt.Errorf("decoding base64 string: %w", err)
	}
	err = json.Unmarshal(d, v)
	if err != nil {
		return fmt.Errorf("decoding item stored in cache: %w", err)
	}
	return nil
}

// CleanUp performs maintenance on the cache by removing expired elements from
// the cache.
func (c *PersistentCache) CleanUp() int {
	var expired []string
	var entry cacheEntry
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

func (c *PersistentCache) refresh(e *cacheEntry, timeout time.Duration) bool {
	if timeout == 0 {
		timeout = c.timeout
	}
	if timeout > 0 {
		e.Expiration = c.now().Add(timeout).Unix()
		return true
	}
	return false
}

func (c *PersistentCache) expired(entry *cacheEntry) bool {
	return entry.Expiration != 0 && c.now().Unix() > entry.Expiration
}

// StartJanitor starts a goroutine that will periodically invoke the cache's
// CleanUp() method.
func (c *PersistentCache) StartJanitor(interval time.Duration) {
	c.janitorQuit = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.CleanUp()
			case <-c.janitorQuit:
				return
			}
		}
	}()
}

// StopJanitor stops the goroutine created by StartJanitor.
func (c *PersistentCache) StopJanitor() {
	close(c.janitorQuit)
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

type Registry struct {
	mutex    sync.Mutex
	path     string
	registry *statestore.Registry
}

func (r *Registry) OpenStore(logger *logp.Logger, name string) (*statestore.Store, error) {
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
			return nil, fmt.Errorf("opening store for persistent cache: %w", err)
		}
		r.registry = statestore.NewRegistry(backend)
	}

	return r.registry.Get(name)
}

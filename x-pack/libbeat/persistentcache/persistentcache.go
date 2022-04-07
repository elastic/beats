// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/paths"
)

const (
	cacheFile = "cache"
	gcPeriod  = 5 * time.Minute
)

// PersistentCache is a persistent map of keys to values. Elements added to the
// cache are stored until they are explicitly deleted or are expired due to time-based
// eviction based on last access or add time.
type PersistentCache struct {
	log *logp.Logger

	store *Store
	codec codec

	refreshOnAccess bool
	timeout         time.Duration
}

// Options are the options that can be used to customize persistent caches
type Options struct {
	// Length of time before cache elements expire
	Timeout time.Duration

	// If set to true, expiration time of an entry is updated
	// when the object is accessed.
	RefreshOnAccess bool

	// If empty, beats data path is used.
	RootPath string
}

// New creates and returns a new persistent cache.
// Cache returned by this method must be closed with Close() when
// not needed anymore.
func New(name string, opts Options) (*PersistentCache, error) {
	logger := logp.NewLogger("persistentcache")

	rootPath := opts.RootPath
	if rootPath == "" {
		rootPath = paths.Resolve(paths.Data, cacheFile)
	}
	store, err := newStore(logger, rootPath, name)
	if err != nil {
		return nil, err
	}

	return &PersistentCache{
		log:   logger,
		store: store,
		codec: newCBORCodec(),

		refreshOnAccess: opts.RefreshOnAccess,
		timeout:         opts.Timeout,
	}, nil
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
	d, err := c.codec.Encode(v)
	if err != nil {
		return fmt.Errorf("encoding item to store in cache: %w", err)
	}
	if timeout == 0 {
		timeout = c.timeout
	}
	return c.store.Set([]byte(k), d, timeout)
}

// Get the current value associated with a key or nil if the key is not
// present. The last access time of the element is updated.
func (c *PersistentCache) Get(k string, v interface{}) error {
	d, err := c.store.Get([]byte(k))
	if err != nil {
		return err
	}
	if c.refreshOnAccess && c.timeout > 0 {
		c.store.Set([]byte(k), d, c.timeout)
	}
	err = c.codec.Decode(d, v)
	if err != nil {
		return fmt.Errorf("decoding item stored in cache: %w", err)
	}
	return nil
}

// Close releases all resources associated with this cache.
func (c *PersistentCache) Close() error {
	return c.store.Close()
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/redis/apiv1/redispb"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/sqladmin/v1"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Cache represents a generic cache with metadata and mutex
type Cache[T any] struct {
	data            map[string]T
	lastRefreshed   time.Time
	refreshInterval time.Duration
	lock            sync.Mutex
	logger          *logp.Logger
}

func (c *Cache[T]) isExpired() bool {
	return time.Since(c.lastRefreshed) > c.refreshInterval
}

// Get retrieves a value from the cache if it exists.
// It returns the value and a boolean indicating whether the key was found.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	value, found := c.data[key]
	return value, found
}

// EnsureFresh checks if the cache is expired and if so, refreshes it.
func (c *Cache[T]) EnsureFresh(refreshFunc func() (map[string]T, error)) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.isExpired() {
		return nil
	}

	c.logger.Debug("cache expired, refreshing data...")
	newData, err := refreshFunc()
	if err != nil {
		return fmt.Errorf("failed to refresh cache data; calling refreshFunc failed: %w", err)
	}

	c.data = make(map[string]T, len(newData))
	for k, v := range newData {
		c.data[k] = v
	}
	c.lastRefreshed = time.Now()

	return nil
}

// NewCache creates a new cache instance with the given refresh interval
func NewCache[T any](logger *logp.Logger, refreshInterval time.Duration) *Cache[T] {
	return &Cache[T]{
		data:            make(map[string]T),
		refreshInterval: refreshInterval,
		lastRefreshed:   time.Time{},
		logger:          logger,
	}
}

// CacheRegistry holds cached GCP resource information.
type CacheRegistry struct {
	Compute  *Cache[*computepb.Instance]
	CloudSQL *Cache[*sqladmin.DatabaseInstance]
	Redis    *Cache[*redispb.Instance]
	Dataproc *Cache[*dataproc.Cluster]
}

// NewCacheRegistry creates a new cache registry.
func NewCacheRegistry(logger *logp.Logger, refreshInterval time.Duration) *CacheRegistry {
	return &CacheRegistry{
		Compute:  NewCache[*computepb.Instance](logger, refreshInterval),
		CloudSQL: NewCache[*sqladmin.DatabaseInstance](logger, refreshInterval),
		Redis:    NewCache[*redispb.Instance](logger, refreshInterval),
		Dataproc: NewCache[*dataproc.Cluster](logger, refreshInterval),
	}
}

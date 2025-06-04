// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/redis/apiv1/redispb"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/sqladmin/v1"

	"github.com/elastic/elastic-agent-libs/logp"
)

// CacheEntry stores metadata about a specific cache type.
type CacheEntry struct {
	LastRefreshed   time.Time
	RefreshInterval time.Duration
}

// IsExpired checks if the cache entry has exceeded its refresh interval.
func (e *CacheEntry) IsExpired() bool {
	if e.RefreshInterval <= 0 {
		return true
	}
	if e.LastRefreshed.IsZero() {
		return true
	}
	return time.Since(e.LastRefreshed) > e.RefreshInterval
}

// Cache represents a generic cache with metadata and mutex
type Cache[T any] struct {
	data       map[string]T
	meta       CacheEntry
	fetchMutex sync.Mutex
}

// NewCache creates a new cache instance with the given refresh interval
func NewCache[T any](refreshInterval time.Duration) *Cache[T] {
	return &Cache[T]{
		data: make(map[string]T),
		meta: CacheEntry{RefreshInterval: refreshInterval},
	}
}

// CacheRegistry holds cached GCP resource information.
type CacheRegistry struct {
	logger    *logp.Logger
	dataMutex sync.RWMutex

	compute  *Cache[*computepb.Instance]
	cloudsql *Cache[*sqladmin.DatabaseInstance]
	redis    *Cache[*redispb.Instance]
	dataproc *Cache[*dataproc.Cluster]
}

// NewCacheRegistry creates a new cache registry.
func NewCacheRegistry(logger *logp.Logger, refreshInterval time.Duration) *CacheRegistry {
	return &CacheRegistry{
		logger:   logger,
		compute:  NewCache[*computepb.Instance](refreshInterval),
		cloudsql: NewCache[*sqladmin.DatabaseInstance](refreshInterval),
		redis:    NewCache[*redispb.Instance](refreshInterval),
		dataproc: NewCache[*dataproc.Cluster](refreshInterval),
	}
}

// cacheCopy creates a copy of a map
func cacheCopy[T any](instance map[string]T) map[string]T {
	c := make(map[string]T, len(instance))
	for k, v := range instance {
		c[k] = v
	}
	return c
}

// getCache is a generic helper to get cache data
func getCache[T any](r *CacheRegistry, cache *Cache[T]) map[string]T {
	r.dataMutex.RLock()
	defer r.dataMutex.RUnlock()
	return cacheCopy(cache.data)
}

// updateCache is a generic helper to update cache
func updateCache[T any](cache *Cache[T], update map[string]T) {
	if cache.data == nil {
		cache.data = make(map[string]T)
	}
	for k, v := range update {
		cache.data[k] = v
	}
	cache.meta.LastRefreshed = time.Now()
}

// ensureCacheFresh is a generic helper to ensure cache freshness
func ensureCacheFresh[T any](
	r *CacheRegistry,
	cache *Cache[T],
	ctx context.Context,
	cacheName string,
	fetchFunc func(context.Context) (map[string]T, error),
) error {
	r.dataMutex.RLock()
	expired := cache.meta.IsExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debugf("%s cache is fresh.", cacheName)
		return nil
	}

	r.logger.Debugf("%s cache potentially expired, acquiring fetch lock...", cacheName)
	cache.fetchMutex.Lock()
	defer cache.fetchMutex.Unlock()
	r.logger.Debugf("%s fetch lock acquired.", cacheName)

	r.dataMutex.RLock()
	expired = cache.meta.IsExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debugf("%s cache was refreshed by another goroutine while waiting for fetch lock.", cacheName)
		return nil
	}

	r.logger.Infof("%s cache expired, executing fetch function...", cacheName)
	newData, err := fetchFunc(ctx)
	if err != nil {
		r.logger.Errorf("%s cache fetch function failed: %v", cacheName, err)
		return fmt.Errorf("%s cache fetch failed: %w", cacheName, err)
	}

	r.dataMutex.Lock()
	updateCache(cache, newData)
	r.dataMutex.Unlock()

	r.logger.Infof("%s cache successfully refreshed.", cacheName)
	return nil
}

// GetComputeCache returns a copy of the compute instance cache.
func (r *CacheRegistry) GetComputeCache() map[string]*computepb.Instance {
	return getCache(r, r.compute)
}

// UpdateComputeCache appends the provided map to the Compute instance cache.
func (r *CacheRegistry) UpdateComputeCache(update map[string]*computepb.Instance) {
	updateCache(r.compute, update)
}

// GetCloudSQLCache returns a copy of the CloudSQL instance cache.
func (r *CacheRegistry) GetCloudSQLCache() map[string]*sqladmin.DatabaseInstance {
	return getCache(r, r.cloudsql)
}

// UpdateCloudSQLCache appends the provided map to the CloudSQL instance cache.
func (r *CacheRegistry) UpdateCloudSQLCache(update map[string]*sqladmin.DatabaseInstance) {
	updateCache(r.cloudsql, update)
}

// GetRedisCache returns a copy of the Redis instance cache.
func (r *CacheRegistry) GetRedisCache() map[string]*redispb.Instance {
	return getCache(r, r.redis)
}

// UpdateRedisCache appends the provided map to the Redis instance cache.
func (r *CacheRegistry) UpdateRedisCache(update map[string]*redispb.Instance) {
	updateCache(r.redis, update)
}

// GetDataprocCache returns a copy of the Dataproc cluster cache.
func (r *CacheRegistry) GetDataprocCache() map[string]*dataproc.Cluster {
	return getCache(r, r.dataproc)
}

// UpdateDataprocCache appends the provided map to the dataproc instance cache.
func (r *CacheRegistry) UpdateDataprocCache(update map[string]*dataproc.Cluster) {
	updateCache(r.dataproc, update)
}

func (r *CacheRegistry) isComputeCacheExpired() bool {
	return r.compute.meta.IsExpired()
}

func (r *CacheRegistry) isRedisCacheExpired() bool {
	return r.redis.meta.IsExpired()
}

func (r *CacheRegistry) isCloudSQLCacheExpired() bool {
	return r.cloudsql.meta.IsExpired()
}

func (r *CacheRegistry) isDataprocCacheExpired() bool {
	return r.dataproc.meta.IsExpired()
}

// EnsureComputeCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to compute fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureComputeCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*computepb.Instance, error)) error {
	return ensureCacheFresh(r, r.compute, ctx, "Compute", fetchFunc)
}

// EnsureRedisCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to redis fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureRedisCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*redispb.Instance, error)) error {
	return ensureCacheFresh(r, r.redis, ctx, "Redis", fetchFunc)
}

// EnsureCloudSQLCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to cloudsql fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureCloudSQLCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*sqladmin.DatabaseInstance, error)) error {
	return ensureCacheFresh(r, r.cloudsql, ctx, "CloudSQL", fetchFunc)
}

// EnsureDataprocCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to dataproc fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureDataprocCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*dataproc.Cluster, error)) error {
	return ensureCacheFresh(r, r.dataproc, ctx, "Dataproc", fetchFunc)
}

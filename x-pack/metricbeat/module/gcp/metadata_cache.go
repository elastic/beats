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
	"github.com/cenkalti/backoff/v4"
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
	if e.RefreshInterval <= 0 || e.LastRefreshed.IsZero() {
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

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 2 * time.Minute

	backoffWithRetry := backoff.WithMaxRetries(expBackoff, 3)

	var newData map[string]T
	operation := func() error {
		var err error
		newData, err = fetchFunc(ctx)
		if err != nil {
			r.logger.Warnf("%s cache fetch attempt failed: %v", cacheName, err)
			return err
		}
		return nil
	}

	err := backoff.Retry(operation, backoff.WithContext(backoffWithRetry, ctx))
	if err != nil {
		r.logger.Errorf("%s cache fetch function failed after retries: %v", cacheName, err)
		return fmt.Errorf("%s cache fetch failed: %w", cacheName, err)
	}

	r.dataMutex.Lock()
	updateCache(cache, newData)
	r.dataMutex.Unlock()

	r.logger.Infof("%s cache successfully refreshed.", cacheName)
	return nil
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

// getInstanceByID is a generic helper to get a single instance by ID from cache
func getInstanceByID[T any](r *CacheRegistry, cache *Cache[T], id string) (T, bool) {
	r.dataMutex.RLock()
	defer r.dataMutex.RUnlock()
	instance, found := cache.data[id]
	return instance, found
}

// GetComputeInstanceByID returns a compute instance by ID from cache
func (r *CacheRegistry) GetComputeInstanceByID(id string) (*computepb.Instance, bool) {
	return getInstanceByID(r, r.compute, id)
}

// GetCloudSQLInstanceByID returns a CloudSQL instance by ID from cache
func (r *CacheRegistry) GetCloudSQLInstanceByID(id string) (*sqladmin.DatabaseInstance, bool) {
	return getInstanceByID(r, r.cloudsql, id)
}

// GetRedisInstanceByID returns a Redis instance by ID from cache
func (r *CacheRegistry) GetRedisInstanceByID(id string) (*redispb.Instance, bool) {
	return getInstanceByID(r, r.redis, id)
}

// GetDataprocClusterByID returns a Dataproc cluster by ID from cache
func (r *CacheRegistry) GetDataprocClusterByID(id string) (*dataproc.Cluster, bool) {
	return getInstanceByID(r, r.dataproc, id)
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

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
	"github.com/elastic/elastic-agent-libs/logp"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/sqladmin/v1"
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

// CacheRegistry holds cached GCP resource information.
type CacheRegistry struct {
	logger    *logp.Logger
	dataMutex sync.RWMutex

	compute  map[string]*computepb.Instance
	cloudsql map[string]*sqladmin.DatabaseInstance
	redis    map[string]*redispb.Instance
	dataproc map[string]*dataproc.Cluster

	computeMeta  CacheEntry
	cloudsqlMeta CacheEntry
	redisMeta    CacheEntry
	dataprocMeta CacheEntry

	computeFetchMutex  sync.Mutex
	cloudsqlFetchMutex sync.Mutex
	redisFetchMutex    sync.Mutex
	dataprocFetchMutex sync.Mutex
}

// NewCacheRegistry creates a new cache registry.
func NewCacheRegistry(logger *logp.Logger, refreshInterval time.Duration) *CacheRegistry {
	return &CacheRegistry{
		logger:       logger,
		compute:      make(map[string]*computepb.Instance),
		cloudsql:     make(map[string]*sqladmin.DatabaseInstance),
		redis:        make(map[string]*redispb.Instance),
		dataproc:     make(map[string]*dataproc.Cluster),
		computeMeta:  CacheEntry{RefreshInterval: refreshInterval},
		cloudsqlMeta: CacheEntry{RefreshInterval: refreshInterval},
		redisMeta:    CacheEntry{RefreshInterval: refreshInterval},
		dataprocMeta: CacheEntry{RefreshInterval: refreshInterval},
	}
}

// GetComputeCache returns a copy of the compute instance cache.
func (r *CacheRegistry) GetComputeCache() map[string]*computepb.Instance {
	r.dataMutex.RLock()
	defer r.dataMutex.RUnlock()

	cacheCopy := make(map[string]*computepb.Instance, len(r.compute))
	for k, v := range r.compute {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// UpdateComputeCache appends the provided map to the Compute instance cache.
func (r *CacheRegistry) UpdateComputeCache(update map[string]*computepb.Instance) {
	if r.compute == nil {
		r.compute = make(map[string]*computepb.Instance)
	}

	for k, v := range update {
		r.compute[k] = v
	}

	r.computeMeta.LastRefreshed = time.Now()
}

// GetCloudSQLCache returns a copy of the CloudSQL instance cache.
func (r *CacheRegistry) GetCloudSQLCache() map[string]*sqladmin.DatabaseInstance {
	r.dataMutex.RLock()
	defer r.dataMutex.RUnlock()

	cacheCopy := make(map[string]*sqladmin.DatabaseInstance, len(r.cloudsql))
	for k, v := range r.cloudsql {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// UpdateCloudSQLCache appends the provided map to the CloudSQL instance cache.
func (r *CacheRegistry) UpdateCloudSQLCache(update map[string]*sqladmin.DatabaseInstance) {
	if r.cloudsql == nil {
		r.cloudsql = make(map[string]*sqladmin.DatabaseInstance)
	}

	for k, v := range update {
		r.cloudsql[k] = v
	}

	r.cloudsqlMeta.LastRefreshed = time.Now()
}

// GetRedisCache returns a copy of the Redis instance cache.
func (r *CacheRegistry) GetRedisCache() map[string]*redispb.Instance {
	r.dataMutex.RLock()
	defer r.dataMutex.RUnlock()

	cacheCopy := make(map[string]*redispb.Instance, len(r.redis))
	for k, v := range r.redis {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// UpdateRedisCache appends the provided map to the Redis instance cache.
func (r *CacheRegistry) UpdateRedisCache(update map[string]*redispb.Instance) {
	if r.redis == nil {
		r.redis = make(map[string]*redispb.Instance)
	}

	for k, v := range update {
		r.redis[k] = v
	}

	r.redisMeta.LastRefreshed = time.Now()
}

// GetDataprocCache returns a copy of the Dataproc cluster cache.
func (r *CacheRegistry) GetDataprocCache() map[string]*dataproc.Cluster {
	r.dataMutex.RLock()
	defer r.dataMutex.RUnlock()

	cacheCopy := make(map[string]*dataproc.Cluster, len(r.dataproc))
	for k, v := range r.dataproc {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// UpdateDataprocCache appends the provided map to the dataproc instance cache.
func (r *CacheRegistry) UpdateDataprocCache(update map[string]*dataproc.Cluster) {
	if r.dataproc == nil {
		r.dataproc = make(map[string]*dataproc.Cluster)
	}

	for k, v := range update {
		r.dataproc[k] = v
	}

	r.dataprocMeta.LastRefreshed = time.Now()
}

func (r *CacheRegistry) isComputeCacheExpired() bool {
	return r.computeMeta.IsExpired()
}

func (r *CacheRegistry) isRedisCacheExpired() bool {
	return r.redisMeta.IsExpired()
}

func (r *CacheRegistry) isCloudSQLCacheExpired() bool {
	return r.cloudsqlMeta.IsExpired()
}

func (r *CacheRegistry) isDataprocCacheExpired() bool {
	return r.dataprocMeta.IsExpired()
}

// EnsureComputeCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to compute fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureComputeCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*computepb.Instance, error)) error {
	r.dataMutex.RLock()
	expired := r.isComputeCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("Compute cache is fresh.")
		return nil
	}

	r.logger.Debug("Compute cache potentially expired, acquiring fetch lock...")
	r.computeFetchMutex.Lock()
	defer r.computeFetchMutex.Unlock()
	r.logger.Debug("Compute fetch lock acquired.")

	r.dataMutex.RLock()
	expired = r.isComputeCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("Compute cache was refreshed by another goroutine while waiting for fetch lock.")
		return nil
	}

	r.logger.Info("Compute cache expired, executing fetch function...")
	newData, err := fetchFunc(ctx)
	if err != nil {
		r.logger.Errorf("Compute cache fetch function failed: %v", err)
		return fmt.Errorf("compute cache fetch failed: %w", err)
	}

	r.dataMutex.Lock()
	r.UpdateComputeCache(newData)
	r.dataMutex.Unlock()

	r.logger.Info("Compute cache successfully refreshed.")
	return nil
}

// EnsureRedisCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to redis fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureRedisCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*redispb.Instance, error)) error {
	r.dataMutex.RLock()
	expired := r.isRedisCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("Redis cache is fresh.")
		return nil
	}

	r.logger.Debug("Redis cache potentially expired, acquiring fetch lock...")
	r.redisFetchMutex.Lock()
	defer r.redisFetchMutex.Unlock()
	r.logger.Debug("Redis fetch lock acquired.")

	r.dataMutex.RLock()
	expired = r.isRedisCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("Redis cache was refreshed by another goroutine while waiting for fetch lock.")
		return nil
	}

	r.logger.Info("Redis cache expired, executing fetch function...")
	newData, err := fetchFunc(ctx)
	if err != nil {
		r.logger.Errorf("Redis cache fetch function failed: %v", err)
		return fmt.Errorf("redis cache fetch failed: %w", err)
	}

	r.dataMutex.Lock()
	r.UpdateRedisCache(newData)
	r.dataMutex.Unlock()

	r.logger.Info("Redis cache successfully refreshed.")
	return nil
}

// EnsureCloudSQLCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to cloudsql fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureCloudSQLCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*sqladmin.DatabaseInstance, error)) error {
	r.dataMutex.RLock()
	expired := r.isCloudSQLCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("CloudSQL cache is fresh.")
		return nil
	}

	r.logger.Debug("CloudSQL cache potentially expired, acquiring fetch lock...")
	r.cloudsqlFetchMutex.Lock()
	defer r.cloudsqlFetchMutex.Unlock()
	r.logger.Debug("CloudSQL fetch lock acquired.")

	r.dataMutex.RLock()
	expired = r.isCloudSQLCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("CloudSQL cache was refreshed by another goroutine while waiting for fetch lock.")
		return nil
	}

	r.logger.Info("CloudSQL cache expired, executing fetch function...")
	newData, err := fetchFunc(ctx)
	if err != nil {
		r.logger.Errorf("CloudSQL cache fetch function failed: %v", err)
		return fmt.Errorf("cloudsql cache fetch failed: %w", err)
	}

	r.dataMutex.Lock()
	r.UpdateCloudSQLCache(newData)
	r.dataMutex.Unlock()

	r.logger.Info("CloudSQL cache successfully refreshed.")
	return nil
}

// EnsureDataprocCacheFresh checks if the cache is fresh. If not, it acquires a
// lock specific to dataproc fetching, re-checks, and if still expired, calls
// the provided fetchFunc to update the cache.
func (r *CacheRegistry) EnsureDataprocCacheFresh(ctx context.Context, fetchFunc func(context.Context) (map[string]*dataproc.Cluster, error)) error {
	r.dataMutex.RLock()
	expired := r.isDataprocCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("Dataproc cache is fresh.")
		return nil
	}

	r.logger.Debug("Dataproc cache potentially expired, acquiring fetch lock...")
	r.dataprocFetchMutex.Lock()
	defer r.dataprocFetchMutex.Unlock()
	r.logger.Debug("Dataproc fetch lock acquired.")

	r.dataMutex.RLock()
	expired = r.isDataprocCacheExpired()
	r.dataMutex.RUnlock()

	if !expired {
		r.logger.Debug("Dataproc cache was refreshed by another goroutine while waiting for fetch lock.")
		return nil
	}

	r.logger.Info("Dataproc cache expired, executing fetch function...")
	newData, err := fetchFunc(ctx)
	if err != nil {
		r.logger.Errorf("Dataproc cache fetch function failed: %v", err)
		return fmt.Errorf("dataproc cache fetch failed: %w", err)
	}

	r.dataMutex.Lock()
	r.UpdateDataprocCache(newData)
	r.dataMutex.Unlock()

	r.logger.Info("Dataproc cache successfully refreshed.")
	return nil
}

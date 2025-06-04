// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/redis/apiv1/redispb"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/sqladmin/v1"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		entry    CacheEntry
		expected bool
	}{
		{
			name: "zero refresh interval",
			entry: CacheEntry{
				RefreshInterval: 0,
				LastRefreshed:   time.Now(),
			},
			expected: true,
		},
		{
			name: "negative refresh interval",
			entry: CacheEntry{
				RefreshInterval: -1 * time.Hour,
				LastRefreshed:   time.Now(),
			},
			expected: true,
		},
		{
			name: "zero last refreshed",
			entry: CacheEntry{
				RefreshInterval: time.Hour,
				LastRefreshed:   time.Time{},
			},
			expected: true,
		},
		{
			name: "expired cache",
			entry: CacheEntry{
				RefreshInterval: time.Hour,
				LastRefreshed:   time.Now().Add(-2 * time.Hour),
			},
			expected: true,
		},
		{
			name: "fresh cache",
			entry: CacheEntry{
				RefreshInterval: time.Hour,
				LastRefreshed:   time.Now().Add(-30 * time.Minute),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.IsExpired())
		})
	}
}

func TestNewCacheRegistry(t *testing.T) {
	logger := logp.NewLogger("test")
	refreshInterval := 5 * time.Minute

	registry := NewCacheRegistry(logger, refreshInterval)

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.compute)
	assert.NotNil(t, registry.cloudsql)
	assert.NotNil(t, registry.redis)
	assert.NotNil(t, registry.dataproc)
	assert.Equal(t, refreshInterval, registry.compute.meta.RefreshInterval)
	assert.Equal(t, refreshInterval, registry.cloudsql.meta.RefreshInterval)
	assert.Equal(t, refreshInterval, registry.redis.meta.RefreshInterval)
	assert.Equal(t, refreshInterval, registry.dataproc.meta.RefreshInterval)
}

func TestComputeCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	cache := registry.GetComputeCache()
	assert.Empty(t, cache)

	testData := map[string]*computepb.Instance{
		"instance-1": {Name: stringPtr("test-instance-1")},
		"instance-2": {Name: stringPtr("test-instance-2")},
	}

	registry.UpdateComputeCache(testData)

	cache = registry.GetComputeCache()
	assert.Len(t, cache, 2)
	assert.Equal(t, "test-instance-1", *cache["instance-1"].Name)
	assert.Equal(t, "test-instance-2", *cache["instance-2"].Name)

	cache["instance-3"] = &computepb.Instance{Name: stringPtr("test-instance-3")}
	newCache := registry.GetComputeCache()
	assert.Len(t, newCache, 2) // Original cache should still have 2 items
}

func TestRedisCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	cache := registry.GetRedisCache()
	assert.Empty(t, cache)

	testData := map[string]*redispb.Instance{
		"redis-1": {Name: "projects/test/locations/us-central1/instances/redis-1"},
		"redis-2": {Name: "projects/test/locations/us-central1/instances/redis-2"},
	}

	registry.UpdateRedisCache(testData)

	cache = registry.GetRedisCache()
	assert.Len(t, cache, 2)
	assert.Equal(t, "projects/test/locations/us-central1/instances/redis-1", cache["redis-1"].Name)
}

func TestCloudSQLCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	testData := map[string]*sqladmin.DatabaseInstance{
		"sql-1": {Name: "sql-instance-1"},
		"sql-2": {Name: "sql-instance-2"},
	}

	registry.UpdateCloudSQLCache(testData)
	cache := registry.GetCloudSQLCache()
	assert.Len(t, cache, 2)
	assert.Equal(t, "sql-instance-1", cache["sql-1"].Name)
}

func TestDataprocCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	testData := map[string]*dataproc.Cluster{
		"cluster-1": {ClusterName: "dataproc-cluster-1"},
		"cluster-2": {ClusterName: "dataproc-cluster-2"},
	}

	registry.UpdateDataprocCache(testData)
	cache := registry.GetDataprocCache()
	assert.Len(t, cache, 2)
	assert.Equal(t, "dataproc-cluster-1", cache["cluster-1"].ClusterName)
}

func TestEnsureComputeCacheFresh(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 100*time.Millisecond)

	fetchCallCount := 0
	fetchFunc := func(ctx context.Context) (map[string]*computepb.Instance, error) {
		fetchCallCount++
		return map[string]*computepb.Instance{
			"instance-1": {Name: stringPtr("fetched-instance")},
		}, nil
	}

	// First call should fetch
	err := registry.EnsureComputeCacheFresh(context.Background(), fetchFunc)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchCallCount)

	// Second call should not fetch (cache is fresh)
	err = registry.EnsureComputeCacheFresh(context.Background(), fetchFunc)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchCallCount)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should fetch again
	err = registry.EnsureComputeCacheFresh(context.Background(), fetchFunc)
	require.NoError(t, err)
	assert.Equal(t, 2, fetchCallCount)
}

func TestEnsureCacheFresh_Error(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 100*time.Millisecond)

	expectedErr := fmt.Errorf("fetch failed")
	fetchFunc := func(ctx context.Context) (map[string]*computepb.Instance, error) {
		return nil, expectedErr
	}

	err := registry.EnsureComputeCacheFresh(context.Background(), fetchFunc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetch failed")
}

func TestConcurrentAccess(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 50*time.Millisecond)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				registry.UpdateComputeCache(map[string]*computepb.Instance{
					fmt.Sprintf("instance-%d-%d", id, j): {Name: stringPtr(fmt.Sprintf("test-%d-%d", id, j))},
				})
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = registry.GetComputeCache()
			}
		}()
	}

	// Concurrent freshness checkers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fetchFunc := func(ctx context.Context) (map[string]*computepb.Instance, error) {
				return map[string]*computepb.Instance{
					fmt.Sprintf("fresh-%d", id): {Name: stringPtr(fmt.Sprintf("fresh-instance-%d", id))},
				}, nil
			}
			for j := 0; j < iterations/10; j++ {
				_ = registry.EnsureComputeCacheFresh(context.Background(), fetchFunc)
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is not corrupted
	cache := registry.GetComputeCache()
	assert.NotNil(t, cache)
	// Should have some data
	assert.NotEmpty(t, cache)
}

func TestMultipleCacheTypes(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	registry.UpdateComputeCache(map[string]*computepb.Instance{
		"compute-1": {Name: stringPtr("compute-instance")},
	})
	registry.UpdateRedisCache(map[string]*redispb.Instance{
		"redis-1": {Name: "redis-instance"},
	})
	registry.UpdateCloudSQLCache(map[string]*sqladmin.DatabaseInstance{
		"sql-1": {Name: "sql-instance"},
	})
	registry.UpdateDataprocCache(map[string]*dataproc.Cluster{
		"cluster-1": {ClusterName: "dataproc-cluster"},
	})

	// Verify each cache is independent
	computeCache := registry.GetComputeCache()
	assert.Len(t, computeCache, 1)
	assert.Contains(t, computeCache, "compute-1")

	redisCache := registry.GetRedisCache()
	assert.Len(t, redisCache, 1)
	assert.Contains(t, redisCache, "redis-1")

	sqlCache := registry.GetCloudSQLCache()
	assert.Len(t, sqlCache, 1)
	assert.Contains(t, sqlCache, "sql-1")

	dataprocCache := registry.GetDataprocCache()
	assert.Len(t, dataprocCache, 1)
	assert.Contains(t, dataprocCache, "cluster-1")
}

func TestCacheExpiredCheck(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 100*time.Millisecond)

	// Initially all caches should be expired
	assert.True(t, registry.isComputeCacheExpired())
	assert.True(t, registry.isRedisCacheExpired())
	assert.True(t, registry.isCloudSQLCacheExpired())
	assert.True(t, registry.isDataprocCacheExpired())

	// Update caches
	registry.UpdateComputeCache(map[string]*computepb.Instance{"test": {}})
	registry.UpdateRedisCache(map[string]*redispb.Instance{"test": {}})
	registry.UpdateCloudSQLCache(map[string]*sqladmin.DatabaseInstance{"test": {}})
	registry.UpdateDataprocCache(map[string]*dataproc.Cluster{"test": {}})

	// Now caches should not be expired
	assert.False(t, registry.isComputeCacheExpired())
	assert.False(t, registry.isRedisCacheExpired())
	assert.False(t, registry.isCloudSQLCacheExpired())
	assert.False(t, registry.isDataprocCacheExpired())

	time.Sleep(150 * time.Millisecond)

	// Caches should be expired again
	assert.True(t, registry.isComputeCacheExpired())
	assert.True(t, registry.isRedisCacheExpired())
	assert.True(t, registry.isCloudSQLCacheExpired())
	assert.True(t, registry.isDataprocCacheExpired())
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

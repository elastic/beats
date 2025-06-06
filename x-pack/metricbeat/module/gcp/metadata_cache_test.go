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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
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

	_, found := registry.GetComputeInstanceByID("instance-1")
	assert.False(t, found)

	testData := map[string]*computepb.Instance{
		"instance-1": {Name: stringPtr("test-instance-1")},
		"instance-2": {Name: stringPtr("test-instance-2")},
	}

	registry.UpdateComputeCache(testData)

	instance1, found := registry.GetComputeInstanceByID("instance-1")
	assert.True(t, found)
	assert.Equal(t, "test-instance-1", *instance1.Name)

	instance2, found := registry.GetComputeInstanceByID("instance-2")
	assert.True(t, found)
	assert.Equal(t, "test-instance-2", *instance2.Name)

	_, found = registry.GetComputeInstanceByID("instance-3")
	assert.False(t, found)
}

func TestRedisCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	_, found := registry.GetRedisInstanceByID("redis-1")
	assert.False(t, found)

	testData := map[string]*redispb.Instance{
		"redis-1": {Name: "projects/test/locations/us-central1/instances/redis-1"},
		"redis-2": {Name: "projects/test/locations/us-central1/instances/redis-2"},
	}

	registry.UpdateRedisCache(testData)

	instance1, found := registry.GetRedisInstanceByID("redis-1")
	assert.True(t, found)
	assert.Equal(t, "projects/test/locations/us-central1/instances/redis-1", instance1.Name)

	instance2, found := registry.GetRedisInstanceByID("redis-2")
	assert.True(t, found)
	assert.Equal(t, "projects/test/locations/us-central1/instances/redis-2", instance2.Name)
}

func TestCloudSQLCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	testData := map[string]*sqladmin.DatabaseInstance{
		"sql-1": {Name: "sql-instance-1"},
		"sql-2": {Name: "sql-instance-2"},
	}

	registry.UpdateCloudSQLCache(testData)

	instance1, found := registry.GetCloudSQLInstanceByID("sql-1")
	assert.True(t, found)
	assert.Equal(t, "sql-instance-1", instance1.Name)

	instance2, found := registry.GetCloudSQLInstanceByID("sql-2")
	assert.True(t, found)
	assert.Equal(t, "sql-instance-2", instance2.Name)
}

func TestDataprocCache(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	testData := map[string]*dataproc.Cluster{
		"cluster-1": {ClusterName: "dataproc-cluster-1"},
		"cluster-2": {ClusterName: "dataproc-cluster-2"},
	}

	registry.UpdateDataprocCache(testData)

	cluster1, found := registry.GetDataprocClusterByID("cluster-1")
	assert.True(t, found)
	assert.Equal(t, "dataproc-cluster-1", cluster1.ClusterName)

	cluster2, found := registry.GetDataprocClusterByID("cluster-2")
	assert.True(t, found)
	assert.Equal(t, "dataproc-cluster-2", cluster2.ClusterName)
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

	instance, found := registry.GetComputeInstanceByID("instance-1")
	assert.True(t, found)
	assert.Equal(t, "fetched-instance", *instance.Name)

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

	var addedIDs sync.Map

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				instanceID := fmt.Sprintf("instance-%d-%d", id, j)
				registry.UpdateComputeCache(map[string]*computepb.Instance{
					instanceID: {Name: stringPtr(fmt.Sprintf("test-%d-%d", id, j))},
				})
				addedIDs.Store(instanceID, true)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				instanceID := fmt.Sprintf("instance-%d-%d", j%5, j)
				_, _ = registry.GetComputeInstanceByID(instanceID)
			}
		}()
	}

	// Concurrent freshness checkers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fetchFunc := func(ctx context.Context) (map[string]*computepb.Instance, error) {
				instanceID := fmt.Sprintf("fresh-%d", id)
				addedIDs.Store(instanceID, true)
				return map[string]*computepb.Instance{
					instanceID: {Name: stringPtr(fmt.Sprintf("fresh-instance-%d", id))},
				}, nil
			}
			for j := 0; j < iterations/10; j++ {
				_ = registry.EnsureComputeCacheFresh(context.Background(), fetchFunc)
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	instancesFound := 0
	addedIDs.Range(func(key, value interface{}) bool {
		if instance, found := registry.GetComputeInstanceByID(key.(string)); found && instance != nil {
			instancesFound++
		}
		return true
	})
	assert.Greater(t, instancesFound, 0, "Should have found at least some instances")
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
	computeInstance, found := registry.GetComputeInstanceByID("compute-1")
	assert.True(t, found)
	assert.Equal(t, "compute-instance", *computeInstance.Name)

	redisInstance, found := registry.GetRedisInstanceByID("redis-1")
	assert.True(t, found)
	assert.Equal(t, "redis-instance", redisInstance.Name)

	sqlInstance, found := registry.GetCloudSQLInstanceByID("sql-1")
	assert.True(t, found)
	assert.Equal(t, "sql-instance", sqlInstance.Name)

	dataprocCluster, found := registry.GetDataprocClusterByID("cluster-1")
	assert.True(t, found)
	assert.Equal(t, "dataproc-cluster", dataprocCluster.ClusterName)

	_, found = registry.GetComputeInstanceByID("redis-1")
	assert.False(t, found)

	_, found = registry.GetRedisInstanceByID("compute-1")
	assert.False(t, found)
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

func TestGetInstanceByID_NotFound(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	_, found := registry.GetComputeInstanceByID("non-existent")
	assert.False(t, found)

	_, found = registry.GetRedisInstanceByID("non-existent")
	assert.False(t, found)

	_, found = registry.GetCloudSQLInstanceByID("non-existent")
	assert.False(t, found)

	_, found = registry.GetDataprocClusterByID("non-existent")
	assert.False(t, found)
}

func TestEnsureCacheFresh_WithRetries(t *testing.T) {
	tests := []struct {
		name              string
		failureCount      int
		expectedAttempts  int
		expectError       bool
		contextTimeout    time.Duration
		expectedErrString string
	}{
		{
			name:             "succeeds on first attempt",
			failureCount:     0,
			expectedAttempts: 1,
			expectError:      false,
		},
		{
			name:             "succeeds after 1 retry",
			failureCount:     1,
			expectedAttempts: 2,
			expectError:      false,
		},
		{
			name:             "succeeds after 2 retries",
			failureCount:     2,
			expectedAttempts: 3,
			expectError:      false,
		},
		{
			name:             "succeeds on last retry (3rd attempt)",
			failureCount:     3,
			expectedAttempts: 4,
			expectError:      false,
		},
		{
			name:              "fails after max retries",
			failureCount:      4,
			expectedAttempts:  4, // 1 initial + 3 retries
			expectError:       true,
			expectedErrString: "fetch failed",
		},
		{
			name:              "context cancelled during retry",
			failureCount:      10,
			expectedAttempts:  2,
			expectError:       true,
			contextTimeout:    1500 * time.Millisecond,
			expectedErrString: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logp.NewLogger("test")
			registry := NewCacheRegistry(logger, 5*time.Minute)

			attemptCount := 0
			fetchFunc := func(ctx context.Context) (map[string]*computepb.Instance, error) {
				attemptCount++
				if attemptCount <= tt.failureCount {
					return nil, fmt.Errorf("simulated fetch error #%d", attemptCount)
				}
				return map[string]*computepb.Instance{
					"instance-1": {Name: stringPtr("successful-instance")},
				}, nil
			}

			ctx := context.Background()
			if tt.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
			}

			err := registry.EnsureComputeCacheFresh(ctx, fetchFunc)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrString != "" {
					assert.Contains(t, err.Error(), tt.expectedErrString)
				}
			} else {
				assert.NoError(t, err)
				instance, found := registry.GetComputeInstanceByID("instance-1")
				assert.True(t, found)
				assert.Equal(t, "successful-instance", *instance.Name)
			}

			assert.Equal(t, tt.expectedAttempts, attemptCount,
				"Expected %d attempts but got %d", tt.expectedAttempts, attemptCount)
		})
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

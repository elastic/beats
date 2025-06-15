// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/redis/apiv1/redispb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/sqladmin/v1"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestNewCache(t *testing.T) {
	logger := logp.NewLogger("test")
	refreshInterval := 5 * time.Minute

	cache := NewCache[string](logger, refreshInterval)

	require.NotNil(t, cache)
	assert.NotNil(t, cache.data)
	assert.Equal(t, refreshInterval, cache.refreshInterval)
	assert.Equal(t, time.Time{}, cache.lastRefreshed)
	assert.Equal(t, logger, cache.logger)
}

func TestCache_Get(t *testing.T) {
	logger := logp.NewLogger("test")
	cache := NewCache[string](logger, 5*time.Minute)

	value, found := cache.Get("key1")
	assert.False(t, found)
	assert.Equal(t, "", value)

	cache.data["key1"] = "value1"
	cache.data["key2"] = "value2"

	value, found = cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	value, found = cache.Get("key3")
	assert.False(t, found)
	assert.Equal(t, "", value)
}

func TestCache_isExpired(t *testing.T) {
	logger := logp.NewLogger("test")
	cache := NewCache[string](logger, 100*time.Millisecond)

	// initially should be expired (zero time)
	assert.True(t, cache.isExpired())

	// set last refreshed to now
	cache.lastRefreshed = time.Now()
	assert.False(t, cache.isExpired())

	// wait for expiration
	time.Sleep(150 * time.Millisecond)
	assert.True(t, cache.isExpired())
}

func TestCache_EnsureFresh(t *testing.T) {
	logger := logp.NewLogger("test")
	cache := NewCache[string](logger, 1*time.Hour)

	refreshCount := 0
	refreshFunc := func() (map[string]string, error) {
		refreshCount++
		return map[string]string{
			"key1": "value1",
			"key2": "value2",
		}, nil
	}

	// first call should refresh - cache is expired
	err := cache.EnsureFresh(refreshFunc)
	require.NoError(t, err)
	assert.Equal(t, 1, refreshCount)
	assert.Len(t, cache.data, 2)
	assert.Equal(t, "value1", cache.data["key1"])
	assert.Equal(t, "value2", cache.data["key2"])

	// second call should not refresh - cache is fresh
	err = cache.EnsureFresh(refreshFunc)
	require.NoError(t, err)
	assert.Equal(t, 1, refreshCount)
}

func TestCache_EnsureFresh_Error(t *testing.T) {
	logger := logp.NewLogger("test")
	cache := NewCache[string](logger, 1*time.Hour)

	refreshFunc := func() (map[string]string, error) {
		return nil, errors.New("refresh failed")
	}

	err := cache.EnsureFresh(refreshFunc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to refresh cache data")
	assert.Contains(t, err.Error(), "refresh failed")
}

func TestCache_EnsureFresh_UpdatesData(t *testing.T) {
	logger := logp.NewLogger("test")
	cache := NewCache[string](logger, 100*time.Millisecond)

	cache.data["old"] = "oldValue"
	cache.lastRefreshed = time.Now()

	// wait for expiration
	time.Sleep(150 * time.Millisecond)

	refreshFunc := func() (map[string]string, error) {
		return map[string]string{
			"new": "newValue",
		}, nil
	}

	err := cache.EnsureFresh(refreshFunc)
	require.NoError(t, err)

	_, found := cache.Get("old")
	assert.False(t, found)

	value, found := cache.Get("new")
	assert.True(t, found)
	assert.Equal(t, "newValue", value)
}

func TestCache_EnsureFresh_ZeroRefreshInterval(t *testing.T) {
	// this test simulates how the cache's EnsureFresh call behaves
	// when the cache is actually disabled. we still use the cache's map to store metadata,
	// but with the refresh interval set to 0, the data always expires and gets cleared out,
	// so it's repopulated every time.

	logger := logp.NewLogger("test")
	cache := NewCache[string](logger, 0)

	cache.data["old"] = "oldValue"
	cache.lastRefreshed = time.Now()

	refreshFunc := func() (map[string]string, error) {
		return map[string]string{
			"new": "newValue",
		}, nil
	}

	err := cache.EnsureFresh(refreshFunc)
	require.NoError(t, err)

	_, found := cache.Get("old")
	assert.False(t, found)

	refreshFunc = func() (map[string]string, error) {
		return map[string]string{
			"new2": "newValue",
		}, nil
	}

	err = cache.EnsureFresh(refreshFunc)
	require.NoError(t, err)

	_, found = cache.Get("old")
	assert.False(t, found)

	_, found = cache.Get("new")
	assert.False(t, found)

	value, found := cache.Get("new2")
	assert.True(t, found)
	assert.Equal(t, "newValue", value)
}

func TestNewCacheRegistry(t *testing.T) {
	logger := logp.NewLogger("test")
	refreshInterval := 5 * time.Minute

	registry := NewCacheRegistry(logger, refreshInterval)

	require.NotNil(t, registry)
	assert.NotNil(t, registry.Compute)
	assert.NotNil(t, registry.CloudSQL)
	assert.NotNil(t, registry.Redis)
	assert.NotNil(t, registry.Dataproc)
}

func TestCacheRegistry_TypedCaches(t *testing.T) {
	logger := logp.NewLogger("test")
	registry := NewCacheRegistry(logger, 5*time.Minute)

	computeInstance := &computepb.Instance{Name: stringPtr("test-instance")}
	registry.Compute.data["instance1"] = computeInstance
	retrieved, found := registry.Compute.Get("instance1")
	assert.True(t, found)
	assert.Equal(t, computeInstance, retrieved)

	sqlInstance := &sqladmin.DatabaseInstance{Name: "test-db"}
	registry.CloudSQL.data["db1"] = sqlInstance
	retrievedSQL, found := registry.CloudSQL.Get("db1")
	assert.True(t, found)
	assert.Equal(t, sqlInstance, retrievedSQL)

	redisInstance := &redispb.Instance{Name: "test-redis"}
	registry.Redis.data["redis1"] = redisInstance
	retrievedRedis, found := registry.Redis.Get("redis1")
	assert.True(t, found)
	assert.Equal(t, redisInstance, retrievedRedis)

	dataprocCluster := &dataproc.Cluster{ClusterName: "test-cluster"}
	registry.Dataproc.data["cluster1"] = dataprocCluster
	retrievedCluster, found := registry.Dataproc.Get("cluster1")
	assert.True(t, found)
	assert.Equal(t, dataprocCluster, retrievedCluster)
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

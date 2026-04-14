// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

const (
	DefaultLRUCacheSize = 100
	// DefaultLRUCacheTTL defines the default time-to-live for entries in the LRU cache.
	DefaultLRUCacheTTL = 1 * time.Minute
)

// A Simple LRU cache implementation for ntfs tables.
// Joins and complex queries can result in multiple calls to getVolumes and getPartitions, which can be expensive.
// By caching the results of these functions, we can improve performance for subsequent calls with the same parameters.
// The cache is implemented as a simple LRU cache with a configurable size and TTL.
type NtfsCache struct {
	volumes    *expirable.LRU[string, *Volume]
	partitions *expirable.LRU[string, []*Partition]
}

var (
	ntfsCache *NtfsCache
	once      sync.Once
)

// newNtfsCache creates a new NtfsCache instance with the default configuration.
func newNtfsCache() *NtfsCache {
	closeVolume := func(key string, value *Volume) {
		value.Close()
	}
	cache := &NtfsCache{
		// volumes cache is small since we typically have a limited number of mounted volumes, and we want to ensure timely cleanup of NTFS sessions when volumes are evicted.
		volumes: expirable.NewLRU[string, *Volume](DefaultLRUCacheSize, closeVolume, 2*time.Minute),

		// partitions cache can be larger since we may have many physical drives with multiple partitions, and the data is less expensive to keep around.
		partitions: expirable.NewLRU[string, []*Partition](DefaultLRUCacheSize, nil, 2*time.Minute),
	}
	return cache
}

// GetNtfsCache returns the singleton NtfsCache instance.
func GetNtfsCache() *NtfsCache {
	once.Do(func() {
		ntfsCache = newNtfsCache()
	})
	return ntfsCache
}

func CacheVolume(driveLetter string, volume *Volume) {
	cache := GetNtfsCache()
	cache.volumes.Add(driveLetter, volume)
}

func GetCachedVolumes(driveLetter string) (*Volume, bool) {
	cache := GetNtfsCache()
	volume, found := cache.volumes.Get(driveLetter)
	return volume, found
}

func GetCachedPartitions(physicalDrive string) ([]*Partition, bool) {
	cache := GetNtfsCache()
	partitions, found := cache.partitions.Get(physicalDrive)
	return partitions, found
}

func CachePartitions(physicalDrive string, partitions []*Partition) {
	cache := GetNtfsCache()
	cache.partitions.Add(physicalDrive, partitions)
}

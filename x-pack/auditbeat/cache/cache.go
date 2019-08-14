// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cache

// Cache is just a map being used as a cache.
type Cache struct {
	hashMap map[uint64]Cacheable
}

// Cacheable is the interface items stored in Cache need to implement.
type Cacheable interface {
	Hash() uint64
}

// New creates a new cache.
func New() *Cache {
	return &Cache{
		hashMap: make(map[uint64]Cacheable),
	}
}

// IsEmpty checks if the cache is empty.
func (cache *Cache) IsEmpty() bool {
	return len(cache.hashMap) == 0
}

// DiffAndUpdateCache takes a list of new items to cache, compares them to the current
// cache contents, and returns both items new to the cache and items that are in the cache
// but missing in the new data.
func (cache *Cache) DiffAndUpdateCache(current []Cacheable) (new, missing []interface{}) {
	// Create hashmap of incoming Cacheables to avoid calling Hash() on each one many times
	currentMap := make(map[uint64]Cacheable, len(current))

	for _, currentValue := range current {
		currentMap[currentValue.Hash()] = currentValue
	}

	// Check for and delete missing - what is no longer in current that was in the cache
	for cacheHash, cacheValue := range cache.hashMap {
		_, found := currentMap[cacheHash]

		if !found {
			missing = append(missing, cacheValue)
			delete(cache.hashMap, cacheHash)
		}
	}

	// Check for new - what is in current but not in cache
	for currentHash, currentValue := range currentMap {
		if _, contains := cache.hashMap[currentHash]; !contains {
			new = append(new, currentValue)
			cache.hashMap[currentHash] = currentValue
		}
	}

	return
}

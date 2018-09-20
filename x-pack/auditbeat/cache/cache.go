// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cache

// Cache is just a map being used as a cache.
type Cache struct {
	hashMap map[string]Cacheable
}

// Cacheable is the interface items stored in Cache need to implement.
type Cacheable interface {
	Hash() string
}

// New creates a new cache.
func New() *Cache {
	return &Cache{
		hashMap: make(map[string]Cacheable),
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
	// Check for and delete missing - what is no longer in current that was in the cache
	for cacheKey, cacheValue := range cache.hashMap {
		found := false
		for _, currentValue := range current {
			if currentValue.Hash() == cacheKey {
				found = true
				break
			}
		}

		if !found {
			missing = append(missing, cacheValue)
			delete(cache.hashMap, cacheKey)
		}
	}

	// Check for new - what is in current but not in cache
	for _, currentValue := range current {
		if _, contains := cache.hashMap[currentValue.Hash()]; !contains {
			new = append(new, currentValue)
			cache.hashMap[currentValue.Hash()] = currentValue
		}
	}

	return
}

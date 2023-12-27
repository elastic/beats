// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dns

import (
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

type successRecord struct {
	data    []string
	expires time.Time
}

func (r successRecord) IsExpired(now time.Time) bool {
	return now.After(r.expires)
}

type successCache struct {
	sync.RWMutex
	data          map[string]successRecord
	maxSize       int
	minSuccessTTL time.Duration
}

func (c *successCache) set(now time.Time, key string, result *result) {
	c.Lock()
	defer c.Unlock()

	if len(c.data) >= c.maxSize {
		c.evict()
	}

	c.data[key] = successRecord{
		data:    result.Data,
		expires: now.Add(time.Duration(result.TTL) * time.Second),
	}
}

// evict removes a single random key from the cache.
func (c *successCache) evict() {
	var key string
	for k := range c.data {
		key = k
		break
	}
	delete(c.data, key)
}

func (c *successCache) get(now time.Time, key string) *result {
	c.RLock()
	defer c.RUnlock()

	r, found := c.data[key]
	if found && !r.IsExpired(now) {
		return &result{r.data, uint32(r.expires.Sub(now) / time.Second)}
	}
	return nil
}

type failureRecord struct {
	error
	expires time.Time
}

func (r failureRecord) IsExpired(now time.Time) bool {
	return now.After(r.expires)
}

type failureCache struct {
	sync.RWMutex
	data       map[string]failureRecord
	maxSize    int
	failureTTL time.Duration
}

func (c *failureCache) set(now time.Time, key string, err error) {
	c.Lock()
	defer c.Unlock()
	if len(c.data) >= c.maxSize {
		c.evict()
	}

	c.data[key] = failureRecord{
		error:   err,
		expires: now.Add(c.failureTTL),
	}
}

// evict removes a single random key from the cache.
func (c *failureCache) evict() {
	var key string
	for k := range c.data {
		key = k
		break
	}
	delete(c.data, key)
}

func (c *failureCache) get(now time.Time, key string) error {
	c.RLock()
	defer c.RUnlock()

	r, found := c.data[key]
	if found && !r.IsExpired(now) {
		return r.error
	}
	return nil
}

type cachedError struct {
	err error
}

func (ce *cachedError) Error() string { return ce.err.Error() + " (from failure cache)" }
func (ce *cachedError) Cause() error  { return ce.err }

// lookupCache is a cache for storing and retrieving the results of
// DNS queries. It caches the results of queries regardless of their
// outcome (success or failure).
type lookupCache struct {
	success  *successCache
	failure  *failureCache
	resolver resolver
	stats    cacheStats
}

type cacheStats struct {
	Hit  *monitoring.Int
	Miss *monitoring.Int
}

// newLookupCache returns a new cache.
func newLookupCache(reg *monitoring.Registry, conf cacheConfig, resolver resolver) (*lookupCache, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	c := &lookupCache{
		success: &successCache{
			data:          make(map[string]successRecord, conf.SuccessCache.InitialCapacity),
			maxSize:       conf.SuccessCache.MaxCapacity,
			minSuccessTTL: conf.SuccessCache.MinTTL,
		},
		failure: &failureCache{
			data:       make(map[string]failureRecord, conf.FailureCache.InitialCapacity),
			maxSize:    conf.FailureCache.MaxCapacity,
			failureTTL: conf.FailureCache.TTL,
		},
		resolver: resolver,
		stats: cacheStats{
			Hit:  monitoring.NewInt(reg, "hits"),
			Miss: monitoring.NewInt(reg, "misses"),
		},
	}

	return c, nil
}

// Lookup performs a lookup on the given query string. A cached result
// will be returned if it is contained in the cache, otherwise a lookup is
// performed.
func (c lookupCache) Lookup(q string, qt queryType) (*result, error) {
	now := time.Now()

	r := c.success.get(now, q)
	if r != nil {
		c.stats.Hit.Inc()
		return r, nil
	}

	err := c.failure.get(now, q)
	if err != nil {
		c.stats.Hit.Inc()
		return nil, err
	}
	c.stats.Miss.Inc()

	r, err = c.resolver.Lookup(q, qt)
	if err != nil {
		c.failure.set(now, q, &cachedError{err})
		return nil, err
	}

	// We set the result TTL to the minimum TTL in case it is less than that.
	r.TTL = max(r.TTL, uint32(c.success.minSuccessTTL/time.Second))

	c.success.set(now, q, r)
	return r, nil
}

func max(a, b uint32) uint32 {
	if a >= b {
		return a
	}
	return b
}

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

	"github.com/elastic/beats/libbeat/monitoring"
)

type ptrRecord struct {
	host    string
	expires time.Time
}

func (r ptrRecord) IsExpired(now time.Time) bool {
	return now.After(r.expires)
}

type ptrCache struct {
	sync.RWMutex
	data    map[string]ptrRecord
	maxSize int
}

func (c *ptrCache) set(now time.Time, key string, ptr *PTR) {
	c.Lock()
	defer c.Unlock()

	if len(c.data) >= c.maxSize {
		c.evict()
	}

	c.data[key] = ptrRecord{
		host:    ptr.Host,
		expires: now.Add(time.Duration(ptr.TTL) * time.Second),
	}
}

// evict removes a single random key from the cache.
func (c *ptrCache) evict() {
	var key string
	for k := range c.data {
		key = k
		break
	}
	delete(c.data, key)
}

func (c *ptrCache) get(now time.Time, key string) *PTR {
	c.RLock()
	defer c.RUnlock()

	r, found := c.data[key]
	if found && !r.IsExpired(now) {
		return &PTR{r.host, uint32(r.expires.Sub(now) / time.Second)}
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

// PTRLookupCache is a cache for storing and retrieving the results of
// reverse DNS queries. It caches the results of queries regardless of their
// outcome (success or failure).
type PTRLookupCache struct {
	success    *ptrCache
	failure    *failureCache
	failureTTL time.Duration
	resolver   PTRResolver
	stats      cacheStats
}

type cacheStats struct {
	Hit  *monitoring.Int
	Miss *monitoring.Int
}

// NewPTRLookupCache returns a new cache.
func NewPTRLookupCache(reg *monitoring.Registry, conf CacheConfig, resolver PTRResolver) (*PTRLookupCache, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	c := &PTRLookupCache{
		success: &ptrCache{
			data:    make(map[string]ptrRecord, conf.SuccessCache.InitialCapacity),
			maxSize: conf.SuccessCache.MaxCapacity,
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

// LookupPTR performs a reverse lookup on the given IP address. A cached result
// will be returned if it is contained in the cache, otherwise a lookup is
// performed.
func (c PTRLookupCache) LookupPTR(ip string) (*PTR, error) {
	now := time.Now()

	ptr := c.success.get(now, ip)
	if ptr != nil {
		c.stats.Hit.Inc()
		return ptr, nil
	}

	err := c.failure.get(now, ip)
	if err != nil {
		c.stats.Hit.Inc()
		return nil, err
	}
	c.stats.Miss.Inc()

	ptr, err = c.resolver.LookupPTR(ip)
	if err != nil {
		c.failure.set(now, ip, &cachedError{err})
		return nil, err
	}

	c.success.set(now, ip, ptr)
	return ptr, nil
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

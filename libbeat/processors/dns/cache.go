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
	data    map[string]ptrRecord
	maxSize int
}

func (c *ptrCache) set(now time.Time, key string, ptr *PTR) {
	if len(c.data) >= c.maxSize {
		c.evict()
	}

	c.data[key] = ptrRecord{
		host:    ptr.Host,
		expires: now.Add(time.Duration(ptr.TTL) * time.Second),
	}
}

func (c *ptrCache) evict() {
	var key string
	for k := range c.data {
		key = k
		break
	}
	delete(c.data, key)
}

func (c *ptrCache) get(now time.Time, key string) *PTR {
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
	data       map[string]failureRecord
	maxSize    int
	failureTTL time.Duration
	stats      cacheStats
}

func (c *failureCache) set(now time.Time, key string, err error) {
	if len(c.data) >= c.maxSize {
		c.evict()
	}

	c.data[key] = failureRecord{
		error:   err,
		expires: now.Add(c.failureTTL),
	}
}

func (c *failureCache) evict() {
	var key string
	for k := range c.data {
		key = k
		break
	}
	delete(c.data, key)
}

func (c *failureCache) get(now time.Time, key string) error {
	r, found := c.data[key]
	if found && !r.IsExpired(now) {
		return r.error
	}
	return nil
}

// PTRLookupCache is a cache for storing and retrieving the results of
// reverse DNS queries. It caches the results of queries regardless of their
// outcome (success or failure).
type PTRLookupCache struct {
	success    *ptrCache
	failure    *failureCache
	failureTTL time.Duration
	resolver   PTRResolver
	log        Logger
	stats      cacheStats
}

type cacheStats struct {
	Hit  *monitoring.Int
	Miss *monitoring.Int
}

// Logger logs debug messages.
type Logger interface {
	Debugw(msg string, keysAndValues ...interface{})
}

// NewPTRLookupCache returns a new cache.
func NewPTRLookupCache(reg *monitoring.Registry, l Logger, conf CacheConfig, resolver PTRResolver) *PTRLookupCache {
	c := &PTRLookupCache{
		success: &ptrCache{
			data:    make(map[string]ptrRecord, conf.SuccessCache.InitialCapacity),
			maxSize: max(100, max(conf.SuccessCache.InitialCapacity, conf.SuccessCache.MaxCapacity)),
		},
		failure: &failureCache{
			data:       make(map[string]failureRecord, conf.FailureCache.InitialCapacity),
			maxSize:    max(100, max(conf.FailureCache.InitialCapacity, conf.FailureCache.MaxCapacity)),
			failureTTL: conf.FailureCache.TTL,
		},
		resolver: resolver,
		log:      l,
		stats: cacheStats{
			Hit:  monitoring.NewInt(reg, "hits"),
			Miss: monitoring.NewInt(reg, "misses"),
		},
	}

	return c
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
		c.log.Debugw("Reverse DNS lookup failed.", "error", err, "ip", ip)
		if _, cacheable := err.(*dnsError); cacheable {
			c.failure.set(now, ip, err)
		}
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

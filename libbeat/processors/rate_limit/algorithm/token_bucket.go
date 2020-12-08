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

package algorithm

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

func init() {
	register("token_bucket", newTokenBucket)
}

type bucket struct {
	tokens        float64
	lastReplenish time.Time
}

type tokenBucket struct {
	limit   Rate
	depth   float64
	buckets sync.Map

	// GC thresholds and metrics
	gc struct {
		thresholds tokenBucketGCConfig
		metrics    tokenBucketGCConfig
	}
}

type tokenBucketGCConfig struct {
	// NumCalls is the number of calls made to IsAllowed. When more than
	// the specified number of calls are made, GC is performed.
	NumCalls int `config:"num_calls"`

	// NumBuckets is the number of buckets being utilized by the token
	// bucket algorithm. When more than the specified number are utilized,
	// GC is performed.
	NumBuckets int `config:"num_buckets"`
}

type tokenBucketConfig struct {
	BurstMultiplier float64 `config:"burst_multiplier"`

	// GC governs when completely filled token buckets must be deleted
	// to free up memory. GC is performed when _any_ of the GC conditions
	// below are met. After each GC, counters corresponding to _each_ of
	// the GC conditions below are reset.
	GC tokenBucketGCConfig `config:"gc"`
}

func newTokenBucket(config Config) (Algorithm, error) {
	cfg := tokenBucketConfig{
		BurstMultiplier: 1.0,
		GC: tokenBucketGCConfig{
			NumCalls:   10000,
			NumBuckets: 1000,
		},
	}

	if err := config.Config.Unpack(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not unpack token_bucket algorithm configuration")
	}

	return &tokenBucket{
		config.Limit,
		config.Limit.value * cfg.BurstMultiplier,
		sync.Map{},
		struct {
			thresholds tokenBucketGCConfig
			metrics    tokenBucketGCConfig
		}{
			thresholds: tokenBucketGCConfig{
				NumCalls:   cfg.GC.NumCalls,
				NumBuckets: cfg.GC.NumBuckets,
			},
		},
	}, nil
}

func (t *tokenBucket) IsAllowed(key uint64) bool {
	t.runGC()

	b := t.getBucket(key)
	allowed := b.withdraw()

	t.gc.metrics.NumCalls++
	return allowed
}

func (t *tokenBucket) getBucket(key uint64) *bucket {
	v, exists := t.buckets.LoadOrStore(key, &bucket{
		tokens:        t.depth,
		lastReplenish: time.Now(),
	})
	b := v.(*bucket)

	if exists {
		b.replenish(t.limit)
		return b
	}

	t.gc.metrics.NumBuckets++
	return b
}

func (b *bucket) withdraw() bool {
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (b *bucket) replenish(rate Rate) {
	secsSinceLastReplenish := time.Now().Sub(b.lastReplenish).Seconds()
	tokensToReplenish := secsSinceLastReplenish * rate.valuePerSecond()

	b.tokens += tokensToReplenish
	b.lastReplenish = time.Now()
}

func (t *tokenBucket) runGC() {
	// Don't run GC if thresholds haven't been crossed.
	if (t.gc.metrics.NumBuckets < t.gc.thresholds.NumBuckets) &&
		(t.gc.metrics.NumCalls < t.gc.thresholds.NumCalls) {
		return
	}

	// Add tokens to all buckets according to the rate limit
	// and flag full buckets for deletion.
	toDelete := make([]uint64, 0)
	numBuckets := 0
	t.buckets.Range(func(k, v interface{}) bool {
		key := k.(uint64)
		b := v.(*bucket)

		b.replenish(t.limit)

		if b.tokens >= t.depth {
			toDelete = append(toDelete, key)
		}

		numBuckets++

		return true
	})

	// Cleanup full buckets to free up memory
	for _, key := range toDelete {
		t.buckets.Delete(key)
		numBuckets--
	}

	// Reset GC metrics
	t.gc.metrics.NumCalls = 0
	t.gc.metrics.NumBuckets = numBuckets
}

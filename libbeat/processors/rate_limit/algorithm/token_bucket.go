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
	"time"

	"github.com/pkg/errors"
)

func init() {
	Register("token_bucket", newTokenBucket)
}

type bucket struct {
	tokens        float64
	lastReplenish time.Time
}

type tokenBucket struct {
	limit   Rate
	depth   float64
	buckets map[string]bucket
}

func newTokenBucket(config Config) (Algorithm, error) {
	var cfg struct {
		BurstMultipler float64 `config:"burst_multiplier"`
	}

	if err := config.Config.Unpack(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not unpack token_bucket algorithm configuration")
	}

	return &tokenBucket{
		config.Limit,
		config.Limit.value * cfg.BurstMultipler,
		make(map[string]bucket, 0),
	}, nil
}

func (t *tokenBucket) IsAllowed(key string) bool {
	t.replenishBuckets()

	b := t.getBucket(key)
	allowed := b.withdraw()

	return allowed
}

func (t *tokenBucket) getBucket(key string) bucket {
	b, exists := t.buckets[key]
	if !exists {
		b = bucket{
			tokens:        t.depth,
			lastReplenish: time.Now(),
		}
		t.buckets[key] = b
	}

	return b

}

func (b *bucket) withdraw() bool {
	if b.tokens == 0 {
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

func (t *tokenBucket) replenishBuckets() {
	toDelete := make([]string, 0)

	// Replenish all buckets with tokens at the rate limit
	for key, b := range t.buckets {
		b.replenish(t.limit)

		// If bucket is full, flag it for deletion
		if b.tokens >= t.depth {
			toDelete = append(toDelete, key)
		}
	}

	// Cleanup full buckets to free up memory
	for _, key := range toDelete {
		delete(t.buckets, key)
	}
}

package algorithm

import (
	"time"
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

func newTokenBucket(config Config) Algorithm {
	return &tokenBucket{
		config.Limit,
		config.Limit.value * 1, // TODO: replace 1 with burstability multiplier
		make(map[string]bucket, 0),
	}
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

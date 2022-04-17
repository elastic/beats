// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// CounterCache keeps a cache of the last value of all given counters
// and allows to calculate their rate since the last call.
// All methods are thread-unsafe and must not be called concurrently
type CounterCache interface {
	// Start the cache cleanup worker. It mus be called once before start using
	// the cache
	Start()

	// Stop the cache cleanup worker. It mus be called when the cache is disposed
	Stop()

	// RateUint64 returns, for a given counter name, the difference between the given value
	// and the value that was given in a previous call, and true if a previous value existed.
	// It will return 0 and false on the first call.
	RateUint64(counterName string, value uint64) (uint64, bool)

	// RateFloat64 returns, for a given counter name, the difference between the given value
	// and the value that was given in a previous call, and true if a previous value existed.
	// It will return 0 and false on the first call.
	RateFloat64(counterName string, value float64) (float64, bool)
}

type counterCache struct {
	ints    *common.Cache
	floats  *common.Cache
	timeout time.Duration
}

// NewCounterCache initializes and returns a CounterCache. The timeout parameter will be
// used to automatically expire counters that hasn't been updated in a whole timeout period
func NewCounterCache(timeout time.Duration) CounterCache {
	return &counterCache{
		ints:    common.NewCache(timeout, 0),
		floats:  common.NewCache(timeout, 0),
		timeout: timeout,
	}
}

// RateUint64 returns, for a given counter name, the difference between the given value
// and the value that was given in a previous call, and true if a previous value existed.
// It will return 0 and false on the first call.
func (c *counterCache) RateUint64(counterName string, value uint64) (uint64, bool) {
	prev := c.ints.PutWithTimeout(counterName, value, c.timeout)
	if prev != nil {
		if prev.(uint64) > value {
			// counter reset
			return 0, true
		}
		return value - prev.(uint64), true
	}

	// first put for this value, return rate of 0
	return 0, false
}

// RateFloat64 returns, for a given counter name, the difference between the given value
// and the value that was given in a previous call, and true if a previous value existed.
// It will return 0 and false on the first call.
func (c *counterCache) RateFloat64(counterName string, value float64) (float64, bool) {
	prev := c.floats.PutWithTimeout(counterName, value, c.timeout)
	if prev != nil {
		if prev.(float64) > value {
			// counter reset
			return 0, true
		}
		return value - prev.(float64), true
	}

	// first put for this value, return rate of 0
	return 0, false
}

// Start the cache cleanup worker. It mus be called once before start using
// the cache
func (c *counterCache) Start() {
	c.ints.StartJanitor(c.timeout)
	c.floats.StartJanitor(c.timeout)
}

// Stop the cache cleanup worker. It mus be called when the cache is disposed
func (c *counterCache) Stop() {
	c.ints.StopJanitor()
	c.floats.StopJanitor()
}

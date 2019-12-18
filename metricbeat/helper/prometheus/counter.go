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

package prometheus

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
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

	// RateUUint64 returns, for a given counter name, the difference between the given value
	// and the value that was given in a previous call. It will return 0 on the first call
	RateUint64(counterName string, value uint64) uint64

	// RateFloat64 returns, for a given counter name, the difference between the given value
	// and the value that was given in a previous call. It will return 0.0 on the first call
	RateFloat64(counterName string, value float64) float64
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
// and the value that was given in a previous call. It will return 0 on the first call
func (c *counterCache) RateUint64(counterName string, value uint64) uint64 {
	prev := c.ints.PutWithTimeout(counterName, value, c.timeout)
	if prev != nil {
		if prev.(uint64) > value {
			// counter reset
			return 0
		}
		return value - prev.(uint64)
	}

	// first put for this value, return rate of 0
	return 0
}

// RateFloat64 returns, for a given counter name, the difference between the given value
// and the value that was given in a previous call. It will return 0.0 on the first call
func (c *counterCache) RateFloat64(counterName string, value float64) float64 {
	prev := c.floats.PutWithTimeout(counterName, value, c.timeout)
	if prev != nil {
		if prev.(float64) > value {
			// counter reset
			return 0
		}
		return value - prev.(float64)
	}

	// first put for this value, return rate of 0
	return 0
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

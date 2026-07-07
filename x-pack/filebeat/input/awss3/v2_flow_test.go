// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyObserver_InitialValue(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers: 10,
	})
	// Starts at half capacity.
	assert.Equal(t, 5, co.concurrencyLevel())
}

func TestConcurrencyObserver_InitialValue_Minimum(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers: 1,
	})
	assert.Equal(t, 1, co.concurrencyLevel())
}

func TestConcurrencyObserver_OnSuccess_ScalesUp(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     4,
		AdjustCooldown: 0, // no cooldown for testing
	})
	// Starts at 2 (4/2).
	assert.Equal(t, 2, co.concurrencyLevel())

	co.OnSuccess()
	assert.Equal(t, 3, co.concurrencyLevel())

	co.OnSuccess()
	assert.Equal(t, 4, co.concurrencyLevel())

	// At max, should not increase further.
	co.OnSuccess()
	assert.Equal(t, 4, co.concurrencyLevel())
}

func TestConcurrencyObserver_OnBackpressure_ScalesDown(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     8,
		AdjustCooldown: 0,
	})
	// Starts at 4 (8/2).
	assert.Equal(t, 4, co.concurrencyLevel())

	co.OnBackpressure()
	assert.Equal(t, 2, co.concurrencyLevel())

	co.OnBackpressure()
	assert.Equal(t, 1, co.concurrencyLevel())

	// Cannot go below 1.
	co.OnBackpressure()
	assert.Equal(t, 1, co.concurrencyLevel())
}

func TestConcurrencyObserver_Cooldown(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     10,
		AdjustCooldown: time.Hour, // effectively blocks adjustment
	})
	// First call succeeds (lastAdjust is zero time, so elapsed > cooldown).
	co.OnSuccess()
	afterFirst := co.concurrencyLevel()

	// Second call should be blocked by cooldown.
	co.OnSuccess()
	assert.Equal(t, afterFirst, co.concurrencyLevel(), "second adjustment blocked by cooldown")
}

func TestConcurrencyObserver_AIMD_Pattern(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     16,
		AdjustCooldown: 0,
	})
	// Start at 8. Scale up to max.
	for co.concurrencyLevel() < 16 {
		co.OnSuccess()
	}
	require.Equal(t, 16, co.concurrencyLevel())

	// Backpressure halves: 16 -> 8.
	co.OnBackpressure()
	assert.Equal(t, 8, co.concurrencyLevel())

	// Recover additively: 8 -> 9 -> 10 ...
	co.OnSuccess()
	assert.Equal(t, 9, co.concurrencyLevel())
}

func TestConcurrencyObserver_Metrics(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     4,
		AdjustCooldown: 0,
	})
	co.OnSuccess()
	co.OnSuccess()
	co.OnBackpressure()

	assert.Equal(t, uint64(2), co.ScaleUps())
	assert.Equal(t, uint64(1), co.ScaleDowns())
}

func TestPublishWithBackpressure_Fast(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     4,
		AdjustCooldown: 0,
	})
	initial := co.concurrencyLevel()

	publishWithBackpressure(co, 100*time.Millisecond, func() {
		// Fast publish — no delay.
	})

	// Should scale up (fast publish = success signal).
	assert.Greater(t, co.concurrencyLevel(), initial)
}

func TestPublishWithBackpressure_Slow(t *testing.T) {
	co := newConcurrencyObserver(concurrencyObserverConfig{
		MaxWorkers:     8,
		AdjustCooldown: 0,
	})
	// Start at 4.
	assert.Equal(t, 4, co.concurrencyLevel())

	publishWithBackpressure(co, 1*time.Millisecond, func() {
		time.Sleep(5 * time.Millisecond)
	})

	// Should scale down (slow publish = backpressure).
	assert.Less(t, co.concurrencyLevel(), 4)
}

func (o *concurrencyObserver) concurrencyLevel() int {
	return int(o.level.Get())
}

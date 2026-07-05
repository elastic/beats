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

func TestConcurrencyController_InitialValue(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers: 10,
	})
	// Starts at half capacity.
	assert.Equal(t, 5, cc.concurrencyLevel())
}

func TestConcurrencyController_InitialValue_Minimum(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers: 1,
	})
	assert.Equal(t, 1, cc.concurrencyLevel())
}

func TestConcurrencyController_OnSuccess_ScalesUp(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     4,
		AdjustCooldown: 0, // no cooldown for testing
	})
	// Starts at 2 (4/2).
	assert.Equal(t, 2, cc.concurrencyLevel())

	cc.OnSuccess()
	assert.Equal(t, 3, cc.concurrencyLevel())

	cc.OnSuccess()
	assert.Equal(t, 4, cc.concurrencyLevel())

	// At max, should not increase further.
	cc.OnSuccess()
	assert.Equal(t, 4, cc.concurrencyLevel())
}

func TestConcurrencyController_OnBackpressure_ScalesDown(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     8,
		AdjustCooldown: 0,
	})
	// Starts at 4 (8/2).
	assert.Equal(t, 4, cc.concurrencyLevel())

	cc.OnBackpressure()
	assert.Equal(t, 2, cc.concurrencyLevel())

	cc.OnBackpressure()
	assert.Equal(t, 1, cc.concurrencyLevel())

	// Cannot go below 1.
	cc.OnBackpressure()
	assert.Equal(t, 1, cc.concurrencyLevel())
}

func TestConcurrencyController_Cooldown(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     10,
		AdjustCooldown: time.Hour, // effectively blocks adjustment
	})
	// First call succeeds (lastAdjust is zero time, so elapsed > cooldown).
	cc.OnSuccess()
	afterFirst := cc.concurrencyLevel()

	// Second call should be blocked by cooldown.
	cc.OnSuccess()
	assert.Equal(t, afterFirst, cc.concurrencyLevel(), "second adjustment blocked by cooldown")
}

func TestConcurrencyController_AIMD_Pattern(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     16,
		AdjustCooldown: 0,
	})
	// Start at 8. Scale up to max.
	for cc.concurrencyLevel() < 16 {
		cc.OnSuccess()
	}
	require.Equal(t, 16, cc.concurrencyLevel())

	// Backpressure halves: 16 -> 8.
	cc.OnBackpressure()
	assert.Equal(t, 8, cc.concurrencyLevel())

	// Recover additively: 8 -> 9 -> 10 ...
	cc.OnSuccess()
	assert.Equal(t, 9, cc.concurrencyLevel())
}

func TestConcurrencyController_Metrics(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     4,
		AdjustCooldown: 0,
	})
	cc.OnSuccess()
	cc.OnSuccess()
	cc.OnBackpressure()

	assert.Equal(t, uint64(2), cc.ScaleUps())
	assert.Equal(t, uint64(1), cc.ScaleDowns())
}

func TestPublishWithBackpressure_Fast(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     4,
		AdjustCooldown: 0,
	})
	initial := cc.concurrencyLevel()

	publishWithBackpressure(cc, 100*time.Millisecond, func() {
		// Fast publish — no delay.
	})

	// Should scale up (fast publish = success signal).
	assert.Greater(t, cc.concurrencyLevel(), initial)
}

func TestPublishWithBackpressure_Slow(t *testing.T) {
	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     8,
		AdjustCooldown: 0,
	})
	// Start at 4.
	assert.Equal(t, 4, cc.concurrencyLevel())

	publishWithBackpressure(cc, 1*time.Millisecond, func() {
		time.Sleep(5 * time.Millisecond)
	})

	// Should scale down (slow publish = backpressure).
	assert.Less(t, cc.concurrencyLevel(), 4)
}

func (cc *concurrencyController) concurrencyLevel() int {
	return int(cc.level.Get())
}

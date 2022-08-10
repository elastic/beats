// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package backoff

import (
	"math/rand"
	"time"
)

// EqualJitterBackoff implements an equal jitter strategy, meaning the wait time will consist of two parts,
// the first will be exponential and the other half will be random and will provide the jitter
// necessary to distribute the wait on remote endpoint.
type EqualJitterBackoff struct {
	duration time.Duration
	done     <-chan struct{}

	init time.Duration
	max  time.Duration

	last time.Time
}

// NewEqualJitterBackoff returns a new EqualJitter object.
func NewEqualJitterBackoff(done <-chan struct{}, init, max time.Duration) Backoff {
	return &EqualJitterBackoff{
		duration: init * 2, // Allow to sleep at least the init period on the first wait.
		done:     done,
		init:     init,
		max:      max,
	}
}

// Reset resets the duration of the backoff.
func (b *EqualJitterBackoff) Reset() {
	// Allow to sleep at least the init period on the first wait.
	b.duration = b.init * 2
}

// Wait block until either the timer is completed or channel is done.
func (b *EqualJitterBackoff) Wait() bool {
	// Make sure we have always some minimal back off and jitter.
	temp := int64(b.duration / 2)
	backoff := time.Duration(temp + rand.Int63n(temp))

	// increase duration for next wait.
	b.duration *= 2
	if b.duration > b.max {
		b.duration = b.max
	}

	select {
	case <-b.done:
		return false
	case <-time.After(backoff):
		b.last = time.Now()
		return true
	}
}

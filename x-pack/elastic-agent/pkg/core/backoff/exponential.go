// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package backoff

import (
	"time"
)

// ExpBackoff exponential backoff, will wait an initial time and exponentially
// increases the wait time up to a predefined maximum. Resetting Backoff will reset the next sleep
// timer to the initial backoff duration.
type ExpBackoff struct {
	duration time.Duration
	done     <-chan struct{}

	init time.Duration
	max  time.Duration

	last time.Time
}

// NewExpBackoff returns a new exponential backoff.
func NewExpBackoff(done <-chan struct{}, init, max time.Duration) Backoff {
	return &ExpBackoff{
		duration: init,
		done:     done,
		init:     init,
		max:      max,
	}
}

// Reset resets the duration of the backoff.
func (b *ExpBackoff) Reset() {
	b.duration = b.init
}

// Wait block until either the timer is completed or channel is done.
func (b *ExpBackoff) Wait() bool {
	backoff := b.duration
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

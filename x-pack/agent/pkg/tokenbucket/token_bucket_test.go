// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tokenbucket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/scheduler"
)

func TestTokenBucket(t *testing.T) {
	dropAmount := 1
	bucketSize := 3

	t.Run("when way below the bucket size it should not block", func(t *testing.T) {
		stepper := scheduler.NewStepper()

		b, err := newTokenBucketWithScheduler(
			bucketSize,
			dropAmount,
			stepper,
		)

		assert.NoError(t, err, "initiating a bucket failed")

		// Below the bucket size and should not block.
		b.Add()
	})

	t.Run("when below the bucket size it should not block", func(t *testing.T) {
		stepper := scheduler.NewStepper()

		b, err := newTokenBucketWithScheduler(
			bucketSize,
			dropAmount,
			stepper,
		)

		assert.NoError(t, err, "initiating a bucket failed")

		// Below the bucket size and should not block.
		b.Add()
		b.Add()
	})

	t.Run("when we hit the bucket size it should block", func(t *testing.T) {
		stepper := scheduler.NewStepper()

		b, err := newTokenBucketWithScheduler(
			bucketSize,
			dropAmount,
			stepper,
		)

		assert.NoError(t, err, "initiating a bucket failed")

		// Same as the bucket size and should block.
		b.Add()
		b.Add()
		b.Add()

		// Out of bound unblock calls
		unblock := func() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				wg.Done()

				// will unblock the next Add after a second.
				<-time.After(1 * time.Second)
				stepper.Next()
			}(&wg)
			wg.Wait()
		}

		unblock()
		b.Add() // Should block and be unblocked, if not unblock test will timeout.
		unblock()
		b.Add() // Should block and be unblocked, if not unblock test will timeout.
	})

	t.Run("When we use a timer scheduler we can unblock", func(t *testing.T) {
		d := 1 * time.Second
		b, err := NewTokenBucket(
			bucketSize,
			dropAmount,
			d,
		)

		assert.NoError(t, err, "initiating a bucket failed")

		// Same as the bucket size and should block.
		b.Add()
		b.Add()
		b.Add()
		b.Add() // Should block and be unblocked, if not unblock test will timeout.
	})
}

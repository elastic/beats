// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package backoff

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type factory func(<-chan struct{}) Backoff

func TestBackoff(t *testing.T) {
	t.Run("test close channel", testCloseChannel)
	t.Run("test unblock after some time", testUnblockAfterInit)
}

func testCloseChannel(t *testing.T) {
	init := 2 * time.Second
	max := 5 * time.Minute

	tests := map[string]factory{
		"ExpBackoff": func(done <-chan struct{}) Backoff {
			return NewExpBackoff(done, init, max)
		},
		"EqualJitterBackoff": func(done <-chan struct{}) Backoff {
			return NewEqualJitterBackoff(done, init, max)
		},
	}

	for name, f := range tests {
		t.Run(name, func(t *testing.T) {
			c := make(chan struct{})
			b := f(c)
			close(c)
			assert.False(t, b.Wait())
		})
	}
}

func testUnblockAfterInit(t *testing.T) {
	init := 1 * time.Second
	max := 5 * time.Minute

	tests := map[string]factory{
		"ExpBackoff": func(done <-chan struct{}) Backoff {
			return NewExpBackoff(done, init, max)
		},
		"EqualJitterBackoff": func(done <-chan struct{}) Backoff {
			return NewEqualJitterBackoff(done, init, max)
		},
	}

	for name, f := range tests {
		t.Run(name, func(t *testing.T) {
			c := make(chan struct{})
			defer close(c)

			b := f(c)

			startedAt := time.Now()
			assert.True(t, WaitOnError(b, errors.New("bad bad")))
			assert.True(t, time.Now().Sub(startedAt) >= init)
		})
	}
}

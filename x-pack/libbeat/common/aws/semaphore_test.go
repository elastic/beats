// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSemaphore(t *testing.T) {
	s := NewSem(5)

	assert.Equal(t, s.Acquire(5), 5)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Asks for 2, and blocks because 0 are available.
		// It unblocks and returns 1 when Release(1) is called.
		assert.Equal(t, s.Acquire(2), 1)
	}()

	// None are available until Release().
	assert.Equal(t, s.Available(), 0)

	s.Release(1)
	wg.Wait()
}

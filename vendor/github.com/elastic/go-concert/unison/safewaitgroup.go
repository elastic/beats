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

package unison

import (
	"context"
	"errors"
	"sync"

	"github.com/elastic/go-concert/ctxtool"
)

// SafeWaitGroup provides a safe alternative to WaitGroup, that instead of
// panicing returns an error when Wait has been called.
type SafeWaitGroup struct {
	mu     sync.RWMutex
	wg     sync.WaitGroup
	cancel context.CancelFunc
	closed bool
}

// ErrGroupClosed indicates that the WaitGroup is currently closed, and no more
// routines can be started.
var ErrGroupClosed = errors.New("group closed")

// SafeWaitGroupWithCancel creates a SafeWaitGroup that will be closed when
// the given canceler signals shutdown.
//
// Associated resources are cleaned when the parent context is cancelled, or Stop is called.
func SafeWaitGroupWithCancel(parent Canceler) *SafeWaitGroup {
	grp := &SafeWaitGroup{}
	_, cancel := ctxtool.WithFunc(parent, grp.Close)
	grp.cancel = cancel
	return grp
}

// Add adds the delta to the WaitGroup counter.
// If the counter becomes 0, all goroutines are blocked on Wait will continue.
//
// Add returns an error if 'Wait' has already been called, indicating that no more
// go-routines should be started.
func (s *SafeWaitGroup) Add(n int) error {
	if n < 0 {
		s.wg.Add(n)
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrGroupClosed
	}

	s.wg.Add(n)
	return nil
}

// Done decrements the WaitGroup counter.
func (s *SafeWaitGroup) Done() {
	s.wg.Done()
}

// Close marks the wait group as closed. All calls to Add will fail with ErrGroupClosed after
// close has been called. Close does not wait until the WaitGroup counter has
// reached zero, but will return immediately. Use Wait to wait for the counter to become 0.
func (s *SafeWaitGroup) Close() {
	// When the context is cancelled, either by the parent context or by calling
	// 'cancel' directly, Close will be called.
	// The `cancel` function must always be called in order to clean up the context resources.
	// Due to `cancel` calling `Close`, we better be sure to have the mutex
	// released before calling cancel.
	// Although `cancel` is likely to be run in another go-routine, we don't want
	// to make any assumptions about implementation details of the context and cancel function.
	var wasClosed bool
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		wasClosed, s.closed = s.closed, true
	}()

	if !wasClosed && s.cancel != nil {
		s.cancel()
	}
}

// Wait closes the WaitGroup and blocks until the WaitGroup counter is zero.
// Add will return errors the moment 'Wait' has been called.
func (s *SafeWaitGroup) Wait() {
	s.Close()
	s.wg.Wait()
}

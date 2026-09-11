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

package scheduler

import (
	"context"
	"sync"
)

// jobLimitSemaphore is a resizable, per-type job limiter. semaphore.Weighted
// is suitable for the immutable global scheduler limit, but cannot shrink
// safely while jobs are holding slots. This limiter drains an over-subscribed
// type to its new cap without interrupting in-flight jobs.
type jobLimitSemaphore struct {
	mu      sync.Mutex
	limit   int64
	running int64
	waiters []chan struct{}
}

func newJobLimitSemaphore(limit int64) *jobLimitSemaphore {
	return &jobLimitSemaphore{limit: limit}
}

func (s *jobLimitSemaphore) acquire(ctx context.Context) error {
	s.mu.Lock()
	if len(s.waiters) == 0 && s.canAcquire() {
		s.running++
		s.mu.Unlock()
		return nil
	}

	waiter := make(chan struct{})
	s.waiters = append(s.waiters, waiter)
	s.mu.Unlock()

	select {
	case <-waiter:
		return nil
	case <-ctx.Done():
		s.mu.Lock()
		if s.removeWaiter(waiter) {
			s.mu.Unlock()
			return ctx.Err()
		}

		// A slot was assigned concurrently with cancellation. Return that
		// slot before honoring cancellation so a later waiter cannot deadlock.
		s.running--
		s.grantWaiters()
		s.mu.Unlock()
		return ctx.Err()
	}
}

func (s *jobLimitSemaphore) release() {
	s.mu.Lock()
	s.running--
	s.grantWaiters()
	s.mu.Unlock()
}

func (s *jobLimitSemaphore) setLimit(limit int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.limit == limit {
		return false
	}
	s.limit = limit
	s.grantWaiters()
	return true
}

func (s *jobLimitSemaphore) canAcquire() bool {
	return s.limit == 0 || s.running < s.limit
}

func (s *jobLimitSemaphore) grantWaiters() {
	for len(s.waiters) > 0 && s.canAcquire() {
		waiter := s.waiters[0]
		s.waiters = s.waiters[1:]
		s.running++
		close(waiter)
	}
}

func (s *jobLimitSemaphore) removeWaiter(waiter chan struct{}) bool {
	for i, queued := range s.waiters {
		if queued == waiter {
			s.waiters = append(s.waiters[:i], s.waiters[i+1:]...)
			return true
		}
	}
	return false
}

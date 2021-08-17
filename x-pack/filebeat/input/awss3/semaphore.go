// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"sync"
)

type sem struct {
	mutex     *sync.Mutex
	cond      sync.Cond
	available int
}

func newSem(n int) *sem {
	var m sync.Mutex
	return &sem{
		available: n,
		mutex:     &m,
		cond: sync.Cond{
			L: &m,
		},
	}
}

func (s *sem) AcquireContext(n int, ctx context.Context) (int, error) {
	acquireC := make(chan int, 1)
	go func() {
		defer close(acquireC)
		acquireC <- s.Acquire(n)
	}()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case n := <-acquireC:
		return n, nil
	}
}

func (s *sem) Acquire(n int) int {
	if n <= 0 {
		return 0
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.available == 0 {
		s.cond.Wait()
	}

	if n >= s.available {
		rtn := s.available
		s.available = 0
		return rtn
	}

	s.available -= n
	return n
}

func (s *sem) Release(n int) {
	if n <= 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.available += n
	s.cond.Signal()
}

func (s *sem) Available() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.available
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"sync"
)

type Sem struct {
	cond      sync.Cond
	available int
}

func NewSem(n int) *Sem {
	return &Sem{
		available: n,
		cond:      sync.Cond{L: &sync.Mutex{}},
	}
}

func (s *Sem) AcquireContext(n int, ctx context.Context) (int, error) {
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

func (s *Sem) Acquire(n int) int {
	if n <= 0 {
		return 0
	}

	s.cond.L.Lock()
	defer s.cond.L.Unlock()

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

func (s *Sem) Release(n int) {
	if n <= 0 {
		return
	}

	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	s.available += n
	s.cond.Signal()
}

func (s *Sem) Available() int {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	return s.available
}

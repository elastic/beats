// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// CrashChecker checks agent for crash pattern in Elastic Agent lifecycle.
type CrashChecker struct {
	notifyChan chan error
}

// NewCrashChecker creates a new crash checker.
func NewCrashChecker(ch chan error) *CrashChecker {
	return &CrashChecker{
		notifyChan: ch,
	}
}

// Run runs the checking loop.
func (ch CrashChecker) Run(ctx context.Context) {
	// TODO: finish me

}

func getAgentServicePid() int {
	// TODO: finish me
	return 0
}

type disctintQueue struct {
	q    []int
	size int
	lock sync.Mutex
}

func newDistinctQueue(size int) (*disctintQueue, error) {
	if size < 1 {
		return nil, errors.New("invalid size", errors.TypeUnexpected)
	}
	return &disctintQueue{
		q:    make([]int, 0, size),
		size: size,
	}, nil
}

func (dq *disctintQueue) Push(id int) {
	dq.lock.Lock()
	defer dq.lock.Unlock()
	dq.q = append([]int{id}, dq.q[:dq.size-1]...)
}

func (dq *disctintQueue) Disctinct() int {
	dq.lock.Lock()
	defer dq.lock.Unlock()

	dm := make(map[int]int)

	for _, id := range dq.q {
		dm[id] = 1
	}

	return len(dm)
}

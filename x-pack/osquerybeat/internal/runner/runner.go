// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package runner

import (
	"context"
	"errors"
	"sync"
)

var ErrAlreadyRunning = errors.New("runner is alredy running")

type RunFunc func(ctx context.Context) error

type Runner struct {
	cn context.CancelFunc
	mx sync.RWMutex

	wg sync.WaitGroup
}

func New() *Runner {
	r := &Runner{}
	return r
}

func (r *Runner) Run(ctx context.Context, runfn RunFunc) error {
	r.mx.Lock()
	if r.cn != nil {
		r.mx.Unlock()
		return ErrAlreadyRunning
	}

	ctx, cn := context.WithCancel(ctx)
	defer cn()
	r.cn = cn

	r.mx.Unlock()

	r.wg.Add(1)
	defer r.wg.Done()

	return runfn(ctx)
}

func (r *Runner) Stop() {
	r.mx.RLock()
	if r.cn != nil {
		r.cn()
	}
	r.mx.RUnlock()

	// Wait until stopped
	r.wg.Wait()
}

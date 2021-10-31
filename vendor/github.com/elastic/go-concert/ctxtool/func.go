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

package ctxtool

import (
	"context"
	"sync"
)

type funcContext struct {
	context.Context
	ch  <-chan struct{}
	mu  sync.Mutex
	err error
}

// WithFunc creates a context that will execute the given function when the
// parent context gets cancelled.
func WithFunc(parent canceller, fn func()) (context.Context, context.CancelFunc) {
	ctx := FromCanceller(parent)

	if ctx.Err() != nil {
		// context already cancelled, call fn
		go fn()
		return ctx, func() {}
	}

	chCancel := make(chan struct{})
	chDone := make(chan struct{})
	fnCtx := &funcContext{
		Context: ctx,
		ch:      chDone,
	}

	go fnCtx.wait(chCancel, chDone, fn)

	var closeOnce sync.Once
	return fnCtx, func() {
		closeOnce.Do(func() {
			close(chCancel)
		})
	}
}

func (ctx *funcContext) wait(cancel <-chan struct{}, done chan struct{}, fn func()) {
	defer close(done)
	defer fn()

	select {
	case <-ctx.Context.Done():
		ctx.setErr(ctx.Context.Err())
	case <-cancel:
		ctx.setErr(context.Canceled)
	}
}

func (ctx *funcContext) setErr(err error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.err = err
}

func (ctx *funcContext) Done() <-chan struct{} { return ctx.ch }
func (ctx *funcContext) Err() error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.err
}

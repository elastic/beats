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
	"time"
)

type cancelledContext struct {
	context.Context
	err error
}

type mergeCancelCtx struct {
	context.Context
	cancel canceller
	ch     <-chan struct{}

	mu  sync.Mutex
	err error
}

// cancelOverwriteContext uses the canceller for Done and error calls, and the
// original context for Deadline and Value calls.
type cancelOverwriteContext struct {
	ctx    context.Context
	cancel canceller
}

type mergedDeadlineCtx struct {
	context.Context
	deadline time.Time
}

type mergeValueCtx struct {
	context.Context
	overwrites valuer
}

// MergeContexts merges cancellation and values of 2 contexts.
// The resulting context is canceled by the first context that got canceled.
// The ctx2 overwrites values in ctx1 during value lookup.
func MergeContexts(ctx1, ctx2 context.Context) (context.Context, context.CancelFunc) {
	return MergeCancellation(MergeValues(MergeDeadline(ctx1, ctx2), ctx2), ctx2)
}

// MergeCancellation creates a new context that will be cancelled if one of the
// two input contexts gets canceled. The `Values` and `Deadline` are taken from the first context.
func MergeCancellation(parent, other canceller) (context.Context, context.CancelFunc) {
	ctx := FromCanceller(parent)

	err := ctx.Err()
	if err == nil {
		err = other.Err()
	}
	if err != nil {
		// at least one context is already cancelled
		return &cancelledContext{Context: ctx, err: err}, func() {}
	}

	if ctx.Done() == nil {
		if other.Done() == nil {
			// context is never cancelled.
			return ctx, func() {}
		}
		return &cancelOverwriteContext{ctx: ctx, cancel: other}, func() {}
	}

	chDone := make(chan struct{})
	merged := &mergeCancelCtx{
		Context: ctx,
		cancel:  other,
		ch:      chDone,
	}
	go merged.waitCancel(chDone)

	canceller := func() {
		merged.mu.Lock()
		defer merged.mu.Unlock()
		if merged.err == nil {
			merged.err = context.Canceled
			close(chDone)
		}
	}
	return merged, canceller
}

func (c *cancelledContext) Done() <-chan struct{} {
	return closedChan
}

func (c *cancelledContext) Err() error {
	return c.err
}

func (c *mergeCancelCtx) waitCancel(chDone chan struct{}) {
	var err error
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.err == nil {
			c.err = err
			close(chDone)
		}
	}()

	select {
	case <-chDone: // CancelFunc triggered cleanup

	case <-c.Context.Done():
		err = c.Context.Err()
	case <-c.cancel.Done():
		err = c.cancel.Err()
	}
}

func (c *mergeCancelCtx) Done() <-chan struct{} {
	return c.ch
}

func (c *mergeCancelCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *cancelOverwriteContext) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *cancelOverwriteContext) Done() <-chan struct{} {
	return c.cancel.Done()
}

func (c *cancelOverwriteContext) Err() error {
	return c.cancel.Err()
}

func (c *cancelOverwriteContext) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

// MergeValues merges the values from ctx and overwrites. Value lookup will occur on `overwrites` first.
// Deadline and cancellation are still driven by the first context. In order to merge cancellation use
// MergeCancellation.
func MergeValues(ctx context.Context, overwrites valuer) context.Context {
	return &mergeValueCtx{ctx, overwrites}
}

func (c *mergeValueCtx) Value(key interface{}) interface{} {
	if val := c.overwrites.Value(key); val != nil {
		return val
	}
	return c.Context.Value(key)
}

// MergeDeadline merges the deadline of two contexts. The resulting context
// deadline will be the lesser deadline between the two context.  If neither
// context configures a deadline, the original context is returned.
func MergeDeadline(ctx context.Context, deadliner deadliner) context.Context {
	deadline, ok := deadliner.Deadline()
	if !ok {
		return ctx
	}

	ctxDeadline, ok := ctx.Deadline()
	if ok && ctxDeadline.Before(deadline) {
		return ctx
	}

	return &mergedDeadlineCtx{ctx, deadline}
}

func (ctx mergedDeadlineCtx) Deadline() (time.Time, bool) {
	return ctx.deadline, true
}

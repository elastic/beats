// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
)

// TxTracker implements a transaction tracker. Individual beat events which
// compose a transaction are added to a pending count with Add. As events are
// ACK-ed, the pending count is decremented (see NewTxACKHandler). Calling Wait
// after the last Add will block until all events are ACK-ed or the parent
// context is cancelled.
//
// The pending count may reach zero while events are still being published,
// so reaching zero only releases Wait once Wait has been called.
type TxTracker struct {
	ctx context.Context

	mu      sync.Mutex
	pending int64
	waiting bool
	done    chan struct{} // closed when pending is zero after Wait is called
}

// NewTxTracker will create a new TxTracker using the provided context. It
// is recommended that the parent context has a way of being cancelled,
// otherwise a call to TxTracker.Wait may not be interruptible.
func NewTxTracker(ctx context.Context) *TxTracker {
	return &TxTracker{ctx: ctx, done: make(chan struct{})}
}

// Add increments the pending count.
func (t *TxTracker) Add() {
	t.mu.Lock()
	t.pending++
	t.mu.Unlock()
}

// Ack decrements the pending count. If Wait has been called and pending
// goes to zero, Wait is released.
func (t *TxTracker) Ack() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending--
	if t.waiting && t.pending == 0 {
		t.release()
	}
}

// Wait blocks until the pending count is 0 or the parent context is
// cancelled. It must be called after the last call to Add.
func (t *TxTracker) Wait() {
	t.mu.Lock()
	t.waiting = true
	if t.pending == 0 {
		t.release()
	}
	t.mu.Unlock()

	select {
	case <-t.done:
	case <-t.ctx.Done():
	}
}

// release closes t.done if it is not already closed. It must be called
// with t.mu held.
func (t *TxTracker) release() {
	select {
	case <-t.done:
	default:
		close(t.done)
	}
}

// NewTxACKHandler creates a new beat.EventListener. As events are ACK-ed and if the event
// contains a TxTracker, Ack will be called on the TxTracker.
func NewTxACKHandler() beat.EventListener {
	return acker.ConnectionOnly(acker.EventPrivateReporter(func(acked int, privates []any) {
		for _, private := range privates {
			if t, ok := private.(*TxTracker); ok {
				t.Ack()
			}
		}
	}))
}

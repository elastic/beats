// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

// TxTracker implements a transaction tracker. Individual beat events which
// compose a transaction can be added to a pending count via Ack. As events are
// ACK-ed, the pending count is decremented (see NewTxACKHandler). Calling Wait
// will block until all events are ACK-ed or the parent context is cancelled.
type TxTracker struct {
	pending atomic.Int
	ctx     context.Context
	cancel  context.CancelFunc
}

// Add increments the pending count.
func (t *TxTracker) Add() {
	t.pending.Inc()
}

// Ack decrements the pending count. If pending goes to zero, then the
// context on TxTracker is cancelled.
func (t *TxTracker) Ack() {
	if t.pending.Dec() == 0 {
		t.cancel()
	}
}

// Wait will block until the pending count is 0. If pending is zero, then
// the context on TxTracker is cancelled.
func (t *TxTracker) Wait() {
	if t.pending.Load() == 0 {
		t.cancel()
	}

	<-t.ctx.Done()
}

// NewTxTracker will create a new TxTracker using the provided context. It
// is recommended that the parent context has a way of being cancelled,
// otherwise a call to TxTracker.Wait may not be interruptable.
func NewTxTracker(ctx context.Context) *TxTracker {
	t := TxTracker{}
	t.ctx, t.cancel = context.WithCancel(ctx)

	return &t
}

// NewTxACKHandler creates a new beat.EventListener. As events are ACK-ed and if the event
// contains a TxTracker, Ack will be called on the TxTracker.
func NewTxACKHandler() beat.EventListener {
	return acker.ConnectionOnly(acker.EventPrivateReporter(func(acked int, privates []interface{}) {
		for _, private := range privates {
			if t, ok := private.(*TxTracker); ok {
				t.Ack()
			}
		}
	}))
}

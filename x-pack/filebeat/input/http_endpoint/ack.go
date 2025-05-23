// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
)

// newEventACKHandler returns a beat ACKer that can receive callbacks when
// an event has been ACKed an output. If the event contains a private metadata
// pointing to a batchACKTracker then it will invoke the tracker's ACK() method
// to decrement the number of pending ACKs.
func newEventACKHandler() beat.EventListener {
	return acker.ConnectionOnly(
		acker.EventPrivateReporter(func(_ int, privates []interface{}) {
			for _, private := range privates {
				if ack, ok := private.(*batchACKTracker); ok {
					ack.ACK()
				}
			}
		}),
	)
}

// batchACKTracker invokes batchACK when all events associated to the batch
// have been published and acknowledged by an output.
type batchACKTracker struct {
	batchACK func()

	mu      sync.Mutex
	pending int64
}

// newBatchACKTracker returns a new batchACKTracker. The provided batchACK function
// is invoked after the full batch has been acknowledged. Ready() must be invoked
// after all events in the batch are published.
func newBatchACKTracker(fn func()) *batchACKTracker {
	return &batchACKTracker{
		batchACK: fn,
		pending:  1, // Ready() must be called to consume this "1".
	}
}

// Ready signals that the batch has been fully consumed. Only
// after the batch is marked as "ready" can the batch be ACKed.
// This prevents the batch from being ACKed prematurely.
func (t *batchACKTracker) Ready() {
	t.ACK()
}

// Add increments the number of pending ACKs.
func (t *batchACKTracker) Add() {
	t.mu.Lock()
	t.pending++
	t.mu.Unlock()
}

// ACK decrements the number of pending event ACKs. When all pending ACKs are
// received then the event batch is ACKed.
func (t *batchACKTracker) ACK() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.pending <= 0 {
		panic("misuse detected: negative ACK counter")
	}

	t.pending--
	if t.pending == 0 {
		t.batchACK()
	}
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
)

// batchACKTracker invokes batchACK when all events associated to the batch
// have been published and acknowledged by an output.
type batchACKTracker struct {
	batchACK func()

	mutex       sync.Mutex // mutex synchronizes access to pendingACKs.
	pendingACKs int64      // Number of Beat events in lumberjack batch that are pending ACKs.
}

// newBatchACKTracker returns a new batchACKTracker. The provided batchACK function
// is invoked after the full batch has been acknowledged. Ready() must be invoked
// after all events in the batch are published.
func newBatchACKTracker(batchACKCallback func()) *batchACKTracker {
	return &batchACKTracker{
		batchACK:    batchACKCallback,
		pendingACKs: 1, // Ready() must be called to consume this "1".
	}
}

// Ready signals that the batch has been fully consumed. Only
// after the batch is marked as "ready" can the lumberjack batch
// be ACKed. This prevents the batch from being ACKed prematurely.
func (t *batchACKTracker) Ready() {
	t.ACK()
}

// Add increments the number of pending ACKs.
func (t *batchACKTracker) Add() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.pendingACKs++
}

// ACK decrements the number of pending event ACKs. When all pending ACKs are
// received then the lumberjack batch is ACKed.
func (t *batchACKTracker) ACK() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.pendingACKs <= 0 {
		panic("misuse detected: negative ACK counter")
	}

	t.pendingACKs--
	if t.pendingACKs == 0 {
		t.batchACK()
	}
}

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

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

// EventACKTracker tracks the publishing state of S3 objects. Specifically
// it tracks the number of message acknowledgements that are pending from the
// output. It can be used to wait until all ACKs have been received for one or
// more S3 objects.
type EventACKTracker struct {
	sync.Mutex
	PendingACKs *atomic.Uint64
}

func NewEventACKTracker() *EventACKTracker {
	return &EventACKTracker{PendingACKs: atomic.NewUint64(0)}
}

// Add increments the number of pending ACKs.
func (a *EventACKTracker) Add() {
	a.PendingACKs.Inc()
}

// ACK decrements the number of pending ACKs.
func (a *EventACKTracker) ACK() {
	if a.PendingACKs.Load() <= 0 {
		panic("misuse detected: negative ACK counter")
	}

	a.PendingACKs.Dec()
}

// Wait waits for the number of pending ACKs to be zero.
// Wait must be called only after every expected
// `Add` calls are made. Failing to do so could reset the pendingACKs
// property to 0 and would results in Wait returning after additional
// calls to `Add` are made without a corresponding `ACK` call.
func (a *EventACKTracker) Wait() {

	// If there were no any pending ACKs returns.
	for {
		if a.PendingACKs.Load() == 0 {
			return
		}
	}
}

// NewEventACKHandler returns a beat ACKer that can receive callbacks when
// an event has been ACKed an output. If the event contains a private metadata
// pointing to an eventACKTracker then it will invoke the trackers ACK() method
// to decrement the number of pending ACKs.
func NewEventACKHandler() beat.EventListener {
	return acker.ConnectionOnly(
		acker.EventPrivateReporter(func(_ int, privates []interface{}) {
			for _, private := range privates {
				if ack, ok := private.(*EventACKTracker); ok {
					ack.ACK()
				}
			}
		}),
	)
}

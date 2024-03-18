// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

var ackerIDCounter *atomic.Uint64

func init() {
	ackerIDCounter = atomic.NewUint64(0)
}

// EventACKTracker tracks the publishing state of S3 objects. Specifically
// it tracks the number of message acknowledgements that are pending from the
// output. It can be used to wait until all ACKs have been received for one or
// more S3 objects.
type EventACKTracker struct {
	ID uint64

	eventsAcked     atomic.Uint64
	eventsToBeAcked atomic.Uint64

	ctx        context.Context
	cancel     context.CancelFunc
	deletionWg *sync.WaitGroup

	msg             *types.Message
	ReceiveCount    int
	start           time.Time
	processingErr   error
	Handles         []s3ObjectHandler
	keepaliveCancel context.CancelFunc
	keepaliveWg     *sync.WaitGroup
	msgHandler      sqsProcessor
	log             *logp.Logger
	metrics         *inputMetrics
}

func NewEventACKTracker(ctx context.Context, deletionWg *sync.WaitGroup) *EventACKTracker {
	ctx, cancel := context.WithCancel(ctx)
	acker := &EventACKTracker{
		ID:         ackerIDCounter.Inc(),
		ctx:        ctx,
		cancel:     cancel,
		deletionWg: deletionWg,
	}

	go func() {
		t := time.NewTicker(500 * time.Microsecond)
		defer t.Stop()

		for {
			<-t.C
			if !acker.FullyTracked() {
				continue
			}

			acker.cancelAndFlush()
			return
		}
	}()

	return acker
}

func (a *EventACKTracker) cancelAndFlush() {
	a.deleteSQS()
	a.cancel()
}

// MarkSQSProcessedWithData Every call after the first one is a no-op
func (a *EventACKTracker) MarkSQSProcessedWithData(msg *types.Message, publishedEvent uint64, receiveCount int, start time.Time, processingErr error, handles []s3ObjectHandler, keepaliveCancel context.CancelFunc, keepaliveWg *sync.WaitGroup, msgHandler sqsProcessor, log *logp.Logger, metrics *inputMetrics) {
	// We want to execute the logic of this call only once, when the ack mutex was locked on init
	if a.eventsToBeAcked.Load() > 0 {
		return
	}

	a.msg = msg
	a.eventsToBeAcked.Store(publishedEvent)
	a.ReceiveCount = receiveCount
	a.start = start
	a.processingErr = processingErr
	a.Handles = handles
	a.keepaliveCancel = keepaliveCancel
	a.keepaliveWg = keepaliveWg
	a.msgHandler = msgHandler
	a.log = log
	a.metrics = metrics
}

func (a *EventACKTracker) FullyTracked() bool {
	eventsToBeTracked := a.eventsToBeAcked.Load()
	if eventsToBeTracked == 0 {
		return false
	}

	// This is eating its own tail: we should check for dropped+published, but then we won't wait for acked.
	// Acked might not be equal to published?
	return a.eventsAcked.Load() == eventsToBeTracked
}

// deleteSQS delete related SQS message
func (a *EventACKTracker) deleteSQS() {
	// Stop keepalive visibility routine before deleting.
	a.keepaliveCancel()
	a.keepaliveWg.Wait()

	a.metrics.sqsMessagesInflight.Dec()

	err := a.msgHandler.DeleteSQS(a.msg, a.ReceiveCount, a.processingErr, a.Handles)
	if err != nil {
		a.log.Warnw("Failed deleting SQS message.",
			"error", err,
			"message_id", *a.msg.MessageId,
			"elapsed_time_ns", time.Since(a.start))
	} else {
		a.log.Debugw("Success deleting SQS message.",
			"message_id", *a.msg.MessageId,
			"elapsed_time_ns", time.Since(a.start))
	}

	a.deletionWg.Done()

}

// ACK increments the number of eventsAcked.
func (a *EventACKTracker) ACK() {
	a.eventsAcked.Inc()
}

// NewEventACKHandler returns a beat ACKer that can receive callbacks when
// an event has been ACKed an output. If the event contains a private metadata
// pointing to an eventACKTracker then it will invoke the trackers ACK() method
// to decrement the number of pending ACKs.
func NewEventACKHandler() beat.EventListener {
	return acker.ConnectionOnly(
		acker.EventPrivateReporter(func(_ int, privates []interface{}) {
			for _, private := range privates {
				if acker, ok := private.(*EventACKTracker); ok {
					acker.ACK()
				}
			}
		}))
}

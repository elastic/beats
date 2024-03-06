// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

var totMessageDeleted *atomic.Uint64

func init() {
	totMessageDeleted = atomic.NewUint64(0)
}

// EventACKTracker tracks the publishing state of S3 objects. Specifically
// it tracks the number of message acknowledgements that are pending from the
// output. It can be used to wait until all ACKs have been received for one or
// more S3 objects.
type EventACKTracker struct {
	DeletionWg *sync.WaitGroup

	EventsToBeAcked  *atomic.Uint64
	TotalEventsAcked *atomic.Uint64

	isSQSAcker bool

	ctx    context.Context
	cancel context.CancelFunc

	msg             *types.Message
	ReceiveCount    int
	start           time.Time
	processingErr   error
	Handles         []s3ObjectHandler
	keepaliveCancel context.CancelFunc
	keepaliveWg     *sync.WaitGroup
	msgHandler      sqsProcessor
	log             *logp.Logger
}

func NewEventACKTracker(ctx context.Context, deletionWg *sync.WaitGroup) *EventACKTracker {
	ctx, cancel := context.WithCancel(ctx)
	return &EventACKTracker{
		ctx:              ctx,
		cancel:           cancel,
		DeletionWg:       deletionWg,
		TotalEventsAcked: atomic.NewUint64(0),
		EventsToBeAcked:  atomic.NewUint64(0),
	}
}

func (a *EventACKTracker) AddSQSDeletionData(msg *types.Message, publishedEvent uint64, receiveCount int, start time.Time, processingErr error, handles []s3ObjectHandler, keepaliveCancel context.CancelFunc, keepaliveWg *sync.WaitGroup, msgHandler sqsProcessor, log *logp.Logger) {
	a.isSQSAcker = true

	a.msg = msg
	a.EventsToBeAcked = atomic.NewUint64(publishedEvent)
	a.ReceiveCount = receiveCount
	a.start = start
	a.processingErr = processingErr
	a.Handles = handles
	a.keepaliveCancel = keepaliveCancel
	a.keepaliveWg = keepaliveWg
	a.msgHandler = msgHandler
	a.log = log
}

func (a *EventACKTracker) FullyAcked() bool {
	return a.TotalEventsAcked.Load() == a.EventsToBeAcked.Load()

}
func (a *EventACKTracker) WaitForS3() {
	// If it's fully acked then cancel the context.
	if a.FullyAcked() {
		a.cancel()
	}

	// Wait.
	<-a.ctx.Done()
}

// FlushForSQS delete related SQS message
func (a *EventACKTracker) FlushForSQS() {
	if !a.isSQSAcker {
		return
	}

	// Stop keepalive visibility routine before deleting.
	a.keepaliveCancel()
	a.keepaliveWg.Wait()

	err := a.msgHandler.DeleteSQS(a.msg, a.ReceiveCount, a.processingErr, a.Handles)
	a.DeletionWg.Done()
	totMessageDeleted.Inc()

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
}

// ACK decrements the number of total Events ACKed.
func (a *EventACKTracker) ACK() {
	if a.FullyAcked() {
		panic("misuse detected: ACK call on fully acked")
	}

	a.TotalEventsAcked.Inc()

	if a.FullyAcked() {
		a.cancel()
	}
}

// NewEventACKHandler returns a beat ACKer that can receive callbacks when
// an event has been ACKed an output. If the event contains a private metadata
// pointing to an eventACKTracker then it will invoke the trackers ACK() method
// to decrement the number of pending ACKs.
func NewEventACKHandler() beat.EventListener {
	return acker.ConnectionOnly(
		acker.EventPrivateReporter(func(_ int, privates []interface{}) {
			for _, current := range privates {
				if ackTracker, ok := current.(*EventACKTracker); ok {
					ackTracker.ACK()

					if ackTracker.FullyAcked() {
						ackTracker.FlushForSQS()
					}
				}
			}
		}),
	)
}

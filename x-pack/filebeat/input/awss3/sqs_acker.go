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

	EventsToBeAcked *atomic.Uint64

	ctx        context.Context
	cancel     context.CancelFunc
	deletionWg *sync.WaitGroup

	ackMutex             *sync.RWMutex
	ackMutexLockedOnInit *atomic.Bool

	isSQSAcker      bool
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
	ackMutex := new(sync.RWMutex)
	// We need to lock on ack mutex, in order to know that we have passed the info to the acker about the total to be acked
	// Lock it as soon as we create the acker. It will be unlocked and in either MarkS3FromListingProcessedWithData or MarkSQSProcessedWithData
	ackMutex.Lock()
	return &EventACKTracker{
		ID:                   ackerIDCounter.Inc(),
		ctx:                  ctx,
		cancel:               cancel,
		deletionWg:           deletionWg,
		ackMutex:             ackMutex,
		ackMutexLockedOnInit: atomic.NewBool(true),
		EventsToBeAcked:      atomic.NewUint64(0),
	}
}

// MarkS3FromListingProcessedWithData has to be used when the acker is used when the input is in s3 direct listing mode, instead of MarkSQSProcessedWithData
// Specifically we both Swap the value of EventACKTracker.ackMutexLockedOnInit initialised in NewEventACKTracker
func (a *EventACKTracker) MarkS3FromListingProcessedWithData(s3EventsCreatedTotal uint64) {
	// We want to execute the logic of this call only once, when the ack mutex was locked on init
	if !a.ackMutexLockedOnInit.Swap(false) {
		return
	}

	a.EventsToBeAcked.Add(s3EventsCreatedTotal)

	a.ackMutex.Unlock()
}

// MarkSQSProcessedWithData has to be used when the acker is used when the input is in sqs-s3 mode, instead of MarkS3FromListingProcessedWithData
// Specifically we both Swap the value of EventACKTracker.ackMutexLockedOnInit initialised in NewEventACKTracker
func (a *EventACKTracker) MarkSQSProcessedWithData(msg *types.Message, publishedEvent uint64, receiveCount int, start time.Time, processingErr error, handles []s3ObjectHandler, keepaliveCancel context.CancelFunc, keepaliveWg *sync.WaitGroup, msgHandler sqsProcessor, log *logp.Logger) {
	// We want to execute the logic of this call only once, when the ack mutex was locked on init
	if !a.ackMutexLockedOnInit.Swap(false) {
		return
	}

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

	a.ackMutex.Unlock()
}

func (a *EventACKTracker) FullyAcked() bool {
	return a.EventsToBeAcked.Load() == 0

}

// WaitForS3 must be called after MarkS3FromListingProcessedWithData
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
	a.deletionWg.Done()

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
	// We need to lock on ack mutex, in order to know that we have passed the info to the acker about the total to be acked
	// But we want to do it only before the info have been passed, once they did, no need anymore to lock on the ack mutext
	if a.ackMutexLockedOnInit.Load() {
		a.ackMutex.Lock()
		defer a.ackMutex.Unlock()
	}

	if a.FullyAcked() {
		panic("misuse detected: ACK call on fully acked")
	}

	a.EventsToBeAcked.Dec()

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

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"github.com/zyedidia/generic/queue"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/common/acker"
)

type awsACKHandler struct {
	pending    *queue.Queue[pendingACK]
	ackedCount int

	pendingChan chan pendingACK
	ackChan     chan int
}

type pendingACK struct {
	eventCount  int
	ackCallback func()
}

func newAWSACKHandler() *awsACKHandler {
	handler := &awsACKHandler{
		pending: queue.New[pendingACK](),

		// Channel buffer sizes are somewhat arbitrary: synchronous channels
		// would be safe, but buffers slightly reduce scheduler overhead since
		// the ack loop goroutine doesn't need to wake up as often.
		//
		// pendingChan receives one message each time an S3/SQS worker goroutine
		// finishes processing an object. If it is full, workers will not be able
		// to advance to the next object until the ack loop wakes up.
		//
		// ackChan receives approximately one message every time an acknowledged
		// batch of events contains at least one event from this input. (Sometimes
		// fewer if messages can be coalesced.) If it is full, acknowledgement
		// notifications for inputs/queue will stall until the ack loop wakes up.
		// (This is a much worse consequence than pendingChan, but ackChan also
		// receives fewer messages than pendingChan by a factor of ~thousands,
		// so in practice it's still low-impact.)
		pendingChan: make(chan pendingACK, 10),
		ackChan:     make(chan int, 10),
	}
	go handler.run()
	return handler
}

func (ah *awsACKHandler) Add(eventCount int, ackCallback func()) {
	ah.pendingChan <- pendingACK{
		eventCount:  eventCount,
		ackCallback: ackCallback,
	}
}

// Called when a worker is closing, to indicate to the ack handler that it
// should shut down as soon as the current pending list is acknowledged.
func (ah *awsACKHandler) Close() {
	close(ah.pendingChan)
}

func (ah *awsACKHandler) pipelineEventListener() beat.EventListener {
	return acker.TrackingCounter(func(_ int, total int) {
		// Notify the ack handler goroutine
		ah.ackChan <- total
	})
}

// Listener that handles both incoming metadata and ACK
// confirmations.
func (ah *awsACKHandler) run() {
	for {
		select {
		case result, ok := <-ah.pendingChan:
			if ok {
				ah.pending.Enqueue(result)
			} else {
				// Channel is closed, reset so we don't receive any more values
				ah.pendingChan = nil
			}
		case count := <-ah.ackChan:
			ah.ackedCount += count
		}

		// Finalize any objects that are now completed
		for !ah.pending.Empty() && ah.ackedCount >= ah.pending.Peek().eventCount {
			result := ah.pending.Dequeue()
			ah.ackedCount -= result.eventCount
			// Run finalization asynchronously so we don't block the SQS worker
			// or the queue by ignoring the ack handler's input channels. Ordering
			// is no longer important at this point.
			if result.ackCallback != nil {
				go result.ackCallback()
			}
		}

		// If the input is closed and all acks are completed, we're done
		if ah.pending.Empty() && ah.pendingChan == nil {
			return
		}
	}
}

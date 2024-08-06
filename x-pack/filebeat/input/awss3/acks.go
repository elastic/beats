// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/fifo"
)

type awsACKHandler struct {
	pending    fifo.FIFO[pendingACK]
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
				ah.pending.Add(result)
			} else {
				// Channel is closed, reset so we don't receive any more values
				ah.pendingChan = nil
			}
		case count := <-ah.ackChan:
			ah.ackedCount += count
		}

		// Finalize any objects that are now completed
		for !ah.pending.Empty() && ah.ackedCount >= ah.pending.First().eventCount {
			result := ah.pending.ConsumeFirst()
			ah.ackedCount -= result.eventCount
			// Run finalization asynchronously so we don't block the SQS worker
			// or the queue by ignoring the ack handler's input channels. Ordering
			// is no longer important at this point.
			go result.ackCallback()
		}

		// If the input is closed and all acks are completed, we're done
		if ah.pending.Empty() && ah.pendingChan == nil {
			return
		}
	}
}

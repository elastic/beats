// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"github.com/zyedidia/generic/queue"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
)

type ackHandler struct {
	pending    *queue.Queue[pendingACK]
	ackedCount int

	pendingChan chan pendingACK
	ackChan     chan int
}

type pendingACK struct {
	eventCount  int
	ackCallback func()
}

func newACKHandler() *ackHandler {
	handler := &ackHandler{
		pending:     queue.New[pendingACK](),
		pendingChan: make(chan pendingACK, 10),
		ackChan:     make(chan int, 10),
	}
	go handler.run()
	return handler
}

// Add registers a pending ACK entry. The callback fires after eventCount
// events have been acknowledged by the output pipeline.
func (ah *ackHandler) Add(eventCount int, ackCallback func()) {
	ah.pendingChan <- pendingACK{
		eventCount:  eventCount,
		ackCallback: ackCallback,
	}
}

// Close signals the ACK handler to shut down once all pending entries are
// drained.
func (ah *ackHandler) Close() {
	close(ah.pendingChan)
}

// pipelineEventListener returns a beat.EventListener that feeds ACK
// notifications into this handler.
func (ah *ackHandler) pipelineEventListener() beat.EventListener {
	return acker.TrackingCounter(func(_ int, total int) {
		ah.ackChan <- total
	})
}

func (ah *ackHandler) run() {
	for {
		select {
		case result, ok := <-ah.pendingChan:
			if ok {
				ah.pending.Enqueue(result)
			} else {
				ah.pendingChan = nil
			}
		case count := <-ah.ackChan:
			ah.ackedCount += count
		}

		for !ah.pending.Empty() && ah.ackedCount >= ah.pending.Peek().eventCount {
			result := ah.pending.Dequeue()
			ah.ackedCount -= result.eventCount
			if result.ackCallback != nil {
				go result.ackCallback()
			}
		}

		if ah.pending.Empty() && ah.pendingChan == nil {
			return
		}
	}
}

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package proxyqueue

import (
	"io"
	"sync"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

type broker struct {
	doneChan chan struct{}

	logger *logp.Logger

	// The maximum number of events in any pending batch
	batchSize int

	///////////////////////////
	// api channels

	// Producers send queue entries to pushChan to add them to the next batch.
	pushChan chan *pushRequest

	// Consumers send requests to getChan to read entries from the queue.
	getChan chan getRequest

	// A callback that should be invoked when ACKs are processed.
	// This is used to forward notifications back to the pipeline observer,
	// which updates the beats registry if needed. This callback is included
	// in batches created by the proxy queue, so they can invoke it when they
	// receive a Done call.
	ackCallback func(eventCount int)

	// Internal state for the broker's run loop.
	queuedEntries      []queueEntry
	blockedRequests    blockedRequests
	outstandingBatches batchList

	// wait group for worker shutdown
	wg sync.WaitGroup
}

type Settings struct {
	BatchSize int
}

type queueEntry struct {
	event interface{}

	// The producer that generated this event, or nil if this producer does
	// not require ack callbacks.
	producer *producer
}

type blockedRequest struct {
	next    *blockedRequest
	request *pushRequest
}

// linked list helper to store an ordered list of blocked requests
type blockedRequests struct {
	first *blockedRequest
	last  *blockedRequest
}

const QueueType = "proxy"

// FactoryForSettings is a simple wrapper around NewQueue so a concrete
// Settings object can be wrapped in a queue-agnostic interface for
// later use by the pipeline.
func FactoryForSettings(settings Settings) queue.QueueFactory {
	return func(
		logger *logp.Logger,
		ackCallback func(eventCount int),
		inputQueueSize int,
		encoderFactory queue.EncoderFactory,
	) (queue.Queue, error) {
		return NewQueue(logger, ackCallback, settings, encoderFactory), nil
	}
}

// NewQueue creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewQueue(
	logger *logp.Logger,
	ackCallback func(eventCount int),
	settings Settings,
	encoderFactory queue.EncoderFactory,
) *broker {
	if logger == nil {
		logger = logp.NewLogger("proxyqueue")
	}

	b := &broker{
		doneChan:  make(chan struct{}),
		logger:    logger,
		batchSize: settings.BatchSize,

		// broker API channels
		pushChan: make(chan *pushRequest),
		getChan:  make(chan getRequest),

		ackCallback: ackCallback,
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.run()
	}()

	return b
}

func (b *broker) Close() error {
	close(b.doneChan)
	b.wg.Wait()
	return nil
}

func (b *broker) QueueType() string {
	return QueueType
}

func (b *broker) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{}
}

func (b *broker) Producer(cfg queue.ProducerConfig) queue.Producer {
	return newProducer(b, cfg.ACK)
}

func (b *broker) Get(_ int) (queue.Batch, error) {
	// The response channel needs a buffer size of 1 to guarantee that the
	// broker routine will not block when sending the response.
	responseChan := make(chan *batch, 1)
	select {
	case <-b.doneChan:
		return nil, io.EOF
	case b.getChan <- getRequest{responseChan: responseChan}:
	}

	// if request has been sent, we are guaranteed a response
	return <-responseChan, nil
}

// Metrics returns an empty response because the proxy queue
// doesn't accumulate batches; for the real metadata, use either the
// Beats pipeline metrics, or the queue metrics in the shipper, which
// is where pending events are really queued when the proxy queue is
// in use.
func (b *broker) Metrics() (queue.Metrics, error) {
	return queue.Metrics{}, nil
}

func (b *broker) run() {
	for {
		var getChan chan getRequest
		// Get requests are enabled if the current pending batch is nonempty.
		if len(b.queuedEntries) > 0 {
			getChan = b.getChan
		}

		select {
		case <-b.doneChan:
			// The queue is closing, reject any requests that were blocked
			// waiting for space in the queue.
			blocked := b.blockedRequests
			for req := blocked.next(); req != nil; req = blocked.next() {
				req.responseChan <- false
			}
			return

		case req := <-b.pushChan: // producer pushing new event
			b.handlePushRequest(req)

		case req := <-getChan: // consumer asking for next batch
			b.handleGetRequest(req)

		case <-b.outstandingBatches.nextDoneChan():
			ackedBatch := b.outstandingBatches.remove()
			// Notify any listening producers
			for _, ack := range ackedBatch.producerACKs {
				ack.producer.ackHandler(ack.count)
			}
			// Notify the pipeline's metrics reporter
			//nolint:typecheck // this nil check is ok
			if b.ackCallback != nil {
				b.ackCallback(ackedBatch.originalEntryCount)
			}
		}
	}
}

func (b *broker) handlePushRequest(req *pushRequest) {
	if len(b.queuedEntries) < b.batchSize {
		b.queuedEntries = append(b.queuedEntries,
			queueEntry{event: req.event, producer: req.producer})
		if req.producer != nil {
			req.producer.producedCount++
		}
		req.responseChan <- true
	} else if req.canBlock {
		// If there isn't room for the event, but the producer wants
		// to block until there is, add it to the queue.
		b.blockedRequests.add(req)
	} else {
		// The pending batch is full, the producer doesn't want to
		// block, so return immediate failure.
		req.responseChan <- false
	}
}

func (b *broker) handleGetRequest(req getRequest) {
	acks := acksForEntries(b.queuedEntries)

	newBatch := &batch{
		entries:            b.queuedEntries,
		originalEntryCount: len(b.queuedEntries),
		producerACKs:       acks,
		doneChan:           make(chan struct{}),
	}
	b.outstandingBatches.add(newBatch)
	req.responseChan <- newBatch

	// Unblock any pending requests we can fit into the new batch.
	entries := []queueEntry{}
	for len(entries) < b.batchSize {
		req := b.blockedRequests.next()
		if req == nil {
			// No more blocked requests
			break
		}

		entries = append(entries,
			queueEntry{event: req.event, producer: req.producer})
		if req.producer != nil {
			req.producer.producedCount++
		}
		req.responseChan <- true
	}

	// Reset the pending entries
	b.queuedEntries = entries
}

// Adds a new request to the end of the current list.
func (b *blockedRequests) add(request *pushRequest) {
	blockedReq := &blockedRequest{request: request}
	if b.first == nil {
		b.first = blockedReq
	} else {
		b.last.next = blockedReq
	}
	b.last = blockedReq
}

// Removes the oldest request from the list and returns it.
func (b *blockedRequests) next() *pushRequest {
	var result *pushRequest
	if b.first != nil {
		result = b.first.request
		b.first = b.first.next
		if b.first == nil {
			b.last = nil
		}
	}
	return result
}

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

package queue

import (
	"github.com/elastic/elastic-agent-libs/logp"
)

// Entry is a placeholder type for the objects contained by the queue, which
// can be anything (but right now is always a publisher.Event). We could just
// use interface{} everywhere but this makes the API's intentions clearer
// and reduces accidental type mismatches.
type Entry interface{}

// Queue is responsible for accepting, forwarding and ACKing events.
// A queue will receive and buffer single events from its producers.
// Consumers will receive events in batches from the queues buffers.
// Once a consumer has finished processing a batch, it must ACK the batch, for
// the queue to advance its buffers. Events in progress or ACKed are not readable
// from the queue.
// When the queue decides it is safe to progress (events have been ACKed by
// consumer or flush to some other intermediate storage), it will send an ACK signal
// with the number of ACKed events to the Producer (ACK happens in batches).
type Queue interface {
	// Close signals the queue to shut down, but it may keep handling requests
	// and acknowledgments for events that are already in progress.
	Close() error

	// Done returns a channel that unblocks when the queue is closed and all
	// its events are persisted or acknowledged.
	Done() <-chan struct{}

	QueueType() string
	BufferConfig() BufferConfig

	Producer(cfg ProducerConfig) Producer

	// Get retrieves a batch of up to eventCount events. If eventCount <= 0,
	// there is no bound on the number of returned events.
	Get(eventCount int) (Batch, error)
}

// If encoderFactory is provided, then the resulting queue must use it to
// encode queued events before returning them.
type QueueFactory func(
	logger *logp.Logger,
	observer Observer,
	inputQueueSize int,
	encoderFactory EncoderFactory,
) (Queue, error)

// BufferConfig returns the pipelines buffering settings,
// for the pipeline to use.
// In case of the pipeline itself storing events for reporting ACKs to clients,
// but still dropping events, the pipeline can use the buffer information,
// to define an upper bound of events being active in the pipeline.
type BufferConfig struct {
	// MaxEvents is the maximum number of events the queue can hold at capacity.
	// A value <= 0 means there is no fixed limit.
	MaxEvents int
}

// ProducerConfig as used by the Pipeline to configure some custom callbacks
// between pipeline and queue.
type ProducerConfig struct {
	// if ACK is set, the callback will be called with number of events produced
	// by the producer instance and being ACKed by the queue.
	ACK func(count int)
}

type EntryID uint64

// Producer is an interface to be used by the pipelines client to forward
// events to a queue.
type Producer interface {
	// Publish adds an entry to the queue, blocking if necessary, and returns
	// the new entry's id and true on success.
	Publish(entry Entry) (EntryID, bool)

	// TryPublish adds an entry to the queue if doing so will not block the
	// caller, otherwise it immediately returns. The reasons a publish attempt
	// might block are defined by the specific queue implementation and its
	// configuration. If the event was successfully added, returns true with
	// the event's assigned ID, and false otherwise.
	TryPublish(entry Entry) (EntryID, bool)

	// Close closes this Producer endpoint.
	// Note: A queue may still send ACK signals even after Close is called on
	// the originating Producer. The pipeline client must accept these ACKs.
	Close()
}

// Batch of entries (usually publisher.Event) to be returned to Consumers.
// The `Done` method will tell the queue that the batch has been consumed and
// its entries can be acknowledged and discarded.
type Batch interface {
	Count() int
	Entry(i int) Entry
	Done()
}

// Outputs can provide an EncoderFactory to enable early encoding, in which
// case the queue will run the given encoder on events before they reach
// consumers.
// Encoders are provided as factories so each worker goroutine can have its own
type EncoderFactory func() Encoder

type Encoder interface {
	// Return the encoded form of the entry that the output workers can use,
	// and the in-memory size of the encoded buffer.
	// EncodeEntry should return a valid Entry when given one, even if the
	// encoding fails. In that case, the returned Entry should contain the
	// metadata needed to report the error when the entry is consumed.
	EncodeEntry(Entry) (Entry, int)
}

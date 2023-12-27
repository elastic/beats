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
	"errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/opt"
)

// Metrics is a set of basic-user friendly metrics that report the current state of the queue. These metrics are meant to be relatively generic and high-level, and when reported directly, can be comprehensible to a user.
type Metrics struct {
	//EventCount is the total events currently in the queue
	EventCount opt.Uint
	//ByteCount is the total byte size of the queue
	ByteCount opt.Uint
	//ByteLimit is the user-configured byte limit of the queue
	ByteLimit opt.Uint
	//EventLimit is the user-configured event limit of the queue
	EventLimit opt.Uint

	//UnackedConsumedEvents is the count of events that an output consumer has read, but not yet ack'ed
	UnackedConsumedEvents opt.Uint

	//OldestActiveTimestamp is the timestamp of the oldest item in the queue.
	OldestActiveTimestamp common.Time

	// OldestActiveID is ID of the oldest unacknowledged event in the queue, or
	// the next ID that will be assigned if the queue is empty.
	OldestEntryID EntryID
}

// ErrMetricsNotImplemented is a hopefully temporary type to mark queue metrics as not yet implemented
var ErrMetricsNotImplemented = errors.New("Queue metrics not implemented")

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
	Close() error

	QueueType() string
	BufferConfig() BufferConfig

	Producer(cfg ProducerConfig) Producer

	// Get retrieves a batch of up to eventCount events. If eventCount <= 0,
	// there is no bound on the number of returned events.
	Get(eventCount int) (Batch, error)

	Metrics() (Metrics, error)
}

type QueueFactory func(logger *logp.Logger, ack func(eventCount int), inputQueueSize int) (Queue, error)

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

	// OnDrop is called to report events being silently dropped by
	// the queue. Currently this can only happen when a Publish call is sent
	// to the memory queue's request channel but the producer is cancelled
	// before it reaches the queue buffer.
	OnDrop func(interface{})

	// DropOnCancel is a hint to the queue to drop events if the producer disconnects
	// via Cancel.
	DropOnCancel bool
}

type EntryID uint64

// Producer is an interface to be used by the pipelines client to forward
// events to a queue.
type Producer interface {
	// Publish adds an event to the queue, blocking if necessary, and returns
	// the new entry's id and true on success.
	Publish(event interface{}) (EntryID, bool)

	// TryPublish adds an event to the queue if doing so will not block the
	// caller, otherwise it immediately returns. The reasons a publish attempt
	// might block are defined by the specific queue implementation and its
	// configuration. If the event was successfully added, returns true with
	// the event's assigned ID, and false otherwise.
	TryPublish(event interface{}) (EntryID, bool)

	// Cancel closes this Producer endpoint. If the producer is configured to
	// drop its events on Cancel, the number of dropped events is returned.
	// Note: A queue may still send ACK signals even after Cancel is called on
	//       the originating Producer. The pipeline client must accept and
	//       discard these ACKs.
	Cancel() int
}

// Batch of events to be returned to Consumers. The `Done` method will tell the
// queue that the batch has been consumed and its events can be discarded.
type Batch interface {
	Count() int
	Entry(i int) interface{}
	// Release the internal references to the contained events.
	// Count() and Entry() cannot be used after this call.
	// This is only guaranteed to release references when using the
	// proxy queue, where it is used to avoid keeping multiple copies
	// of events that have already been queued by the shipper.
	FreeEntries()
	Done()
}

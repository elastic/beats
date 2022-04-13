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
	"io"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Factory for creating a queue used by a pipeline instance.
type Factory func(ACKListener, *logp.Logger, *common.Config, int) (Queue, error)

// ACKListener listens to special events to be send by queue implementations.
type ACKListener interface {
	OnACK(eventCount int) // number of consecutively published events acked by producers
}

//Metrics is a set of basic-user friendly metrics that report the current state of the queue. These metrics are meant to be relatively generic and high-level, and when reported directly, can be comprehensible to a user.
type Metrics struct {
	//QueueLimitCount is the count of times that the queue has reached it's user-configured limit, either in terms of storage space, or count of events.
	QueueLimitCount uint64
	//QueueIsFull is a simple bool value that indicates if the queue is currently full as per it's user-configured limits.
	QueueIsFull bool
	//QueueLevelPct is the current "full level" of the queue, expressed as percentage from 0.0 to 1.0.
	QueueLevelPct float64
	//QueueLevelCurrent is the current capacity of the queue, expressed in the "native" units of the queue implementation, be it event count, MBs, etc.
	QueueLevelCurrent uint64
	//LongestWaitingItem is the timestamp of the oldest item in the queue.
	LongestWaitingItem common.Time
	//QueueLag is the difference between the consumer and producer position in the queue.
}

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
	io.Closer

	BufferConfig() BufferConfig

	Producer(cfg ProducerConfig) Producer
	Consumer() Consumer

	Metrics() (Metrics, error)
}

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

	// OnDrop provided to the queue, to report events being silently dropped by
	// the queue. For example an async producer close and publish event,
	// with close happening early might result in the event being dropped. The callback
	// gives a queue user a chance to keep track of total number of events
	// being buffered by the queue.
	OnDrop func(beat.Event)

	// DropOnCancel is a hint to the queue to drop events if the producer disconnects
	// via Cancel.
	DropOnCancel bool
}

// Producer is an interface to be used by the pipelines client to forward
// events to a queue.
type Producer interface {
	// Publish adds an event to the queue, blocking if necessary, and returns
	// true on success.
	Publish(event publisher.Event) bool

	// TryPublish adds an event to the queue if doing so will not block the
	// caller, otherwise it immediately returns. The reasons a publish attempt
	// might block are defined by the specific queue implementation and its
	// configuration. Returns true if the event was successfully added, false
	// otherwise.
	TryPublish(event publisher.Event) bool

	// Cancel closes this Producer endpoint. If the producer is configured to
	// drop its events on Cancel, the number of dropped events is returned.
	// Note: A queue may still send ACK signals even after Cancel is called on
	//       the originating Producer. The pipeline client must accept and
	//       discard these ACKs.
	Cancel() int
}

// Consumer is an interface to be used by the pipeline output workers,
// used to read events from the head of the queue.
type Consumer interface {
	// Get retrieves a batch of up to eventCount events. If eventCount <= 0,
	// there is no bound on the number of returned events.
	Get(eventCount int) (Batch, error)

	// Close closes this Consumer. Returns an error if the Consumer is
	// already closed.
	Close() error
}

// Batch of events to be returned to Consumers. The `ACK` method will send the
// ACK signal to the queue.
type Batch interface {
	Events() []publisher.Event
	ACK()
}

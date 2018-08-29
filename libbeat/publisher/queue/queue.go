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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

// Factory for creating a queue used by a pipeline instance.
type Factory func(Eventer, *logp.Logger, *common.Config) (Queue, error)

// Eventer listens to special events to be send by queue implementations.
type Eventer interface {
	OnACK(int) // number of consecutively published messages, acked by producers
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
}

// BufferConfig returns the pipelines buffering settings,
// for the pipeline to use.
// In case of the pipeline itself storing events for reporting ACKs to clients,
// but still dropping events, the pipeline can use the buffer information,
// to define an upper bound of events being active in the pipeline.
type BufferConfig struct {
	Events int // can be <= 0, if queue can not determine limit
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

// Producer interface to be used by the pipelines client to forward events to be
// published to the queue.
// When a producer calls `Cancel`, it's up to the queue to send or remove
// events not yet ACKed.
// Note: A queue is still allowed to send the ACK signal after Cancel. The
//       pipeline client must filter out ACKs after cancel.
type Producer interface {
	Publish(event publisher.Event) bool
	TryPublish(event publisher.Event) bool
	Cancel() int
}

// Consumer interface to be used by the pipeline output workers.
// The `Get` method retrieves a batch of events up to size `sz`. If sz <= 0,
// the batch size is up to the queue.
type Consumer interface {
	Get(sz int) (Batch, error)
	Close() error
}

// Batch of events to be returned to Consumers. The `ACK` method will send the
// ACK signal to the queue.
type Batch interface {
	Events() []publisher.Event
	ACK()
}

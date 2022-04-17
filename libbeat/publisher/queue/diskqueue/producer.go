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

package diskqueue

import (
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue"
)

type diskQueueProducer struct {
	// The disk queue that created this producer.
	queue *diskQueue

	// The configuration this producer was created with.
	config queue.ProducerConfig

	encoder *eventEncoder

	// When a producer is cancelled, cancelled is set to true and the done
	// channel is closed. (We could get by with just a done channel, but we
	// need to make sure that calling Cancel repeatedly doesn't close an
	// already-closed channel, which would panic.)
	cancelled bool
	done      chan struct{}
}

// A request sent from a producer to the core loop to add a frame to the queue.
type producerWriteRequest struct {
	frame        *writeFrame
	shouldBlock  bool
	responseChan chan bool
}

//
// diskQueueProducer implementation of the queue.Producer interface
//

func (producer *diskQueueProducer) Publish(event publisher.Event) bool {
	return producer.publish(event, true)
}

func (producer *diskQueueProducer) TryPublish(event publisher.Event) bool {
	return producer.publish(event, false)
}

func (producer *diskQueueProducer) publish(
	event publisher.Event, shouldBlock bool,
) bool {
	if producer.cancelled {
		return false
	}
	serialized, err := producer.encoder.encode(&event)
	if err != nil {
		producer.queue.logger.Errorf(
			"Couldn't serialize incoming event: %v", err)
		return false
	}
	request := producerWriteRequest{
		frame: &writeFrame{
			serialized: serialized,
			producer:   producer,
		},
		shouldBlock: shouldBlock,
		// This response channel will be used by the core loop, so it must have
		// buffer size 1 to guarantee that the core loop will not need to block.
		responseChan: make(chan bool, 1),
	}

	select {
	case producer.queue.producerWriteRequestChan <- request:
		// The request has been sent, and we are now guaranteed to get a result on
		// the response channel, so we must read from it immediately to avoid
		// blocking the core loop.
		response := <-request.responseChan
		return response
	case <-producer.queue.done:
		return false
	case <-producer.done:
		return false
	}
}

func (producer *diskQueueProducer) Cancel() int {
	if producer.cancelled {
		return 0
	}
	producer.cancelled = true
	close(producer.done)

	// TODO (possibly?): message the core loop to remove any pending events that
	// were sent through this producer. If we do, return the number of cancelled
	// events here instead of zero.
	return 0
}

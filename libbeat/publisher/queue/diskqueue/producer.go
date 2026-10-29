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
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
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

	// ackWait is closed when Close is called. The disk queue persists events
	// durably and does not track per-producer in-memory acknowledgments, so
	// there is nothing to wait for beyond Close itself.
	ackWait chan struct{}
	ackOnce sync.Once
}

var _ queue.ContextProducer[publisher.Event] = (*diskQueueProducer)(nil)

// A request sent from a producer to the core loop to add a frame to the queue.
type producerWriteRequest struct {
	frame       *writeFrame
	shouldBlock bool
	// responseChan gets exactly one value once the core loop has received the
	// request: nil if the frame was added, else why it was not. That is an
	// error wrapping queue.ErrRejected for a frame the queue can never take,
	// errQueueFull, queue.ErrClosed or errCancelled.
	responseChan chan error
}

var (
	// errQueueFull answers a non-blocking request that does not fit.
	errQueueFull = errors.New("diskqueue: queue is full")
	// errCancelled answers a blocked request withdrawn through
	// diskQueue.producerCancelChan.
	errCancelled = errors.New("diskqueue: request cancelled")
)

//
// diskQueueProducer implementation of the queue.Producer interface
//

func (producer *diskQueueProducer) Publish(event publisher.Event) (queue.EntryID, bool) {
	return 0, producer.publish(context.Background(), event, true) == nil
}

// PublishContext is Publish, but stops waiting for room when ctx is done. See
// queue.ContextProducer.
func (producer *diskQueueProducer) PublishContext(ctx context.Context, event publisher.Event) (queue.EntryID, error) {
	return 0, producer.publish(ctx, event, true)
}

func (producer *diskQueueProducer) TryPublish(event publisher.Event) (queue.EntryID, bool) {
	return 0, producer.publish(context.Background(), event, false) == nil
}

func (producer *diskQueueProducer) publish(
	ctx context.Context, event publisher.Event, shouldBlock bool,
) error {
	if producer.cancelled {
		return queue.ErrClosed
	}
	serialized, err := producer.encoder.encode(event)
	if err != nil {
		producer.queue.logger.Errorf(
			"Couldn't serialize incoming event: %v", err)
		return fmt.Errorf("%w: serializing event: %w", queue.ErrRejected, err)
	}
	request := producerWriteRequest{
		frame: &writeFrame{
			serialized: serialized,
			producer:   producer,
		},
		shouldBlock: shouldBlock,
		// This response channel will be used by the core loop, so it must have
		// buffer size 1 to guarantee that the core loop will not need to block.
		responseChan: make(chan error, 1),
	}

	select {
	case producer.queue.producerWriteRequestChan <- request:
	case <-ctx.Done():
		return ctx.Err()
	case <-producer.queue.close:
		return queue.ErrClosed
	case <-producer.done:
		return queue.ErrClosed
	}

	// The request has been sent, so the core loop will answer it exactly once.
	select {
	case err := <-request.responseChan:
		return err
	case <-ctx.Done():
	}
	// Ask the core loop to withdraw the request. Either way it answers: with
	// errCancelled if it withdrew the request, or with the result it had
	// already sent. Shutdown answers every blocked request, so this cannot hang
	// once the core loop has stopped.
	select {
	case producer.queue.producerCancelChan <- request.responseChan:
		err = <-request.responseChan
	case err = <-request.responseChan:
	}
	if errors.Is(err, errCancelled) {
		return ctx.Err()
	}
	return err
}

func (producer *diskQueueProducer) Close() {
	// Resolve ackWait immediately rather than waiting for the queue to drain.
	// The disk queue is durable and can hold a very large (multi-GB) backlog.
	producer.ackOnce.Do(func() { close(producer.ackWait) })
	if producer.cancelled {
		return
	}
	producer.cancelled = true
	close(producer.done)
}

func (producer *diskQueueProducer) ACKWaitChan() <-chan struct{} {
	return producer.ackWait
}

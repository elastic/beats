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
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/common/atomic"
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue"
)

type diskQueueConsumer struct {
	queue  *diskQueue
	closed atomic.Bool
	done   chan struct{}
}

type diskQueueBatch struct {
	queue  *diskQueue
	frames []*readFrame
}

//
// diskQueueConsumer implementation of the queue.Consumer interface
//

func (consumer *diskQueueConsumer) Get(eventCount int) (queue.Batch, error) {
	// We can always eventually read at least one frame unless the queue or the
	// consumer is closed.
	var frames []*readFrame
	select {
	case frame, ok := <-consumer.queue.readerLoop.output:
		if !ok {
			return nil, fmt.Errorf("tried to read from a closed disk queue")
		}
		frames = []*readFrame{frame}
	case <-consumer.done:
		return nil, fmt.Errorf("tried to read from a closed disk queue consumer")
	}
eventLoop:
	for eventCount <= 0 || len(frames) < eventCount {
		select {
		case frame, ok := <-consumer.queue.readerLoop.output:
			if !ok {
				// The queue was closed while we were reading it, just send back
				// what we have so far.
				break eventLoop
			}
			frames = append(frames, frame)
		default:
			// We can't read any more frames without blocking, so send back
			// what we have now.
			break eventLoop
		}
	}

	// There is a mild race condition here based on queue closure: events
	// written to readerLoop.output may have been buffered before the
	// queue was closed, and we may be reading its leftovers afterwards.
	// We could try to detect this case here by checking the
	// consumer.queue.done channel, and return nothing if it's been closed.
	// But this gives rise to another race: maybe the queue was
	// closed _after_ we read those frames, and we _ought_ to return them
	// to the reader. The queue interface doesn't specify the proper
	// behavior in this case.
	//
	// Lacking formal requirements, we elect to be permissive: if we have
	// managed to read frames, then the queue already knows and considers them
	// "read," so we lose no consistency by returning them. If someone closes
	// the queue while we are draining the channel, nothing changes functionally
	// except that any ACKs after that point will be ignored. A well-behaved
	// Beats shutdown will always ACK / close its consumers before closing the
	// queue itself, so we expect this corner case not to arise in practice, but
	// if it does it is innocuous.
	return &diskQueueBatch{
		queue:  consumer.queue,
		frames: frames,
	}, nil
}

func (consumer *diskQueueConsumer) Close() error {
	if consumer.closed.Swap(true) {
		return fmt.Errorf("already closed")
	}
	close(consumer.done)
	return nil
}

//
// diskQueueBatch implementation of the queue.Batch interface
//

func (batch *diskQueueBatch) Events() []publisher.Event {
	events := make([]publisher.Event, len(batch.frames))
	for i, frame := range batch.frames {
		events[i] = frame.event
	}
	return events
}

func (batch *diskQueueBatch) ACK() {
	batch.queue.acks.addFrames(batch.frames)
}

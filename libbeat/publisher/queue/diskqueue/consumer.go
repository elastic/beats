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

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type diskQueueBatch struct {
	queue  *diskQueue
	frames []*readFrame
}

func (dq *diskQueue) Get(eventCount int, _ int) (queue.Batch, error) {
	// We can always eventually read at least one frame unless the queue or the
	// consumer is closed.
	frame, ok := <-dq.readerLoop.output
	if !ok {
		return nil, fmt.Errorf("tried to read from a closed disk queue")
	}
	frames := []*readFrame{frame}

eventLoop:
	for eventCount <= 0 || len(frames) < eventCount {
		select {
		case frame, ok := <-dq.readerLoop.output:
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
	// queue.done channel, and return nothing if it's been closed.
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
	// Beats shutdown will always ACK its batches before closing the
	// queue itself, so we expect this corner case not to arise in practice, but
	// if it does it is innocuous.
	return &diskQueueBatch{
		queue:  dq,
		frames: frames,
	}, nil
}

//
// diskQueueBatch implementation of the queue.Batch interface
//

func (batch *diskQueueBatch) Count() int {
	return len(batch.frames)
}

func (batch *diskQueueBatch) Entry(i int) queue.Entry {
	return batch.frames[i].event
}

func (batch *diskQueueBatch) FreeEntries() {
}

func (batch *diskQueueBatch) Done() {
	batch.queue.acks.addFrames(batch.frames)
}

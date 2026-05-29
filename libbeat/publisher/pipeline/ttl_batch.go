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

package pipeline

import (
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type retryer interface {
	retry(batch *ttlBatch, decreaseTTL bool)
}

type ttlBatch struct {
	// The callback to inform the queue (and possibly the producer)
	// that this batch has been acknowledged.
	done func()

	// The internal hook back to the eventConsumer, used to implement the
	// publisher.Batch retry interface.
	retryer retryer

	// How many retries until we drop this batch. -1 means it can't be dropped.
	ttl int

	// The cached events returned from original.Events(). If some but not
	// all of the events are ACKed, those ones are removed from the list.
	events []publisher.Event

	// If split is non-nil, this batch was created by splitting another
	// batch when the output determined it was too large. In this case,
	// all split batches descending from the same original batch will
	// point to the same metadata.
	split *batchSplitData
}

type batchSplitData struct {
	// The original done callback, to be invoked when all split batches
	// descending from it have been completed.
	originalDone func()

	// The number of events awaiting acknowledgment from the original
	// batch. When this reaches zero, originalDone should be invoked.
	outstandingEvents atomic.Int64
}

func newBatch(retryer retryer, original queue.Batch[publisher.Event], ttl int) *ttlBatch {
	if original == nil {
		panic("empty batch")
	}

	count := original.Count()
	events := make([]publisher.Event, 0, count)
	for i := 0; i < count; i++ {
		events = append(events, original.Entry(i))
	}
	original.FreeEntries()

	b := &ttlBatch{
		done:    original.Done,
		retryer: retryer,
		ttl:     ttl,
		events:  events,
	}
	return b
}

func (b *ttlBatch) Events() []publisher.Event {
	return b.events
}

func (b *ttlBatch) ACK() {
	// Help the garbage collector clean up the event data a little faster
	b.events = nil
	b.done()
}

func (b *ttlBatch) Drop() {
	// Help the garbage collector clean up the event data a little faster
	b.events = nil
	b.done()
}

// SplitRetry is called by the output to report that the batch is
// too large to ingest. It splits the events into two separate batches
// and sends both of them back to the retryer. Returns false if the
// batch could not be split.
func (b *ttlBatch) SplitRetry() bool {
	if len(b.events) < 2 {
		// This batch is already as small as it can get
		return false
	}
	splitData := b.split
	if splitData == nil {
		// Splitting a previously unsplit batch, create the metadata
		splitData = &batchSplitData{
			originalDone: b.done,
		}
		// Initialize to the number of events in the original batch
		splitData.outstandingEvents.Add(int64(len(b.events)))
	}
	splitIndex := len(b.events) / 2
	events1 := b.events[:splitIndex]
	events2 := b.events[splitIndex:]
	b.retryer.retry(&ttlBatch{
		events:  events1,
		done:    splitData.doneCallback(len(events1)),
		retryer: b.retryer,
		ttl:     b.ttl,
		split:   splitData,
	}, false)
	b.retryer.retry(&ttlBatch{
		events:  events2,
		done:    splitData.doneCallback(len(events2)),
		retryer: b.retryer,
		ttl:     b.ttl,
		split:   splitData,
	}, false)
	return true
}

// splitByDestination splits b into one batch per distinct event Source so each
// destination can be published and acknowledged independently. All resulting
// batches share completion accounting with the original (via batchSplitData),
// so the underlying queue read is acknowledged only after every destination's
// batch has completed. This keeps the queue's event cap unchanged no matter how
// many destinations a batch fans out to: the events stay accounted against the
// queue until all of them are done. It returns nil when the batch targets a
// single destination, in which case the caller should use b unchanged.
func (b *ttlBatch) splitByDestination() []*ttlBatch {
	groups := groupEventsBySource(b.events)
	if len(groups) < 2 {
		// Single destination (or empty): no split needed.
		return nil
	}

	splitData := b.split
	if splitData == nil {
		// Splitting a previously unsplit batch, create the metadata.
		splitData = &batchSplitData{
			originalDone: b.done,
		}
		// Initialize to the number of events in the original batch.
		splitData.outstandingEvents.Add(int64(len(b.events)))
	}

	batches := make([]*ttlBatch, len(groups))
	for i, events := range groups {
		batches[i] = &ttlBatch{
			events:  events,
			done:    splitData.doneCallback(len(events)),
			retryer: b.retryer,
			ttl:     b.ttl,
			split:   splitData,
		}
	}
	return batches
}

// groupEventsBySource partitions events into one slice per distinct
// publisher.Event.Source, preserving the order in which sources first appear.
// Events with a nil Source are grouped together. The number of distinct sources
// (one per pipeline sharing the queue) is small, so a linear scan is used and
// each group is allocated to its exact size.
func groupEventsBySource(events []publisher.Event) [][]publisher.Event {
	var sources []*beat.Info
	var counts []int
	indexOf := func(source *beat.Info) int {
		for i, s := range sources {
			if s == source {
				return i
			}
		}
		return -1
	}

	// First pass: discover the distinct sources and how many events each holds.
	for i := range events {
		source := events[i].Source
		idx := indexOf(source)
		if idx < 0 {
			sources = append(sources, source)
			counts = append(counts, 0)
			idx = len(sources) - 1
		}
		counts[idx]++
	}

	groups := make([][]publisher.Event, len(sources))
	for i := range groups {
		groups[i] = make([]publisher.Event, 0, counts[i])
	}

	// Second pass: place each event into its source's group.
	for i := range events {
		idx := indexOf(events[i].Source)
		groups[idx] = append(groups[idx], events[i])
	}
	return groups
}

// returns a callback to acknowledge the given number of events from
// a batch fragment.
func (splitData *batchSplitData) doneCallback(eventCount int) func() {
	return func() {
		remaining := splitData.outstandingEvents.Add(-int64(eventCount))
		if remaining == 0 {
			splitData.originalDone()
		}
	}
}

func (b *ttlBatch) Retry() {
	b.retryer.retry(b, true)
}

func (b *ttlBatch) Cancelled() {
	b.retryer.retry(b, false)
}

func (b *ttlBatch) RetryEvents(events []publisher.Event) {
	b.events = events
	b.Retry()
}

// reduceTTL reduces the time to live for all events that have no 'guaranteed'
// sending requirements.  reduceTTL returns true if the batch is still alive.
func (b *ttlBatch) reduceTTL() bool {
	if b.ttl <= 0 {
		return true
	}

	b.ttl--
	if b.ttl > 0 {
		return true
	}

	// filter for events with guaranteed send flags
	events := b.events[:0]
	for _, event := range b.events {
		if event.Guaranteed() {
			events = append(events, event)
		}
	}
	b.events = events

	if len(b.events) > 0 {
		b.ttl = -1 // we need infinite retry for all events left in this batch
		return true
	}

	// all events have been dropped:
	return false
}

///////////////////////////////////////////////////////////////////////
// Testing support helpers

// NewBatchForTesting creates a ttlBatch (exposed through its publisher
// interface). This is exposed publicly to support testing of ttlBatch
// with other pipeline components, it should never be used to create
// a batch in live pipeline code.
//
//   - events: the publisher events contained in the batch
//   - ttl: the number of retries left until the batch is dropped. -1 means it
//     can't be dropped.
//   - retryCallback: the callback invoked when a batch needs to be retried.
//     In a live pipeline, this points to the retry method on eventConsumer,
//     the helper object that distributes pending batches to output workers.
//   - done: the callback invoked on receiving batch.Done
func NewBatchForTesting(
	events []publisher.Event,
	retryCallback func(batch publisher.Batch),
	done func(),
) publisher.Batch {
	return &ttlBatch{
		events:  events,
		done:    done,
		retryer: testingRetryer{retryCallback},
	}
}

// testingRetryer is a simple wrapper of the retryer interface that is
// used by NewBatchForTesting, to allow tests in other packages to interoperate
// with the internal type ttlBatch.
type testingRetryer struct {
	retryCallback func(batch publisher.Batch)
}

func (tr testingRetryer) retry(batch *ttlBatch, _ bool) {
	tr.retryCallback(batch)
}

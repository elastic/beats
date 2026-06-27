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

	// release is the queue.Batch.Release callback: returns backing
	// storage to the queue without firing producer ACK callbacks. Set
	// by newBatch from the wrapped queue.Batch.
	release func()

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
	// originalDone is the wrapped queue.Batch.Done — invoked when every
	// descendant of the original batch has been Done()'d (no abandonment).
	originalDone func()

	// originalRelease is the wrapped queue.Batch.Release — invoked when
	// the last descendant of the original batch finishes AND at least one
	// descendant was Release()'d (abandoned). This keeps the at-least-
	// once contract: a partially abandoned batch must not signal
	// successful delivery to the input via originalDone.
	originalRelease func()

	// outstandingEvents counts events still awaiting completion from
	// either Done or Release. When it reaches zero, one of
	// originalDone/originalRelease fires (chosen by anyReleased).
	outstandingEvents atomic.Int64

	// anyReleased is set true the first time a descendant invokes its
	// release callback. If set when outstandingEvents hits zero,
	// originalRelease fires instead of originalDone.
	anyReleased atomic.Bool
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
		release: original.Release,
		retryer: retryer,
		ttl:     ttl,
		events:  events,
	}
	return b
}

// Release returns the batch's backing storage to the underlying queue
// without firing producer ACK callbacks. It is the abandonment path,
// to be used only on pipeline shutdown — see queue.Batch.Release for
// the caller contract. ACK / Drop are the normal completion paths.
//
// For a batch created by SplitRetry, the release callback decrements
// the shared outstanding-events counter and, when all descendants
// finish, invokes the original queue.Batch.Release if any descendant
// was Released (preserving at-least-once for partially abandoned
// splits).
func (b *ttlBatch) Release() {
	b.events = nil
	if b.release != nil {
		b.release()
	}
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
			originalDone:    b.done,
			originalRelease: b.release,
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
		release: splitData.releaseCallback(len(events1)),
		retryer: b.retryer,
		ttl:     b.ttl,
		split:   splitData,
	}, false)
	b.retryer.retry(&ttlBatch{
		events:  events2,
		done:    splitData.doneCallback(len(events2)),
		release: splitData.releaseCallback(len(events2)),
		retryer: b.retryer,
		ttl:     b.ttl,
		split:   splitData,
	}, false)
	return true
}

// doneCallback returns a callback used as a descendant batch's done
// hook: it decrements outstandingEvents by eventCount and, when the
// last descendant finishes, invokes the original Done — unless any
// sibling was Released, in which case originalRelease fires instead.
func (splitData *batchSplitData) doneCallback(eventCount int) func() {
	return func() {
		remaining := splitData.outstandingEvents.Add(-int64(eventCount))
		if remaining == 0 {
			splitData.completeOriginal()
		}
	}
}

// releaseCallback returns a callback used as a descendant batch's
// release hook: it records that abandonment occurred and, on the last
// descendant, invokes originalRelease (never originalDone — once any
// descendant is abandoned, the whole batch is treated as abandoned for
// at-least-once correctness).
func (splitData *batchSplitData) releaseCallback(eventCount int) func() {
	return func() {
		splitData.anyReleased.Store(true)
		remaining := splitData.outstandingEvents.Add(-int64(eventCount))
		if remaining == 0 {
			splitData.completeOriginal()
		}
	}
}

// completeOriginal invokes the appropriate terminal callback on the
// wrapped queue.Batch — Release if any descendant was abandoned,
// otherwise Done. Called from doneCallback / releaseCallback exactly
// once, when outstandingEvents reaches zero.
func (splitData *batchSplitData) completeOriginal() {
	if splitData.anyReleased.Load() {
		if splitData.originalRelease != nil {
			splitData.originalRelease()
		}
		return
	}
	splitData.originalDone()
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

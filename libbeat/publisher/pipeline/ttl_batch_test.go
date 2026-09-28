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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestBatchSplitRetry(t *testing.T) {
	// SplitRetry should:
	// - send two batches to the retryer, each with half the events
	// - give each a callback that will fall through to the original one only
	//   after all descendant batches are acknowledged

	retryer := &mockRetryer{}
	events := make([]publisher.Event, 2)
	doneWasCalled := false

	rootBatch := ttlBatch{
		events:  events,
		retryer: retryer,
		done:    func() { doneWasCalled = true },
	}

	rootBatch.SplitRetry()

	require.Len(t, retryer.batches, 2, "SplitRetry should retry 2 batches")
	require.Len(t, retryer.batches[0].events, 1, "Retried batches should have one event each")
	require.Len(t, retryer.batches[1].events, 1, "Retried batches should have one event each")
	assert.Same(t, &events[0], &retryer.batches[0].events[0], "Retried batch events should match original")
	assert.Same(t, &events[1], &retryer.batches[1].events[0], "Retried batch events should match original")

	assert.False(t, doneWasCalled, "No batch callbacks should be received yet")
	retryer.batches[0].done()
	assert.False(t, doneWasCalled, "Original callback shouldn't be invoked until both children are")
	retryer.batches[1].done()
	assert.True(t, doneWasCalled, "Original callback should be invoked when all children are")
}

func TestNestedBatchSplit(t *testing.T) {
	// Test splitting the same original batch multiple times

	retryer := &mockRetryer{}
	events := make([]publisher.Event, 4)
	doneWasCalled := false

	rootBatch := ttlBatch{
		events:  events,
		retryer: retryer,
		done:    func() { doneWasCalled = true },
	}

	rootBatch.SplitRetry()
	require.Len(t, retryer.batches, 2, "SplitRetry should retry 2 batches")
	// Clear out the first-level batches from the retry buffer and retry both of them
	batches := retryer.batches
	retryer.batches = []*ttlBatch{}
	batches[0].SplitRetry()
	batches[1].SplitRetry()

	require.Len(t, retryer.batches, 4, "two SplitRetry calls should generate four retrys")

	for i := range 4 {
		assert.False(t, doneWasCalled, "Original callback shouldn't be invoked until all children are")
		require.Len(t, retryer.batches[i].events, 1, "Retried batches should have one event each")

		// We expect the indices in the retried batches to match because we retried them in order
		assert.Same(t, &events[i], &retryer.batches[i].events[0], "Retried batch events should match original")
		retryer.batches[i].done()
	}
	assert.True(t, doneWasCalled, "Original callback should be invoked when all children are")
}

// TestSplitRetryAllChildrenDone verifies the unchanged happy path: when
// every descendant of a split batch is Done()'d, originalDone fires and
// originalRelease does not.
func TestSplitRetryAllChildrenDone(t *testing.T) {
	retryer := &mockRetryer{}
	events := make([]publisher.Event, 4)
	doneCalled := false
	releaseCalled := false

	root := &ttlBatch{
		events:  events,
		retryer: retryer,
		done:    func() { doneCalled = true },
		release: func() { releaseCalled = true },
	}
	require.True(t, root.SplitRetry(), "SplitRetry should succeed")
	require.Len(t, retryer.batches, 2)

	// Done every descendant.
	for _, b := range retryer.batches {
		b.done()
	}
	assert.True(t, doneCalled, "all-Done → originalDone fires")
	assert.False(t, releaseCalled, "all-Done → originalRelease must NOT fire")
}

// TestSplitRetryAnyChildReleased verifies the at-least-once contract:
// if at least one descendant of a split batch is Released (abandoned),
// originalRelease fires instead of originalDone — even if siblings were
// Done()'d normally. Without this, a partially abandoned split would
// report a successful delivery and falsely advance the input registry.
func TestSplitRetryAnyChildReleased(t *testing.T) {
	retryer := &mockRetryer{}
	events := make([]publisher.Event, 4)
	doneCalled := false
	releaseCalled := false

	root := &ttlBatch{
		events:  events,
		retryer: retryer,
		done:    func() { doneCalled = true },
		release: func() { releaseCalled = true },
	}
	require.True(t, root.SplitRetry())
	require.Len(t, retryer.batches, 2)

	// One descendant is Released (e.g. consumer shut down while it was
	// in retryBatches), the other gets Done.
	retryer.batches[0].release()
	assert.False(t, doneCalled, "Original must not fire until all descendants finish")
	assert.False(t, releaseCalled, "Original must not fire until all descendants finish")

	retryer.batches[1].done()
	assert.False(t, doneCalled, "originalDone must NOT fire when any descendant was Released")
	assert.True(t, releaseCalled, "originalRelease must fire when any descendant was Released")
}

// TestSplitRetryAllChildrenReleased verifies the fully-abandoned path:
// every descendant Released → originalRelease fires exactly once.
func TestSplitRetryAllChildrenReleased(t *testing.T) {
	retryer := &mockRetryer{}
	events := make([]publisher.Event, 4)
	doneCalled := false
	releaseCount := 0

	root := &ttlBatch{
		events:  events,
		retryer: retryer,
		done:    func() { doneCalled = true },
		release: func() { releaseCount++ },
	}
	require.True(t, root.SplitRetry())

	for _, b := range retryer.batches {
		b.release()
	}
	assert.False(t, doneCalled, "originalDone must not fire on full release")
	assert.Equal(t, 1, releaseCount, "originalRelease must fire exactly once")
}

// TestSplitRetryReleaseInheritedThroughNestedSplit verifies that the
// release path survives splitting a child batch again. The leaf
// batches still share the original's batchSplitData and a Release on
// any leaf still trips originalRelease at completion.
func TestSplitRetryReleaseInheritedThroughNestedSplit(t *testing.T) {
	retryer := &mockRetryer{}
	events := make([]publisher.Event, 4)
	doneCalled := false
	releaseCalled := false

	root := &ttlBatch{
		events:  events,
		retryer: retryer,
		done:    func() { doneCalled = true },
		release: func() { releaseCalled = true },
	}
	require.True(t, root.SplitRetry())
	firstLevel := retryer.batches
	retryer.batches = nil

	for _, b := range firstLevel {
		require.True(t, b.SplitRetry(), "nested SplitRetry should succeed")
	}
	require.Len(t, retryer.batches, 4)

	// Three leaves Done, one Released.
	for i, b := range retryer.batches {
		if i == 0 {
			b.release()
		} else {
			b.done()
		}
	}
	assert.False(t, doneCalled, "originalDone must NOT fire when any leaf was Released")
	assert.True(t, releaseCalled, "originalRelease must fire when any leaf was Released, even across nested splits")
}

func TestBatchCallsDoneAndFreesEvents(t *testing.T) {
	doneCalled := false
	batch := &ttlBatch{
		done:   func() { doneCalled = true },
		events: []publisher.Event{{}},
	}
	require.NotNil(t, batch.events, "Initial batch events must be non-nil")
	batch.ACK()
	require.Nil(t, batch.events, "Calling batch.ACK should clear the events array")
	require.True(t, doneCalled, "Calling batch.ACK should invoke the done callback")

	doneCalled = false
	batch.events = []publisher.Event{{}}
	require.NotNil(t, batch.events, "Initial batch events must be non-nil")
	batch.Drop()
	require.Nil(t, batch.events, "Calling batch.Drop should clear the events array")
	require.True(t, doneCalled, "Calling batch.Drop should invoke the done callback")
}

func TestNewBatchFreesEvents(t *testing.T) {
	queueBatch := &mockQueueBatch{}
	_ = newBatch(nil, queueBatch, 0)
	assert.Equal(t, 1, queueBatch.freeEntriesCalled, "Creating a new ttlBatch should call FreeEntries on the underlying queue.Batch")
}

// TestTTLBatchRetryDoesNotReleaseSlots verifies that retrying a ttlBatch
// wrapping an slabqueue Batch keeps the underlying slots reserved for the
// entire retry chain — slots are only returned to the pool on final
// ACK/Drop. This is what guarantees the queue's max-events budget can never
// be violated even when batches bounce through the retry path repeatedly.
func TestTTLBatchRetryDoesNotReleaseSlots(t *testing.T) {
	pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{})
	for i := range 4 {
		_, ok := p.Publish(publisher.Event{Content: beat.Event{Private: i}})
		require.True(t, ok)
	}
	require.Equal(t, 0, pool.Available(), "pool should be full after 4 publishes")

	queueBatch, err := q.Get(0)
	require.NoError(t, err)
	require.Equal(t, 4, queueBatch.Count())

	retryer := &mockRetryer{}
	batch := newBatch(retryer, queueBatch, 3) // ttl=3

	// Retry several times — TTL decreases via reduceTTL but slots remain
	// reserved because the queue.Batch's Done has not been invoked.
	for i := range 3 {
		batch.Retry()
		require.Equal(t, 0, pool.Available(), "slots must stay reserved during retry %d", i+1)
	}
	require.Len(t, retryer.batches, 3, "retryer should have received 3 retry attempts")

	// ACK releases the slots in one shot.
	batch.ACK()
	assert.Equal(t, 4, pool.Available(), "all slots must be released after ACK")
}

// TestTTLBatchDropReleasesSlots verifies that Drop also releases slots, so
// when an output decides to drop a batch (e.g. permanent error or TTL
// exhausted) the queue is not leaked.
func TestTTLBatchDropReleasesSlots(t *testing.T) {
	pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 2}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{})
	p.Publish(publisher.Event{Content: beat.Event{Private: 1}})
	p.Publish(publisher.Event{Content: beat.Event{Private: 2}})
	require.Equal(t, 0, pool.Available())

	queueBatch, err := q.Get(0)
	require.NoError(t, err)

	batch := newBatch(&mockRetryer{}, queueBatch, 1)
	batch.Drop()
	assert.Equal(t, 2, pool.Available(), "slots must be released after Drop")
}

type mockQueueBatch struct {
	freeEntriesCalled int
}

func (b *mockQueueBatch) Count() int {
	return 1
}

func (b *mockQueueBatch) Done() {
}

func (b *mockQueueBatch) Release() {
}

func (b *mockQueueBatch) Entry(i int) publisher.Event {
	return publisher.Event{
		Content: beat.Event{
			Fields: mapstr.M{
				"message": fmt.Sprintf("event %v", i),
			},
		},
	}
}

func (b *mockQueueBatch) FreeEntries() {
	b.freeEntriesCalled++
}

type mockRetryer struct {
	batches []*ttlBatch
}

func (r *mockRetryer) retry(batch *ttlBatch, decreaseTTL bool) {
	r.batches = append(r.batches, batch)
}

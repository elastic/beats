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

	for i := 0; i < 4; i++ {
		assert.False(t, doneWasCalled, "Original callback shouldn't be invoked until all children are")
		require.Len(t, retryer.batches[i].events, 1, "Retried batches should have one event each")

		// We expect the indices in the retried batches to match because we retried them in order
		assert.Same(t, &events[i], &retryer.batches[i].events[0], "Retried batch events should match original")
		retryer.batches[i].done()
	}
	assert.True(t, doneWasCalled, "Original callback should be invoked when all children are")
}

func TestBatchSplitByDestination(t *testing.T) {
	source1 := &beat.Info{Name: "s1"}
	source2 := &beat.Info{Name: "s2"}

	privateIDs := func(events []publisher.Event) []int {
		ids := make([]int, len(events))
		for i, e := range events {
			ids[i] = e.Content.Private.(int)
		}
		return ids
	}

	t.Run("splits by source and shares completion accounting", func(t *testing.T) {
		retryer := &mockRetryer{}
		// events 0 and 2 belong to source1, event 1 to source2 (interleaved).
		events := []publisher.Event{
			{Content: beat.Event{Private: 0}, Source: source1},
			{Content: beat.Event{Private: 1}, Source: source2},
			{Content: beat.Event{Private: 2}, Source: source1},
		}
		doneWasCalled := false
		root := &ttlBatch{
			events:  events,
			retryer: retryer,
			ttl:     3,
			done:    func() { doneWasCalled = true },
		}

		batches := root.splitByDestination()
		require.Len(t, batches, 2, "should create one batch per distinct source")

		// Groups preserve first-seen source order, with each group's events.
		assert.Equal(t, []int{0, 2}, privateIDs(batches[0].events))
		assert.Equal(t, []int{1}, privateIDs(batches[1].events))
		assert.Same(t, source1, batches[0].events[0].Source)
		assert.Same(t, source2, batches[1].events[0].Source)

		// Children carry over the retryer and ttl and share split metadata.
		assert.Equal(t, 3, batches[0].ttl)
		assert.Equal(t, 3, batches[1].ttl)
		assert.NotNil(t, batches[0].split)
		assert.Same(t, batches[0].split, batches[1].split, "children must share completion accounting")

		// The underlying queue read is acknowledged only after every child
		// completes, so the queue's event cap is held until all destinations are
		// done regardless of how many there are.
		batches[0].done()
		assert.False(t, doneWasCalled, "original done must wait for all children")
		batches[1].done()
		assert.True(t, doneWasCalled, "original done fires once all children complete")
	})

	t.Run("single source is not split", func(t *testing.T) {
		root := &ttlBatch{events: []publisher.Event{{Source: source1}, {Source: source1}}}
		assert.Nil(t, root.splitByDestination(), "a single-source batch should not be split")
	})

	t.Run("untagged events are a single destination", func(t *testing.T) {
		root := &ttlBatch{events: []publisher.Event{{}, {}}}
		assert.Nil(t, root.splitByDestination(), "an all-nil-source batch should not be split")
	})
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

type mockQueueBatch struct {
	freeEntriesCalled int
}

func (b *mockQueueBatch) Count() int {
	return 1
}

func (b *mockQueueBatch) Done() {
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

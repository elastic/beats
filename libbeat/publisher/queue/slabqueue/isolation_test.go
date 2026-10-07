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

package slabqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// A receiver retains its slots until output acknowledges its events, including
// after Get removes them from its FIFO. A stalled output must not consume the
// configured capacity of another receiver whose queue is empty.
func TestCappedQueueDoesNotStarveEmptyReceiver(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	t.Cleanup(pool.Shutdown)
	stalled := pool.Connect()
	stalled.SetTarget(4)
	healthy := pool.Connect()
	healthy.SetTarget(2)
	p1 := stalled.Producer(queue.ProducerConfig{})
	p2 := healthy.Producer(queue.ProducerConfig{})

	for i := range 4 {
		_, ok := p1.TryPublish(i)
		require.True(t, ok, "stalled receiver should admit its configured event budget")
	}
	b, err := stalled.Get(0)
	require.NoError(t, err, "stalled receiver should hand its events to the output")
	t.Cleanup(b.Done)
	require.Equal(t, 4, b.Count(), "all stalled events should remain in flight")

	for i := range 2 {
		_, ok := p2.TryPublish(100 + i)
		assert.True(t, ok, "an empty receiver must retain its full configured capacity while its sibling is stalled")
	}
}

func TestCappedQueueRetainsBacklogCapacityWhenLowered(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	t.Cleanup(pool.Shutdown)
	stalled := pool.Connect()
	stalled.SetTarget(4)
	healthy := pool.Connect()
	healthy.SetTarget(2)
	p1 := stalled.Producer(queue.ProducerConfig{})
	for i := range 4 {
		_, ok := p1.TryPublish(i)
		require.True(t, ok, "events within the original cap should be admitted")
	}
	b, err := stalled.Get(0)
	require.NoError(t, err, "the backlog should become an in-flight output batch")

	stalled.SetTarget(1)
	assert.Equal(t, 6, pool.Target(), "retained backlog must not consume the healthy receiver's capacity")
	p2 := healthy.Producer(queue.ProducerConfig{})
	for i := range 2 {
		_, ok := p2.TryPublish(100 + i)
		assert.True(t, ok, "the healthy receiver should keep its entire cap during a live shrink")
	}
	for i := range 4 {
		assert.Equal(t, i, b.Entry(i), "lowering the cap must preserve every retained event")
	}
	b.Done()
	assert.Equal(t, 3, pool.Target(), "draining the excess backlog should return capacity to the new sum of caps")
}

func TestClosingQueueRetainsInFlightCapacity(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "graceful", true: "forced"}[force], func(t *testing.T) {
			pool := NewPool[int](Settings{Events: 4}, nil)
			t.Cleanup(pool.Shutdown)
			stalled := pool.Connect()
			stalled.SetTarget(4)
			healthy := pool.Connect()
			healthy.SetTarget(2)
			p1 := stalled.Producer(queue.ProducerConfig{})
			for i := range 4 {
				_, ok := p1.TryPublish(i)
				require.True(t, ok, "events within the cap should be admitted")
			}
			b, err := stalled.Get(0)
			require.NoError(t, err, "the output batch should retain the first receiver's slots")
			require.NoError(t, stalled.Close(force), "closing the receiver should succeed while its batch is in flight")
			assert.Equal(t, 1, pool.ConnectedQueues(), "the closing receiver should be disconnected")
			assert.Equal(t, 6, pool.Target(), "retained batch slots should stay in the pool budget until released")
			p2 := healthy.Producer(queue.ProducerConfig{})
			for i := range 2 {
				_, ok := p2.TryPublish(100 + i)
				assert.True(t, ok, "the remaining receiver should publish while its departed sibling retains output slots")
			}
			b.Done()
			assert.Equal(t, 2, pool.Target(), "the closed receiver should return its budget when its batch finishes")
			assert.Empty(t, pool.draining, "fully drained receivers should not remain referenced by the pool")
		})
	}
}

func TestUncappedQueuesKeepSharedBudget(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	t.Cleanup(pool.Shutdown)
	p1 := pool.Connect().Producer(queue.ProducerConfig{})
	p2 := pool.Connect().Producer(queue.ProducerConfig{})
	for i := range 4 {
		_, ok := p1.TryPublish(i)
		require.True(t, ok, "an uncapped queue may use the entire initial pool")
	}
	_, ok := p2.TryPublish(100)
	assert.False(t, ok, "uncapped queues share the initial budget without reserving capacity")
	assert.Equal(t, 4, pool.Target(), "connecting uncapped queues should keep the initial target")
}

func TestAggregateCapacityOverflow(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	assert.Equal(t, maxInt, addQueueCapacity(maxInt-2, 2), "a representable sum should remain exact")
	assert.PanicsWithValue(t, "slabqueue: combined queue capacities exceed int range", func() {
		addQueueCapacity(maxInt-2, 3)
	}, "an unrepresentable aggregate must not wrap into a small pool budget")
	assert.Equal(t, ((maxInt-1)>>slabChunkShift)+1, numChunks(maxInt), "chunk rounding must not overflow at the integer limit")
}

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestCappedQueuesResizeAndShutdownConcurrently(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	t.Cleanup(pool.Shutdown)
	const receivers = 4
	queues := make([]*Queue[int], receivers)
	var workers sync.WaitGroup
	var published, consumed atomic.Int64
	for i := range receivers {
		q := pool.Connect()
		q.SetTarget(4)
		queues[i] = q
		p := q.Producer(queue.ProducerConfig{})
		workers.Go(func() {
			for event := 0; ; event++ {
				if _, ok := p.Publish(event); !ok {
					return
				}
				published.Add(1)
			}
		})
		workers.Go(func() {
			for {
				b, err := q.Get(2)
				if err != nil {
					return
				}
				consumed.Add(int64(b.Count()))
				b.Done()
			}
		})
	}
	require.Eventually(t, func() bool { return published.Load() >= receivers*4 }, time.Second, time.Millisecond,
		"all receivers should have live traffic before concurrent resize and shutdown")
	for _, q := range queues {
		workers.Go(func() {
			for i := range 100 {
				q.SetTarget(1 + i%8)
			}
			_ = q.Close(false)
		})
	}
	workers.Go(pool.Shutdown)
	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("capped receiver traffic, resizing, and shutdown must complete without deadlocking")
	}
	assert.LessOrEqual(t, consumed.Load(), published.Load(), "shutdown must never deliver unpublished events")
	assert.Equal(t, 0, pool.ConnectedQueues(), "shutdown should disconnect every receiver")
}

func TestReservationRechecksConcurrentCapReduction(t *testing.T) {
	for _, oldCap := range []int{0, 4} {
		t.Run(map[int]string{0: "uncapped", 4: "capped"}[oldCap], func(t *testing.T) {
			pool := NewPool[int](Settings{Events: 4}, nil)
			t.Cleanup(pool.Shutdown)
			q := pool.Connect()
			q.SetTarget(oldCap)
			healthy := pool.Connect()
			healthy.SetTarget(2)
			p := q.Producer(queue.ProducerConfig{})
			_, ok := p.TryPublish(1)
			require.True(t, ok, "the first event should be admitted before the cap changes")

			// Reproduce a producer that already read oldCap: SetTarget scans
			// the old live count, then that producer increments it. Neither
			// the CAS nor the uncapped Add path may acquire a pool slot now.
			q.SetTarget(1)
			reserved := q.live.Add(1)
			assert.False(t, q.validateReservation(reserved), "a stale admission above the new cap must roll back")
			assert.Equal(t, int64(1), q.live.Load(), "rollback should restore the original live count")
			assert.Equal(t, 3, pool.Target(), "the rolled-back reservation must not leave an inflated budget")
			p2 := healthy.Producer(queue.ProducerConfig{})
			for i := range 2 {
				_, ok := p2.TryPublish(100 + i)
				assert.True(t, ok, "the sibling should retain every configured slot after the stale reservation")
			}
		})
	}
}

func TestDrainingQueueReferencesAreReleased(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "last capped queue drains", true: "pool shuts down"}[shutdown], func(t *testing.T) {
			pool := NewPool[int](Settings{Events: 2}, nil)
			t.Cleanup(pool.Shutdown)
			q := pool.Connect()
			q.SetTarget(2)
			p := q.Producer(queue.ProducerConfig{})
			_, ok := p.TryPublish(1)
			require.True(t, ok, "the receiver should admit its event")
			b, err := q.Get(0)
			require.NoError(t, err, "the event should remain in an output batch")
			require.NoError(t, q.Close(false), "the last capped queue should begin draining")
			require.Len(t, pool.draining, 1, "the pool should retain the draining contribution while output holds its event")
			if shutdown {
				pool.Shutdown()
				assert.Empty(t, pool.draining, "shutdown should release capacity-accounting references even while output retains a batch")
			}
			assert.Equal(t, 1, b.Entry(0), "removing accounting references must preserve the output-owned event")
			b.Done()
			assert.Empty(t, pool.draining, "releasing the last batch should remove the draining reference even with no caps left")
		})
	}
}

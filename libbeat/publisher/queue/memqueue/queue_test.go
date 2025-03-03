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

package memqueue

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
)

var seed int64

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 1024
	minEvents := 32

	randGen := rand.New(rand.NewSource(seed))
	events := randGen.Intn(maxEvents-minEvents) + minEvents
	batchSize := randGen.Intn(events-8) + 4
	bufferSize := randGen.Intn(batchSize*2) + 4

	// events := 4
	// batchSize := 1
	// bufferSize := 2

	t.Log("seed: ", seed)
	t.Log("events: ", events)
	t.Log("batchSize: ", batchSize)
	t.Log("bufferSize: ", bufferSize)

	testWith := func(factory queuetest.QueueFactory) func(t *testing.T) {
		return func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				t.Parallel()
				queuetest.TestSingleProducerConsumer(t, events, batchSize, factory)
			})
			t.Run("multi", func(t *testing.T) {
				t.Parallel()
				queuetest.TestMultiProducerConsumer(t, events, batchSize, factory)
			})
		}
	}

	t.Run("direct", testWith(makeTestQueue(bufferSize, 0, 0)))
	t.Run("flush", testWith(makeTestQueue(bufferSize, batchSize/2, 100*time.Millisecond)))
}

// TestProducerDoesNotBlockWhenQueueClosed ensures the producer Publish
// does not block indefinitely during queue shutdown.
//
// Once we get a producer `p` from the queue `q` we want to ensure
// that if p.Publish is called and blocks it will unblock once
// `q.Close` is called.
//
// For this test we start a queue with size 2 and try to add more
// than 2 events to it, p.Publish will block, once we call q.Close,
// we ensure the 3rd event was not successfully published.
func TestProducerDoesNotBlockWhenQueueClosed(t *testing.T) {
	q := NewQueue(nil, nil,
		Settings{
			Events:        2, // Queue size
			MaxGetRequest: 1, // make sure the queue won't buffer events
			FlushTimeout:  time.Millisecond,
		}, 0, nil)

	p := q.Producer(queue.ProducerConfig{
		// We do not read from the queue, so the callbacks are never called
		ACK: func(count int) {},
	})

	success := atomic.Bool{}
	publishCount := atomic.Int32{}
	go func() {
		// Publish 2 events, this will make the queue full, but
		// both will be accepted
		for i := 0; i < 2; i++ {
			id, ok := p.Publish(fmt.Sprintf("Event %d", i))
			if !ok {
				t.Errorf("failed to publish to the queue, event ID: %v", id)
				return
			}
			publishCount.Add(1)
		}
		_, ok := p.Publish("Event 3")
		if ok {
			t.Errorf("publishing the 3rd event must fail")
			return
		}

		// Flag the test as successful
		success.Store(true)
	}()

	// Allow the producer to run and the queue to do its thing.
	// Two events should be accepted and the third call to p.Publish
	// must block
	// time.Sleep(100 * time.Millisecond)

	// Ensure we published two events
	require.Eventually(
		t,
		func() bool { return publishCount.Load() == 2 },
		200*time.Millisecond,
		time.Millisecond,
		"the first two events were not successfully published")

	// Close the queue, this should unblock the pending Publish call.
	// It's not enough to just cancel the producer: once the producer
	// has successfully sent a request to the queue, it must wait for
	// the response unless the queue shuts down, otherwise the pipeline
	// event totals will be wrong.
	q.Close()

	require.Eventually(
		t,
		success.Load,
		200*time.Millisecond,
		1*time.Millisecond,
		"test not flagged as successful, p.Publish likely blocked indefinitely")
}

func TestProducerClosePreservesEventCount(t *testing.T) {
	// Check for https://github.com/elastic/beats/issues/37702, a problem
	// where canceling a producer while it was waiting on a response
	// to an insert request could lead to inaccurate event totals.

	var activeEvents atomic.Int64

	q := NewQueue(nil, nil,
		Settings{
			Events:        3, // Queue size
			MaxGetRequest: 2,
			FlushTimeout:  10 * time.Millisecond,
		}, 1, nil)

	p := q.Producer(queue.ProducerConfig{
		ACK: func(count int) {
			activeEvents.Add(-int64(count))
		},
	})

	// Asynchronously, send 4 events to the queue.
	// Three will be enqueued, and one will be buffered,
	// until we start reading from the queue.
	// This needs to run in a goroutine because the buffered
	// event will block until the queue handles it.
	var wgProducer sync.WaitGroup
	wgProducer.Add(1)
	go func() {
		for i := 0; i < 4; i++ {
			event := i
			// For proper navigation of the race conditions inherent to this
			// test: increment active events before the publish attempt, then
			// decrement afterwards if it failed (otherwise the event count
			// could become negative even under correct queue operation).
			activeEvents.Add(1)
			_, ok := p.Publish(event)
			if !ok {
				activeEvents.Add(-1)
			}
		}
		wgProducer.Done()
	}()

	// This sleep is regrettable, but there's no deterministic way to know when
	// the producer code has buffered an event in the queue's channel.
	// However, the test is written to produce false negatives only:
	// - If this test fails, it _always_ indicates a bug.
	// - If there is a bug, this test will _often_ fail.
	time.Sleep(20 * time.Millisecond)

	// Cancel the producer, then read and acknowledge two batches. If the
	// Publish calls and the queue code are working, activeEvents should
	// _usually_ end up as 0, but _always_ end up non-negative.
	p.Close()

	// The queue reads also need to be done in a goroutine, in case the
	// producer cancellation signal went through before the Publish
	// requests -- if only 2 events entered the queue, then the second
	// Get call will block until the queue itself is cancelled.
	go func() {
		for i := 0; i < 2; i++ {
			batch, err := q.Get(2)
			// Only error to worry about is queue closing, which isn't
			// a test failure.
			if err == nil {
				batch.Done()
			}
		}
	}()

	// One last sleep to let things percolate, then we close the queue
	// to unblock any helpers and verify that the final active event
	// count isn't negative.
	time.Sleep(10 * time.Millisecond)
	q.Close()
	assert.False(t, activeEvents.Load() < 0, "active event count should never be negative")
}

func makeTestQueue(sz, minEvents int, flushTimeout time.Duration) queuetest.QueueFactory {
	return func(_ *testing.T) queue.Queue {
		return NewQueue(nil, nil, Settings{
			Events:        sz,
			MaxGetRequest: minEvents,
			FlushTimeout:  flushTimeout,
		}, 0, nil)
	}
}

func TestAdjustInputQueueSize(t *testing.T) {
	t.Run("zero yields default value (main queue size=0)", func(t *testing.T) {
		assert.Equal(t, minInputQueueSize, AdjustInputQueueSize(0, 0))
	})
	t.Run("zero yields default value (main queue size=10)", func(t *testing.T) {
		assert.Equal(t, minInputQueueSize, AdjustInputQueueSize(0, 10))
	})
	t.Run("can't go below min", func(t *testing.T) {
		assert.Equal(t, minInputQueueSize, AdjustInputQueueSize(1, 0))
	})
	t.Run("can set any value within bounds", func(t *testing.T) {
		for q, mainQueue := minInputQueueSize+1, 4096; q < int(float64(mainQueue)*maxInputQueueSizeRatio); q += 10 {
			assert.Equal(t, q, AdjustInputQueueSize(q, mainQueue))
		}
	})
	t.Run("can set any value if no upper bound", func(t *testing.T) {
		for q := minInputQueueSize + 1; q < math.MaxInt32; q *= 2 {
			assert.Equal(t, q, AdjustInputQueueSize(q, 0))
		}
	})
	t.Run("can't go above upper bound", func(t *testing.T) {
		mainQueue := 4096
		assert.Equal(t, int(float64(mainQueue)*maxInputQueueSizeRatio), AdjustInputQueueSize(mainQueue, mainQueue))
	})
}

func TestBatchFreeEntries(t *testing.T) {
	const queueSize = 10
	const batchSize = 5
	// 1. Add 10 events to the queue, request two batches with 5 events each
	// 2. Make sure the queue buffer has 10 non-nil events
	// 3. Call FreeEntries on the second batch
	// 4. Make sure only events 6-10 are nil
	// 5. Call FreeEntries on the first batch
	// 6. Make sure all events are nil
	testQueue := NewQueue(nil, nil, Settings{Events: queueSize, MaxGetRequest: batchSize, FlushTimeout: time.Second}, 0, nil)
	producer := testQueue.Producer(queue.ProducerConfig{})
	for i := 0; i < queueSize; i++ {
		_, ok := producer.Publish(i)
		require.True(t, ok, "Queue publish must succeed")
	}
	batch1, err := testQueue.Get(batchSize)
	require.NoError(t, err, "Queue read must succeed")
	require.Equal(t, batchSize, batch1.Count(), "Returned batch size must match request")
	batch2, err := testQueue.Get(batchSize)
	require.NoError(t, err, "Queue read must succeed")
	require.Equal(t, batchSize, batch2.Count(), "Returned batch size must match request")
	// Slight concurrency subtlety: we check events are non-nil after the queue
	// reads, since if we do it before we have no way to be sure the insert
	// has been completed.
	for i := 0; i < queueSize; i++ {
		require.NotNil(t, testQueue.buf[i].event, "All queue events must be non-nil")
	}
	batch2.FreeEntries()
	for i := 0; i < batchSize; i++ {
		require.NotNilf(t, testQueue.buf[i].event, "Queue index %v: batch 1's events should be unaffected by calling FreeEntries on Batch 2", i)
		require.Nilf(t, testQueue.buf[batchSize+i].event, "Queue index %v: batch 2's events should be nil after FreeEntries", batchSize+i)
	}
	batch1.FreeEntries()
	for i := 0; i < queueSize; i++ {
		require.Nilf(t, testQueue.buf[i].event, "Queue index %v: all events should be nil after calling FreeEntries on both batches")
	}
}

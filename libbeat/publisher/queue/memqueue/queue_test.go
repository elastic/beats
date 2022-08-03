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
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var seed int64

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 1024
	minEvents := 32

	rand.Seed(seed)
	events := rand.Intn(maxEvents-minEvents) + minEvents
	batchSize := rand.Intn(events-8) + 4
	bufferSize := rand.Intn(batchSize*2) + 4

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

func TestQueueMetricsDirect(t *testing.T) {
	eventsToTest := 5
	maxEvents := 10

	// Test the directEventLoop
	directSettings := Settings{
		Events:         maxEvents,
		FlushMinEvents: 1,
		FlushTimeout:   0,
	}
	t.Logf("Testing directEventLoop")
	queueTestWithSettings(t, directSettings, eventsToTest, "directEventLoop")

}

func TestQueueMetricsBuffer(t *testing.T) {
	eventsToTest := 5
	maxEvents := 10
	// Test Buffered Event Loop
	bufferedSettings := Settings{
		Events:         maxEvents,
		FlushMinEvents: eventsToTest, // The buffered event loop can only return FlushMinEvents per Get()
		FlushTimeout:   time.Millisecond,
	}
	t.Logf("Testing bufferedEventLoop")
	queueTestWithSettings(t, bufferedSettings, eventsToTest, "bufferedEventLoop")
}

func queueTestWithSettings(t *testing.T, settings Settings, eventsToTest int, testName string) {
	testQueue := NewQueue(nil, settings)
	defer testQueue.Close()

	// Send events to queue
	producer := testQueue.Producer(queue.ProducerConfig{})
	for i := 0; i < eventsToTest; i++ {
		producer.Publish(queuetest.MakeEvent(mapstr.M{"count": i}))
	}
	queueMetricsAreValid(t, testQueue, 5, settings.Events, 0, fmt.Sprintf("%s - First send of metrics to queue", testName))

	// Read events, don't yet ack them
	batch, err := testQueue.Get(eventsToTest)
	assert.NilError(t, err, "error in Get")
	t.Logf("Got batch of %d events", batch.Count())

	queueMetricsAreValid(t, testQueue, 5, settings.Events, 5, fmt.Sprintf("%s - Producer Getting events, no ACK", testName))

	// Test metrics after ack
	batch.Done()

	queueMetricsAreValid(t, testQueue, 0, settings.Events, 0, fmt.Sprintf("%s - Producer Getting events, no ACK", testName))

}

func queueMetricsAreValid(t *testing.T, q queue.Queue, evtCount, evtLimit, occupied int, test string) {
	// wait briefly to avoid races across all the queue channels
	time.Sleep(time.Millisecond * 100)
	testMetrics, err := q.Metrics()
	assert.NilError(t, err, "error calling metrics for test %s", test)
	assert.Equal(t, testMetrics.EventCount.ValueOr(0), uint64(evtCount), "incorrect EventCount for %s", test)
	assert.Equal(t, testMetrics.EventLimit.ValueOr(0), uint64(evtLimit), "incorrect EventLimit for %s", test)
	assert.Equal(t, testMetrics.UnackedConsumedEvents.ValueOr(0), uint64(occupied), "incorrect OccupiedRead for %s", test)
}

func TestProducerCancelRemovesEvents(t *testing.T) {
	queuetest.TestProducerCancelRemovesEvents(t, makeTestQueue(1024, 0, 0))
}

func makeTestQueue(sz, minEvents int, flushTimeout time.Duration) queuetest.QueueFactory {
	return func(_ *testing.T) queue.Queue {
		return NewQueue(nil, Settings{
			Events:         sz,
			FlushMinEvents: minEvents,
			FlushTimeout:   flushTimeout,
		})
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

func TestEntryIDs(t *testing.T) {
	entryCount := 100

	testForward := func(q queue.Queue) {
		producer := q.Producer(queue.ProducerConfig{})
		for i := 0; i < entryCount; i++ {
			id, success := producer.Publish(nil)
			assert.Equal(t, success, true, "Queue publish should succeed")
			assert.Equal(t, id, queue.EntryID(i), "Entry ID should match publication order")
		}

		for i := 0; i < entryCount; i++ {
			batch, err := q.Get(1)
			assert.NilError(t, err, "Queue read should succeed")
			assert.Equal(t, batch.Count(), 1, "Returned batch should have 1 entry")
			assert.Equal(t, batch.ID(0), queue.EntryID(i), "Consumed entry IDs should be ordered the same as when they were produced")

			time.Sleep(100 * time.Millisecond)
			metrics, err := q.Metrics()
			assert.NilError(t, err, "Queue metrics call should succeed")
			assert.Equal(t, metrics.OldestEntryID, queue.EntryID(i),
				fmt.Sprintf("Oldest entry ID before ACKing event %v should be %v", i, i))

			batch.Done()
			// Hard to remove this delay since the Done signal is propagated
			// asynchronously to the queue...
			time.Sleep(1 * time.Millisecond)
			metrics, err = q.Metrics()
			assert.NilError(t, err, "Queue metrics call should succeed")
			assert.Equal(t, metrics.OldestEntryID, queue.EntryID(i+1),
				fmt.Sprintf("Oldest entry ID after ACKing event %v should be %v", i, i+1))

		}
	}

	testBackward := func(q queue.Queue) {
		producer := q.Producer(queue.ProducerConfig{})
		for i := 0; i < entryCount; i++ {
			id, success := producer.Publish(nil)
			assert.Equal(t, success, true, "Queue publish should succeed")
			assert.Equal(t, id, queue.EntryID(i), "Entry ID should match publication order")
		}

		batches := []queue.Batch{}

		for i := 0; i < entryCount; i++ {
			batch, err := q.Get(1)
			assert.NilError(t, err, "Queue read should succeed")
			assert.Equal(t, batch.Count(), 1, "Returned batch should have 1 entry")
			assert.Equal(t, batch.ID(0), queue.EntryID(i), "Consumed entry IDs should be ordered the same as when they were produced")
			batches = append(batches, batch)
		}

		for i := entryCount - 1; i > 0; i-- {
			batches[i].Done()

			// Hard to remove this delay since the Done signal is propagated
			// asynchronously to the queue...
			time.Sleep(10 * time.Millisecond)
			metrics, err := q.Metrics()
			assert.NilError(t, err, "Queue metrics call should succeed")
			assert.Equal(t, metrics.OldestEntryID, queue.EntryID(0),
				fmt.Sprintf("Oldest entry ID after ACKing event %v should be 0", i))
		}
		// ACK the first batch, which should unblock all the later ones
		batches[0].Done()
		time.Sleep(1 * time.Millisecond)
		metrics, err := q.Metrics()
		assert.NilError(t, err, "Queue metrics call should succeed")
		assert.Equal(t, metrics.OldestEntryID, queue.EntryID(100),
			fmt.Sprintf("Oldest entry ID after ACKing event 0 should be %v", queue.EntryID(entryCount)))

	}

	t.Run("acking in forward order with directEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, Settings{Events: 1000})
		testForward(testQueue)
	})

	t.Run("acking in reverse order with directEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, Settings{Events: 1000})
		testBackward(testQueue)
	})

	t.Run("acking in forward order with bufferedEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, Settings{Events: 1000, FlushMinEvents: 2, FlushTimeout: time.Microsecond})
		testForward(testQueue)
	})

	t.Run("acking in reverse order with bufferedEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, Settings{Events: 1000, FlushMinEvents: 2, FlushTimeout: time.Microsecond})
		testBackward(testQueue)
	})
}

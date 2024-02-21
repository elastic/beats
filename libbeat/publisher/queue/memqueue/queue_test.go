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

	"gotest.tools/assert"

	"github.com/stretchr/testify/require"

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
		}, 0)

	p := q.Producer(queue.ProducerConfig{
		// We do not read from the queue, so the callbacks are never called
		ACK:          func(count int) {},
		OnDrop:       func(e interface{}) {},
		DropOnCancel: false,
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

	// Close the queue, this should unblock the pending Publish call
	q.Close()

	require.Eventually(
		t,
		success.Load,
		200*time.Millisecond,
		1*time.Millisecond,
		"test not flagged as successful, p.Publish likely blocked indefinitely")
}

func TestQueueMetricsDirect(t *testing.T) {
	eventsToTest := 5
	maxEvents := 10

	// Test the directEventLoop
	directSettings := Settings{
		Events:        maxEvents,
		MaxGetRequest: 1,
		FlushTimeout:  0,
	}
	t.Logf("Testing directEventLoop")
	queueTestWithSettings(t, directSettings, eventsToTest, "directEventLoop")

}

func TestQueueMetricsBuffer(t *testing.T) {
	eventsToTest := 5
	maxEvents := 10
	// Test Buffered Event Loop
	bufferedSettings := Settings{
		Events:        maxEvents,
		MaxGetRequest: eventsToTest, // The buffered event loop can only return FlushMinEvents per Get()
		FlushTimeout:  time.Millisecond,
	}
	t.Logf("Testing bufferedEventLoop")
	queueTestWithSettings(t, bufferedSettings, eventsToTest, "bufferedEventLoop")
}

func queueTestWithSettings(t *testing.T, settings Settings, eventsToTest int, testName string) {
	testQueue := NewQueue(nil, nil, settings, 0)
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
		return NewQueue(nil, nil, Settings{
			Events:        sz,
			MaxGetRequest: minEvents,
			FlushTimeout:  flushTimeout,
		}, 0)
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
		waiter := &producerACKWaiter{}
		producer := q.Producer(queue.ProducerConfig{ACK: waiter.ack})
		for i := 0; i < entryCount; i++ {
			id, success := producer.Publish(nil)
			assert.Equal(t, success, true, "Queue publish should succeed")
			assert.Equal(t, id, queue.EntryID(i), "Entry ID should match publication order")
		}

		for i := 0; i < entryCount; i++ {
			batch, err := q.Get(1)
			assert.NilError(t, err, "Queue read should succeed")
			assert.Equal(t, batch.Count(), 1, "Returned batch should have 1 entry")

			metrics, err := q.Metrics()
			assert.NilError(t, err, "Queue metrics call should succeed")
			assert.Equal(t, metrics.OldestEntryID, queue.EntryID(i),
				fmt.Sprintf("Oldest entry ID before ACKing event %v should be %v", i, i))

			batch.Done()
			waiter.waitForEvents(1)
			metrics, err = q.Metrics()
			assert.NilError(t, err, "Queue metrics call should succeed")
			assert.Equal(t, metrics.OldestEntryID, queue.EntryID(i+1),
				fmt.Sprintf("Oldest entry ID after ACKing event %v should be %v", i, i+1))

		}
	}

	testBackward := func(q queue.Queue) {
		waiter := &producerACKWaiter{}
		producer := q.Producer(queue.ProducerConfig{ACK: waiter.ack})
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
			batches = append(batches, batch)
		}

		for i := entryCount - 1; i > 0; i-- {
			batches[i].Done()

			// It's hard to remove this delay since the Done signal is propagated
			// asynchronously to the queue, and since this test is ensuring that the
			// queue _doesn't_ advance we can't use a callback to gate the comparison
			// like we do in testForward. However:
			// - While this race condition could sometimes let a buggy implementation
			//   pass, it will not produce a false failure (so it won't contribute
			//   to general test flakiness)
			// - That notwithstanding, when the ACK _does_ cause an incorrect
			//   metrics update, this delay is enough to recognize it approximately
			//   100% of the time, so this test is still a good signal despite
			//   the slight nondeterminism.
			time.Sleep(1 * time.Millisecond)
			metrics, err := q.Metrics()
			assert.NilError(t, err, "Queue metrics call should succeed")
			assert.Equal(t, metrics.OldestEntryID, queue.EntryID(0),
				fmt.Sprintf("Oldest entry ID after ACKing event %v should be 0", i))
		}
		// ACK the first batch, which should unblock all the later ones
		batches[0].Done()
		waiter.waitForEvents(100)
		metrics, err := q.Metrics()
		assert.NilError(t, err, "Queue metrics call should succeed")
		assert.Equal(t, metrics.OldestEntryID, queue.EntryID(100),
			fmt.Sprintf("Oldest entry ID after ACKing event 0 should be %v", queue.EntryID(entryCount)))

	}

	t.Run("acking in forward order with directEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, nil, Settings{Events: 1000}, 0)
		testForward(testQueue)
	})

	t.Run("acking in reverse order with directEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, nil, Settings{Events: 1000}, 0)
		testBackward(testQueue)
	})

	t.Run("acking in forward order with bufferedEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, nil, Settings{Events: 1000, MaxGetRequest: 2, FlushTimeout: time.Microsecond}, 0)
		testForward(testQueue)
	})

	t.Run("acking in reverse order with bufferedEventLoop reports the right event IDs", func(t *testing.T) {
		testQueue := NewQueue(nil, nil, Settings{Events: 1000, MaxGetRequest: 2, FlushTimeout: time.Microsecond}, 0)
		testBackward(testQueue)
	})
}

// producerACKWaiter is a helper that can listen to queue producer callbacks
// and wait on them from the test thread, so we can test the queue's asynchronous
// behavior without relying on time.Sleep.
type producerACKWaiter struct {
	sync.Mutex

	// The number of acks received from a producer callback.
	acked int

	// The number of acks that callers have waited for in waitForEvents.
	waited int

	// When non-nil, this channel is being listened to by a test thread
	// blocking on ACKs, and incoming producer callbacks are forwarded
	// to it.
	ackChan chan int
}

func (w *producerACKWaiter) ack(count int) {
	w.Lock()
	defer w.Unlock()
	w.acked += count
	if w.ackChan != nil {
		w.ackChan <- count
	}
}

func (w *producerACKWaiter) waitForEvents(count int) {
	w.Lock()
	defer w.Unlock()
	if w.ackChan != nil {
		panic("don't call producerACKWaiter.waitForEvents from multiple goroutines")
	}

	avail := w.acked - w.waited
	if count <= avail {
		w.waited += count
		return
	}
	w.waited = w.acked
	count -= avail
	// We have advanced as far as we can, we have to wait for
	// more incoming ACKs.
	// Set a listener and unlock, so ACKs can come in on another
	// goroutine.
	w.ackChan = make(chan int)
	w.Unlock()

	newAcked := 0
	for newAcked < count {
		newAcked += <-w.ackChan
	}
	// When we're done, turn off the listener channel and update
	// the number of events waited on.
	w.Lock()
	w.ackChan = nil
	w.waited += count
}

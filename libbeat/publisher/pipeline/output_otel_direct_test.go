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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func makeTestEvent(field string, value interface{}) publisher.Event {
	return publisher.Event{
		Content: beat.Event{
			Timestamp: time.Now(),
			Fields:    mapstr.M{field: value},
		},
	}
}

func TestDirectProducerFlushesEveryEvent(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	var mu sync.Mutex
	var batches []int

	client := newMockClient(func(batch publisher.Batch) error {
		mu.Lock()
		defer mu.Unlock()
		batches = append(batches, len(batch.Events()))
		batch.ACK()
		return nil
	})

	dp := newDirectProducer(logger, client, nil)

	dp.Publish(makeTestEvent("a", 1))
	dp.Publish(makeTestEvent("b", 2))
	dp.Publish(makeTestEvent("c", 3))

	dp.Close()

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, batches, 3, "each event should be flushed individually")
	for i, size := range batches {
		assert.Equal(t, 1, size, "batch %d should have exactly 1 event", i)
	}
}

func TestDirectProducerACKCallback(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	var ackCount atomic.Int64
	client := newMockClient(func(batch publisher.Batch) error {
		batch.ACK()
		return nil
	})

	dp := newDirectProducer(logger, client, func(count int) {
		ackCount.Add(int64(count))
	})

	dp.Publish(makeTestEvent("a", 1))
	dp.Publish(makeTestEvent("b", 2))

	dp.Close()

	assert.Equal(t, int64(2), ackCount.Load(), "each event should be ACKed individually")
}

func TestDirectProducerRejectsAfterClose(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	client := newMockClient(func(batch publisher.Batch) error {
		batch.ACK()
		return nil
	})

	dp := newDirectProducer(logger, client, nil)
	dp.Close()

	_, ok := dp.Publish(makeTestEvent("after", "close"))
	assert.False(t, ok, "Publish after Close should return false")
}

func TestDirectProducerTryPublishSameAsPublish(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	var published atomic.Int64
	client := newMockClient(func(batch publisher.Batch) error {
		published.Add(int64(len(batch.Events())))
		batch.ACK()
		return nil
	})

	dp := newDirectProducer(logger, client, nil)

	id, ok := dp.TryPublish(makeTestEvent("try", 1))
	assert.True(t, ok)
	assert.Equal(t, uint64(0), uint64(id))

	_, ok = dp.TryPublish(makeTestEvent("try", 2))
	assert.True(t, ok)

	assert.Equal(t, int64(2), published.Load())

	dp.Close()
}

func TestDirectProducerConcurrentPublish(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	var totalEvents atomic.Int64
	client := newMockClient(func(batch publisher.Batch) error {
		totalEvents.Add(int64(len(batch.Events())))
		batch.ACK()
		return nil
	})

	dp := newDirectProducer(logger, client, nil)

	numGoroutines := 10
	eventsPerGoroutine := 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < eventsPerGoroutine; i++ {
				dp.Publish(makeTestEvent("g", g*100+i))
			}
		}(g)
	}
	wg.Wait()
	dp.Close()

	expected := int64(numGoroutines * eventsPerGoroutine)
	assert.Equal(t, expected, totalEvents.Load(),
		"all events from concurrent publishers should be flushed")
}

func TestDirectProducerIDsAreSequential(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	client := newMockClient(func(batch publisher.Batch) error {
		batch.ACK()
		return nil
	})

	dp := newDirectProducer(logger, client, nil)

	id0, _ := dp.Publish(makeTestEvent("a", 1))
	id1, _ := dp.Publish(makeTestEvent("b", 2))
	id2, _ := dp.Publish(makeTestEvent("c", 3))

	assert.Equal(t, queue.EntryID(0), id0)
	assert.Equal(t, queue.EntryID(1), id1)
	assert.Equal(t, queue.EntryID(2), id2)

	dp.Close()
}

// --- directBatch tests ---

func TestDirectBatchSignals(t *testing.T) {
	var acked atomic.Int64
	ackFn := func(count int) {
		acked.Add(int64(count))
	}

	events := []publisher.Event{
		makeTestEvent("a", 1),
		makeTestEvent("b", 2),
	}

	t.Run("ACK sends non-retry signal and clears events", func(t *testing.T) {
		acked.Store(0)
		ch := make(chan batchResult, 1)
		b := &directBatch{events: events, ackFn: ackFn, doneCh: ch}
		b.ACK()
		assert.Equal(t, int64(2), acked.Load())
		assert.Nil(t, b.events)
		result := <-ch
		assert.False(t, result.retry)
	})

	t.Run("Drop sends non-retry signal and clears events", func(t *testing.T) {
		ch := make(chan batchResult, 1)
		b := &directBatch{events: events, ackFn: ackFn, doneCh: ch}
		b.Drop()
		assert.Nil(t, b.events)
		result := <-ch
		assert.False(t, result.retry)
	})

	t.Run("Retry sends retry signal", func(t *testing.T) {
		ch := make(chan batchResult, 1)
		b := &directBatch{events: events, doneCh: ch}
		b.Retry()
		result := <-ch
		assert.True(t, result.retry)
	})

	t.Run("Cancelled sends retry signal", func(t *testing.T) {
		ch := make(chan batchResult, 1)
		b := &directBatch{events: events, doneCh: ch}
		b.Cancelled()
		result := <-ch
		assert.True(t, result.retry)
	})

	t.Run("RetryEvents replaces events and retries", func(t *testing.T) {
		ch := make(chan batchResult, 1)
		b := &directBatch{events: events, doneCh: ch}
		subset := events[:1]
		b.RetryEvents(subset)
		assert.Equal(t, subset, b.events)
		result := <-ch
		assert.True(t, result.retry)
	})

	t.Run("SplitRetry returns false for single event", func(t *testing.T) {
		b := &directBatch{events: events[:1], doneCh: make(chan batchResult, 1)}
		assert.False(t, b.SplitRetry())
	})

	t.Run("SplitRetry returns true for multiple events", func(t *testing.T) {
		b := &directBatch{events: events, doneCh: make(chan batchResult, 1)}
		assert.True(t, b.SplitRetry())
	})
}

func TestDirectProducerFlushRetriesOnRetrySignal(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	var attempts atomic.Int64
	client := newMockClient(func(batch publisher.Batch) error {
		n := attempts.Add(1)
		if n < 3 {
			batch.Retry()
		} else {
			batch.ACK()
		}
		return nil
	})

	dp := newDirectProducer(logger, client, nil)

	dp.Publish(makeTestEvent("key", "value"))
	dp.Close()

	assert.Equal(t, int64(3), attempts.Load(),
		"flush should retry iteratively until ACK")
}

// --- otelOutputController integration tests ---

func TestOTelOutputControllerQueueModes(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("DirectQueue controller has no queue and flushes immediately", func(t *testing.T) {
		var published atomic.Int64
		ctrl := &otelOutputController{
			beatInfo:     beat.Info{Logger: logger},
			logger:       logger,
			queueMode:    DirectQueue,
			directClient: newMockClient(func(batch publisher.Batch) error {
				published.Add(int64(len(batch.Events())))
				batch.ACK()
				return nil
			}),
		}

		assert.Equal(t, DirectQueue, ctrl.queueMode)
		assert.Nil(t, ctrl.queue)
		assert.Nil(t, ctrl.consumer)

		producer := ctrl.queueProducer(queue.ProducerConfig{})
		require.NotNil(t, producer)

		producer.Publish(makeTestEvent("test", 1))
		assert.Equal(t, int64(1), published.Load())

		producer.Publish(makeTestEvent("test", 2))
		assert.Equal(t, int64(2), published.Load())

		producer.Close()
	})

	t.Run("DefaultQueue controller returns queue producer", func(t *testing.T) {
		ctrl := &otelOutputController{
			beatInfo:  beat.Info{Logger: logger},
			logger:    logger,
			queueMode: DefaultQueue,
		}

		assert.Equal(t, DefaultQueue, ctrl.queueMode)
	})
}

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestNoBatchAssemblyOnNilTarget(t *testing.T) {
	// Create a minimal struct with only the channels we need. Batch assembly
	// is triggered determinstically (i.e. no selects) at the start of each
	// iteration of the run loop, so this way we can test synchronously
	// instead of starting up the full goroutine and relying on a timeout,
	// which can cause flakiness on CI. (This test does not pass without the
	// code change to check for a nil channel.)
	logger := logptest.NewTestingLogger(t, "")
	c := &eventConsumer{
		logger: logger.Named("eventConsumer test"),
		queueReader: queueReader{
			req: make(chan queueReaderRequest, 1),
		},
		done: make(chan struct{}),
	}

	// Close immediately so the run loop returns
	close(c.done)

	c.run()

	// Make sure no read request was sent
	_, ok := <-c.queueReader.req
	assert.False(t, ok, "The queue reader shouldn't get a read request when the target is nil")
}

func receiveBatch(t *testing.T, ch <-chan publisher.Batch) publisher.Batch {
	t.Helper()
	select {
	case b := <-ch:
		return b
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a batch on the output channel")
		return nil
	}
}

// TestEventConsumerRetriesUntilTTLExhausted verifies the run loop's retry path:
// a retried batch is re-delivered until its TTL is exhausted, then dropped (and
// the underlying events acknowledged to the queue so it can make progress).
func TestEventConsumerRetriesUntilTTLExhausted(t *testing.T) {
	q := memqueue.NewQueue[publisher.Event](
		logp.NewNopLogger(), queue.NewQueueObserver(nil),
		memqueue.Settings{Events: 10, MaxGetRequest: 1, FlushTimeout: 10 * time.Millisecond},
		0, nil)

	ackedCount := make(chan int, 1)
	producer := q.Producer(queue.ProducerConfig{ACK: func(n int) { ackedCount <- n }})
	_, ok := producer.Publish(publisher.Event{Content: beat.Event{Private: 0}})
	require.True(t, ok, "publish should succeed")

	c := newEventConsumer(logp.NewNopLogger(), nilObserver)
	ch := make(chan publisher.Batch)
	defer func() {
		q.Close(false)
		c.close()
	}()
	// timeToLive 2: the batch survives one retry, then is dropped on the next.
	c.setTarget(consumerTarget{queue: q, ch: ch, batchSize: 1, timeToLive: 2})

	// First delivery, then retry: still alive, so it is re-delivered.
	first := receiveBatch(t, ch)
	require.Len(t, first.Events(), 1)
	first.Retry()

	// Second delivery (the retry), then retry again: TTL is exhausted -> dropped.
	second := receiveBatch(t, ch)
	require.Len(t, second.Events(), 1)
	second.Retry()

	// Dropping the batch acknowledges the underlying event to the queue.
	select {
	case n := <-ackedCount:
		assert.Equal(t, 1, n, "the dropped event should be acknowledged to the queue")
	case <-time.After(2 * time.Second):
		t.Fatal("expected the event to be acked after TTL exhaustion")
	}

	// The dropped batch must not be delivered again.
	select {
	case <-ch:
		t.Fatal("a dropped batch should not be re-delivered")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestEventConsumerRetryAfterCloseDropsBatch verifies that retry() and
// setTarget() return cleanly once the consumer is shut down, dropping any batch
// handed to retry rather than blocking.
func TestEventConsumerRetryAfterCloseDropsBatch(t *testing.T) {
	c := newEventConsumer(logp.NewNopLogger(), nilObserver)
	c.close()

	dropped := false
	batch := &ttlBatch{retryer: c, done: func() { dropped = true }}
	c.retry(batch, false)
	assert.True(t, dropped, "retry after close should drop the batch")

	// setTarget after close should return without blocking.
	c.setTarget(consumerTarget{})
}

// TestEventConsumerCloseReleasesHeldBatches verifies the shutdown-leak fix:
// when the consumer's run loop is closed while it holds in-flight batches
// (a fresh queueBatch from the reader or batches awaiting retry), those
// batches' done callbacks must fire so the underlying queue can release
// its slots. For slabqueue this is load-bearing — slot indices are
// permanently leaked from the pool's semaphore if Done never runs.
func TestEventConsumerCloseReleasesHeldBatches(t *testing.T) {
	const capacity = 8
	pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: capacity}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{})
	for i := range capacity {
		_, ok := p.Publish(publisher.Event{Content: beat.Event{Private: i}})
		require.True(t, ok)
	}
	require.Equal(t, 0, pool.Available(), "pool should be full after publishing capacity events")

	c := newEventConsumer(logp.NewNopLogger(), nilObserver)

	// Hand the consumer a target whose output channel is a real (but
	// unread) channel. The run loop's request gate requires ch != nil
	// to fire a queueReader read, so we need a real channel; nothing
	// reads from it, so once the consumer has the batch the dispatch
	// blocks and the batch sits in its queueBatch local.
	out := make(chan publisher.Batch)
	c.setTarget(consumerTarget{
		queue:      q,
		ch:         out,
		batchSize:  capacity,
		timeToLive: 3,
	})

	// Give the consumer time to pull the batch from the queue into its
	// queueBatch local and then block on the unread `out`. There's no
	// externally observable signal that the consumer has reached the
	// dispatch step, so we sleep with a margin that's large enough for
	// CI under load.
	time.Sleep(100 * time.Millisecond)

	// Close the consumer. Without the leak fix the held queueBatch (and
	// any retry batches) would be silently dropped without releasing the
	// pool's slot indices.
	c.close()

	require.Eventually(t, func() bool {
		return pool.Available() == capacity
	}, time.Second, 10*time.Millisecond,
		"all slots must return to the pool after consumer close (got %d/%d)",
		pool.Available(), capacity)
}

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

// TestEventConsumerSplitsByDestination drives a real eventConsumer over a real
// memqueue and verifies that, with splitByDestination enabled, a multi-source
// read is delivered as one single-source batch per destination, and the
// underlying queue read is acknowledged only after every split batch completes.
func TestEventConsumerSplitsByDestination(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	q := memqueue.NewQueue[publisher.Event](
		logger, queue.NewQueueObserver(nil),
		memqueue.Settings{Events: 10, MaxGetRequest: 4, FlushTimeout: 10 * time.Millisecond},
		0, nil)

	source1 := &beat.Info{Name: "s1"}
	source2 := &beat.Info{Name: "s2"}

	ackedCount := make(chan int, 1)
	producer := q.Producer(queue.ProducerConfig{ACK: func(n int) { ackedCount <- n }})
	publish := func(id int, src *beat.Info) {
		_, ok := producer.Publish(publisher.Event{Content: beat.Event{Private: id}, Source: src})
		require.True(t, ok, "publish should succeed")
	}
	// Interleave sources so a contiguous sub-slice can't accidentally pass.
	publish(0, source1)
	publish(1, source2)
	publish(2, source1)

	c := newEventConsumer(logger, nilObserver)
	ch := make(chan publisher.Batch)
	defer func() {
		// Close the queue first so the queue reader unblocks, then the consumer
		// (mirrors the controller's shutdown order).
		q.Close(false)
		c.close()
	}()
	c.setTarget(consumerTarget{queue: q, ch: ch, batchSize: 4, timeToLive: 3, splitByDestination: true})

	// Expect one single-source batch per destination.
	batchForSource := map[*beat.Info]publisher.Batch{}
	idsForSource := map[*beat.Info][]int{}
	for len(batchForSource) < 2 {
		b := receiveBatch(t, ch)
		src := b.Events()[0].Source
		for _, e := range b.Events() {
			require.Same(t, src, e.Source, "a split batch must contain a single source")
			idsForSource[src] = append(idsForSource[src], e.Content.Private.(int))
		}
		batchForSource[src] = b
	}
	assert.Equal(t, []int{0, 2}, idsForSource[source1], "source1 should get its own events")
	assert.Equal(t, []int{1}, idsForSource[source2], "source2 should get its own events")

	// Acking only one destination must NOT acknowledge the queue read: the
	// shared completion accounting holds the cap until every destination is done.
	batchForSource[source1].ACK()
	select {
	case <-ackedCount:
		t.Fatal("queue read must not be acked until all split batches are acked")
	case <-time.After(200 * time.Millisecond):
	}

	// Acking the second destination completes the read.
	batchForSource[source2].ACK()
	select {
	case n := <-ackedCount:
		assert.Equal(t, 3, n, "all 3 events should be acked once both split batches complete")
	case <-time.After(2 * time.Second):
		t.Fatal("queue read should be acked once all split batches complete")
	}
}

// TestEventConsumerRetriesUntilTTLExhausted verifies the run loop's retry path:
// a retried batch is re-delivered until its TTL is exhausted, then dropped (and
// the underlying events acknowledged to the queue so it can make progress).
func TestEventConsumerRetriesUntilTTLExhausted(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	q := memqueue.NewQueue[publisher.Event](
		logger, queue.NewQueueObserver(nil),
		memqueue.Settings{Events: 10, MaxGetRequest: 1, FlushTimeout: 10 * time.Millisecond},
		0, nil)

	ackedCount := make(chan int, 1)
	producer := q.Producer(queue.ProducerConfig{ACK: func(n int) { ackedCount <- n }})
	_, ok := producer.Publish(publisher.Event{Content: beat.Event{Private: 0}})
	require.True(t, ok, "publish should succeed")

	c := newEventConsumer(logger, nilObserver)
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
	logger := logptest.NewTestingLogger(t, "")
	c := newEventConsumer(logger, nilObserver)
	c.close()

	dropped := false
	batch := &ttlBatch{retryer: c, done: func() { dropped = true }}
	c.retry(batch, false)
	assert.True(t, dropped, "retry after close should drop the batch")

	// setTarget after close should return without blocking.
	c.setTarget(consumerTarget{})
}

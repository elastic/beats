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

//go:build !nooteloutput

package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// nopLogConsumer is a consumer.Logs that drops every batch it receives.
func nopLogConsumer(t *testing.T) consumer.Logs {
	t.Helper()
	c, err := consumer.NewLogs(func(context.Context, plog.Logs) error { return nil })
	require.NoError(t, err)
	return c
}

// beatInfoForTest returns a beat.Info wired with a nop LogConsumer so the
// otelconsumer worker does not panic when receiving batches.
func beatInfoForTest(t *testing.T) beat.Info {
	return beat.Info{Logger: logp.NewNopLogger(), LogConsumer: nopLogConsumer(t)}
}

// beatInfoNoDrain returns a beat.Info whose LogConsumer blocks until its
// context is cancelled (which the output worker does on Close). This keeps the
// controller's background consumer and workers from acking events while a test
// inspects the queue or pool budget: nopLogConsumer acks almost immediately,
// which races budget assertions and makes them flaky. Published events stay
// live until cleanup, where waitClose cancels the worker context to release it.
func beatInfoNoDrain(t *testing.T) beat.Info {
	t.Helper()
	c, err := consumer.NewLogs(func(ctx context.Context, _ plog.Logs) error {
		<-ctx.Done()
		return ctx.Err()
	})
	require.NoError(t, err)
	return beat.Info{Logger: logp.NewNopLogger(), LogConsumer: c}
}

// monitorsForTest returns a fresh Monitors with its own metrics registry. Each
// controller must get its own registry because loadOutput registers stats
// under fixed names that would collide if reused.
func monitorsForTest() Monitors {
	return Monitors{Logger: logp.NewNopLogger(), Metrics: monitoring.NewRegistry()}
}

func TestOTelQueueMetrics(t *testing.T) {
	// More thorough testing of queue metrics is in the queue package; here we
	// just want to make sure that they appear under the right monitoring
	// namespace. Uses a test-unique intake queue ID so it doesn't share the
	// global pool with any other test.
	reg := monitoring.NewRegistry()
	settings := memqueue.Settings{Events: 1000}
	controller, err := newOTelOutputController(
		beatInfoForTest(t),
		Monitors{
			Logger:  logp.NewNopLogger(),
			Metrics: reg,
		},
		nilObserver,
		"TestOTelQueueMetrics",
		nil, // queueFactory unused on the slabqueue pool path
		settings,
	)
	require.NoError(t, err, "creating OTel output controller should succeed")
	defer controller.waitClose(context.Background(), true)
	entry := reg.Get("pipeline.queue.max_events")
	require.NotNil(t, entry, "pipeline.queue.max_events must exist")
	value, ok := entry.(*monitoring.Uint)
	require.True(t, ok, "pipeline.queue.max_events must be a *monitoring.Uint")
	assert.Equal(t, uint64(1000), value.Get(), "pipeline.queue.max_events should match the events configuration key")
}

// TestEmptyIntakeQueueIDJoinsGlobalPool verifies that two receivers started
// without an explicit intake queue ID share the same (global default) pool,
// matching what an unconfigured production deployment sees.
func TestEmptyIntakeQueueIDJoinsGlobalPool(t *testing.T) {
	settings := memqueue.Settings{Events: 8}

	c1, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"",
		nil, // queueFactory unused on the slabqueue pool path
		settings,
	)
	require.NoError(t, err)
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"",
		nil, // queueFactory unused on the slabqueue pool path
		settings,
	)
	require.NoError(t, err)
	defer c2.waitClose(cancelledContext(), false)

	assert.Same(t, c1.poolForTest(), c2.poolForTest(),
		"empty intake queue ID must share the global default pool across receivers")
}

// TestSharedPoolBudgetIsolatesPipelines verifies that pipelines connected to
// the same intake queue ID share a single event budget, but each pipeline has
// its own independent Queue façade so a slow consumer on one pipeline cannot
// block deliveries on another.
func TestSharedPoolBudgetIsolatesPipelines(t *testing.T) {
	const flushTimeout = time.Second
	settings := memqueue.Settings{
		Events:        5,
		MaxGetRequest: 2,
		FlushTimeout:  flushTimeout,
	}

	c1, err := newOTelOutputController(
		beatInfoNoDrain(t),
		monitorsForTest(),
		nilObserver,
		"sharedID",
		nil, // queueFactory unused on the slabqueue pool path
		settings,
	)
	require.NoError(t, err, "first controller creation should succeed")
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beatInfoNoDrain(t),
		monitorsForTest(),
		nilObserver,
		"sharedID",
		nil, // queueFactory unused on the slabqueue pool path
		settings,
	)
	require.NoError(t, err, "second controller creation should succeed")
	defer c2.waitClose(cancelledContext(), false)

	assert.Same(t, c1.poolForTest(), c2.poolForTest(),
		"controllers with the same intake queue ID must share the same pool")

	prod1 := c1.queueProducer(queue.ProducerConfig{})
	prod2 := c2.queueProducer(queue.ProducerConfig{})

	// Fill the entire pool budget through c1, leaving zero slots for c2.
	for i := 0; i < settings.Events; i++ {
		_, ok := prod1.TryPublish(testEvent(i))
		require.True(t, ok, "TryPublish should succeed while the pool has slots")
	}

	// A further TryPublish on either producer must fail because the pool
	// budget is fully consumed.
	_, ok := prod2.TryPublish(testEvent(100))
	assert.False(t, ok, "TryPublish should fail when the shared pool budget is exhausted")
}

// TestSharedIntakeQueueGrowsToMax verifies that receivers requesting different
// event budgets for the same intake queue ID no longer conflict: the shared
// pool grows to the largest requested budget and every receiver connects.
func TestSharedIntakeQueueGrowsToMax(t *testing.T) {
	c1, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"mismatchID",
		nil, // queueFactory unused on the slabqueue pool path
		memqueue.Settings{Events: 5},
	)
	require.NoError(t, err, "first output controller creation should succeed")
	defer c1.waitClose(cancelledContext(), false)

	require.Equal(t, 5, c1.poolForTest().Target(), "pool starts at the first receiver's budget")

	// A second receiver asks for a larger budget on the same ID. Instead of
	// erroring on the mismatch, the shared pool must grow to the maximum.
	c2, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"mismatchID",
		nil,
		memqueue.Settings{Events: 10},
	)
	require.NoError(t, err, "a differing budget must grow the shared pool, not fail")
	defer c2.waitClose(cancelledContext(), false)

	require.Same(t, c1.poolForTest(), c2.poolForTest(), "both receivers share the pool")
	assert.Equal(t, 10, c1.poolForTest().Target(), "pool grows to the largest requested budget")
	assert.Equal(t, 10, c1.poolForTest().Capacity(), "growth is immediate")

	// A smaller late joiner rides the larger pool without changing the budget.
	c3, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"mismatchID",
		nil,
		memqueue.Settings{Events: 3},
	)
	require.NoError(t, err)
	defer c3.waitClose(cancelledContext(), false)
	assert.Equal(t, 10, c1.poolForTest().Target(), "a smaller joiner does not shrink the pool")
}

// TestSharedIntakeQueueShrinksWhenLargestLeaves verifies that when the receiver
// holding the maximum budget leaves, the pool's target drops to the new running
// maximum (shrink converges lazily under the hood).
func TestSharedIntakeQueueShrinksWhenLargestLeaves(t *testing.T) {
	c1, err := newOTelOutputController(
		beatInfoForTest(t), monitorsForTest(), nilObserver, "shrinkID", nil,
		memqueue.Settings{Events: 4},
	)
	require.NoError(t, err)
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beatInfoForTest(t), monitorsForTest(), nilObserver, "shrinkID", nil,
		memqueue.Settings{Events: 20},
	)
	require.NoError(t, err)

	pool := c1.poolForTest()
	require.Equal(t, 20, pool.Target(), "pool sized to the larger receiver")

	// The larger receiver leaves; the budget must fall back to the smaller one.
	require.NoError(t, c2.waitClose(cancelledContext(), false))
	assert.Equal(t, 4, pool.Target(), "budget drops to the remaining receiver's request")
	assert.Eventually(t, func() bool { return pool.Capacity() == 4 }, time.Second, time.Millisecond,
		"capacity converges down to the new target once high slots are free")
}

// TestSharedIntakeQueueCapsPerReceiver verifies that, on a shared pool sized to
// the largest receiver, each receiver's own queue is still capped at its own
// requested size: the smaller receiver cannot use more than its budget even
// though the pool has room.
func TestSharedIntakeQueueCapsPerReceiver(t *testing.T) {
	c1, err := newOTelOutputController(
		beatInfoNoDrain(t), monitorsForTest(), nilObserver, "capID", nil,
		memqueue.Settings{Events: 4},
	)
	require.NoError(t, err)
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beatInfoNoDrain(t), monitorsForTest(), nilObserver, "capID", nil,
		memqueue.Settings{Events: 8},
	)
	require.NoError(t, err)
	defer c2.waitClose(cancelledContext(), false)

	pool := c1.poolForTest()
	require.Same(t, pool, c2.poolForTest())
	require.Equal(t, 8, pool.Target(), "pool sized to the larger receiver")

	// The small receiver (Events=4) must cap at 4 live events even though the
	// pool has 8 slots.
	p1 := c1.queueProducer(queue.ProducerConfig{})
	for i := 0; i < 4; i++ {
		_, ok := p1.TryPublish(testEvent(i))
		require.True(t, ok, "publish %d within the small receiver's cap should succeed", i)
	}
	_, ok := p1.TryPublish(testEvent(99))
	assert.False(t, ok, "the small receiver must cap at 4 even though the pool has room")
	assert.Equal(t, 4, pool.Available(), "pool still has 4 free slots; only the per-queue cap blocked it")

	// The larger receiver (Events=8) can use the remaining pool budget.
	p2 := c2.queueProducer(queue.ProducerConfig{})
	for i := 0; i < 4; i++ {
		_, ok := p2.TryPublish(testEvent(i))
		require.True(t, ok, "the larger receiver can use the rest of the shared pool")
	}
	assert.Equal(t, 0, pool.Available(), "pool now fully used: 4 + 4 = 8")
}

// TestNonMemQueueOptsOutOfPool verifies that a non-memory queue config
// (e.g. queue.disk in production) opts the receiver out of the slabqueue
// pool and uses the user-supplied queueFactory instead. The receiver
// controller must not be backed by a pool in this mode.
func TestNonMemQueueOptsOutOfPool(t *testing.T) {
	type fakeNonMemqueueSettings struct{ Path string }

	// On the non-mem path the controller calls queueFactory; we substitute
	// an in-memory factory so the test doesn't perform real disk I/O. The
	// point of the test is that the slabqueue pool is skipped, not what
	// the factory actually returns.
	stubFactory := memqueue.FactoryForSettings[publisher.Event](memqueue.Settings{Events: 4})
	c, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"", // non-mem queue cannot be shared, so intake queue ID must be empty
		stubFactory,
		fakeNonMemqueueSettings{Path: "/tmp/dq"},
	)
	require.NoError(t, err, "non-mem queue config must take the queueFactory path")
	defer c.waitClose(cancelledContext(), false)

	assert.Nil(t, c.poolForTest(),
		"non-mem queue receiver must not be backed by an slabqueue pool")
}

func TestSharedIntakeQueueRequiresMemqueue(t *testing.T) {
	// Sharing an intake queue is meaningless for a non-memory queue (each
	// receiver writes to its own on-disk path), so the combination is
	// rejected at startup.
	type fakeNonMemqueueSettings struct{ Path string }

	_, err := newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"non-mem-id",
		nil, // factory unreachable: shared-non-mem combination is rejected
		fakeNonMemqueueSettings{},
	)
	require.Error(t, err, "shared intake queue must reject non-memory queue configs")
	assert.Contains(t, err.Error(), "queue.mem", "error should explain the requirement")
}

// TestSharedPoolRefCount verifies that the pool registered for an intake
// queue ID is only shut down once the last pipeline disconnects.
func TestSharedPoolRefCount(t *testing.T) {
	settings := memqueue.Settings{Events: 4}

	c1, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "refcount-id", nil, settings)
	require.NoError(t, err)
	c2, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "refcount-id", nil, settings)
	require.NoError(t, err)

	pool := c1.poolForTest()
	require.Same(t, pool, c2.poolForTest(), "both controllers should share the pool")

	// First close: pool must remain registered for the second connection.
	require.NoError(t, c1.waitClose(cancelledContext(), false))
	allOTelPools.Lock()
	_, stillRegistered := allOTelPools.lookup["refcount-id"]
	allOTelPools.Unlock()
	assert.True(t, stillRegistered, "pool must remain registered while at least one pipeline uses it")

	// Second close: pool must be deregistered.
	require.NoError(t, c2.waitClose(cancelledContext(), false))
	allOTelPools.Lock()
	_, stillRegistered = allOTelPools.lookup["refcount-id"]
	allOTelPools.Unlock()
	assert.False(t, stillRegistered, "pool must be deregistered once the last pipeline disconnects")
}

// TestPipelineQueuesAreIndependent verifies that closing or stalling one
// pipeline's Queue does not block the other pipeline. We close c1's queue and
// confirm c2 can still publish through its own queue.
func TestPipelineQueuesAreIndependent(t *testing.T) {
	settings := memqueue.Settings{Events: 10}

	c1, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "isolation-id", nil, settings)
	require.NoError(t, err)
	c2, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "isolation-id", nil, settings)
	require.NoError(t, err)
	defer c2.waitClose(cancelledContext(), false)

	// Close c1's pipeline-local queue. The shared pool stays alive (c2 still
	// holds a ref) and c2's queue must remain usable.
	require.NoError(t, c1.waitClose(cancelledContext(), false))

	prod2 := c2.queueProducer(queue.ProducerConfig{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, ok := prod2.TryPublish(testEvent(1))
		assert.True(t, ok, "c2 must still be able to publish after c1 is closed")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("c2 publish did not return; pipeline queues are not independent")
	}
}

// TestConcurrentControllerCreate stresses the acquire/release path under
// concurrency to exercise the global registry locking.
func TestConcurrentControllerCreate(t *testing.T) {
	settings := memqueue.Settings{Events: 4}

	const n = 16
	controllers := make([]*otelOutputController, n)
	infos := make([]beat.Info, n)
	mons := make([]Monitors, n)
	for i := 0; i < n; i++ {
		infos[i] = beatInfoForTest(t)
		mons[i] = monitorsForTest()
	}
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			c, err := newOTelOutputController(infos[i], mons[i], nilObserver, "concurrent-id", nil, settings)
			require.NoError(t, err)
			controllers[i] = c
		}()
	}
	wg.Wait()

	// All controllers should share the same pool.
	first := controllers[0].poolForTest()
	for _, c := range controllers[1:] {
		assert.Same(t, first, c.poolForTest(), "all controllers must share the same pool")
	}

	// Release all of them; the pool must end up deregistered.
	for _, c := range controllers {
		require.NoError(t, c.waitClose(cancelledContext(), false))
	}
	allOTelPools.Lock()
	_, stillRegistered := allOTelPools.lookup["concurrent-id"]
	allOTelPools.Unlock()
	assert.False(t, stillRegistered, "pool must be deregistered once every controller closes")
}

func testEvent(i int) publisher.Event {
	return publisher.Event{
		Content: beat.Event{Private: i},
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

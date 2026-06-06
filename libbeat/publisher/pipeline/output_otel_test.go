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
	conf "github.com/elastic/elastic-agent-libs/config"
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
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"sharedID",
		nil, // queueFactory unused on the slabqueue pool path
		settings,
	)
	require.NoError(t, err, "first controller creation should succeed")
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beatInfoForTest(t),
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

func TestSharedIntakeQueueConfigMismatch(t *testing.T) {
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

	_, err = newOTelOutputController(
		beatInfoForTest(t),
		monitorsForTest(),
		nilObserver,
		"mismatchID",
		nil,
		memqueue.Settings{Events: 10},
	)
	require.Error(t, err, "connecting to a shared intake queue with a different queue config should fail")
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

// TestProducerTrackingUntracksOnClose verifies that a producer the client
// closes itself is removed from the controller's tracking set, so the
// controller does not hold or re-close producers that are already gone.
func TestProducerTrackingUntracksOnClose(t *testing.T) {
	settings := memqueue.Settings{Events: 8}
	c, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "track-close-id", nil, settings)
	require.NoError(t, err)
	defer c.waitClose(cancelledContext(), false)

	prod := c.queueProducer(queue.ProducerConfig{})
	assert.Equal(t, 1, c.trackedProducerCountForTest(), "vended producer must be tracked")

	prod.Close()
	assert.Equal(t, 0, c.trackedProducerCountForTest(), "closing a producer must untrack it")

	// Close is idempotent and must not corrupt the tracking set.
	prod.Close()
	assert.Equal(t, 0, c.trackedProducerCountForTest(), "double close must be a no-op")
}

// TestWaitCloseClosesOpenProducers verifies that producers a client never
// closed are closed by the controller when the pipeline disconnects: the
// tracking set is emptied and the closed producers reject further publishes.
func TestWaitCloseClosesOpenProducers(t *testing.T) {
	settings := memqueue.Settings{Events: 8}
	c, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "track-waitclose-id", nil, settings)
	require.NoError(t, err)

	prod := c.queueProducer(queue.ProducerConfig{})
	require.Equal(t, 1, c.trackedProducerCountForTest(), "vended producer must be tracked")

	// Disconnect the pipeline without the client closing its producer.
	require.NoError(t, c.waitClose(cancelledContext(), false))

	assert.Equal(t, 0, c.trackedProducerCountForTest(),
		"waitClose must close and untrack every still-open producer")
	_, ok := prod.TryPublish(testEvent(1))
	assert.False(t, ok, "a producer closed by waitClose must reject further publishes")
}

// TestWaitCloseGracefulSuccess verifies the graceful drain path: with this
// pipeline's events resolved, waitClose completes via success (its producers'
// ACKWaitChans close and the queue drains) before the context deadline, rather
// than via the force-close timeout path.
func TestWaitCloseGracefulSuccess(t *testing.T) {
	settings := memqueue.Settings{Events: 8}
	c, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "phase4-graceful-id", nil, settings)
	require.NoError(t, err)

	// A vended-but-unused producer: waitClose closes it (0 events, so its
	// ACKWaitChan resolves immediately) and the empty queue's Done fires at
	// once, so a graceful waitClose succeeds well within the deadline.
	c.queueProducer(queue.ProducerConfig{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, c.waitClose(ctx, false))

	assert.NoError(t, ctx.Err(), "graceful waitClose must complete before the context deadline (success path, not timeout)")
	assert.Equal(t, 0, c.trackedProducerCountForTest(), "producers must be closed and untracked")
}

// TestWaitCloseIsBoundedByContext verifies that disconnecting a pipeline with
// still-pending events does not hang: waitClose is bounded by the context, and
// when it expires it force-closes this pipeline's queue, untracks its producers
// and releases the shared pool — without ever blocking on another pipeline.
func TestWaitCloseIsBoundedByContext(t *testing.T) {
	settings := memqueue.Settings{Events: 8}
	c1, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "phase4-bound-id", nil, settings)
	require.NoError(t, err)
	c2, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "phase4-bound-id", nil, settings)
	require.NoError(t, err)
	defer c2.waitClose(cancelledContext(), false)

	// Publish events into c1 that may not drain before the deadline.
	prod := c1.queueProducer(queue.ProducerConfig{})
	for i := 0; i < 4; i++ {
		_, _ = prod.TryPublish(testEvent(i))
	}
	require.Equal(t, 1, c1.trackedProducerCountForTest(), "vended producer must be tracked")

	// An already-expired context must not cause waitClose to hang.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = c1.waitClose(cancelledContext(), false)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("waitClose must be bounded by its context and not hang on pending events")
	}

	assert.Equal(t, 0, c1.trackedProducerCountForTest(), "all producers must be closed and untracked after waitClose")

	// The shared pool must remain registered for the still-connected c2.
	allOTelPools.Lock()
	_, stillRegistered := allOTelPools.lookup["phase4-bound-id"]
	allOTelPools.Unlock()
	assert.True(t, stillRegistered, "disconnecting one pipeline must not shut down the shared pool")
}

// TestCloseProducersClosesTracked verifies closeProducers closes and untracks
// every still-open producer (the safety-net path that runs when producers were
// not already closed earlier in waitClose).
func TestCloseProducersClosesTracked(t *testing.T) {
	settings := memqueue.Settings{Events: 4}
	c, err := newOTelOutputController(beatInfoForTest(t), monitorsForTest(), nilObserver, "closeproducers-id", nil, settings)
	require.NoError(t, err)
	defer c.waitClose(cancelledContext(), false)

	c.queueProducer(queue.ProducerConfig{})
	require.Equal(t, 1, c.trackedProducerCountForTest(), "vended producer must be tracked")

	c.closeProducers()
	assert.Equal(t, 0, c.trackedProducerCountForTest(), "closeProducers must close and untrack tracked producers")
}

// TestNewForReceiverConnects verifies the receiver pipeline constructor wires up
// a working pipeline backed by the slabqueue pool and can be disconnected.
func TestNewForReceiverConnects(t *testing.T) {
	p, err := NewForReceiver(beatInfoForTest(t), monitorsForTest(), conf.Namespace{}, Settings{}, "receiver-ctor-id")
	require.NoError(t, err, "NewForReceiver should succeed with a default (in-memory) queue config")
	require.NotNil(t, p)

	client, err := p.ConnectWith(beat.ClientConfig{})
	require.NoError(t, err)
	require.NoError(t, client.Close())

	require.NoError(t, p.Disconnect(context.Background()))
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

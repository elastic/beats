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
	"sync/atomic"
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
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// These tests hold guaranteed events inside the output consumer, rather than
// only filling the FIFO, to reproduce downstream backpressure at the receiver
// boundary. Capacity must remain available until the stalled output ACKs.
func TestReceiverPublishesWhileSiblingRetainsGuaranteedEvents(t *testing.T) {
	info, entered, release := retainingConsumerForIsolation(t)
	stalled, _ := controllerForIsolation(t, info, 4)
	fillGuaranteedReceiver(t, stalled, 4)
	awaitIsolationSignal(t, entered, "the stalled receiver should enter its downstream consumer")
	require.Equal(t, 4, stalled.pool.Target(), "the sole receiver should start with exactly its requested capacity")
	require.Equal(t, 0, stalled.pool.Available(), "guaranteed output backlog should retain all four slots")

	// Join after the first receiver has exhausted its budget.
	healthy, _ := controllerForIsolation(t, beatInfoForTest(t), 2)
	require.Same(t, stalled.pool, healthy.pool, "the late joiner should use the same pool")
	assert.Equal(t, 6, healthy.pool.Target(), "the late joiner should add its own capacity")
	publishAndAckForIsolation(t, healthy)
	assert.Equal(t, 4, stalled.pool.Capacity()-stalled.pool.Available(), "healthy output delivery should leave the stalled backlog intact")
	close(release)
}

func TestReceiverCapacitySurvivesSiblingLiveShrink(t *testing.T) {
	info, entered, release := retainingConsumerForIsolation(t)
	stalled, _ := controllerForIsolation(t, info, 4)
	healthy, _ := controllerForIsolation(t, beatInfoForTest(t), 2)
	var acked atomic.Int64
	p := stalled.queueProducer(queue.ProducerConfig{ACK: func(n int) { acked.Add(int64(n)) }})
	for i := range 4 {
		_, ok := p.TryPublish(guaranteedIsolationEvent(i))
		require.True(t, ok, "the receiver should admit its original guaranteed backlog")
	}
	p.Close()
	awaitIsolationSignal(t, entered, "the guaranteed backlog should be retained by output")
	q, ok := stalled.queue.(*slabqueue.Queue[publisher.Event])
	require.True(t, ok, "an in-memory receiver should expose its slab queue")
	q.SetTarget(1)
	assert.Equal(t, 6, stalled.pool.Target(), "live excess backlog must keep capacity outside the sibling budget")
	publishAndAckForIsolation(t, healthy)
	close(release)
	awaitIsolationSignal(t, p.ACKWaitChan(), "all guaranteed backlog should ACK after output recovers")
	assert.Equal(t, int64(4), acked.Load(), "shrinking must preserve acknowledgments for every retained event")
	require.Eventually(t, func() bool { return stalled.pool.Target() == 3 }, time.Second, time.Millisecond,
		"capacity should return to the new sum of receiver caps after the excess drains")
}

func TestReceiverCapacitySurvivesSiblingDeparture(t *testing.T) {
	info, entered, release := retainingConsumerForIsolation(t)
	stalled, closeStalled := controllerForIsolation(t, info, 4)
	healthy, _ := controllerForIsolation(t, beatInfoForTest(t), 2)
	fillGuaranteedReceiver(t, stalled, 4)
	awaitIsolationSignal(t, entered, "the guaranteed backlog should reach the output")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		_ = closeStalled(ctx)
	}()
	require.Eventually(t, func() bool { return healthy.pool.ConnectedQueues() == 1 }, time.Second, time.Millisecond,
		"the stalled receiver should disconnect while its graceful shutdown drains")
	assert.Equal(t, 6, healthy.pool.Target(), "the departing receiver's retained slots must stay outside the sibling budget")
	publishAndAckForIsolation(t, healthy)
	close(release)
	awaitIsolationSignal(t, closed, "graceful shutdown should finish after output recovers")
	assert.NoError(t, ctx.Err(), "the departure should drain successfully before its timeout")
	assert.Equal(t, 2, healthy.pool.Target(), "the departed receiver should return its capacity after draining")
	publishAndAckForIsolation(t, healthy)
}

func controllerForIsolation(t *testing.T, info beat.Info, events int) (*otelOutputController, func(context.Context) error) {
	t.Helper()
	c, err := newOTelOutputController(info, monitorsForTest(), nilObserver, nil, memqueue.Settings{Events: events})
	require.NoError(t, err, "the receiver output controller should initialize")
	var once sync.Once
	var closeErr error
	closeController := func(ctx context.Context) error {
		once.Do(func() { closeErr = c.waitClose(ctx, false) })
		return closeErr
	}
	t.Cleanup(func() { _ = closeController(cancelledContext()) })
	return c, closeController
}

func retainingConsumerForIsolation(t *testing.T) (beat.Info, <-chan struct{}, chan struct{}) {
	t.Helper()
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	c, err := consumer.NewLogs(func(ctx context.Context, _ plog.Logs) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	require.NoError(t, err, "the retaining output consumer should initialize")
	return beat.Info{Logger: logp.NewNopLogger(), LogConsumer: c}, entered, release
}

func fillGuaranteedReceiver(t *testing.T, c *otelOutputController, events int) {
	t.Helper()
	p := c.queueProducer(queue.ProducerConfig{})
	for i := range events {
		_, ok := p.TryPublish(guaranteedIsolationEvent(i))
		require.True(t, ok, "the receiver should admit guaranteed event %d within its cap", i)
	}
}

func guaranteedIsolationEvent(i int) publisher.Event {
	e := testEvent(i)
	e.Flags = publisher.GuaranteedSend
	return e
}

func publishAndAckForIsolation(t *testing.T, c *otelOutputController) {
	t.Helper()
	p := c.queueProducer(queue.ProducerConfig{})
	published := make(chan bool, 1)
	go func() {
		_, ok := p.Publish(guaranteedIsolationEvent(100))
		p.Close()
		published <- ok
	}()
	select {
	case ok := <-published:
		require.True(t, ok, "the healthy receiver should publish while its sibling retains guaranteed backlog")
	case <-time.After(time.Second):
		t.Fatal("the healthy receiver blocked on its sibling's backlog")
	}
	awaitIsolationSignal(t, p.ACKWaitChan(), "the healthy receiver should deliver and acknowledge its own event")
}

func awaitIsolationSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

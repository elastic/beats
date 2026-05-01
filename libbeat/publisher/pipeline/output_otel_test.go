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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestOTelQueueMetrics(t *testing.T) {
	// More thorough testing of queue metrics are in the queue package,
	// here we just want to make sure that they appear under the right
	// monitoring namespace.
	reg := monitoring.NewRegistry()
	logger := logptest.NewTestingLogger(t, "")
	controller, err := newOTelOutputController(
		beat.Info{Logger: logger},
		Monitors{
			Logger:  logger,
			Metrics: reg,
		},
		nilObserver,
		memqueue.FactoryForSettings[publisher.Event](memqueue.Settings{Events: 1000}),
		"")
	require.NoError(t, err, "creating OTel output controller should succeed")
	defer controller.waitClose(context.Background(), true)
	entry := reg.Get("pipeline.queue.max_events")
	require.NotNil(t, entry, "pipeline.queue.max_events must exist")
	value, ok := entry.(*monitoring.Uint)
	require.True(t, ok, "pipeline.queue.max_events must be a *monitoring.Uint")
	assert.Equal(t, uint64(1000), value.Get(), "pipeline.queue.max_events should match the events configuration key")
}

func TestSharedQueue(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	// The shared queue holds at most 5 events, and when read will wait up to
	// 1sec to return a "full" batch with 2 events.
	const flushTimeout = time.Second
	queueSettings := memqueue.Settings{
		Events:        5,
		MaxGetRequest: 2,
		FlushTimeout:  flushTimeout,
	}
	queueFactory := func(
		logger *logp.Logger,
		observer queue.Observer,
		inputQueueSize int,
		encoderFactory queue.EncoderFactory[publisher.Event],
	) (queue.Queue[publisher.Event], error) {
		return memqueue.NewQueue(logger, observer, queueSettings, 0, encoderFactory), nil
	}

	c1, err := newOTelOutputController(
		beat.Info{Logger: logger},
		Monitors{
			Logger:  logger,
			Metrics: monitoring.NewRegistry(),
		},
		nilObserver,
		queueFactory,
		"queueID",
	)
	require.NoError(t, err, "output controller creation should succeed")
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beat.Info{Logger: logger},
		Monitors{
			Logger:  logger,
			Metrics: monitoring.NewRegistry(),
		},
		nilObserver,
		queueFactory,
		"queueID",
	)
	require.NoError(t, err, "output controller creation should succeed")
	defer c2.waitClose(cancelledContext(), false)

	assert.Same(t, c1.otelOutputController, c2.otelOutputController, "output controller handles with the same intake queue ID should reference the same output controller")

	// Close the otelconsumer workers, we will check the batches manually from the worker input channel
	for _, worker := range c1.otelOutputController.workers {
		_ = worker.Close()
	}
	batchChan := c1.otelOutputController.workerChan

	ackChan1 := make(chan int, 1)
	prod1 := c1.queueProducer(queue.ProducerConfig{
		ACK: func(count int) {
			ackChan1 <- count
		},
	})
	ackChan2 := make(chan int, 1)
	prod2 := c1.queueProducer(queue.ProducerConfig{
		ACK: func(count int) {
			ackChan2 <- count
		},
	})

	var events []publisher.Event
	for i := range 6 {
		events = append(events, testEvent(i))
	}

	// Publish one event through each handle, verify that they can be read as a single batch
	prod1.Publish(events[0])
	prod2.Publish(events[1])

	// Two events published with a two-event batch size should make a batch
	// ~immediately available on the worker channel.
	var ackFirstBatch func()
	select {
	case batch := <-batchChan:
		batchEvents := batch.Events()
		require.Len(t, batchEvents, 2, "batch should contain 2 events")
		assert.Equal(t, events[0], batchEvents[0])
		assert.Equal(t, events[1], batchEvents[1])

		// Save the ack callback to trigger after the queue is blocked
		ackFirstBatch = batch.ACK
	case <-time.After(flushTimeout / 2):
		require.Fail(t, "expected 2-event batch on worker channel, no batch received")
	}

	// Publish 3 more events through controller 1. After this,
	// the queue should be full.
	prod1.Publish(events[2])
	prod1.Publish(events[3])
	prod1.Publish(events[4])

	// publishedChan will be closed when events[5] is published.
	publishedChan := make(chan struct{})
	go func() {
		defer close(publishedChan)
		prod2.Publish(events[5])
	}()

	select {
	case <-time.After(time.Second):
		// All is well, the event was blocked by the other pipeline's events as expected
	case <-publishedChan:
		require.Fail(t, "Publish call to full shared queue should block")
	}

	// Acknowledging the first batch should unblock the queue
	ackFirstBatch()
	select {
	case <-time.After(time.Second):
		require.Fail(t, "Acknowledging the first batch should unblock the pending Publish call to shared queue")
	case <-publishedChan:
	}
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

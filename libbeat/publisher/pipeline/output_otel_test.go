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
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestOTelQueueMetrics(t *testing.T) {
	// More thorough testing of queue metrics are in the queue package,
	// here we just want to make sure that they appear under the right
	// monitoring namespace.
	reg := monitoring.NewRegistry()
	controller, err := newOTelOutputController(
		beat.Info{Logger: logp.NewNopLogger()},
		Monitors{
			Logger:  logp.NewNopLogger(),
			Metrics: reg,
		},
		nilObserver,
		memqueue.FactoryForSettings[publisher.Event](memqueue.Settings{Events: 1000}),
		"",
		nil)
	require.NoError(t, err, "creating OTel output controller should succeed")
	defer controller.waitClose(context.Background(), true)
	entry := reg.Get("pipeline.queue.max_events")
	require.NotNil(t, entry, "pipeline.queue.max_events must exist")
	value, ok := entry.(*monitoring.Uint)
	require.True(t, ok, "pipeline.queue.max_events must be a *monitoring.Uint")
	assert.Equal(t, uint64(1000), value.Get(), "pipeline.queue.max_events should match the events configuration key")
}

func TestSharedQueue(t *testing.T) {

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
		beat.Info{Logger: logp.NewNopLogger()},
		Monitors{
			Logger:  logp.NewNopLogger(),
			Metrics: monitoring.NewRegistry(),
		},
		nilObserver,
		queueFactory,
		"queueID",
		queueSettings,
	)
	require.NoError(t, err, "output controller creation should succeed")
	defer c1.waitClose(cancelledContext(), false)

	c2, err := newOTelOutputController(
		beat.Info{Logger: logp.NewNopLogger()},
		Monitors{
			Logger:  logp.NewNopLogger(),
			Metrics: monitoring.NewRegistry(),
		},
		nilObserver,
		queueFactory,
		"queueID",
		queueSettings,
	)
	require.NoError(t, err, "output controller creation should succeed")
	defer c2.waitClose(cancelledContext(), false)

	assert.Same(t, c1.otelOutputController, c2.otelOutputController, "output controller handles with the same intake queue ID should reference the same output controller")

	// Close the otelconsumer workers, we will check the batches manually from the worker input channel
	for _, worker := range c1.workers {
		_ = worker.Close()
	}
	batchChan := c1.workerChan

	prod1 := c1.queueProducer(queue.ProducerConfig{})
	prod2 := c2.queueProducer(queue.ProducerConfig{})

	var events []publisher.Event
	for i := range 6 {
		events = append(events, testEvent(i))
	}

	// Publish one event through each handle. They are read together (the queue's
	// MaxGetRequest is 2) and the pipeline splits them into one single-source
	// batch per destination.
	prod1.Publish(events[0])
	prod2.Publish(events[1])

	// Collect the two split batches and their ACK callbacks, keyed by source.
	acks := map[*beat.Info]func(){}
	for range 2 {
		select {
		case batch := <-batchChan:
			batchEvents := batch.Events()
			require.Len(t, batchEvents, 1, "each split batch should target a single source")
			acks[batchEvents[0].Source] = batch.ACK
		case <-time.After(flushTimeout / 2):
			require.Fail(t, "expected a split batch on worker channel, no batch received")
		}
	}
	require.Contains(t, acks, &c1.beatInfo, "prod1's event should be routed to c1's destination")
	require.Contains(t, acks, &c2.beatInfo, "prod2's event should be routed to c2's destination")

	// Fill the rest of the shared queue (cap 5) through controller 1: the two
	// events above still occupy their slots until acknowledged.
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
		// All is well, the event was blocked by the full shared queue as expected.
	case <-publishedChan:
		require.Fail(t, "Publish call to full shared queue should block")
	}

	// The first queue read holds its slots until *both* destinations' split
	// batches are acknowledged, so the shared queue never exceeds its event cap
	// regardless of the number of destinations. Acking only one must not unblock.
	acks[&c1.beatInfo]()
	select {
	case <-time.After(250 * time.Millisecond):
		// Still blocked: one destination's ack is not enough to free the read.
	case <-publishedChan:
		require.Fail(t, "Publish should stay blocked until all split batches are acknowledged")
	}

	// Acking the second destination completes the read and frees its slots.
	acks[&c2.beatInfo]()
	select {
	case <-time.After(time.Second):
		require.Fail(t, "Acknowledging both split batches should unblock the pending Publish")
	case <-publishedChan:
	}
}

func TestSharedQueueConfigMismatch(t *testing.T) {
	queueFactory := func(
		logger *logp.Logger,
		observer queue.Observer,
		inputQueueSize int,
		encoderFactory queue.EncoderFactory[publisher.Event],
	) (queue.Queue[publisher.Event], error) {
		return memqueue.NewQueue(logger, observer, memqueue.Settings{Events: 5}, 0, encoderFactory), nil
	}
	monitors := Monitors{Logger: logp.NewNopLogger(), Metrics: monitoring.NewRegistry()}

	c1, err := newOTelOutputController(
		beat.Info{Logger: logp.NewNopLogger()},
		monitors,
		nilObserver,
		queueFactory,
		"mismatchID",
		memqueue.Settings{Events: 5},
	)
	require.NoError(t, err, "first output controller creation should succeed")
	defer c1.waitClose(cancelledContext(), false)

	_, err = newOTelOutputController(
		beat.Info{Logger: logp.NewNopLogger()},
		monitors,
		nilObserver,
		queueFactory,
		"mismatchID",
		memqueue.Settings{Events: 10},
	)
	require.Error(t, err, "connecting to a shared intake queue with a different queue config should fail")
}

func TestSharedIntakeQueueRequiresMemqueue(t *testing.T) {
	// publisher.Event.Source is not serialized by the disk queue, so a shared
	// intake queue backed by anything but the memory queue would silently
	// misroute events between pipelines. The controller must reject such a
	// misconfiguration at startup.
	queueFactory := func(
		logger *logp.Logger,
		observer queue.Observer,
		inputQueueSize int,
		encoderFactory queue.EncoderFactory[publisher.Event],
	) (queue.Queue[publisher.Event], error) {
		return memqueue.NewQueue(logger, observer, memqueue.Settings{Events: 5}, 0, encoderFactory), nil
	}
	monitors := Monitors{Logger: logp.NewNopLogger(), Metrics: monitoring.NewRegistry()}

	// queueConfig is anything other than memqueue.Settings; the parsed
	// diskqueue settings are the realistic case, modeled here as a struct
	// literal so the test doesn't take a new package dependency.
	type fakeNonMemqueueSettings struct{ Path string }

	_, err := newOTelOutputController(
		beat.Info{Logger: logp.NewNopLogger()},
		monitors,
		nilObserver,
		queueFactory,
		"non-mem-id",
		fakeNonMemqueueSettings{},
	)
	require.Error(t, err, "shared intake queue must reject non-memory queue configs")
	assert.Contains(t, err.Error(), "in-memory queue", "error should explain the requirement")
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

// recordingProducer is a queue.Producer that records the last published event.
type recordingProducer struct {
	published    publisher.Event
	tryPublished publisher.Event
}

func (p *recordingProducer) Publish(event publisher.Event) (queue.EntryID, bool) {
	p.published = event
	return 0, true
}

func (p *recordingProducer) TryPublish(event publisher.Event) (queue.EntryID, bool) {
	p.tryPublished = event
	return 0, true
}

func (p *recordingProducer) Close() {}

func TestSourceTaggingProducer(t *testing.T) {
	source := &beat.Info{Name: "src"}
	rec := &recordingProducer{}
	prod := &sourceTaggingProducer{Producer: rec, source: source}

	prod.Publish(testEvent(1))
	assert.Same(t, source, rec.published.Source, "Publish should tag the event with the source")

	prod.TryPublish(testEvent(2))
	assert.Same(t, source, rec.tryPublished.Source, "TryPublish should tag the event with the source")
}

func TestHandleQueueProducerTagging(t *testing.T) {
	queueFactory := func(
		logger *logp.Logger,
		observer queue.Observer,
		inputQueueSize int,
		encoderFactory queue.EncoderFactory[publisher.Event],
	) (queue.Queue[publisher.Event], error) {
		return memqueue.NewQueue(logger, observer, memqueue.Settings{Events: 5}, 0, encoderFactory), nil
	}
	monitors := Monitors{Logger: logp.NewNopLogger(), Metrics: monitoring.NewRegistry()}

	// A non-shared controller has a single destination, so it must not wrap the
	// producer with source tagging.
	plain, err := newOTelOutputController(beat.Info{Logger: logp.NewNopLogger()}, monitors, nilObserver, queueFactory, "", nil)
	require.NoError(t, err)
	defer plain.waitClose(cancelledContext(), false)
	_, tagging := plain.queueProducer(queue.ProducerConfig{}).(*sourceTaggingProducer)
	assert.False(t, tagging, "non-shared handle should not tag events")

	// A shared controller must wrap the producer so events can be routed back
	// to the pipeline that produced them.
	shared, err := newOTelOutputController(beat.Info{Logger: logp.NewNopLogger()}, monitors, nilObserver, queueFactory, "tagging-id", memqueue.Settings{Events: 5})
	require.NoError(t, err)
	defer shared.waitClose(cancelledContext(), false)
	_, tagging = shared.queueProducer(queue.ProducerConfig{}).(*sourceTaggingProducer)
	assert.True(t, tagging, "shared handle should tag events")
}

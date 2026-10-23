// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type blockedProfileOutput struct {
	entered chan struct{}
	release chan struct{}
	started sync.Once
	closed  sync.Once
}

func (o *blockedProfileOutput) Publish(ctx context.Context, batch publisher.Batch) error {
	o.started.Do(func() { close(o.entered) })
	select {
	case <-o.release:
	case <-ctx.Done():
	}
	batch.ACK()
	return nil
}

func (o *blockedProfileOutput) Close() error {
	o.closed.Do(func() { close(o.release) })
	return nil
}
func (*blockedProfileOutput) String() string { return "blocked profile test output" }

// blockedPipeline returns a real pipeline with the given queue settings whose
// output withholds acknowledgments until it is closed.
func blockedPipeline(t *testing.T, queue map[string]any) (*pipeline.Pipeline, *blockedProfileOutput) {
	t.Helper()
	var cfg pipeline.Config
	require.NoError(t, conf.MustNewConfigFrom(queue).Unpack(&cfg), "configure a small queue")
	output := &blockedProfileOutput{entered: make(chan struct{}), release: make(chan struct{})}
	pipe, err := pipeline.LoadWithSettings(
		beat.Info{Name: "osquerybeat", Logger: logptest.NewTestingLogger(t, "pipeline")},
		pipeline.Monitors{}, cfg,
		func(outputs.Observer) (string, outputs.Group, error) {
			return "blocked", outputs.Group{BatchSize: 1, Clients: []outputs.Client{output}}, nil
		},
		pipeline.Settings{WaitCloseMode: pipeline.WaitOnPipelineCloseThenForce},
	)
	require.NoError(t, err, "create the real publisher pipeline")
	return pipe, output
}

// fillQueue publishes capacity results through the guaranteed-delivery client
// and waits until the output holds the first of them unacknowledged.
func fillQueue(t *testing.T, p *Publisher, output *blockedProfileOutput, capacity int) {
	t.Helper()
	hits := make([]map[string]any, capacity)
	for i := range hits {
		hits[i] = map[string]any{"value": i}
	}
	p.Publish("index", "id", "schedule_id", "response", "", "", "", "query", nil, hits, nil, nil)
	select {
	case <-output.entered:
	case <-time.After(time.Second):
		t.Fatal("output did not receive the first result")
	}
}

// TestScheduledProfileSinkDoesNotBlockOnFullPool uses a real pipeline whose
// slab queue is full because the output withholds acknowledgments. It covers
// the slab queue only: on the memory queue a DropIfFull publish can wait for a
// slot by design, so this test would block there.
func TestScheduledProfileSinkDoesNotBlockOnFullPool(t *testing.T) {
	// Build a real pipeline with a tiny slab queue and an output that never acks.
	const capacity = 32 // The slab queue rejects smaller pools.
	pipe, output := blockedPipeline(t, map[string]any{"queue.slab.events": capacity})

	// Wrap the pipeline in a publisher and make sure the output is always released.
	p := New(&beat.Beat{Publisher: pipe}, logptest.NewTestingLogger(t, "pub"))
	t.Cleanup(func() {
		// Always release the output, including if a regression fails the test.
		_ = output.Close()
		p.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, pipe.Disconnect(ctx), "disconnect the test publisher pipeline")
	})

	// Capture the scheduled profile sink that will be offered to the full queue.
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the scheduled profile destination")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the scheduled profile destination")

	// Fill every slot through the guaranteed-delivery client, with no acks.
	fillQueue(t, p, output, capacity)

	// Offer many profiles to the full queue: none may block, and each must be counted as dropped.
	const attempts = 1024
	var accepted int
	for range attempts {
		published := make(chan bool, 1)
		go func() { published <- sink.Publish("scheduled", "schedule", "response", "", 1, nil) }()
		select {
		case ok := <-published:
			if ok {
				accepted++
			}
		case <-time.After(time.Second):
			t.Fatal("a profile Publish waited for queue space")
		}
	}
	assert.Zero(t, accepted, "a profile offered to a full queue must report that it was dropped")
	assert.Equal(t, uint64(attempts), p.scheduledProfileDropped.Get(), "count every profile dropped on the full queue")
	assert.True(t, sink.Valid(), "dropping profiles must not retire the lease")
}

// profileClientPipeline tracks the DropIfFull clients it creates, which only
// the scheduled profile sink requests.
type profileClientPipeline struct {
	beat.Pipeline
	clients []*trackedProfileClient
}

func (p *profileClientPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	c, err := p.Pipeline.ConnectWith(cfg)
	if err != nil || cfg.PublishMode != beat.DropIfFull {
		return c, err
	}
	tracked := &trackedProfileClient{Client: c, entered: make(chan struct{})}
	p.clients = append(p.clients, tracked)
	return tracked, nil
}

// trackedProfileClient reports when Publish is first called and counts Close calls.
type trackedProfileClient struct {
	beat.Client
	entered chan struct{}
	once    sync.Once
	closes  atomic.Int32
}

func (c *trackedProfileClient) Publish(e beat.Event) {
	c.once.Do(func() { close(c.entered) })
	c.Client.Publish(e)
}

func (c *trackedProfileClient) Close() error {
	c.closes.Add(1)
	return c.Client.Close()
}

// TestScheduledProfileSinkRetireDoesNotWaitForStuckPublish uses a full memory
// queue, where a DropIfFull publish waits for a slot. Retiring the lease must
// not wait for that publish.
func TestScheduledProfileSinkRetireDoesNotWaitForStuckPublish(t *testing.T) {
	// Build a real pipeline with a tiny memory queue and an output that withholds acks.
	const capacity = 32
	pipe, output := blockedPipeline(t, map[string]any{
		"queue.mem.events":           capacity,
		"queue.mem.flush.min_events": 1,
		"queue.mem.flush.timeout":    "0s",
	})
	tracked := &profileClientPipeline{Pipeline: pipe}
	p := New(&beat.Beat{Publisher: tracked}, logptest.NewTestingLogger(t, "pub"))
	t.Cleanup(func() {
		// Release the output first so a regression's stuck publish can return.
		_ = output.Close()
		p.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, pipe.Disconnect(ctx), "disconnect the test publisher pipeline")
	})

	// Capture the sink and fill every queue slot with results.
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the scheduled profile destination")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the scheduled profile destination")
	require.Len(t, tracked.clients, 1, "the sink should own one profile client")
	old := tracked.clients[0]
	fillQueue(t, p, output, capacity)

	// Start a profile publish and wait until it is inside the client.
	published := make(chan bool, 1)
	go func() { published <- sink.Publish("scheduled", "schedule", "response", "", 1, nil) }()
	select {
	case <-old.entered:
	case <-time.After(time.Second):
		t.Fatal("the profile publish did not reach the client")
	}
	select {
	case <-published:
		t.Fatal("the profile publish returned; the test needs it to wait for a queue slot")
	case <-time.After(50 * time.Millisecond):
	}

	// A later publish must not wait behind it, or every daemon restart would
	// strand another collector.
	second := make(chan bool, 1)
	go func() { second <- sink.Publish("scheduled", "schedule", "response", "", 1, nil) }()
	select {
	case ok := <-second:
		assert.False(t, ok, "a publish behind a stuck one must be dropped")
	case <-time.After(time.Second):
		t.Fatal("a second profile publish waited behind a stuck one")
	}
	assert.Equal(t, uint64(1), p.scheduledProfileDropped.Get(), "count the profile dropped behind a stuck publish")

	// A policy change and shutdown must not wait for the stuck publish.
	configured := make(chan error, 1)
	go func() { configured <- p.Configure(scheduledProfileInputs("default")) }()
	select {
	case err := <-configured:
		assert.NoError(t, err, "replace the policy while a profile publish is stuck")
	case <-time.After(time.Second):
		t.Fatal("Configure waited for a stuck profile publish")
	}
	closed := make(chan struct{})
	go func() {
		p.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close waited for a stuck profile publish")
	}
	assert.Zero(t, old.closes.Load(), "the stuck publish still owns the retired client")

	// Releasing the output lets the publish return, and it closes the retired client.
	_ = output.Close()
	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("the stuck profile publish did not return after the output recovered")
	}
	assert.Equal(t, int32(1), old.closes.Load(), "the retired client must be closed exactly once")
}

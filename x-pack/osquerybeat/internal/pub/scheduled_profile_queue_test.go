// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"context"
	"sync"
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

// TestScheduledProfileSinkDoesNotBlockOnFullPool uses a real pipeline whose
// slab queue is full because the output withholds acknowledgments. It covers
// the slab queue only: on the memory queue a DropIfFull publish can wait for a
// slot by design, so this test would block there.
func TestScheduledProfileSinkDoesNotBlockOnFullPool(t *testing.T) {
	// Build a real pipeline with a tiny slab queue and an output that never acks.
	const capacity = 32 // The slab queue rejects smaller pools.
	var cfg pipeline.Config
	require.NoError(t, conf.MustNewConfigFrom(map[string]any{
		"queue.slab.events": capacity,
	}).Unpack(&cfg), "configure a small slab queue")
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

	// Offer many profiles to the full queue: none may block, and each must be counted as dropped.
	const attempts = 1024
	var slowest time.Duration
	var accepted int
	for range attempts {
		start := time.Now()
		if sink.Publish("scheduled", "schedule", "response", "", 1, nil) {
			accepted++
		}
		slowest = max(slowest, time.Since(start))
	}
	assert.Zero(t, accepted, "a profile offered to a full queue must report that it was dropped")
	assert.Less(t, slowest, time.Second, "no Publish call may wait for queue space")
	assert.Equal(t, uint64(attempts), p.scheduledProfileDropped.Get(), "count every profile dropped on the full queue")
	assert.True(t, sink.Valid(), "dropping profiles must not retire the lease")
}

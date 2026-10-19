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
	"github.com/elastic/elastic-agent-libs/logp"
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

type observedProfileClient struct {
	beat.Client
	entered chan struct{}
}

func (c *observedProfileClient) Publish(event beat.Event) {
	close(c.entered)
	c.Client.Publish(event)
}

func TestScheduledProfileFullQueueDoesNotBlockAdmissionOrClose(t *testing.T) {
	// Fill a real memory queue with results and withhold output acknowledgments.
	var cfg pipeline.Config
	require.NoError(t, conf.MustNewConfigFrom(map[string]any{
		"queue.mem.events":           32,
		"queue.mem.flush.min_events": 1,
		"queue.mem.flush.timeout":    time.Millisecond,
	}).Unpack(&cfg), "configure a small publisher queue")
	output := &blockedProfileOutput{entered: make(chan struct{}), release: make(chan struct{})}
	pipe, err := pipeline.LoadWithSettings(
		beat.Info{Name: "osquerybeat", Logger: logp.NewNopLogger()},
		pipeline.Monitors{}, cfg,
		func(outputs.Observer) (string, outputs.Group, error) {
			return "blocked", outputs.Group{BatchSize: 1, Clients: []outputs.Client{output}}, nil
		},
		pipeline.Settings{WaitCloseMode: pipeline.WaitOnPipelineCloseThenForce},
	)
	require.NoError(t, err, "create the real publisher pipeline")
	p := New(&beat.Beat{Publisher: pipe}, logp.NewNopLogger())
	t.Cleanup(func() {
		// Always release the output, including if a shutdown regression fails.
		_ = output.Close()
		p.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, pipe.Disconnect(ctx), "disconnect the test publisher pipeline")
		closeScheduledProfilePublisher(t, p)
	})
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the scheduled profile destination")
	sink := p.CaptureScheduledProfileSink("default").(*scheduledProfileSink)
	observed := &observedProfileClient{Client: sink.client, entered: make(chan struct{})}
	sink.client = observed
	hits := make([]map[string]any, 32)
	for i := range hits {
		hits[i] = map[string]any{"value": i}
	}
	p.Publish("index", "schedule", "schedule_id", "response", "", "", "", "query", nil, hits, nil, nil)
	select {
	case <-output.entered:
	case <-time.After(time.Second):
		t.Fatal("output did not receive the first result")
	}
	require.True(t, sink.Publish("scheduled", "schedule", "response", "", 1, nil), "offer one profile to the saturated pipeline")
	select {
	case <-observed.entered:
	case <-time.After(time.Second):
		t.Fatal("the independent publisher did not attempt admission")
	}

	// This must finish while the one real Publish call is still blocked.
	finished := make(chan struct{})
	go func() {
		for range 1024 {
			sink.Publish("scheduled", "schedule", "response", "", 1, nil)
		}
		p.Close()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("scheduled profile offers or publisher shutdown waited for the full queue")
	}
	assert.Equal(t, uint64(1024), p.scheduledProfileWorker.dropped.Get(), "drop profiles locally while the single publication worker is busy")
	assert.False(t, sink.Valid(), "shutdown must retire the profile destination")
	select {
	case <-p.scheduledProfileWorker.done:
		t.Fatal("the saturated queue should still hold the one in-flight publication")
	default:
	}

	// Pipeline shutdown must release the remaining worker without an output ACK.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	require.NoError(t, pipe.Disconnect(ctx), "force-close the full queue through the existing pipeline lifecycle")
	select {
	case <-p.scheduledProfileWorker.done:
	case <-time.After(time.Second):
		t.Fatal("pipeline closure did not release the remaining publication worker")
	}
}

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
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type blockedProfileOutput struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (o *blockedProfileOutput) Publish(_ context.Context, batch publisher.Batch) error {
	o.once.Do(func() { close(o.entered) })
	<-o.release
	batch.ACK()
	return nil
}

func (*blockedProfileOutput) Close() error   { return nil }
func (*blockedProfileOutput) String() string { return "blocked profile test output" }

func TestScheduledProfileFullQueueDoesNotBlockAdmissionOrClose(t *testing.T) {
	// Keep a real memory queue full by withholding acknowledgments in its output.
	var cfg pipeline.Config
	err := conf.MustNewConfigFrom(map[string]any{
		"queue.mem.events":           32,
		"queue.mem.flush.min_events": 1,
		"queue.mem.flush.timeout":    time.Millisecond,
	}).Unpack(&cfg)
	require.NoError(t, err, "configure a small publisher queue")
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
	p := New(&beat.Beat{Publisher: pipe}, logptest.NewTestingLogger(t, "full_profile_queue"))
	t.Cleanup(func() {
		// Unblock the output even if admission regresses to a blocking publish.
		close(output.release)
		p.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, pipe.Disconnect(ctx), "disconnect the test publisher pipeline")
	})
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the scheduled profile destination")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the active profile sink")
	require.True(t, sink.Publish("scheduled", "schedule", "response", "", 1, nil), "start the output worker")
	select {
	case <-output.entered:
	case <-time.After(time.Second):
		t.Fatal("output did not receive the first profile")
	}

	// Admission must finish while the output remains blocked, with observable drops.
	published := make(chan struct{})
	go func() {
		for range 1024 {
			sink.Publish("scheduled", "schedule", "response", "", 1, nil)
		}
		close(published)
	}()
	select {
	case <-published:
	case <-time.After(time.Second):
		t.Fatal("scheduled profile admission waited for space in the real publisher queue")
	}
	assert.Positive(t, p.scheduledProfileDropped.Get(), "the real queue must saturate and report dropped profiles")

	// Client shutdown must also complete before output recovery or acknowledgment.
	closed := make(chan struct{})
	go func() {
		p.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("scheduled profile shutdown waited for the blocked output")
	}
	assert.False(t, sink.Valid(), "shutdown must retire the old profile destination")
}

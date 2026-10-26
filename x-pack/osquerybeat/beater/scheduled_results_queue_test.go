// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

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
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pub"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type outageTestOutput struct {
	release chan struct{}
	once    sync.Once
}

func (o *outageTestOutput) Publish(ctx context.Context, batch publisher.Batch) error {
	select {
	case <-o.release:
	case <-ctx.Done():
	}
	batch.ACK()
	return nil
}
func (o *outageTestOutput) Close() error { o.once.Do(func() { close(o.release) }); return nil }
func (*outageTestOutput) String() string { return "stalled output" }

type observedResultPipeline struct {
	beat.PipelineConnector
	calls   atomic.Int32
	entered chan struct{}
}

func (p *observedResultPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	client, err := p.PipelineConnector.ConnectWith(cfg)
	if err != nil {
		return nil, err
	}
	return &observedResultClient{Client: client, owner: p}, nil
}

type observedResultClient struct {
	beat.Client
	owner *observedResultPipeline
}

func (c *observedResultClient) Publish(event beat.Event) {
	if c.owner.calls.Add(1) == 33 {
		close(c.owner.entered)
	}
	c.Client.Publish(event)
}

type outageQueryClient struct{}

func (outageQueryClient) Query(context.Context, string, time.Duration) ([]map[string]any, error) {
	return nil, nil
}
func (outageQueryClient) ResolveResult(_ context.Context, _ string, rows []map[string]string) ([]map[string]any, error) {
	resolved := make([]map[string]any, len(rows))
	for i, row := range rows {
		resolved[i] = map[string]any{"value": row["value"]}
	}
	return resolved, nil
}

func TestScheduledResultsSaturatedPipelineDoesNotBlockShutdown(t *testing.T) {
	log := logp.NewNopLogger()
	var cfg pipeline.Config
	require.NoError(t, conf.MustNewConfigFrom(map[string]any{
		"queue.mem.events": 32, "queue.mem.flush.min_events": 1, "queue.mem.flush.timeout": time.Millisecond,
	}).Unpack(&cfg), "configure the real memory queue")
	output := &outageTestOutput{release: make(chan struct{})}
	pipe, err := pipeline.LoadWithSettings(beat.Info{Name: "osquerybeat", Logger: log}, pipeline.Monitors{}, cfg,
		func(outputs.Observer) (string, outputs.Group, error) {
			return "blocked", outputs.Group{BatchSize: 1, Clients: []outputs.Client{output}}, nil
		},
		pipeline.Settings{WaitCloseMode: pipeline.WaitOnPipelineCloseThenForce})
	require.NoError(t, err, "create the real publisher pipeline")
	observed := &observedResultPipeline{PipelineConnector: pipe, entered: make(chan struct{})}
	bt := &osquerybeat{log: log}
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents, func(allowed func() bool, dropped func()) osquerybeatPublisher {
		return pub.NewScheduled(&beat.Beat{Publisher: observed}, log, allowed, dropped)
	}, bt.handleScheduledQueryResult)
	t.Cleanup(func() {
		_ = output.Close()
		d.stop()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, pipe.Disconnect(ctx), "close the test pipeline")
		select {
		case <-d.done:
		case <-time.After(time.Second):
			t.Error("publication did not exit after closing the queue")
		}
	})
	inputs := outageInputs("old")
	require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure scheduled-only publisher clients")
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(inputs), "stage the native policy")
	_, err = plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the policy")
	rows := make([]map[string]string, 64)
	for i := range rows {
		rows[i] = map[string]string{"value": "row"}
	}
	result := QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}
	d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	select {
	case <-observed.entered:
	case <-time.After(time.Second):
		t.Fatal("worker did not reach the saturated queue")
	}

	finished := make(chan struct{})
	go func() {
		for range 256 {
			d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
		}
		d.stop()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("admission or shutdown waited for the full queue")
	}
	assert.Positive(t, d.metrics.overflow.Get(), "saturation must cause observable local drops")

	// The existing pipeline lifecycle, after the beater returns, releases its
	// one blocked publication. Retiring the lease stops all remaining rows.
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	require.NoError(t, pipe.Disconnect(ctx), "force-close without restoring output delivery")
	select {
	case <-d.done:
	case <-time.After(time.Second):
		t.Fatal("pipeline closure did not release the scheduled worker")
	}
	assert.Equal(t, int32(33), observed.calls.Load(), "cancelled work must not publish the remaining rows or completion")
}

// pacedOutput acknowledges one batch every 20ms and counts the events that
// are not marked as noise.
type pacedOutput struct {
	scheduled atomic.Int32
}

func (o *pacedOutput) Publish(ctx context.Context, batch publisher.Batch) error {
	select {
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
	}
	for _, e := range batch.Events() {
		if _, noise := e.Content.Fields["noise"]; !noise {
			o.scheduled.Add(1)
		}
	}
	batch.ACK()
	return nil
}
func (*pacedOutput) Close() error   { return nil }
func (*pacedOutput) String() string { return "paced output" }

// TestScheduledWorkerGetsSlotsUnderPoolPressure shares a slab queue pool with
// a client that publishes continuously, as all receivers share one pool on the
// OTel runtime. The scheduled worker must wait for slots rather than drop.
func TestScheduledWorkerGetsSlotsUnderPoolPressure(t *testing.T) {
	log := logp.NewNopLogger()
	var cfg pipeline.Config
	require.NoError(t, conf.MustNewConfigFrom(map[string]any{"queue.slab.events": 64}).Unpack(&cfg), "configure the slab queue pool")
	output := &pacedOutput{}
	pipe, err := pipeline.LoadWithSettings(beat.Info{Name: "osquerybeat", Logger: log}, pipeline.Monitors{}, cfg,
		func(outputs.Observer) (string, outputs.Group, error) {
			return "paced", outputs.Group{BatchSize: 8, Clients: []outputs.Client{output}}, nil
		},
		pipeline.Settings{WaitCloseMode: pipeline.WaitOnPipelineCloseThenForce})
	require.NoError(t, err, "create the real publisher pipeline")

	noise, err := pipe.ConnectWith(beat.ClientConfig{})
	require.NoError(t, err, "connect the competing client")
	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	wg.Go(func() {
		for ctx.Err() == nil {
			noise.Publish(beat.Event{Timestamp: time.Now(), Fields: mapstr.M{"noise": true}})
		}
	})

	bt := &osquerybeat{log: log}
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents, func(allowed func() bool, dropped func()) osquerybeatPublisher {
		return pub.NewScheduled(&beat.Beat{Publisher: pipe}, log, allowed, dropped)
	}, bt.handleScheduledQueryResult)
	t.Cleanup(func() {
		d.stop()
		cancel()
		ctx, cancelDisconnect := context.WithTimeout(context.Background(), time.Second)
		defer cancelDisconnect()
		assert.NoError(t, pipe.Disconnect(ctx), "close the test pipeline")
		wg.Wait()
		select {
		case <-d.done:
		case <-time.After(time.Second):
			t.Error("publication did not exit after closing the queue")
		}
	})
	inputs := append(outageInputs("default"), config.InputConfig{
		Datastream: config.DatastreamConfig{Dataset: config.DefaultActionResponsesDataset, Namespace: "default"},
	})
	require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure scheduled clients")
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(inputs), "stage the native policy")
	_, err = plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the policy")

	rows := make([]map[string]string, 5)
	for i := range rows {
		rows[i] = map[string]string{"value": "row"}
	}
	result := QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}
	for i := range 20 {
		start := time.Now()
		d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
		assert.Less(t, time.Since(start), 50*time.Millisecond, "enqueue %d must not wait for the pool", i)
	}
	const want = 20*5 + 20 // Result rows plus one response per result.
	assert.Eventually(t, func() bool { return output.scheduled.Load() == want }, 10*time.Second, 10*time.Millisecond,
		"every scheduled row and response must be published, got %d of %d", output.scheduled.Load(), want)
	assert.Zero(t, d.metrics.overflowRows.Get(), "no rows may overflow")
	assert.Zero(t, d.metrics.staleRows.Get(), "no rows may be stale")
	assert.Zero(t, d.metrics.expiredRows.Get(), "no rows may expire")
}

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
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pub"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, func(allowed func() bool, dropped func()) osquerybeatPublisher {
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
	require.NoError(t, d.configure(inputs), "configure scheduled-only publisher clients")
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

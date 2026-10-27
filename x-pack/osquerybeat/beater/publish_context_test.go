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
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// These tests use real time: the pipeline's client reaper waits on a channel
// created outside any synctest bubble, so synctest.Wait would never return.

// stuckQueues are the queues of the stuck pipeline tests, each holding 32
// events.
var stuckQueues = []struct {
	name string
	cfg  map[string]any
}{
	{"slabqueue", map[string]any{"queue.slab.events": 32}},
	{"memqueue", map[string]any{"queue.mem.events": 32, "queue.mem.flush.min_events": 1, "queue.mem.flush.timeout": time.Millisecond}},
}

// stuckRows is more rows than a stuck queue holds, so publishing them blocks.
const stuckRows = 64

// cancelBound is how soon a cancelled publish must release its worker.
const cancelBound = 100 * time.Millisecond

// stuckOutput holds every batch until it is unblocked, then counts its
// events as delivered.
type stuckOutput struct {
	release   chan struct{}
	once      sync.Once
	delivered atomic.Int64
}

func (o *stuckOutput) Publish(ctx context.Context, batch publisher.Batch) error {
	select {
	case <-o.release:
		o.delivered.Add(int64(len(batch.Events())))
	case <-ctx.Done():
	}
	batch.ACK()
	return nil
}

func (o *stuckOutput) Close() error { o.unblock(); return nil }
func (*stuckOutput) String() string { return "stuck output" }
func (o *stuckOutput) unblock()     { o.once.Do(func() { close(o.release) }) }

// stuckPipeline is a real pipeline whose output holds every batch until it
// is unblocked. It counts the publishes its clients start and finish.
type stuckPipeline struct {
	*pipeline.Pipeline
	output *stuckOutput
	// plain makes its clients implement only beat.Client, as a client
	// wrapper that does not forward PublishContext does.
	plain             bool
	entered, returned atomic.Int64
}

func newStuckPipeline(t *testing.T, queue map[string]any) *stuckPipeline {
	t.Helper()
	log := logp.NewNopLogger()
	var cfg pipeline.Config
	require.NoError(t, conf.MustNewConfigFrom(queue).Unpack(&cfg), "configure the queue")
	output := &stuckOutput{release: make(chan struct{})}
	pipe, err := pipeline.LoadWithSettings(beat.Info{Name: "osquerybeat", Logger: log}, pipeline.Monitors{}, cfg,
		func(outputs.Observer) (string, outputs.Group, error) {
			return "stuck", outputs.Group{BatchSize: 1, Clients: []outputs.Client{output}}, nil
		},
		pipeline.Settings{WaitCloseMode: pipeline.WaitOnPipelineCloseThenForce})
	require.NoError(t, err, "create the real publisher pipeline")
	return &stuckPipeline{Pipeline: pipe, output: output}
}

func (p *stuckPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	c, err := p.Pipeline.ConnectWith(cfg)
	if err != nil {
		return nil, err
	}
	counted := &countedClient{Client: c, owner: p}
	if p.plain {
		return counted, nil
	}
	return &countedContextClient{counted}, nil
}

// waitBlocked waits until a publish has been in flight for 50ms.
func (p *stuckPipeline) waitBlocked(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool {
		r := p.returned.Load()
		if p.entered.Load() == r {
			return false
		}
		time.Sleep(50 * time.Millisecond)
		return p.returned.Load() == r
	}, 5*time.Second, time.Millisecond, "a publish must block on the full queue")
}

// idle reports whether no publish is in flight.
func (p *stuckPipeline) idle() bool {
	return p.entered.Load() == p.returned.Load()
}

// close unblocks the output and closes the pipeline, which waits for the
// queued events to be delivered.
func (p *stuckPipeline) close(t *testing.T) {
	p.output.unblock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assert.NoError(t, p.Disconnect(ctx), "close the test pipeline")
}

type countedClient struct {
	beat.Client
	owner *stuckPipeline
}

func (c *countedClient) Publish(e beat.Event) {
	c.owner.entered.Add(1)
	defer c.owner.returned.Add(1)
	c.Client.Publish(e)
}

type countedContextClient struct{ *countedClient }

func (c *countedContextClient) PublishContext(ctx context.Context, e beat.Event) error {
	c.owner.entered.Add(1)
	defer c.owner.returned.Add(1)
	return c.Client.(beat.ContextPublisher).PublishContext(ctx, e)
}

// stuckScheduledDispatcher returns a dispatcher publishing to pipe under a
// policy with result and action response streams in the default namespace,
// and a scheduled result of stuckRows rows plus its response.
func stuckScheduledDispatcher(t *testing.T, pipe *stuckPipeline) (*scheduledResultDispatcher, *ConfigPlugin, QueryResult) {
	t.Helper()
	log := logp.NewNopLogger()
	bt := &osquerybeat{log: log}
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
		func(ctx context.Context, allowed func() bool, dropped func(), cut func(int)) osquerybeatPublisher {
			return pub.NewScheduled(ctx, &beat.Beat{Publisher: pipe}, log, allowed, dropped, cut)
		}, bt.handleScheduledQueryResult)
	inputs := stuckScheduledInputs("default")
	require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure scheduled clients")
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(inputs), "stage the native policy")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the policy")
	rows := make([]map[string]string, stuckRows)
	for i := range rows {
		rows[i] = map[string]string{"value": "row"}
	}
	return d, plugin, QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}
}

func stuckScheduledInputs(namespace string) []config.InputConfig {
	return append(outageInputs(namespace), config.InputConfig{
		Datastream: config.DatastreamConfig{Dataset: config.DefaultActionResponsesDataset, Namespace: namespace},
	})
}

func TestScheduledWorkerCancelsBlockedPublishOnShutdown(t *testing.T) {
	for _, q := range stuckQueues {
		t.Run(q.name, func(t *testing.T) {
			pipe := newStuckPipeline(t, q.cfg)
			d, plugin, result := stuckScheduledDispatcher(t, pipe)
			d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
			pipe.waitBlocked(t)

			d.stop()
			select {
			case <-d.done:
			case <-time.After(cancelBound):
				t.Errorf("the worker blocked on the full queue did not return within %v of shutdown", cancelBound)
			}

			pipe.close(t)
			<-d.done
			dropped := int64(d.metrics.droppedRows[dropShutdown].Get())
			assert.Equal(t, uint64(1), d.metrics.dropped[dropShutdown].Get(), "count the cut item once")
			assert.Zero(t, d.metrics.droppedRows[dropStale].Get(), "a shutdown is not a stale drop")
			assert.Positive(t, pipe.output.delivered.Load(), "rows queued before the shutdown are delivered")
			assert.Equal(t, int64(stuckRows+1), pipe.output.delivered.Load()+dropped,
				"each row and the response is either delivered or counted once as a shutdown drop, never both")
		})
	}
}

func TestScheduledPolicyChangeCancelsBlockedPublish(t *testing.T) {
	for _, q := range stuckQueues {
		t.Run(q.name, func(t *testing.T) {
			pipe := newStuckPipeline(t, q.cfg)
			d, plugin, result := stuckScheduledDispatcher(t, pipe)
			d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
			pipe.waitBlocked(t)

			// The query moves to another namespace, so the new policy no
			// longer routes the item in flight.
			require.NoError(t, d.configure(stuckScheduledInputs("other"), config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "change the policy")
			// The worker counts the cut item when it finishes the item.
			assert.Eventually(t, func() bool { return d.metrics.dropped[dropStale].Get() == 1 }, cancelBound, time.Millisecond,
				"a policy that no longer routes the item must release the worker blocked on the full queue")

			d.stop()
			pipe.close(t)
			<-d.done
			dropped := int64(d.metrics.droppedRows[dropStale].Get())
			assert.Equal(t, uint64(1), d.metrics.dropped[dropStale].Get(), "count the cut item once")
			assert.Zero(t, d.metrics.droppedRows[dropShutdown].Get(), "a policy change is not a shutdown")
			assert.Equal(t, int64(stuckRows+1), pipe.output.delivered.Load()+dropped,
				"each row and the response is either delivered or counted once as a stale drop, never both")
		})
	}
}

func TestScheduledPolicyChangeKeepsRoutedPublish(t *testing.T) {
	pipe := newStuckPipeline(t, stuckQueues[0].cfg)
	d, plugin, result := stuckScheduledDispatcher(t, pipe)
	d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	pipe.waitBlocked(t)

	require.NoError(t, d.configure(stuckScheduledInputs("default"), config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "change the policy")
	time.Sleep(cancelBound)
	assert.False(t, pipe.idle(), "a policy that still routes the item must not cancel its publish")

	pipe.output.unblock()
	assert.Eventually(t, func() bool { return pipe.output.delivered.Load() == stuckRows+1 }, 5*time.Second, time.Millisecond,
		"every row and the response must be delivered once the output recovers")
	assert.Zero(t, d.metrics.droppedRows[dropStale].Get(), "no rows may be cut")

	d.stop()
	pipe.close(t)
	<-d.done
}

// TestScheduledWorkerWithoutContextPublisherBlocks checks the fallback: with
// clients that do not implement beat.ContextPublisher, shutdown leaves the
// worker in Publish until the output frees room.
func TestScheduledWorkerWithoutContextPublisherBlocks(t *testing.T) {
	pipe := newStuckPipeline(t, stuckQueues[0].cfg)
	pipe.plain = true
	d, plugin, result := stuckScheduledDispatcher(t, pipe)
	d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	pipe.waitBlocked(t)

	d.stop()
	select {
	case <-d.done:
		t.Error("a blocking Publish cannot be cancelled, so the worker must still wait")
	case <-time.After(cancelBound):
	}

	pipe.close(t)
	select {
	case <-d.done:
	case <-time.After(5 * time.Second):
		t.Error("the worker must return once its Publish returns")
	}
}

func TestLiveTimeoutCancelsBlockedPublish(t *testing.T) {
	for _, q := range stuckQueues {
		t.Run(q.name, func(t *testing.T) {
			log := logp.NewNopLogger()
			pipe := newStuckPipeline(t, q.cfg)
			c := &liveClients{newPublisher: func() liveQueryPublisher { return pub.New(&beat.Beat{Publisher: pipe}, log) }}
			require.NoError(t, c.Configure([]config.InputConfig{{Datastream: config.DatastreamConfig{Dataset: config.DefaultDataset}}}), "configure live clients")
			d := newLiveResultDispatcher(monitoring.NewRegistry(), log, 200*time.Millisecond)
			hits := make([]map[string]any, stuckRows)
			for i := range hits {
				hits[i] = map[string]any{"row": i}
			}

			n, err := d.publishResults(t.Context(), c, liveJob{actionID: "a1", hits: hits})
			assert.ErrorIs(t, err, errLivePublishTimeout, "the stuck output must time out the action")
			assert.Positive(t, n, "rows queued before the queue filled are handed over")
			assert.Equal(t, uint64(stuckRows-n), d.metrics.droppedRows[dropTimeout].Get(), "count the rows not handed to the output")
			assert.Eventually(t, pipe.idle, cancelBound, time.Millisecond,
				"the timeout must release the worker blocked on the full queue")

			c.Close()
			d.stop()
			pipe.close(t)
			assert.Equal(t, int64(n), pipe.output.delivered.Load(), "only the rows handed over before the timeout are delivered, none late")
		})
	}
}

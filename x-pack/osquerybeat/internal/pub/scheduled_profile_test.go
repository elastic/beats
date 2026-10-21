// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func scheduledProfileInputs(namespace string) []config.InputConfig {
	return []config.InputConfig{
		inputWithNamespace(config.DefaultQueryProfileDataset, "profiles"),
		inputWithNamespace(config.DefaultActionResponsesDataset, namespace),
		inputWithNamespace(config.DefaultDataset, namespace),
	}
}

func TestScheduledProfileSinkPreservesSchemaAndDeliveryModes(t *testing.T) {
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(func() { closeScheduledProfilePublisher(t, p) })
	require.NoError(t, p.Configure(scheduledProfileInputs("results")), "configure streams by dataset rather than input order")
	require.Len(t, recorder.configs, 4, "scheduled profiling should own a separate client")
	for i := range 3 {
		assert.Equal(t, beat.DefaultGuarantees, recorder.configs[i].PublishMode, "result, action, and live/RRULE profile delivery must stay unchanged")
	}
	assert.Equal(t, beat.DropIfFull, recorder.configs[3].PublishMode, "only scheduled profiles should request best-effort queue admission")
	assert.Equal(t, "profiles", recorder.namespaceOf(t, recorder.clients[3]), "the scheduled sink should use the profile input's processors")
	assert.Nil(t, p.CaptureScheduledProfileSink("other"), "an incompatible result namespace must not capture a sink")
	sink := p.CaptureScheduledProfileSink("results")
	require.NotNil(t, sink, "a compatible result namespace should capture its profile destination")
	profile := map[string]any{"source": "scheduled", "executions": int64(3), "duration": int64(12)}
	require.True(t, sink.Publish("uptime", "schedule", "response", "space", 1, profile), "the active lease should publish the collected profile")
	require.Eventually(t, func() bool { return len(recorder.clients[3].snapshot()) == 1 }, time.Second, time.Millisecond,
		"the independent publisher should publish one scheduled profile event")
	event := recorder.clients[3].snapshot()[0]
	assert.Equal(t, queryProfileToEvent("uptime", "", "response", "space", profile, nil, "schedule", 1), event.Fields, "scheduled profiles must preserve the existing event schema and correlation fields")
	assert.Equal(t, config.QueryProfileDatastream("results"), event.Meta[events.FieldMetaRawIndex], "the explicit profile index must preserve the result namespace")
	require.NotNil(t, recorder.configs[3].ClientListener, "output admission loss should be observable")
	recorder.configs[3].ClientListener.DroppedOnPublish(event)
	assert.Equal(t, uint64(1), p.scheduledProfileDropped.Get(), "count profiles dropped on a full output queue")
}

func TestScheduledProfileSinkDoesNotWaitForResultPublisherMutex(t *testing.T) {
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(func() { closeScheduledProfilePublisher(t, p) })
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the separate scheduled profile client")
	p.mx.Lock()
	done := make(chan bool, 1)
	go func() {
		sink := p.CaptureScheduledProfileSink("default")
		done <- sink != nil && sink.Valid() && sink.Publish("query", "schedule", "response", "", 1, map[string]any{"source": "scheduled"})
	}()
	select {
	case ok := <-done:
		p.mx.Unlock()
		assert.True(t, ok, "profile capture and publication should be independent of a blocked result publication")
	case <-time.After(time.Second):
		p.mx.Unlock()
		t.Fatal("scheduled profiling waited for the result publisher mutex")
	}
}

func TestScheduledProfileSinkRetiresOnPolicyReplacement(t *testing.T) {
	for _, replacement := range []struct {
		name   string
		inputs []config.InputConfig
	}{
		{"same namespace", scheduledProfileInputs("old")},
		{"changed namespace", scheduledProfileInputs("new")},
		{"profile stream removed", []config.InputConfig{inputWithNamespace(config.DefaultDataset, "old")}},
		{"all inputs removed", nil},
	} {
		t.Run(replacement.name, func(t *testing.T) {
			p := New(&beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"))
			t.Cleanup(func() { closeScheduledProfilePublisher(t, p) })
			require.NoError(t, p.Configure(scheduledProfileInputs("old")), "configure the original policy")
			old := p.CaptureScheduledProfileSink("old")
			require.NotNil(t, old, "capture a lease before the policy changes")
			require.NoError(t, p.Configure(replacement.inputs), "replace the policy")
			assert.False(t, old.Valid(), "a queued profile must lose its original lease on every policy replacement")
			assert.False(t, old.Publish("query", "schedule", "response", "", 1, nil), "a retired lease must not publish even when its namespace is reused")
		})
	}
}

func TestScheduledProfileSinkRetiresOnPublisherClose(t *testing.T) {
	p := New(&beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"))
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the profile client")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the configured lease")
	p.Close()
	assert.False(t, sink.Valid(), "publisher shutdown must retire retained profile leases")
}

func TestScheduledProfileSinkRetiresBeforeWaitingForResultPublication(t *testing.T) {
	p := New(&beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(func() { closeScheduledProfilePublisher(t, p) })
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the original profile lease")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the original policy destination")
	p.mx.Lock()
	done := make(chan error, 1)
	go func() { done <- p.Configure(nil) }()
	retired := assert.Eventually(t, func() bool { return !sink.Valid() }, time.Second, time.Millisecond,
		"policy removal must retire delayed profiles before waiting for a blocked result publication")
	select {
	case err := <-done:
		t.Errorf("Configure should still wait for the result publisher mutex: %v", err)
	default:
	}
	p.mx.Unlock()
	if !retired {
		t.Fatal("the previous policy lease remained active while configuration was blocked")
	}
	select {
	case err := <-done:
		assert.NoError(t, err, "policy removal should complete when result publication releases the mutex")
	case <-time.After(time.Second):
		t.Fatal("Configure did not finish after releasing the result publisher mutex")
	}
}

type stalledScheduledProfileClient struct {
	entered, release chan struct{}
	closed           atomic.Bool
}

func (c *stalledScheduledProfileClient) Publish(beat.Event) {
	close(c.entered)
	<-c.release
}
func (c *stalledScheduledProfileClient) PublishAll(events []beat.Event) {
	for _, event := range events {
		c.Publish(event)
	}
}
func (c *stalledScheduledProfileClient) Close() error { c.closed.Store(true); return nil }

func TestScheduledProfileLeaseInvalidationDoesNotWaitForPublication(t *testing.T) {
	client := &stalledScheduledProfileClient{entered: make(chan struct{}), release: make(chan struct{})}
	worker := newScheduledProfilePublisher(monitoring.NewUint(monitoring.NewRegistry(), "dropped"))
	sink := &scheduledProfileSink{client: client, publisher: worker, namespace: "default"}
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(client.release) })
		sink.retire()
		worker.stop()
		select {
		case <-worker.done:
		case <-time.After(time.Second):
			t.Error("released publication worker did not stop")
		}
	})
	done := make(chan bool, 1)
	go func() { done <- sink.Publish("scheduled", "schedule", "response", "", 1, nil) }()
	select {
	case <-client.entered:
	case <-time.After(time.Second):
		t.Fatal("profile publication did not enter the stalled client")
	}
	retired := make(chan struct{})
	go func() { sink.retire(); close(retired) }()
	select {
	case <-retired:
		assert.False(t, sink.Valid(), "lease validity must not wait behind an active publication")
		assert.False(t, client.closed.Load(), "the active publisher should own final client closure")
	case <-time.After(time.Second):
		releaseOnce.Do(func() { close(client.release) })
		t.Fatal("lease invalidation waited for stalled publication")
	}
	releaseOnce.Do(func() { close(client.release) })
	select {
	case ok := <-done:
		assert.True(t, ok, "the original publication attempt entered the pipeline before retirement")
		require.Eventually(t, client.closed.Load, time.Second, time.Millisecond, "finishing the active publication should close the retired client")
	case <-time.After(time.Second):
		t.Fatal("the released publisher did not finish and close its retired client")
	}
}

func closeScheduledProfilePublisher(t *testing.T, p *Publisher) {
	t.Helper()
	p.Close()
	if p.scheduledProfileWorker != nil {
		select {
		case <-p.scheduledProfileWorker.done:
		case <-time.After(time.Second):
			t.Error("scheduled profile publisher did not stop after releasing the output")
		}
	}
}

func TestScheduledProfileBusyPublisherDropsAcrossPolicyChanges(t *testing.T) {
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the original policy")
	worker := p.scheduledProfileWorker
	sink, ok := p.CaptureScheduledProfileSink("default").(*scheduledProfileSink)
	require.True(t, ok, "capture the configured scheduled profile sink")
	client := &stalledScheduledProfileClient{entered: make(chan struct{}), release: make(chan struct{})}
	sink.client = client
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(client.release) })
		closeScheduledProfilePublisher(t, p)
	})

	require.True(t, sink.Publish("query", "schedule", "response", "", 1, nil), "offer the first profile")
	select {
	case <-client.entered:
	case <-time.After(time.Second):
		t.Fatal("publication did not enter the stalled client")
	}

	// Replacing policies while publication is stuck must reuse one worker and
	// leave ordinary result and action publishing independent of its mutex.
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for range 20 {
			if err := p.Configure(scheduledProfileInputs("default")); err != nil {
				t.Errorf("replace policy: %v", err)
				return
			}
			if p.scheduledProfileWorker != worker {
				t.Error("policy replacement created another publication worker")
				return
			}
			p.CaptureScheduledProfileSink("default").Publish("query", "schedule", "response", "", 1, nil)
			p.Publish("index", "id", "schedule_id", "response", "", "", "", "query", nil, []map[string]any{{"value": 1}}, nil, nil)
			p.PublishActionResult(map[string]any{}, map[string]any{})
		}
		p.Close()
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("policy replacement, regular publication, or shutdown waited for profiling")
	}
	assert.Equal(t, uint64(20), worker.dropped.Get(), "drop each new profile while the single worker is busy")
	assert.Len(t, recorder.clients[len(recorder.clients)-4].snapshot(), 1, "regular results must still publish")
	assert.Len(t, recorder.clients[len(recorder.clients)-3].snapshot(), 1, "action responses must still publish")
	assert.False(t, client.closed.Load(), "the original in-flight publication still owns its client")
	releaseOnce.Do(func() { close(client.release) })
	select {
	case <-worker.done:
	case <-time.After(time.Second):
		t.Fatal("publication worker did not exit after the client was released")
	}
	assert.True(t, client.closed.Load(), "the worker must close its retired client before exiting")
}

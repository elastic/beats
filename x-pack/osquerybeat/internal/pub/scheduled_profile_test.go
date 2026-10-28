// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func scheduledProfileInputs(namespace string) []config.InputConfig {
	return []config.InputConfig{
		inputWithNamespace(config.DefaultQueryProfileDataset, "profiles"),
		inputWithNamespace(config.DefaultActionResponsesDataset, namespace),
		inputWithNamespace(config.DefaultDataset, namespace),
	}
}

func TestScheduledProfileSinkPreservesSchemaAndDeliveryModes(t *testing.T) {
	// Configure streams by dataset so the test does not depend on input order.
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure(scheduledProfileInputs("results")), "configure streams by dataset rather than input order")

	// Only the scheduled profile client may use DropIfFull; the other three keep guaranteed delivery.
	require.Len(t, recorder.configs, 4, "scheduled profiling should own a separate client")
	for i := range 3 {
		assert.Equal(t, beat.DefaultGuarantees, recorder.configs[i].PublishMode, "result, action, and live/RRULE profile delivery must stay unchanged")
	}
	assert.Equal(t, beat.DropIfFull, recorder.configs[3].PublishMode, "only scheduled profiles should request best-effort queue admission")
	assert.Equal(t, "profiles", recorder.namespaceOf(t, recorder.clients[3]), "the scheduled sink should use the profile input's processors")

	// Capture a sink only for a result namespace that matches the profile stream.
	assert.Nil(t, p.CaptureScheduledProfileSink("other"), "an incompatible result namespace must not capture a sink")
	sink := p.CaptureScheduledProfileSink("results")
	require.NotNil(t, sink, "a compatible result namespace should capture its profile destination")

	// Publish one profile and check the event keeps the existing schema and index.
	profile := map[string]any{"source": "scheduled", "executions": int64(3), "duration": int64(12)}
	require.True(t, sink.Publish("uptime", "schedule", "response", "space", 7, profile), "the active lease should publish the collected profile")
	require.Len(t, recorder.clients[3].snapshot(), 1, "the sink should publish one scheduled profile event synchronously")
	event := recorder.clients[3].snapshot()[0]
	want := mapstr.M{
		"type":                     "osquery_profile",
		"event":                    map[string]any{"module": "osquery_manager"},
		"osquery_profile":          map[string]any{"source": "scheduled", "executions": int64(3), "duration": int64(12)},
		"query":                    map[string]any{"name": "uptime"},
		"response_id":              "response",
		"space_id":                 "space",
		"schedule_id":              "schedule",
		"schedule_execution_count": int64(7),
	}
	assert.Equal(t, want, event.Fields, "scheduled profiles must preserve the existing event schema and correlation fields")
	assert.Equal(t, config.QueryProfileDatastream("results"), event.Meta[events.FieldMetaRawIndex], "the explicit profile index must preserve the result namespace")

	// Simulate the queue dropping the event and check the drop is counted.
	require.NotNil(t, recorder.configs[3].ClientListener, "output admission loss should be observable")
	recorder.configs[3].ClientListener.DroppedOnPublish(event)
	assert.Equal(t, uint64(1), p.scheduledProfileDropped.Get(), "count profiles dropped on a full output queue")
}

func TestScheduledProfileSinkDoesNotWaitForResultPublisherMutex(t *testing.T) {
	// Configure the separate scheduled profile client.
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the separate scheduled profile client")

	// Hold the result publisher mutex and capture and publish from another goroutine.
	p.mx.Lock()
	done := make(chan bool, 1)
	go func() {
		sink := p.CaptureScheduledProfileSink("default")
		done <- sink != nil && sink.Valid() && sink.Publish("query", "schedule", "response", "", 1, map[string]any{"source": "scheduled"})
	}()

	// The goroutine must finish while the mutex is still held.
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
			// Capture a lease under the original policy.
			p := New(&beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"))
			t.Cleanup(p.Close)
			require.NoError(t, p.Configure(scheduledProfileInputs("old")), "configure the original policy")
			old := p.CaptureScheduledProfileSink("old")
			require.NotNil(t, old, "capture a lease before the policy changes")

			// Replace the policy and check the old lease is retired.
			require.NoError(t, p.Configure(replacement.inputs), "replace the policy")
			assert.False(t, old.Valid(), "a queued profile must lose its original lease on every policy replacement")
			assert.False(t, old.Publish("query", "schedule", "response", "", 1, nil), "a retired lease must not publish even when its namespace is reused")
		})
	}
}

func TestScheduledProfileSinkRetiresOnPublisherClose(t *testing.T) {
	// Capture a lease from a configured publisher.
	p := New(&beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"))
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the profile client")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the configured lease")

	// Closing the publisher must retire the lease.
	p.Close()
	assert.False(t, sink.Valid(), "publisher shutdown must retire retained profile leases")
}

// Stop closes the publisher from another goroutine, so the main loop can still
// apply a queued policy afterwards. That policy must not install a lease.
func TestScheduledProfileSinkNotInstalledAfterClose(t *testing.T) {
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	p.Close()
	assert.Error(t, p.Configure(scheduledProfileInputs("default")), "configuring the scheduled profile stream after Close must fail")
	assert.Nil(t, p.CaptureScheduledProfileSink("default"), "a closed publisher must not hand out a lease")
	require.Len(t, recorder.clients, 4, "Configure should have connected the scheduled profile client before failing")
	assert.True(t, recorder.clients[3].closed.Load(), "the rejected scheduled profile client must be closed")
}

func TestScheduledProfileSinkRetiresBeforeWaitingForResultPublication(t *testing.T) {
	// Capture a lease under the original policy.
	p := New(&beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the original profile lease")
	sink := p.CaptureScheduledProfileSink("default")
	require.NotNil(t, sink, "capture the original policy destination")

	// Block Configure on the result publisher mutex while it removes the policy.
	p.mx.Lock()
	done := make(chan error, 1)
	go func() { done <- p.Configure(nil) }()

	// The lease must retire even though Configure is still waiting for the mutex.
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

	// Releasing the mutex must let Configure finish.
	select {
	case err := <-done:
		assert.NoError(t, err, "policy removal should complete when result publication releases the mutex")
	case <-time.After(time.Second):
		t.Fatal("Configure did not finish after releasing the result publisher mutex")
	}
}

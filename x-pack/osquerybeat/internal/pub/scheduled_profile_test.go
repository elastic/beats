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
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure(scheduledProfileInputs("results")), "configure streams by dataset rather than input order")
	require.Len(t, recorder.configs, 4, "scheduled profiling should own a separate client")
	for i := range 3 {
		assert.Equal(t, beat.DefaultGuarantees, recorder.configs[i].PublishMode, "result, action, and live/RRULE profile delivery must stay unchanged")
	}
	assert.Equal(t, beat.DropIfFull, recorder.configs[3].PublishMode, "only scheduled profiles should use nonblocking admission")
	assert.Equal(t, "profiles", recorder.namespaceOf(t, recorder.clients[3]), "the scheduled sink should use the profile input's processors")
	assert.Nil(t, p.CaptureScheduledProfileSink("other"), "an incompatible result namespace must not capture a sink")
	sink := p.CaptureScheduledProfileSink("results")
	require.NotNil(t, sink, "a compatible result namespace should capture its profile destination")
	profile := map[string]any{"source": "scheduled", "executions": int64(3), "duration": int64(12)}
	require.True(t, sink.Publish("uptime", "response", "space", profile), "the active lease should publish the collected profile")
	require.Len(t, recorder.clients[3].events, 1, "publish one scheduled profile event")
	event := recorder.clients[3].events[0]
	assert.Equal(t, queryProfileToEvent("uptime", "", "response", "space", profile, nil), event.Fields, "scheduled profiles must preserve the existing event schema and correlation fields")
	assert.Equal(t, config.QueryProfileDatastream("results"), event.Meta[events.FieldMetaRawIndex], "the explicit profile index must preserve the result namespace")
	require.NotNil(t, recorder.configs[3].ClientListener, "output admission loss should be observable")
	recorder.configs[3].ClientListener.DroppedOnPublish(event)
	assert.Equal(t, uint64(1), p.scheduledProfileDropped.Get(), "count profiles dropped on a full output queue")
}

func TestScheduledProfileSinkDoesNotWaitForResultPublisherMutex(t *testing.T) {
	recorder := &connectRecorder{}
	p := New(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_profile"))
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure(scheduledProfileInputs("default")), "configure the separate scheduled profile client")
	p.mx.Lock()
	done := make(chan bool, 1)
	go func() {
		sink := p.CaptureScheduledProfileSink("default")
		done <- sink != nil && sink.Valid() && sink.Publish("query", "response", "", map[string]any{"source": "scheduled"})
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
			t.Cleanup(p.Close)
			require.NoError(t, p.Configure(scheduledProfileInputs("old")), "configure the original policy")
			old := p.CaptureScheduledProfileSink("old")
			require.NotNil(t, old, "capture a lease before the policy changes")
			require.NoError(t, p.Configure(replacement.inputs), "replace the policy")
			assert.False(t, old.Valid(), "a queued profile must lose its original lease on every policy replacement")
			assert.False(t, old.Publish("query", "response", "", nil), "a retired lease must not publish even when its namespace is reused")
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
	t.Cleanup(p.Close)
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

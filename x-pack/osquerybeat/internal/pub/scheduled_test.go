// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"sync/atomic"
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

func TestScheduledPublisherGuardAndDeliveryMode(t *testing.T) {
	recorder := &connectRecorder{}
	var guarded, dropped atomic.Int32
	var cut atomic.Int64
	p := NewScheduled(t.Context(), &beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled"),
		func() bool { return guarded.Add(1) == 1 }, func() { dropped.Add(1) }, func(rows int) { cut.Add(int64(rows)) })
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure([]config.InputConfig{
		inputWithNamespace(config.DefaultDataset, "default"),
		inputWithNamespace(config.DefaultQueryProfileDataset, "default"),
	}), "configure scheduled result and profile clients")
	require.Len(t, recorder.configs, 2, "scheduled clients should publish results and profiles")
	for i, cfg := range recorder.configs {
		assert.Equal(t, beat.DefaultGuarantees, cfg.PublishMode, "scheduled client %d must block for queue space", i)
		assert.NotNil(t, cfg.ClientListener, "scheduled client %d must report publication after close", i)
	}
	p.Publish("index", "schedule", "schedule_id", "response", "", "", "", "query", nil,
		[]map[string]any{{"value": 1}, {"value": 2}}, nil, nil)
	assert.Len(t, recorder.clients[0].events, 1, "retirement between rows must prevent further publication")
	p.PublishScheduledProfile("index", "query", "schedule", "response", "", 1, nil)
	assert.Empty(t, recorder.clients[1].events, "retirement must prevent profile publication")
	assert.Equal(t, int64(2), cut.Load(), "report the row and the profile skipped after retirement")
	recorder.configs[0].ClientListener.DroppedOnPublish(beat.Event{})
	assert.Equal(t, int32(1), dropped.Load(), "make publication after close observable")

	regular := &connectRecorder{}
	live := New(&beat.Beat{Publisher: regular}, logptest.NewTestingLogger(t, "live"))
	t.Cleanup(live.Close)
	require.NoError(t, live.Configure([]config.InputConfig{inputWithNamespace(config.DefaultDataset, "default")}), "configure the existing live-result client")
	assert.Equal(t, beat.DefaultGuarantees, regular.configs[0].PublishMode, "live query delivery semantics must remain unchanged")
}

func TestScheduledProfilePreservesSchema(t *testing.T) {
	// Configure streams by dataset so the test does not depend on input order.
	p := NewScheduled(t.Context(), &beat.Beat{Publisher: &connectRecorder{}}, logptest.NewTestingLogger(t, "scheduled_profile"),
		func() bool { return true }, func() {}, func(int) {})
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure([]config.InputConfig{
		inputWithNamespace(config.DefaultQueryProfileDataset, "profiles"),
		inputWithNamespace(config.DefaultDataset, "results"),
	}), "configure the scheduled result and profile clients")
	recorder, ok := p.b.Publisher.(*connectRecorder)
	require.True(t, ok, "the publisher should be the connect recorder")
	assert.Equal(t, "profiles", recorder.namespaceOf(t, p.queryProfileClient), "profiles should use the profile input's processors")

	// Publish one profile and check the event keeps the existing schema and index.
	profile := map[string]any{"source": "scheduled", "executions": int64(3), "duration": int64(12)}
	p.PublishScheduledProfile(config.QueryProfileDatastream("results"), "uptime", "schedule", "response", "space", 7, profile)
	profileClient, ok := p.queryProfileClient.(*recordingClient)
	require.True(t, ok, "the profile client should be a recording client")
	published := profileClient.snapshot()
	require.Len(t, published, 1, "publish one scheduled profile event")
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
	assert.Equal(t, want, published[0].Fields, "scheduled profiles must preserve the existing event schema and correlation fields")
	assert.Equal(t, config.QueryProfileDatastream("results"), published[0].Meta[events.FieldMetaRawIndex], "the explicit profile index must preserve the result namespace")
}

func TestScheduledPublisherReportsCutResponse(t *testing.T) {
	recorder := &connectRecorder{}
	var cut atomic.Int64
	p := NewScheduled(t.Context(), &beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_response"),
		func() bool { return false }, func() {}, func(rows int) { cut.Add(int64(rows)) })
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure([]config.InputConfig{
		inputWithNamespace(config.DefaultActionResponsesDataset, "default"),
	}), "configure the scheduled response client")

	p.PublishScheduledResponse("schedule", "", "", "query", "", "response", time.Now(), time.Now(), time.Now(), 0, 1)

	require.NotNil(t, p.actionResponsesClient, "connect the response client")
	responseClient, ok := p.actionResponsesClient.(*recordingClient)
	require.True(t, ok, "the response client should be a recording client")
	assert.Empty(t, responseClient.snapshot(), "a retired publisher must not publish the response")
	assert.Equal(t, int64(1), cut.Load(), "report the skipped response as one cut row")
}

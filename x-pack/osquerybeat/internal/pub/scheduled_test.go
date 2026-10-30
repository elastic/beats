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
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestScheduledPublisherGuardAndDeliveryMode(t *testing.T) {
	recorder := &connectRecorder{}
	var guarded, dropped atomic.Int32
	var cut atomic.Int64
	p := NewScheduled(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled"),
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
	assert.Equal(t, int64(1), cut.Load(), "report the row skipped after retirement")
	recorder.configs[0].ClientListener.DroppedOnPublish(beat.Event{})
	assert.Equal(t, int32(1), dropped.Load(), "make publication after close observable")

	regular := &connectRecorder{}
	live := New(&beat.Beat{Publisher: regular}, logptest.NewTestingLogger(t, "live"))
	t.Cleanup(live.Close)
	require.NoError(t, live.Configure([]config.InputConfig{inputWithNamespace(config.DefaultDataset, "default")}), "configure the existing live-result client")
	assert.Equal(t, beat.DefaultGuarantees, regular.configs[0].PublishMode, "live query delivery semantics must remain unchanged")
}

func TestScheduledPublisherReportsCutResponse(t *testing.T) {
	recorder := &connectRecorder{}
	var cut atomic.Int64
	p := NewScheduled(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled_response"),
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

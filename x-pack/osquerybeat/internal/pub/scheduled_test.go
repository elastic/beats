// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestScheduledPublisherGuardAndDeliveryMode(t *testing.T) {
	recorder := &connectRecorder{}
	var guarded, dropped atomic.Int32
	p := NewScheduled(&beat.Beat{Publisher: recorder}, logptest.NewTestingLogger(t, "scheduled"),
		func() bool { return guarded.Add(1) == 1 }, func() { dropped.Add(1) })
	t.Cleanup(p.Close)
	require.NoError(t, p.Configure([]config.InputConfig{inputWithNamespace(config.DefaultDataset, "default")}), "configure a scheduled result client")
	require.Len(t, recorder.configs, 1, "only the configured result stream should connect")
	assert.Equal(t, beat.DropIfFull, recorder.configs[0].PublishMode, "native scheduled clients must request best-effort admission")
	p.Publish("index", "schedule", "schedule_id", "response", "", "", "", "query", nil,
		[]map[string]any{{"value": 1}, {"value": 2}}, nil, nil)
	assert.Len(t, recorder.clients[0].events, 1, "retirement between rows must prevent further publication")
	recorder.configs[0].ClientListener.DroppedOnPublish(beat.Event{})
	assert.Equal(t, int32(1), dropped.Load(), "make admission losses observable")

	regular := &connectRecorder{}
	live := New(&beat.Beat{Publisher: regular}, logptest.NewTestingLogger(t, "live"))
	t.Cleanup(live.Close)
	require.NoError(t, live.Configure([]config.InputConfig{inputWithNamespace(config.DefaultDataset, "default")}), "configure the existing live-result client")
	assert.Equal(t, beat.DefaultGuarantees, regular.configs[0].PublishMode, "live query delivery semantics must remain unchanged")
}

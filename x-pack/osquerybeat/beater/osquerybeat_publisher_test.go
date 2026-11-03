// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type mockBeatPublisher struct {
	closeCalled bool
}

var _ osquerybeatPublisher = (*mockBeatPublisher)(nil)
var _ scheduledQueryPublisher = (*mockBeatPublisher)(nil)

func (m *mockBeatPublisher) Publish(index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName string, meta map[string]any, hits []map[string]any, ecsm ecs.Mapping, reqData any) {
}

func (m *mockBeatPublisher) PublishEach(index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName string, meta map[string]any, hits []map[string]any, ecsm ecs.Mapping, reqData any, stop func() bool) int {
	return 0
}

func (m *mockBeatPublisher) PublishActionResult(req map[string]any, res map[string]any) {
}

func (m *mockBeatPublisher) PublishScheduledResponse(scheduleID, packID, packName, queryName, spaceID, responseID string, startedAt, completedAt, plannedScheduleTime time.Time, resultCount int, scheduleExecutionCount int64) {
}

func (m *mockBeatPublisher) PublishQueryProfile(index, queryName, actionID, responseID, spaceID string, profile map[string]any, reqData any) {
}

func (m *mockBeatPublisher) PublishScheduledProfile(index, queryName, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any) {
}

func (m *mockBeatPublisher) Configure(inputs []config.InputConfig) error {
	return nil
}

func (m *mockBeatPublisher) Close() {
	m.closeCalled = true
}

func TestConfigurePublishersSetsLiveTimeout(t *testing.T) {
	c, err := conf.NewConfigWithYAML([]byte(`
data_stream.dataset: osquery_manager.result
osquery.elastic_options.publishing.live.timeout: 7s
`), "policy")
	require.NoError(t, err, "parse the policy")
	var input config.InputConfig
	require.NoError(t, c.Unpack(&input), "unpack the policy")

	log := logptest.NewTestingLogger(t, "live-timeout")
	bt := &osquerybeat{log: log, pub: &mockBeatPublisher{}, liveResults: newLiveResultDispatcher(nil, log, config.DefaultLivePublishTimeout)}
	require.NoError(t, bt.configurePublishers([]config.InputConfig{input}), "configure the publishers")
	assert.Equal(t, 7*time.Second, time.Duration(bt.liveResults.timeout.Load()), "the live dispatcher must use publishing.live.timeout")

	require.NoError(t, bt.configurePublishers(nil), "remove the inputs")
	assert.Equal(t, 7*time.Second, time.Duration(bt.liveResults.timeout.Load()), "removing the inputs must keep the live timeout")
}

func TestOsquerybeatClose_ClosesPublisher(t *testing.T) {
	p := &mockBeatPublisher{}
	cancelCalled := false

	bt := &osquerybeat{
		pub: p,
		cancel: func() {
			cancelCalled = true
		},
	}

	bt.close()

	assert.True(t, p.closeCalled)
	assert.True(t, cancelCalled)
}

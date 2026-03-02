// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
)

type mockBeatPublisher struct {
	closeCalled bool
}

var _ osquerybeatPublisher = (*mockBeatPublisher)(nil)
var _ scheduledQueryPublisher = (*mockBeatPublisher)(nil)

func (m *mockBeatPublisher) Publish(index, idValue, idFieldKey, responseID string, meta map[string]interface{}, hits []map[string]interface{}, ecsm ecs.Mapping, reqData interface{}) {
}

func (m *mockBeatPublisher) PublishActionResult(req map[string]interface{}, res map[string]interface{}) {
}

func (m *mockBeatPublisher) PublishScheduledResponse(scheduleID, responseID string, startedAt, completedAt, plannedScheduleTime time.Time, resultCount int, scheduleExecutionCount int64) {
}

func (m *mockBeatPublisher) Configure(inputs []config.InputConfig) error {
	return nil
}

func (m *mockBeatPublisher) Close() {
	m.closeCalled = true
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
	assert.Nil(t, bt.cancel)
}

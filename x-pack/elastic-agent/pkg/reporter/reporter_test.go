// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reporter

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

var result Event

type testReporter struct{}

func (t *testReporter) Close() error { return nil }
func (t *testReporter) Report(_ context.Context, r Event) error {
	result = r
	return nil
}

type info struct{}

func (*info) AgentID() string { return "id" }

type testScenario struct {
	Status        state.Status
	StatusMessage string
	EventType     string
	EventSubType  string
	EventMessage  string
}

func TestTypes(t *testing.T) {
	rep := NewReporter(context.Background(), nil, &info{}, &testReporter{})
	scenarios := []testScenario{
		{
			Status:        state.Stopped,
			StatusMessage: "Stopped",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeStopped,
			EventMessage:  "Application: a-stopped[id]: State changed to STOPPED: Stopped",
		},
		{
			Status:        state.Starting,
			StatusMessage: "Starting",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeStarting,
			EventMessage:  "Application: a-starting[id]: State changed to STARTING: Starting",
		},
		{
			Status:        state.Configuring,
			StatusMessage: "Configuring",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeConfig,
			EventMessage:  "Application: a-configuring[id]: State changed to CONFIG: Configuring",
		},
		{
			Status:        state.Healthy,
			StatusMessage: "Healthy",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeRunning,
			EventMessage:  "Application: a-healthy[id]: State changed to RUNNING: Healthy",
		},
		{
			Status:        state.Degraded,
			StatusMessage: "Degraded",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeRunning,
			EventMessage:  "Application: a-degraded[id]: State changed to DEGRADED: Degraded",
		},
		{
			Status:        state.Failed,
			StatusMessage: "Failed",
			EventType:     EventTypeError,
			EventSubType:  EventSubTypeFailed,
			EventMessage:  "Application: a-failed[id]: State changed to FAILED: Failed",
		},
		{
			Status:        state.Crashed,
			StatusMessage: "Crashed",
			EventType:     EventTypeError,
			EventSubType:  EventSubTypeFailed,
			EventMessage:  "Application: a-crashed[id]: State changed to CRASHED: Crashed",
		},
		{
			Status:        state.Stopping,
			StatusMessage: "Stopping",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeStopping,
			EventMessage:  "Application: a-stopping[id]: State changed to STOPPING: Stopping",
		},
		{
			Status:        state.Restarting,
			StatusMessage: "Restarting",
			EventType:     EventTypeState,
			EventSubType:  EventSubTypeStarting,
			EventMessage:  "Application: a-restarting[id]: State changed to RESTARTING: Restarting",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.StatusMessage, func(t *testing.T) {
			appID := fmt.Sprintf("a-%s", strings.ToLower(scenario.StatusMessage))
			appName := fmt.Sprintf("app-%s", strings.ToLower(scenario.StatusMessage))
			rep.OnStateChange(appID, appName, state.State{
				Status:  scenario.Status,
				Message: scenario.StatusMessage,
			})
			assert.Equal(t, scenario.EventType, result.Type())
			assert.Equal(t, scenario.EventSubType, result.SubType())
			assert.Equal(t, scenario.EventMessage, result.Message())
		})
	}
}

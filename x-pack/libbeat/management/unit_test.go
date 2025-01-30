// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/libbeat/management/status"
)

func TestUnitUpdate(t *testing.T) {

	type StatusUpdate struct {
		status status.Status
		msg    string
	}

	const (
		Healthy  = "Healthy"
		Failed   = "Failed"
		Degraded = "Degraded"
	)

	unitCfg := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "inputLevelState-1",
				Streams: []*proto.Stream{
					{Id: "stream-1"},
					{Id: "stream-2"},
				},
			},
		},
	}

	cases := []struct {
		name               string
		unit               *mockClientUnit
		inputLevelStatus   StatusUpdate
		streamStates       map[string]StatusUpdate
		expectedUnitStatus client.UnitState
		expectedUnitMsg    string
	}{
		{
			name:             "all running",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name:             "inputLevelState failed",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Failed, Failed},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name:             "inputLevelState stopping",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Stopping, ""},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateStopping,
			expectedUnitMsg:    "",
		},
		{
			name:             "inputLevelState configuring",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Configuring, ""},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateConfiguring,
			expectedUnitMsg:    "",
		},
		{
			name:             "inputLevelState starting",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Starting, ""},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateStarting,
			expectedUnitMsg:    "",
		},
		{
			name:             "inputLevelState degraded",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Degraded, Degraded},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name:             "one stream failed the other running",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Failed, Failed},
				"stream-2": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name:             "one stream failed the other degraded",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Failed, Failed},
				"stream-2": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name:             "one stream running the other degraded",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Running, Healthy},
				"stream-2": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name:             "both streams degraded",
			unit:             unitCfg,
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-1": {status.Degraded, Degraded},
				"stream-2": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			aUnit := newAgentUnit(c.unit, nil)
			err := aUnit.UpdateState(c.inputLevelStatus.status, c.inputLevelStatus.msg, nil)
			if err != nil {
				t.Fatal(err)
			}

			for id, state := range c.streamStates {
				aUnit.updateStateForStream(id, state.status, state.msg)
			}

			if c.unit.reportedState != c.expectedUnitStatus {
				t.Errorf("expected unit status %s, got %s", c.expectedUnitStatus, aUnit.inputLevelState.state)
			}

			if c.unit.reportedMsg != c.expectedUnitMsg {
				t.Errorf("expected unit msg %s, got %s", c.expectedUnitStatus, aUnit.inputLevelState.state)
			}
		})
	}
}

type mockClientUnit struct {
	expected      client.Expected
	reportedState client.UnitState
	reportedMsg   string
}

func (u *mockClientUnit) Expected() client.Expected {
	return u.expected
}

func (u *mockClientUnit) UpdateState(state client.UnitState, msg string, _ map[string]interface{}) error {
	u.reportedState = state
	u.reportedMsg = msg
	return nil
}

func (u *mockClientUnit) ID() string {
	return "inputLevelState-1"
}

func (u *mockClientUnit) Type() client.UnitType {
	return client.UnitTypeInput
}

func (u *mockClientUnit) RegisterAction(_ client.Action) {
}

func (u *mockClientUnit) UnregisterAction(_ client.Action) {
}

func (u *mockClientUnit) RegisterDiagnosticHook(_ string, _ string, _ string, _ string, _ client.DiagnosticHook) {
}

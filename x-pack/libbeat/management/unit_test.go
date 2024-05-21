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
	unitCfg := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-1",
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
		inputStatus        status.Status
		streamStates       map[string]status.Status
		expectedUnitStatus client.UnitState
	}{
		{
			name:        "all running",
			unit:        unitCfg,
			inputStatus: status.Running,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateHealthy,
		},
		{
			name:        "input failed",
			unit:        unitCfg,
			inputStatus: status.Failed,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateFailed,
		},
		{
			name:        "input stopping",
			unit:        unitCfg,
			inputStatus: status.Stopping,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateStopping,
		},
		{
			name:        "input configuring",
			unit:        unitCfg,
			inputStatus: status.Configuring,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateConfiguring,
		},
		{
			name:        "input starting",
			unit:        unitCfg,
			inputStatus: status.Starting,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateStarting,
		},
		{
			name:        "input degraded",
			unit:        unitCfg,
			inputStatus: status.Degraded,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateDegraded,
		},
		{
			name:        "one stream failed the other running",
			unit:        unitCfg,
			inputStatus: status.Running,
			streamStates: map[string]status.Status{
				"stream-1": status.Failed,
				"stream-2": status.Running,
			},
			expectedUnitStatus: client.UnitStateFailed,
		},
		{
			name:        "one stream failed the other degraded",
			unit:        unitCfg,
			inputStatus: status.Running,
			streamStates: map[string]status.Status{
				"stream-1": status.Failed,
				"stream-2": status.Degraded,
			},
			expectedUnitStatus: client.UnitStateFailed,
		},
		{
			name:        "one stream running the other degraded",
			unit:        unitCfg,
			inputStatus: status.Running,
			streamStates: map[string]status.Status{
				"stream-1": status.Running,
				"stream-2": status.Degraded,
			},
			expectedUnitStatus: client.UnitStateDegraded,
		},
		{
			name:        "both streams degraded",
			unit:        unitCfg,
			inputStatus: status.Running,
			streamStates: map[string]status.Status{
				"stream-1": status.Degraded,
				"stream-2": status.Degraded,
			},
			expectedUnitStatus: client.UnitStateDegraded,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			aUnit := newAgentUnit(c.unit, nil)
			err := aUnit.UpdateState(c.inputStatus, "", nil)
			if err != nil {
				t.Fatal(err)
			}

			for id, state := range c.streamStates {
				aUnit.updateStateForStream(id, state, "")
			}

			if c.unit.reportedState != c.expectedUnitStatus {
				t.Errorf("expected unit status %s, got %s", c.expectedUnitStatus, aUnit.input.state)
			}
		})
	}
}

type mockClientUnit struct {
	expected      client.Expected
	reportedState client.UnitState
}

func (u *mockClientUnit) Expected() client.Expected {
	return u.expected
}

func (u *mockClientUnit) UpdateState(state client.UnitState, _ string, _ map[string]interface{}) error {
	u.reportedState = state
	return nil
}

func (u *mockClientUnit) ID() string {
	return "input-1"
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

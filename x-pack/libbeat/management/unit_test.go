// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
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

func TestUnitUpdatePayload(t *testing.T) {
	mock := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-1",
			},
		},
	}
	unit := newAgentUnit(mock, nil)

	payloadA := map[string]any{
		"telemetry": map[string]any{"jobs": 1},
	}
	payloadB := map[string]any{
		"telemetry": map[string]any{"jobs": 2},
	}

	err := unit.UpdateState(status.Running, "Healthy", payloadA)
	assert.NoError(t, err, "first UpdateState should succeed")

	err = unit.UpdateState(status.Running, "Healthy", payloadB)
	assert.NoError(t, err, "second UpdateState with different payload should succeed")

	assert.Equal(t, 2, mock.updateCalls, "different payload with same status should forward a second update")
	assert.Equal(t, payloadB, mock.reportedPayload, "client should receive the latest payload")

	payloadBEqual := map[string]any{
		"telemetry": map[string]any{"jobs": 2},
	}
	err = unit.UpdateState(status.Running, "Healthy", payloadBEqual)
	assert.NoError(t, err, "third UpdateState with equal payload should succeed")

	assert.Equal(t, 2, mock.updateCalls, "equal payload with same status should be suppressed")

	nested := map[string]any{"jobs": 1}
	payloadWithNested := map[string]any{"telemetry": nested}
	mockMutation := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-mutation",
			},
		},
	}
	mutationUnit := newAgentUnit(mockMutation, nil)

	err = mutationUnit.UpdateState(status.Running, "Healthy", payloadWithNested)
	assert.NoError(t, err, "initial UpdateState should succeed")

	nested["jobs"] = 999

	payloadUnchanged := map[string]any{
		"telemetry": map[string]any{"jobs": 1},
	}
	err = mutationUnit.UpdateState(status.Running, "Healthy", payloadUnchanged)
	assert.NoError(t, err, "UpdateState with original payload values should succeed")

	assert.Equal(t, 1, mockMutation.updateCalls, "mutating caller map must not alter stored comparison snapshot")
}

func TestUnitUpdatePayloadStreamsSerializable(t *testing.T) {
	mock := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-with-streams",
				Streams: []*proto.Stream{
					{Id: "stream-1"},
				},
			},
		},
	}
	unit := newAgentUnit(mock, nil)

	payload := map[string]any{
		"telemetry": map[string]any{"jobs": 3},
	}

	err := unit.UpdateState(status.Running, "Healthy", payload)
	assert.NoError(t, err, "UpdateState with streams should succeed")

	assert.Equal(t, 1, mock.updateCalls, "first UpdateState should forward to client")

	reportedTelemetry, ok := mock.reportedPayload["telemetry"].(map[string]any)
	assert.True(t, ok, "telemetry should be forwarded as map[string]any")
	assert.Equal(t, 3, reportedTelemetry["jobs"], "telemetry values should be preserved")

	streams, ok := mock.reportedPayload["streams"].(map[string]any)
	assert.True(t, ok, "streams should be present in forwarded payload")
	assert.Contains(t, streams, "stream-1", "stream state should be included")

	_, err = structpb.NewStruct(mock.reportedPayload)
	assert.NoError(t, err, "forwarded payload must be serializable by structpb.NewStruct")
}

func TestStreamUpdateRetainsInputPayload(t *testing.T) {
	mock := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-with-stream",
				Streams: []*proto.Stream{
					{Id: "stream-1"},
				},
			},
		},
	}
	unit := newAgentUnit(mock, nil)
	scheduler := map[string]any{"jobs": 3}
	payload := map[string]any{
		"heartbeat": map[string]any{
			"scheduler": scheduler,
		},
	}

	err := unit.UpdateState(status.Running, "Healthy", payload)
	assert.NoError(t, err, "input-level UpdateState should succeed")

	scheduler["jobs"] = 999
	unit.updateStateForStream("stream-1", status.Running, "Healthy")

	assert.Equal(t, 2, mock.updateCalls, "stream update should forward a second payload")
	heartbeat, ok := mock.reportedPayload["heartbeat"].(map[string]any)
	assert.True(t, ok, "heartbeat telemetry should remain map[string]any")
	reportedScheduler, ok := heartbeat["scheduler"].(map[string]any)
	assert.True(t, ok, "scheduler telemetry should remain map[string]any")
	assert.Equal(t, 3, reportedScheduler["jobs"], "stream update should retain the snapshotted scheduler value")

	streams, ok := mock.reportedPayload["streams"].(map[string]any)
	assert.True(t, ok, "stream update should include streams")
	assert.Equal(t, map[string]any{
		"status": client.UnitStateHealthy.String(),
		"error":  "Healthy",
	}, streams["stream-1"], "stream update should include the latest stream state")

	_, err = structpb.NewStruct(mock.reportedPayload)
	assert.NoError(t, err, "merged stream payload must be serializable by structpb.NewStruct")
}

type mockClientUnit struct {
	id       string
	unitType client.UnitType
	expected client.Expected

	reportedState   client.UnitState
	reportedMsg     string
	reportedPayload map[string]any
	updateCalls     int
}

func (u *mockClientUnit) Expected() client.Expected {
	return u.expected
}

func (u *mockClientUnit) UpdateState(
	state client.UnitState,
	msg string,
	payload map[string]any,
) error {
	u.reportedState = state
	u.reportedMsg = msg
	u.reportedPayload = payload
	u.updateCalls++
	return nil
}

func (u *mockClientUnit) ID() string {
	if u.id != "" {
		return u.id
	}

	return "inputLevelState-1"
}

func (u *mockClientUnit) Type() client.UnitType {
	return u.unitType
}

func (u *mockClientUnit) RegisterAction(_ client.Action) {
}

func (u *mockClientUnit) UnregisterAction(_ client.Action) {
}

func (u *mockClientUnit) RegisterDiagnosticHook(_ string, _ string, _ string, _ string, _ client.DiagnosticHook) {
}

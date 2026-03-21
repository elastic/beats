// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

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

func TestUnitUpdateSuppressHealthDegradation(t *testing.T) {

	type StatusUpdate struct {
		status status.Status
		msg    string
	}

	const (
		Healthy  = "Healthy"
		Failed   = "Failed"
		Degraded = "Degraded"
	)

	suppressedSource, _ := structpb.NewStruct(map[string]interface{}{
		"suppress_health_degradation": true,
	})

	notSuppressedSource, _ := structpb.NewStruct(map[string]interface{}{
		"suppress_health_degradation": false,
	})

	newUnit := func(streams []*proto.Stream) *mockClientUnit {
		return &mockClientUnit{
			expected: client.Expected{
				Config: &proto.UnitExpectedConfig{
					Id:      "input-1",
					Streams: streams,
				},
			},
		}
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
			name: "suppressed stream degraded does not affect unit health",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-suppressed", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required":   {status.Running, Healthy},
				"stream-suppressed": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "suppressed stream failed does not affect unit health",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-suppressed", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required":   {status.Running, Healthy},
				"stream-suppressed": {status.Failed, Failed},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "required stream degraded still affects unit health even with suppressed streams",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-suppressed", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required":   {status.Degraded, Degraded},
				"stream-suppressed": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "required stream failed still affects unit health even with suppressed streams",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-suppressed", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required":   {status.Failed, Failed},
				"stream-suppressed": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name: "all suppressed streams degraded and failed keeps unit healthy",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-suppressed-1", Source: suppressedSource},
				{Id: "stream-suppressed-2", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-suppressed-1": {status.Degraded, Degraded},
				"stream-suppressed-2": {status.Failed, Failed},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "suppress false behaves same as not set",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-explicit-false", Source: notSuppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-explicit-false": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "input level degraded is not affected by suppress flag",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-suppressed", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Degraded, Degraded},
			streamStates: map[string]StatusUpdate{
				"stream-suppressed": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "input level failed is not affected by suppress flag",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-suppressed", Source: suppressedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Failed, Failed},
			streamStates: map[string]StatusUpdate{
				"stream-suppressed": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
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
				t.Errorf("expected unit status %s, got %s", c.expectedUnitStatus, c.unit.reportedState)
			}

			if c.unit.reportedMsg != c.expectedUnitMsg {
				t.Errorf("expected unit msg %q, got %q", c.expectedUnitMsg, c.unit.reportedMsg)
			}
		})
	}
}

func TestGetStreamStatesParsesSuppress(t *testing.T) {
	suppressedSource, _ := structpb.NewStruct(map[string]interface{}{
		"suppress_health_degradation": true,
	})
	notSuppressedSource, _ := structpb.NewStruct(map[string]interface{}{
		"suppress_health_degradation": false,
	})

	expected := client.Expected{
		Config: &proto.UnitExpectedConfig{
			Id: "input-1",
			Streams: []*proto.Stream{
				{Id: "stream-plain"},
				{Id: "stream-suppressed", Source: suppressedSource},
				{Id: "stream-not-suppressed", Source: notSuppressedSource},
			},
		},
	}

	states, ids := getStreamStates(expected)

	if len(states) != 3 {
		t.Fatalf("expected 3 stream states, got %d", len(states))
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 stream IDs, got %d", len(ids))
	}
	if states["stream-plain"].suppressHealthDegradation {
		t.Error("stream without Source should not have suppress set")
	}
	if !states["stream-suppressed"].suppressHealthDegradation {
		t.Error("stream with suppress_health_degradation: true should have suppress set")
	}
	if states["stream-not-suppressed"].suppressHealthDegradation {
		t.Error("stream with suppress_health_degradation: false should not have suppress set")
	}
}

func TestSuppressFlipRecomputesHealth(t *testing.T) {
	// Simulates what update() does when a policy change flips the suppress
	// flag on a stream that is already Degraded. This verifies the
	// recompute-on-suppression-change path in update().

	cu := &mockClientUnitWithPayload{
		mockClientUnit: mockClientUnit{
			expected: client.Expected{
				Config: &proto.UnitExpectedConfig{
					Id: "input-1",
					Streams: []*proto.Stream{
						{Id: "stream-a"},
					},
				},
			},
		},
	}

	aUnit := newAgentUnit(cu, nil)
	_ = aUnit.UpdateState(status.Running, "Healthy", nil)

	// Stream degrades — unit should be Degraded (suppress not set)
	aUnit.updateStateForStream("stream-a", status.Degraded, "connection refused")
	if cu.reportedState != client.UnitStateDegraded {
		t.Fatalf("expected Degraded before flip, got %s", cu.reportedState)
	}

	// Simulate a policy update that adds suppress_health_degradation: true.
	// This is exactly what update() does: flip the flag, then recompute.
	aUnit.mtx.Lock()
	existing := aUnit.streamStates["stream-a"]
	existing.suppressHealthDegradation = true
	aUnit.streamStates["stream-a"] = existing

	state, msg := aUnit.calcState()
	streamsPayload := make(map[string]interface{}, len(aUnit.streamStates))
	for id, ss := range aUnit.streamStates {
		streamsPayload[id] = map[string]interface{}{
			"status": getUnitState(ss.state).String(),
			"error":  ss.msg,
		}
	}
	_ = aUnit.clientUnit.UpdateState(getUnitState(state), msg, map[string]interface{}{"streams": streamsPayload})
	aUnit.mtx.Unlock()

	// Unit should now be Healthy — the suppression flip took effect
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected Healthy after suppress flip, got %s", cu.reportedState)
	}

	// Per-stream payload should still show Degraded
	streams, ok := cu.reportedPayload["streams"].(map[string]interface{})
	if !ok {
		t.Fatal("expected streams payload")
	}
	streamStatus, ok := streams["stream-a"].(map[string]interface{})
	if !ok {
		t.Fatal("expected stream-a payload")
	}
	if streamStatus["status"] != client.UnitStateDegraded.String() {
		t.Errorf("expected per-stream Degraded after flip, got %q", streamStatus["status"])
	}

	// Now flip suppress back to false — unit should go Degraded again
	aUnit.mtx.Lock()
	existing = aUnit.streamStates["stream-a"]
	existing.suppressHealthDegradation = false
	aUnit.streamStates["stream-a"] = existing

	state, msg = aUnit.calcState()
	streamsPayload = make(map[string]interface{}, len(aUnit.streamStates))
	for id, ss := range aUnit.streamStates {
		streamsPayload[id] = map[string]interface{}{
			"status": getUnitState(ss.state).String(),
			"error":  ss.msg,
		}
	}
	_ = aUnit.clientUnit.UpdateState(getUnitState(state), msg, map[string]interface{}{"streams": streamsPayload})
	aUnit.mtx.Unlock()

	if cu.reportedState != client.UnitStateDegraded {
		t.Errorf("expected Degraded after unsuppress flip, got %s", cu.reportedState)
	}
}

func TestSuppressedStreamRecovery(t *testing.T) {
	suppressedSource, _ := structpb.NewStruct(map[string]interface{}{
		"suppress_health_degradation": true,
	})

	cu := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-1",
				Streams: []*proto.Stream{
					{Id: "stream-suppressed", Source: suppressedSource},
				},
			},
		},
	}

	aUnit := newAgentUnit(cu, nil)
	_ = aUnit.UpdateState(status.Running, "Healthy", nil)

	// Stream goes degraded — unit should stay healthy
	aUnit.updateStateForStream("stream-suppressed", status.Degraded, "connection refused")
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected Healthy during degradation, got %s", cu.reportedState)
	}

	// Stream recovers — unit should still be healthy
	aUnit.updateStateForStream("stream-suppressed", status.Running, "Healthy")
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected Healthy after recovery, got %s", cu.reportedState)
	}
}

func TestSuppressedStreamStillReportsPerStreamStatus(t *testing.T) {
	suppressedSource, _ := structpb.NewStruct(map[string]interface{}{
		"suppress_health_degradation": true,
	})

	cu := &mockClientUnitWithPayload{
		mockClientUnit: mockClientUnit{
			expected: client.Expected{
				Config: &proto.UnitExpectedConfig{
					Id: "input-1",
					Streams: []*proto.Stream{
						{Id: "stream-suppressed", Source: suppressedSource},
					},
				},
			},
		},
	}

	aUnit := newAgentUnit(cu, nil)
	_ = aUnit.UpdateState(status.Running, "Healthy", nil)
	aUnit.updateStateForStream("stream-suppressed", status.Degraded, "connection refused")

	// Unit should be healthy (suppressed)
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected unit Healthy, got %s", cu.reportedState)
	}

	// But the per-stream payload should still show Degraded
	streams, ok := cu.reportedPayload["streams"].(map[string]interface{})
	if !ok {
		t.Fatal("expected streams in payload")
	}
	streamStatus, ok := streams["stream-suppressed"].(map[string]interface{})
	if !ok {
		t.Fatal("expected stream-suppressed in streams payload")
	}
	if streamStatus["status"] != client.UnitStateDegraded.String() {
		t.Errorf("expected per-stream status %q, got %q", client.UnitStateDegraded.String(), streamStatus["status"])
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

// mockClientUnitWithPayload extends mockClientUnit to capture the payload
type mockClientUnitWithPayload struct {
	mockClientUnit
	reportedPayload map[string]interface{}
}

func (u *mockClientUnitWithPayload) UpdateState(state client.UnitState, msg string, payload map[string]interface{}) error {
	u.reportedState = state
	u.reportedMsg = msg
	u.reportedPayload = payload
	return nil
}

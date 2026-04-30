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
				t.Errorf("expected unit status %s, got %s", c.expectedUnitStatus, c.unit.reportedState)
			}

			if c.unit.reportedMsg != c.expectedUnitMsg {
				t.Errorf("expected unit msg %q, got %q", c.expectedUnitMsg, c.unit.reportedMsg)
			}
		})
	}
}

func TestUnitUpdateStatusReporting(t *testing.T) {

	type StatusUpdate struct {
		status status.Status
		msg    string
	}

	const (
		Healthy  = "Healthy"
		Failed   = "Failed"
		Degraded = "Degraded"
	)

	// status_reporting with both report_degraded and report_failed set to false
	muteBothSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
			"report_failed":   false,
		},
	})

	// status_reporting with only report_degraded muted
	muteDegradedSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
		},
	})

	// status_reporting with only report_failed muted
	muteFailedSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_failed": false,
		},
	})

	// status_reporting with both explicitly true (same as not set)
	explicitTrueSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": true,
			"report_failed":   true,
		},
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
			name: "muted stream degraded does not affect unit health",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-muted", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required": {status.Running, Healthy},
				"stream-muted":    {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "muted stream failed does not affect unit health",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-muted", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required": {status.Running, Healthy},
				"stream-muted":    {status.Failed, Failed},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "required stream degraded still affects unit health even with muted streams",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-muted", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required": {status.Degraded, Degraded},
				"stream-muted":    {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "required stream failed still affects unit health even with muted streams",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-required"},
				{Id: "stream-muted", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-required": {status.Failed, Failed},
				"stream-muted":    {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name: "all muted streams degraded and failed keeps unit healthy",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-muted-1", Source: muteBothSource},
				{Id: "stream-muted-2", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-muted-1": {status.Degraded, Degraded},
				"stream-muted-2": {status.Failed, Failed},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "explicit true behaves same as not set",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-explicit-true", Source: explicitTrueSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-explicit-true": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "input level degraded is not affected by status_reporting",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-muted", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Degraded, Degraded},
			streamStates: map[string]StatusUpdate{
				"stream-muted": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "input level failed is not affected by status_reporting",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-muted", Source: muteBothSource},
			}),
			inputLevelStatus: StatusUpdate{status.Failed, Failed},
			streamStates: map[string]StatusUpdate{
				"stream-muted": {status.Running, Healthy},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name: "mute degraded only still reports failed",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-a", Source: muteDegradedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-a": {status.Failed, Failed},
			},
			expectedUnitStatus: client.UnitStateFailed,
			expectedUnitMsg:    Failed,
		},
		{
			name: "mute degraded only suppresses degraded",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-a", Source: muteDegradedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-a": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
		},
		{
			name: "mute failed only still reports degraded",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-a", Source: muteFailedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-a": {status.Degraded, Degraded},
			},
			expectedUnitStatus: client.UnitStateDegraded,
			expectedUnitMsg:    Degraded,
		},
		{
			name: "mute failed only suppresses failed",
			unit: newUnit([]*proto.Stream{
				{Id: "stream-a", Source: muteFailedSource},
			}),
			inputLevelStatus: StatusUpdate{status.Running, Healthy},
			streamStates: map[string]StatusUpdate{
				"stream-a": {status.Failed, Failed},
			},
			expectedUnitStatus: client.UnitStateHealthy,
			expectedUnitMsg:    Healthy,
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

func TestGetStreamStatesParsesStatusReporting(t *testing.T) {
	muteBothSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
			"report_failed":   false,
		},
	})
	muteDegradedSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
		},
	})
	explicitTrueSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": true,
			"report_failed":   true,
		},
	})

	expected := client.Expected{
		Config: &proto.UnitExpectedConfig{
			Id: "input-1",
			Streams: []*proto.Stream{
				{Id: "stream-plain"},
				{Id: "stream-mute-both", Source: muteBothSource},
				{Id: "stream-mute-degraded", Source: muteDegradedSource},
				{Id: "stream-explicit-true", Source: explicitTrueSource},
			},
		},
	}

	states, ids := getStreamStates(expected)

	if len(states) != 4 {
		t.Fatalf("expected 4 stream states, got %d", len(states))
	}
	if len(ids) != 4 {
		t.Fatalf("expected 4 stream IDs, got %d", len(ids))
	}

	// No source — both default to true
	if !states["stream-plain"].statusReporting.reportDegraded {
		t.Error("stream without Source should default reportDegraded to true")
	}
	if !states["stream-plain"].statusReporting.reportFailed {
		t.Error("stream without Source should default reportFailed to true")
	}

	// Both muted
	if states["stream-mute-both"].statusReporting.reportDegraded {
		t.Error("stream with report_degraded: false should have reportDegraded false")
	}
	if states["stream-mute-both"].statusReporting.reportFailed {
		t.Error("stream with report_failed: false should have reportFailed false")
	}

	// Only degraded muted, failed defaults to true
	if states["stream-mute-degraded"].statusReporting.reportDegraded {
		t.Error("stream with report_degraded: false should have reportDegraded false")
	}
	if !states["stream-mute-degraded"].statusReporting.reportFailed {
		t.Error("stream without report_failed should default reportFailed to true")
	}

	// Explicit true — same as defaults
	if !states["stream-explicit-true"].statusReporting.reportDegraded {
		t.Error("stream with report_degraded: true should have reportDegraded true")
	}
	if !states["stream-explicit-true"].statusReporting.reportFailed {
		t.Error("stream with report_failed: true should have reportFailed true")
	}
}

func TestGetStreamStatesInputLevelInheritance(t *testing.T) {
	inputSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
			"report_failed":   false,
		},
	})
	streamOverrideSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": true,
		},
	})

	expected := client.Expected{
		Config: &proto.UnitExpectedConfig{
			Id:     "input-1",
			Source: inputSource,
			Streams: []*proto.Stream{
				{Id: "stream-inherit"},
				{Id: "stream-override", Source: streamOverrideSource},
			},
		},
	}

	states, _ := getStreamStates(expected)

	// stream-inherit has no Source, should inherit input-level defaults
	if states["stream-inherit"].statusReporting.reportDegraded {
		t.Error("stream without Source should inherit input-level reportDegraded=false")
	}
	if states["stream-inherit"].statusReporting.reportFailed {
		t.Error("stream without Source should inherit input-level reportFailed=false")
	}

	// stream-override sets report_degraded=true, report_failed falls back to input-level false
	if !states["stream-override"].statusReporting.reportDegraded {
		t.Error("stream with report_degraded: true should override input-level value")
	}
	if states["stream-override"].statusReporting.reportFailed {
		t.Error("stream without report_failed should inherit input-level reportFailed=false")
	}
}

func TestStatusReportingFlipRecomputesHealth(t *testing.T) {
	// Simulates what update() does when a policy change flips the
	// status_reporting flags on a stream that is already Degraded.

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

	// Stream degrades — unit should be Degraded (defaults: both reported)
	aUnit.updateStateForStream("stream-a", status.Degraded, "connection refused")
	if cu.reportedState != client.UnitStateDegraded {
		t.Fatalf("expected Degraded before flip, got %s", cu.reportedState)
	}

	// Simulate a policy update that sets report_degraded: false.
	aUnit.mtx.Lock()
	existing := aUnit.streamStates["stream-a"]
	existing.statusReporting.reportDegraded = false
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

	// Unit should now be Healthy
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected Healthy after muting degraded, got %s", cu.reportedState)
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

	// Flip report_degraded back to true — unit should go Degraded again
	aUnit.mtx.Lock()
	existing = aUnit.streamStates["stream-a"]
	existing.statusReporting.reportDegraded = true
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
		t.Errorf("expected Degraded after re-enabling reporting, got %s", cu.reportedState)
	}
}

func TestMutedStreamRecovery(t *testing.T) {
	mutedSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
			"report_failed":   false,
		},
	})

	cu := &mockClientUnit{
		expected: client.Expected{
			Config: &proto.UnitExpectedConfig{
				Id: "input-1",
				Streams: []*proto.Stream{
					{Id: "stream-muted", Source: mutedSource},
				},
			},
		},
	}

	aUnit := newAgentUnit(cu, nil)
	_ = aUnit.UpdateState(status.Running, "Healthy", nil)

	// Stream goes degraded — unit should stay healthy
	aUnit.updateStateForStream("stream-muted", status.Degraded, "connection refused")
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected Healthy during degradation, got %s", cu.reportedState)
	}

	// Stream recovers — unit should still be healthy
	aUnit.updateStateForStream("stream-muted", status.Running, "Healthy")
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected Healthy after recovery, got %s", cu.reportedState)
	}
}

func TestMutedStreamStillReportsPerStreamStatus(t *testing.T) {
	mutedSource, _ := structpb.NewStruct(map[string]interface{}{
		"status_reporting": map[string]interface{}{
			"report_degraded": false,
			"report_failed":   false,
		},
	})

	cu := &mockClientUnitWithPayload{
		mockClientUnit: mockClientUnit{
			expected: client.Expected{
				Config: &proto.UnitExpectedConfig{
					Id: "input-1",
					Streams: []*proto.Stream{
						{Id: "stream-muted", Source: mutedSource},
					},
				},
			},
		},
	}

	aUnit := newAgentUnit(cu, nil)
	_ = aUnit.UpdateState(status.Running, "Healthy", nil)
	aUnit.updateStateForStream("stream-muted", status.Degraded, "connection refused")

	// Unit should be healthy (muted)
	if cu.reportedState != client.UnitStateHealthy {
		t.Errorf("expected unit Healthy, got %s", cu.reportedState)
	}

	// But the per-stream payload should still show Degraded
	streams, ok := cu.reportedPayload["streams"].(map[string]interface{})
	if !ok {
		t.Fatal("expected streams in payload")
	}
	streamStatus, ok := streams["stream-muted"].(map[string]interface{})
	if !ok {
		t.Fatal("expected stream-muted in streams payload")
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

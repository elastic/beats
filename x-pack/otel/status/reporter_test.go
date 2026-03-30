// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

func TestGroupStatus(t *testing.T) {
	m := &oteltest.MockHost{}
	reporter := NewGroupStatusReporter(m)

	subReporter1, subReporter2, subReporter3 := reporter.GetReporterForRunner("1"), reporter.GetReporterForRunner("2"), reporter.GetReporterForRunner("3")

	subReporter1.UpdateStatus(status.Running, "")
	subReporter2.UpdateStatus(status.Running, "")
	subReporter3.UpdateStatus(status.Running, "")

	require.Equalf(t, componentstatus.StatusOK, m.Evt.Status(), "expected StatusOK, got %v", m.Evt.Status())
	require.NoErrorf(t, m.Evt.Err(), "expected nil, got %v")

	subReporter1.UpdateStatus(status.Degraded, "Degrade Runner1")
	require.Equalf(t, componentstatus.StatusRecoverableError, m.Evt.Status(), "expected StatusDegraded, got %v", m.Evt.Status())
	require.Error(t, m.Evt.Err(), "expected non-nil error, got nil")
	require.Equalf(t, "Degrade Runner1", m.Evt.Err().Error(), "expected 'Degrade Runner1', got %v", m.Evt.Err())

	subReporter3.UpdateStatus(status.Degraded, "Degrade Runner3")
	subReporter2.UpdateStatus(status.Failed, "Failed Runner2")

	require.Equalf(t, componentstatus.StatusPermanentError, m.Evt.Status(), "expected StatusPermanentError, got %v", m.Evt.Status())
	require.Error(t, m.Evt.Err(), "expected non-nil error, got nil")
	require.Equalf(t, "Failed Runner2", m.Evt.Err().Error(), "expected 'Failed Runner1', got %v", m.Evt.Err())

	// group reporter is updated directly
	reporter.UpdateStatus(status.Failed, "beatreceiver failed to start")

	require.Equalf(t, componentstatus.StatusPermanentError, m.Evt.Status(), "expected StatusPermanentError, got %v", m.Evt.Status())
	require.Error(t, m.Evt.Err(), "expected non-nil error, got nil")
	require.Equalf(t, "beatreceiver failed to start", m.Evt.Err().Error(), "expected 'beatreceiver failed to start', got %v", m.Evt.Err())
}

func TestToPdata(t *testing.T) {
	tests := []struct {
		name     string
		state    status.Status
		msg      string
		wantKeys map[string]string
	}{
		{
			name:  "running state with no message",
			state: status.Running,
			msg:   "",
			wantKeys: map[string]string{
				"status": componentstatus.StatusOK.String(),
				"error":  "",
			},
		},
		{
			name:  "degraded state with message",
			state: status.Degraded,
			msg:   "some error occurred",
			wantKeys: map[string]string{
				"status": componentstatus.StatusRecoverableError.String(),
				"error":  "some error occurred",
			},
		},
		{
			name:  "failed state with message",
			state: status.Failed,
			msg:   "critical failure",
			wantKeys: map[string]string{
				"status": componentstatus.StatusPermanentError.String(),
				"error":  "critical failure",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &runnerState{
				state: tt.state,
				msg:   tt.msg,
			}
			result := toPdata(rs)

			for key, expectedVal := range tt.wantKeys {
				val, ok := result.Get(key)
				require.True(t, ok, "expected key %q to exist", key)
				require.Equal(t, expectedVal, val.Str(), "expected %q=%q, got %q", key, expectedVal, val.Str())
			}
		})
	}
}

func TestGetOppositeStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  componentstatus.Status
		expect componentstatus.Status
	}{
		{
			name:   "OK returns RecoverableError",
			input:  componentstatus.StatusOK,
			expect: componentstatus.StatusRecoverableError,
		},
		{
			name:   "RecoverableError returns OK",
			input:  componentstatus.StatusRecoverableError,
			expect: componentstatus.StatusOK,
		},
		{
			name:   "Starting returns None",
			input:  componentstatus.StatusStarting,
			expect: componentstatus.StatusNone,
		},
		{
			name:   "Stopped returns None",
			input:  componentstatus.StatusStopped,
			expect: componentstatus.StatusNone,
		},
		{
			name:   "PermanentError returns None",
			input:  componentstatus.StatusPermanentError,
			expect: componentstatus.StatusNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOppositeStatus(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestInputStatusesInEventAttributes(t *testing.T) {
	m := &oteltest.MockHost{}
	reporter := NewGroupStatusReporter(m)

	subReporter1 := reporter.GetReporterForRunner("runner-1")
	subReporter2 := reporter.GetReporterForRunner("runner-2")

	subReporter1.UpdateStatus(status.Running, "")
	subReporter2.UpdateStatus(status.Degraded, "some warning")

	require.NotNil(t, m.Evt)

	// Verify inputs attribute exists
	attrs := m.Evt.Attributes()
	inputsVal, ok := attrs.Get(inputStatusAttributesKey)
	require.True(t, ok, "expected 'inputs' attribute to exist")

	inputsMap := inputsVal.Map()

	// Check runner-1 status
	runner1Val, ok := inputsMap.Get("runner-1")
	require.True(t, ok, "expected 'runner-1' to exist in inputs")
	runner1Map := runner1Val.Map()

	runner1Status, ok := runner1Map.Get("status")
	require.True(t, ok)
	assert.Equal(t, componentstatus.StatusOK.String(), runner1Status.Str())

	runner1Error, ok := runner1Map.Get("error")
	require.True(t, ok)
	assert.Empty(t, runner1Error.Str())

	// Check runner-2 status
	runner2Val, ok := inputsMap.Get("runner-2")
	require.True(t, ok, "expected 'runner-2' to exist in inputs")
	runner2Map := runner2Val.Map()

	runner2Status, ok := runner2Map.Get("status")
	require.True(t, ok)
	assert.Equal(t, componentstatus.StatusRecoverableError.String(), runner2Status.Str())

	runner2Error, ok := runner2Map.Get("error")
	require.True(t, ok)
	assert.Equal(t, "some warning", runner2Error.Str())
}

func TestDummyStatusEmission(t *testing.T) {
	// Test that the reporter emits a dummy status before the actual status
	// to force the OTel core to process the change.
	// We verify this by checking that when transitioning between OK and RecoverableError,
	// the opposite status is emitted first.

	m := &statusHistoryHost{}
	reporter := NewGroupStatusReporter(m)

	subReporter1 := reporter.GetReporterForRunner("runner-1")

	// First update: Running -> StatusOK
	subReporter1.UpdateStatus(status.Running, "")

	// The first update should emit a dummy RecoverableError before OK
	require.Len(t, m.history, 2, "expected 2 events (dummy + real)")
	assert.Equal(t, componentstatus.StatusRecoverableError, m.history[0].Status(), "first event should be dummy RecoverableError")
	assert.Equal(t, componentstatus.StatusOK, m.history[1].Status(), "second event should be OK")

	// Clear history
	m.history = nil

	// Second update: Degraded -> StatusRecoverableError
	subReporter1.UpdateStatus(status.Degraded, "degraded message")

	// Should emit dummy OK before RecoverableError
	require.Len(t, m.history, 2, "expected 2 events (dummy + real)")
	assert.Equal(t, componentstatus.StatusOK, m.history[0].Status(), "first event should be dummy OK")
	assert.Equal(t, componentstatus.StatusRecoverableError, m.history[1].Status(), "second event should be RecoverableError")

	// Clear history
	m.history = nil

	// Third update: Failed -> StatusPermanentError (no opposite exists for PermanentError)
	subReporter1.UpdateStatus(status.Failed, "failed message")

	// Should only emit one event since there's no opposite for PermanentError
	require.Len(t, m.history, 1, "expected 1 event (no dummy for PermanentError)")
	assert.Equal(t, componentstatus.StatusPermanentError, m.history[0].Status())
}

// statusHistoryHost is a mock host that records all status events
type statusHistoryHost struct {
	history []*componentstatus.Event
}

func (*statusHistoryHost) GetExtensions() map[component.ID]component.Component {
	return nil
}

func (h *statusHistoryHost) Report(evt *componentstatus.Event) {
	h.history = append(h.history, evt)
}

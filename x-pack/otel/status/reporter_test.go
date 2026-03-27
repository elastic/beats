// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

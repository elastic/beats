// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pdata/pcommon"

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

	require.Equalf(t, componentstatus.StatusOK, m.GetEvent().Status(), "expected StatusOK, got %v", m.GetEvent().Status())
	require.NoErrorf(t, m.GetEvent().Err(), "expected nil, got %v", m.GetEvent().Err())

	subReporter1.UpdateStatus(status.Degraded, "Degrade Runner1")
	require.Equalf(t, componentstatus.StatusRecoverableError, m.GetEvent().Status(), "expected StatusDegraded, got %v", m.GetEvent().Status())
	require.Error(t, m.GetEvent().Err(), "expected non-nil error, got nil")
	require.Equalf(t, "Degrade Runner1", m.GetEvent().Err().Error(), "expected 'Degrade Runner1', got %v", m.GetEvent().Err())

	subReporter3.UpdateStatus(status.Degraded, "Degrade Runner3")
	subReporter2.UpdateStatus(status.Failed, "Failed Runner2")

	require.Equalf(t, componentstatus.StatusPermanentError, m.GetEvent().Status(), "expected StatusPermanentError, got %v", m.GetEvent().Status())
	require.Error(t, m.GetEvent().Err(), "expected non-nil error, got nil")
	require.Equalf(t, "Failed Runner2", m.GetEvent().Err().Error(), "expected 'Failed Runner1', got %v", m.GetEvent().Err())

	// group reporter is updated directly
	reporter.UpdateStatus(status.Failed, "beatreceiver failed to start")

	require.Equalf(t, componentstatus.StatusPermanentError, m.GetEvent().Status(), "expected StatusPermanentError, got %v", m.GetEvent().Status())
	require.Error(t, m.GetEvent().Err(), "expected non-nil error, got nil")
	require.Equalf(t, "beatreceiver failed to start", m.GetEvent().Err().Error(), "expected 'beatreceiver failed to start', got %v", m.GetEvent().Err())
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

	require.NotNil(t, m.GetEvent())

	// Verify inputs attribute exists
	attrs := m.GetEvent().Attributes()
	inputsVal, ok := attrs.Get(InputStatusAttributesKey)
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

// TestGroupStatusHasNoInputStatuses checks that a status reported for the
// whole group carries no input statuses, even once runners reported theirs.
// The Heartbeat receiver relies on this to tell a failed monitor from a failed
// receiver.
func TestGroupStatusHasNoInputStatuses(t *testing.T) {
	m := &oteltest.MockHost{}
	reporter := NewGroupStatusReporter(m)

	reporter.GetReporterForRunner("runner-1").UpdateStatus(status.Failed, "runner failed")
	require.NotNil(t, m.GetEvent(), "expected the runner status to be reported")
	_, ok := m.GetEvent().Attributes().Get(InputStatusAttributesKey)
	require.True(t, ok, "expected the runner status event to carry input statuses")

	reporter.UpdateStatus(status.Failed, "beatreceiver failed to start")
	require.NotNil(t, m.GetEvent(), "expected the group status to be reported")
	assert.Equal(t, componentstatus.StatusPermanentError, m.GetEvent().Status(), "expected the group failure to be permanent")
	_, ok = m.GetEvent().Attributes().Get(InputStatusAttributesKey)
	assert.False(t, ok, "expected the group status event to carry no input statuses")
}

func TestRemoveRunner(t *testing.T) {
	remove := func(t *testing.T, r status.StatusReporter) {
		remover, ok := r.(runnerRemover)
		require.True(t, ok, "expected the runner reporter to remove its runner")
		remover.remove()
	}
	inputStatuses := func(t *testing.T, evt *componentstatus.Event) pcommon.Map {
		inputs, ok := evt.Attributes().Get(InputStatusAttributesKey)
		require.True(t, ok, "expected the event to carry input statuses")
		return inputs.Map()
	}

	t.Run("removes the state and reports the group again", func(t *testing.T) {
		m := &oteltest.MockHost{}
		reporter := NewGroupStatusReporter(m)
		healthy := reporter.GetReporterForRunner("healthy")
		failed := reporter.GetReporterForRunner("failed")
		healthy.UpdateStatus(status.Running, "")
		failed.UpdateStatus(status.Failed, "invalid config")
		require.Equal(t, componentstatus.StatusPermanentError, m.GetEvent().Status(), "a failed runner fails the group")

		remove(t, failed)

		evt := m.GetEvent()
		assert.Equal(t, componentstatus.StatusOK, evt.Status(), "a removed runner must not keep the group failed")
		inputs := inputStatuses(t, evt)
		_, ok := inputs.Get("failed")
		assert.False(t, ok, "a removed runner must not be reported")
		_, ok = inputs.Get("healthy")
		assert.True(t, ok, "other runners must still be reported")
	})

	t.Run("ignores updates of a removed runner", func(t *testing.T) {
		m := &oteltest.MockHost{}
		reporter := NewGroupStatusReporter(m)
		runner := reporter.GetReporterForRunner("runner")
		runner.UpdateStatus(status.Running, "")
		remove(t, runner)
		before := len(m.GetEvents())

		runner.UpdateStatus(status.Failed, "late failure")

		assert.Len(t, m.GetEvents(), before, "a removed runner must not report")
	})

	t.Run("keeps the state of a replacement with the same ID", func(t *testing.T) {
		m := &oteltest.MockHost{}
		reporter := NewGroupStatusReporter(m)
		old := reporter.GetReporterForRunner("runner")
		old.UpdateStatus(status.Running, "")
		replacement := reporter.GetReporterForRunner("runner")
		replacement.UpdateStatus(status.Degraded, "slow")
		before := len(m.GetEvents())

		remove(t, old)

		assert.Len(t, m.GetEvents(), before, "removing a replaced runner must not report")
		replacement.UpdateStatus(status.Degraded, "still slow")
		_, ok := inputStatuses(t, m.GetEvent()).Get("runner")
		assert.True(t, ok, "the replacement must still be reported")
	})

	t.Run("reports nothing for a runner that never reported", func(t *testing.T) {
		m := &oteltest.MockHost{}
		reporter := NewGroupStatusReporter(m)

		remove(t, reporter.GetReporterForRunner("runner"))

		assert.Empty(t, m.GetEvents(), "no report expected")
	})
}

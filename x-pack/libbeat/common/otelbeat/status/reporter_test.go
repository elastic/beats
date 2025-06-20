// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"
)

func TestGroupStatus(t *testing.T) {
	m := &oteltest.MockHost{}
	reporter := NewGroupStatusReporter(m)

	subReporter1, subReporter2, subReporter3 := reporter.GetReporterForRunner(1), reporter.GetReporterForRunner(2), reporter.GetReporterForRunner(3)

	subReporter1.UpdateStatus(status.Running, "")
	subReporter2.UpdateStatus(status.Running, "")
	subReporter3.UpdateStatus(status.Running, "")

	require.Equalf(t, componentstatus.StatusOK, m.Evt.Status(), "expected StatusOK, got %v", m.Evt.Status())
	require.Nilf(t, m.Evt.Err(), "expected nil, got %v")

	subReporter1.UpdateStatus(status.Degraded, "Degrade Runner1")
	require.Equalf(t, m.Evt.Status(), componentstatus.StatusRecoverableError, "expected StatusDegraded, got %v", m.Evt.Status())
	require.NotNil(t, m.Evt.Err(), "expected non-nil error, got nil")
	require.Equalf(t, m.Evt.Err().Error(), "Degrade Runner1", "expected 'Degrade Runner1', got %v", m.Evt.Err())

	subReporter3.UpdateStatus(status.Degraded, "Degrade Runner3")
	subReporter2.UpdateStatus(status.Failed, "Failed Runner2")

	require.Equalf(t, m.Evt.Status(), componentstatus.StatusPermanentError, "expected StatusPermanentError, got %v", m.Evt.Status())
	require.NotNil(t, m.Evt.Err(), "expected non-nil error, got nil")
	require.Equalf(t, m.Evt.Err().Error(), "Failed Runner2", "expected 'Failed Runner1', got %v", m.Evt.Err())
}

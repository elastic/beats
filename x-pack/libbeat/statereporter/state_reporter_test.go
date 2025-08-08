// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package statereporter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestConstructor(t *testing.T) {
	t.Run("Nil context reporter results in reporter with debug reporter", func(t *testing.T) {
		reporter := New(nil, logp.NewNopLogger())
		require.NotNil(t, reporter)

		require.IsType(t, &EnhancedStatusReporter{}, reporter)
		require.IsType(t, &debugStatusReporter{}, reporter.statusReporter)
	})
}

func TestStateReporterStatus(t *testing.T) {
	mockReporter := &countedReporter{}
	reporter := New(mockReporter, logp.NewNopLogger())
	require.IsType(t, &countedReporter{}, reporter.statusReporter)

	// check initials
	require.Equal(t, status.Unknown, reporter.current)
	require.Equal(t, 0, mockReporter.count)

	// update status multiple times
	reporter.UpdateStatus(status.Running, "some message")
	reporter.UpdateStatus(status.Running, "some message")

	// check for proxying only the necessary
	require.Equal(t, status.Running, reporter.current)
	require.Equal(t, 1, mockReporter.count)

	// check for change of status
	reporter.UpdateStatus(status.Stopped, "")
	require.Equal(t, status.Stopped, reporter.current)
	require.Equal(t, 2, mockReporter.count)
}

// countedReporter helps with testing to track proxying count
type countedReporter struct {
	count int
}

func (c *countedReporter) UpdateStatus(status status.Status, msg string) {
	c.count++
}

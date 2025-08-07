// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

func Test_constructor(t *testing.T) {
	t.Run("Nil context reporter results in reporter with debug reporter", func(t *testing.T) {
		reporter := newCWStateReporter(v2.Context{}, logp.NewNopLogger())
		require.NotNil(t, reporter)

		require.IsType(t, &cwStateReporter{}, reporter)
		require.IsType(t, &debugCWStatusReporter{}, reporter.reporter)
	})
}

func Test_cwStateReporterStatus(t *testing.T) {
	reporter := newCWStateReporter(v2.Context{StatusReporter: &countedReporter{}}, logp.NewNopLogger())
	require.IsType(t, &countedReporter{}, reporter.reporter)

	counterReporter, ok := reporter.reporter.(*countedReporter)
	require.True(t, ok)

	// check initials
	require.Equal(t, status.Unknown, reporter.current)
	require.Equal(t, 0, counterReporter.count)

	// update status multiple times
	reporter.UpdateStatus(status.Running, "some message")
	reporter.UpdateStatus(status.Running, "some message")

	// check for proxying only the necessary
	require.Equal(t, status.Running, reporter.current)
	require.Equal(t, 1, counterReporter.count)

	// check for change of status
	reporter.UpdateStatus(status.Stopped, "")
	require.Equal(t, status.Stopped, reporter.current)
	require.Equal(t, 2, counterReporter.count)
}

// countedReporter helps with testing to track proxying count
type countedReporter struct {
	count int
}

func (c *countedReporter) UpdateStatus(status status.Status, msg string) {
	c.count++
}

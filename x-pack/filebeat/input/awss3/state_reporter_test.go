// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// TODO: move this to a centralized pkg since it will be used by 2 inputs (awscloudwatch and awss3)
package awss3

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

}

// countedReporter helps with testing to track proxying count
type countedReporter struct {
	count int
}

func (c *countedReporter) UpdateStatus(status status.Status, msg string) {
	c.count++
}

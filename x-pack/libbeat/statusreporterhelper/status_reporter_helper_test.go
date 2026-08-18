// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package statusreporterhelper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestConstructor(t *testing.T) {
	t.Run("Nil context reporter results in reporter with debug reporter", func(t *testing.T) {
		reporter := New(nil, logp.NewNopLogger(), "test")
		require.NotNil(t, reporter)

		require.IsType(t, &StatusReporterHelper{}, reporter)
		require.IsType(t, &debugStatusReporter{}, reporter.statusReporter)
	})
}

func TestReportingWithReporter(t *testing.T) {
	mockReporter := &countedReporter{}
	unusedLogger, unusedLoggerBuffer := logp.NewInMemoryLocal("not_used", logp.ConsoleEncoderConfig())
	reporter := New(mockReporter, unusedLogger, "test")
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

	// assert the passed-in logger was not used when statusresporter is non-nil
	require.Equal(t, 0, unusedLoggerBuffer.Len())
}

func TestReportingWithLoggerPassthrough(t *testing.T) {
	bufferedLogger, buff := logp.NewInMemoryLocal("in_memory_passthrough_logger", logp.ConsoleEncoderConfig())
	reporter := New(nil, bufferedLogger, "test_passthrough_input_name")
	require.IsType(t, &debugStatusReporter{}, reporter.statusReporter)

	// check initial state
	require.Equal(t, 0, buff.Len())

	// update status multiple times
	reporter.UpdateStatus(status.Running, "some message")
	reporter.UpdateStatus(status.Running, "some message")

	// check for proxying only the necessary
	require.Equal(t, status.Running, reporter.current)
	loggerOutput := buff.String()
	require.Equal(t, 1, strings.Count(loggerOutput, "some message"))
	require.Equal(t, 1, strings.Count(loggerOutput, "test_passthrough_input_name input status updated: status:"))

	// check for change of status
	reporter.UpdateStatus(status.Stopped, "")
	require.Equal(t, status.Stopped, reporter.current)
}

// countedReporter helps with testing to track proxying count
type countedReporter struct {
	count int
}

func (c *countedReporter) UpdateStatus(status status.Status, msg string) {
	c.count++
}

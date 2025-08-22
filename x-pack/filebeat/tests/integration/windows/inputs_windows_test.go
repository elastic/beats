// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build win_integration && windows

package windows

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	fbint "github.com/elastic/beats/v7/filebeat/testing/integration"
	lbint "github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestWinInputs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	lbint.EnsureCompiled(ctx, t, "filebeat")

	reportOptions := lbint.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: true,
	}

	t.Run("input can ingest data", func(t *testing.T) {
		evtx, _ := filepath.Abs(filepath.Join("testdata", "1100.evtx"))
		configWinlogTemplate := fmt.Sprintf(`
filebeat.inputs:
  - type: winlog
    id: "test-winlog"
    enabled: true
    name: %s
output.console:
  enabled: true
`, evtx)

		configETWTemplate := `
filebeat.inputs:
  - type: etw
    id: "test-etw"
    enabled: true
    provider.name: "Microsoft-Windows-Kernel-Process"
    session_name: TestSession
output.console:
  enabled: true
`
		tcs := map[string]struct {
			configTemplate string
			expectedFields mapstr.M
		}{
			"winlog": {
				configTemplate: configWinlogTemplate,
				expectedFields: mapstr.M{
					"input.type":      "winlog",
					"winlog.event_id": "1100",
					"message":         "The event logging service has shut down.",
				},
			},
			"etw": {
				configTemplate: configETWTemplate,
				expectedFields: mapstr.M{
					"input.type":              "etw",
					"event.kind":              "event",
					"winlog.provider_message": "Microsoft-Windows-Kernel-Process",
				},
			},
		}
		for name, tc := range tcs {
			t.Run(name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				test := fbint.NewTest(t, fbint.TestOptions{
					Config: tc.configTemplate,
				})

				test.
					ExpectJSONFields(tc.expectedFields).
					WithReportOptions(reportOptions).
					ExpectStart().
					Start(ctx).
					Wait()
			})
		}
	})
}

func TestWinlogIgnoreMissingChannel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	lbint.EnsureCompiled(ctx, t, "filebeat")

	reportOptions := lbint.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: true,
	}

	tcs := map[string]struct {
		configTemplate string
		expectedOutput []string
	}{
		"default behavior ignores missing channels": {
			configTemplate: `
filebeat.inputs:
  - type: winlog
    id: "test-winlog-missing-default"
    enabled: true
    name: "NonExistentChannel1"
output.console:
  enabled: true
logging.level: info
`,
			expectedOutput: []string{
				"ignoring open error",
				"NonExistentChannel1",
			},
		},
		"explicit true ignores missing channels": {
			configTemplate: `
filebeat.inputs:
  - type: winlog
    id: "test-winlog-missing-explicit-true"
    enabled: true
    name: "NonExistentChannel2"
    ignore_missing_channel: true
output.console:
  enabled: true
logging.level: info
`,
			expectedOutput: []string{"ignoring open error", "NonExistentChannel2"},
		},
		"explicit false fails on missing channels": {
			configTemplate: `
filebeat.inputs:
  - type: winlog
    id: "test-winlog-missing-explicit-false"
    enabled: true
    name: "NonExistentChannel3"
    ignore_missing_channel: false
output.console:
  enabled: true
logging.level: debug
`,
			expectedOutput: []string{"NonExistentChannel3", "The specified channel could not be found", "encountered recoverable error"},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			test := fbint.NewTest(t, fbint.TestOptions{
				Config: tc.configTemplate,
			})

			test.
				ExpectOutput(tc.expectedOutput...).
				WithReportOptions(reportOptions).
				ExpectStart().
				Start(ctx).
				Wait()
		})
	}
}

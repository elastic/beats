// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oteltestcol

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          enabled: true
          message: "test message"
          count: 1
    processors: ~
    logging:
      level: debug
    queue.mem.flush.timeout: 0s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - debug
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
`
	col := New(t, cfg)
	require.NotNil(t, col)

	require.Eventually(t, func() bool {
		return col.ObservedLogs().
			FilterMessageSnippet("Publish event").
			FilterMessageSnippet(`"message": "test message"`).Len() == 1
	}, 30*time.Second, 100*time.Millisecond, "Expected debug log with test message not found")
}

func TestAuditbeatReceiver(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := fmt.Sprintf(`receivers:
  auditbeatreceiver:
    auditbeat:
      modules:
        - module: file_integrity
          enabled: true
          paths:
            - %s
          scan_at_start: false
    logging:
      level: debug
    queue.mem.flush.timeout: 0s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - auditbeatreceiver
      exporters:
        - debug
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
`, tmpDir)
	col := New(t, cfg)
	require.NotNil(t, col)

	require.Eventually(t, func() bool {
		return col.ObservedLogs().
			FilterMessageSnippet("Starting metrics logging every 30s").Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "Expected auditbeat receiver to start and log metrics")
}

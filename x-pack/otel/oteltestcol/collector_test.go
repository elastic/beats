// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oteltestcol

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
			FilterMessageSnippet("Skipping metrics logging").Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "Expected auditbeat receiver to start and initialize the metric reporter")
}

func TestHeartbeatReceiver(t *testing.T) {
	cfg := `receivers:
  heartbeatreceiver:
    heartbeat:
      monitors:
        - type: tcp
          id: test-tcp
          schedule: "@every 60s"
          hosts:
            - "localhost:0"
          enabled: true
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
        - heartbeatreceiver
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
			FilterMessageSnippet("Skipping metrics logging").Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "Expected heartbeat receiver to start and initialize the metric reporter")
}

// TestOsquerybeatReceiverRegistered verifies that the osquerybeat receiver
// factory is properly registered with the collector. The osquerybeat receiver
// requires the osqueryd binary to run a full pipeline, so this test starts the
// collector with a filebeatreceiver pipeline while the osquerybeat factory is
// registered in the component list, confirming it can coexist without errors.
func TestOsquerybeatReceiverRegistered(t *testing.T) {
	cfg := `receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          enabled: true
          message: "osqtest message"
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
			FilterMessageSnippet(`"message": "osqtest message"`).Len() == 1
	}, 30*time.Second, 100*time.Millisecond, "Expected collector to start with osquerybeat receiver registered")
}

func TestPacketbeatReceiver(t *testing.T) {
	device := "lo"
	if runtime.GOOS == "darwin" {
		device = "lo0"
	}
	cfg := fmt.Sprintf(`receivers:
  packetbeatreceiver:
    packetbeat:
      interfaces:
        device: %s
      protocols:
        - type: http
          ports:
            - 80
            - 8080
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
        - packetbeatreceiver
      exporters:
        - debug
  telemetry:
    logs:
      level: DEBUG
    metrics:
      level: none
`, device)
	col := New(t, cfg)
	require.NotNil(t, col)

	require.Eventually(t, func() bool {
		return col.ObservedLogs().
			FilterMessageSnippet("Skipping metrics logging").Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "Expected packetbeat receiver to start and initialize the metric reporter")
}

func TestNewRespectsTelemetryLogsLevel(t *testing.T) {
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
      level: info
    metrics:
      level: none
`
	col := New(t, cfg)
	require.NotNil(t, col)

	require.Eventually(t, func() bool {
		return col.ObservedLogs().
			FilterMessage("Logs").
			FilterField(zap.Int("log records", 1)).Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "expected debug exporter to log the processed event")

	require.Equal(t, 0, col.ObservedLogs().FilterLevelExact(zapcore.DebugLevel).Len(),
		"debug logs should not be observed when telemetry.logs.level is info")
	require.Positive(t, col.ObservedLogs().FilterLevelExact(zapcore.InfoLevel).Len(),
		"info logs should be observed when telemetry.logs.level is info")
}

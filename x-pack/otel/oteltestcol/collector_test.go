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

func TestFilebeatReceiver(t *testing.T) {
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
processors:
  beat:
    processors:
      - add_host_metadata:
          when.not.contains.tags: forwarded
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      processors:
        - beat
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

	require.Empty(t,
		col.ObservedLogs().FilterMessageSnippet("pdata fast path disabled").All(),
		"all Filebeat global processors must implement RunPdata — pdata fast path must be active")
}

func TestMetricbeatReceiver(t *testing.T) {
	cfg := `receivers:
  metricbeatreceiver:
    metricbeat:
      max_start_delay: 0s
      modules:
        - module: system
          enabled: true
          period: 1s
          metricsets:
            - cpu
    logging:
      level: debug
    queue.mem.flush.timeout: 0s
processors:
  beat:
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - metricbeatreceiver
      processors:
        - beat
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
	}, 30*time.Second, 100*time.Millisecond, "Expected metricbeat receiver to start and initialize the metric reporter")

	require.Empty(t,
		col.ObservedLogs().FilterMessageSnippet("pdata fast path disabled").All(),
		"all Metricbeat global processors must implement RunPdata — pdata fast path must be active")
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
processors:
  beat:
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - auditbeatreceiver
      processors:
        - beat
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

	require.Empty(t,
		col.ObservedLogs().FilterMessageSnippet("pdata fast path disabled").All(),
		"all Auditbeat global processors must implement RunPdata — pdata fast path must be active")
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

	require.Empty(t,
		col.ObservedLogs().FilterMessageSnippet("pdata fast path disabled").All(),
		"pdata fast path must not be disabled (heartbeat has no global processors)")
}

// TestOsquerybeatReceiverRegistered verifies that the osquerybeat receiver
// factory is properly registered with the collector. The osquerybeat receiver
// requires the osqueryd binary to run a full pipeline, so this test starts the
// collector with a filebeatreceiver pipeline while the osquerybeat factory is
// registered in the component list, confirming it can coexist without errors.
// The beatprocessor is configured with Osquerybeat's global processors to verify
// the pdata fast path is active.
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
processors:
  beat:
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      processors:
        - beat
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

	require.Empty(t,
		col.ObservedLogs().FilterMessageSnippet("pdata fast path disabled").All(),
		"all Osquerybeat global processors must implement RunPdata — pdata fast path must be active")
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
processors:
  beat:
    processors:
      - drop_fields:
          when.contains.tags: forwarded
          fields: [host]
      - add_host_metadata:
          when.not.contains.tags: forwarded
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - detect_mime_type:
          field: http.request.body.content
          target: http.request.mime_type
      - detect_mime_type:
          field: http.response.body.content
          target: http.response.mime_type
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers:
        - packetbeatreceiver
      processors:
        - beat
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

	require.Empty(t,
		col.ObservedLogs().FilterMessageSnippet("pdata fast path disabled").All(),
		"all Packetbeat global processors must implement RunPdata — pdata fast path must be active")
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

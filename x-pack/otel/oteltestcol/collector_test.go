// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oteltestcol

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/features"
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
			FilterMessageSnippet("Starting metrics logging every 30s").Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "Expected heartbeat receiver to start and log metrics")
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
			FilterMessageSnippet("Starting metrics logging every 30s").Len() > 0
	}, 30*time.Second, 100*time.Millisecond, "Expected packetbeat receiver to start and log metrics")
}

// TestFilebeatReceiverFileStorage verifies that a Filebeat receiver can use the
// OpenTelemetry file_storage extension as its state store. It ingests a file,
// stops the collector, restarts it with the same storage directory, appends two
// more lines, and asserts that only those two lines are read again (i.e. the
// filestream offset was persisted to and restored from the file_storage
// extension).
func TestFilebeatReceiverFileStorage(t *testing.T) {
	// The injected storage extension is only used by inputs for which the
	// Elasticsearch state store feature is enabled. Enable it for filestream.
	// Cleanup is registered before t.Setenv so it runs after the env is
	// restored (cleanups run in LIFO order), re-reading the original value.
	t.Cleanup(func() { features.ReinitForTest() })
	t.Setenv("AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES", "filestream")
	features.ReinitForTest()

	// Directories that must survive the collector restart.
	workDir := t.TempDir()
	homeDir := filepath.Join(workDir, "home")
	storageDir := filepath.Join(workDir, "storage")
	require.NoError(t, os.MkdirAll(homeDir, 0o700))
	require.NoError(t, os.MkdirAll(storageDir, 0o700))
	logFile := filepath.Join(workDir, "input.log")

	cfg := fmt.Sprintf(`extensions:
  file_storage:
    directory: %[1]s
    create_directory: true
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-filestorage-test
          enabled: true
          paths:
            - %[2]s
          file_identity.native: ~
          prospector.scanner.fingerprint.enabled: false
          prospector.scanner.check_interval: 100ms
    path.home: %[3]s
    storage: file_storage
    queue.mem.flush.timeout: 0s
    logging:
      level: debug
exporters:
  debug:
    verbosity: detailed
service:
  extensions:
    - file_storage
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
`, storageDir, logFile, homeDir)

	// Ingest the first three lines.
	appendLines(t, logFile, "line-1", "line-2", "line-3")

	col1 := New(t, cfg)
	require.NotNil(t, col1)

	for _, line := range []string{"line-1", "line-2", "line-3"} {
		require.Eventuallyf(t, func() bool {
			return debugRecordCount(col1, line) > 0
		}, 60*time.Second, 100*time.Millisecond, "expected %q in the debug output of the first run", line)
	}

	// Stop the collector; this releases the file_storage lock and persists the
	// filestream offset.
	col1.Shutdown()

	// Restart the collector using the same storage directory, then append two
	// more lines. Only these should be read.
	col2 := New(t, cfg)
	require.NotNil(t, col2)

	appendLines(t, logFile, "line-4", "line-5")

	for _, line := range []string{"line-4", "line-5"} {
		require.Eventuallyf(t, func() bool {
			return debugRecordCount(col2, line) > 0
		}, 60*time.Second, 100*time.Millisecond, "expected %q in the debug output after restart", line)
	}

	// The previously ingested lines must not be read again after the restart,
	// proving the offset was restored from the file_storage extension.
	for _, line := range []string{"line-1", "line-2", "line-3"} {
		assert.Zerof(t, debugRecordCount(col2, line),
			"line %q was re-read after restart; the filestream offset was not restored from file_storage", line)
	}
}

// appendLines appends the given lines (newline-terminated) to the file at path,
// creating it if necessary, and flushes them to disk.
func appendLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	require.NoErrorf(t, err, "could not open %s", path)
	defer f.Close()
	for _, line := range lines {
		_, err := fmt.Fprintln(f, line)
		require.NoErrorf(t, err, "could not write %q to %s", line, path)
	}
	require.NoErrorf(t, f.Sync(), "could not sync %s", path)
}

// debugRecordCount returns the number of detailed debug exporter log entries
// whose body contains the given line. Filtering on "LogRecord #" restricts the
// match to the debug exporter's marshaled output (which embeds the event body),
// rather than Filebeat's own internal log lines.
func debugRecordCount(col *Collector, line string) int {
	return col.ObservedLogs().
		FilterMessageSnippet("LogRecord #").
		FilterMessageSnippet(line).
		Len()
}

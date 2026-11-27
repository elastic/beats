// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/osquery/osquery-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/osqreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// Removing the live logger registration makes native scheduled-result logging
// fail, exercising osqueryd's catastrophic exit rather than a fake process exit.
func TestReceiverRestartsAfterOsqueryExit78(t *testing.T) {
	ensureOsquerydAvailable(t)
	t.Cleanup(func() { oteltest.VerifyNoLeaks(t) })

	// Set up the receiver and capture results identifying the active daemon.
	home := receiverTestHome(t)
	core, observed := observer.New(zapcore.DebugLevel)
	host := &mockActionHost{MockHost: &oteltest.MockHost{}}

	var resultMu sync.Mutex
	var lastPID, socketPath string
	var lastResultTime int64
	var profileScheduleIDs []any

	logConsumer, err := consumer.NewLogs(func(_ context.Context, logs plog.Logs) error {
		for _, resource := range logs.ResourceLogs().All() {
			for _, scope := range resource.ScopeLogs().All() {
				for _, record := range scope.LogRecords().All() {
					fields := record.Body().Map().AsRaw()
					if fields["type"] == "osquery_profile" {
						resultMu.Lock()
						profileScheduleIDs = append(profileScheduleIDs, fields["schedule_id"])
						resultMu.Unlock()
						continue
					}
					result, ok := fields["osquery"].(map[string]any)
					if !ok || result["extension_socket"] == nil || result["pid"] == nil {
						continue
					}

					resultMu.Lock()
					lastPID = fmt.Sprint(result["pid"])
					socketPath = fmt.Sprint(result["extension_socket"])
					if meta, ok := fields["osquery_meta"].(map[string]any); ok {
						lastResultTime, _ = meta["unix_time"].(int64)
					}
					resultMu.Unlock()
				}
			}
		}
		return nil
	})
	require.NoError(t, err, "create restart consumer")

	// Configure a scheduled query that reports the daemon PID and socket. With a
	// query_profile stream, each execution is also profiled outside the logger
	// callback.
	const scheduleID = "restart-schedule"
	factory := osqreceiver.NewFactoryWithSettings(osqreceiver.Settings{Home: home})
	cfg := &osqreceiver.Config{Beatconfig: map[string]any{
		"path.home":                  home,
		"http.enabled":               false,
		"management.otel.enabled":    true,
		"queue.mem.flush.min_events": 1,
		"queue.mem.flush.timeout":    "0s",
		"osquerybeat": map[string]any{"inputs": []any{
			map[string]any{
				"id": "restart-result", "type": "osquery",
				"data_stream": map[string]any{"dataset": "osquery_manager.result"},
				"osquery": map[string]any{
					"options": map[string]any{"schedule_splay_percent": 0},
					"schedule": map[string]any{"restart": map[string]any{
						"query": "SELECT info.pid, flags.value AS extension_socket " +
							"FROM osquery_info info CROSS JOIN osquery_flags flags WHERE flags.name = 'extensions_socket'",
						"interval": 1, "snapshot": true, "schedule_id": scheduleID,
					}},
				},
			},
			map[string]any{
				"id": "restart-response", "type": "osquery",
				"data_stream": map[string]any{"dataset": "osquery_manager.action.responses"},
			},
			map[string]any{
				"id": "restart-profile", "type": "osquery",
				"data_stream": map[string]any{"dataset": "osquery_manager.query_profile"},
			},
		}},
	}}

	// Start the receiver and arrange shutdown and failure diagnostics.
	rec, err := factory.CreateLogs(t.Context(), makeReceiverSettings(factory, "restart", core), cfg, logConsumer)
	require.NoError(t, err, "create receiver")
	require.NoError(t, rec.Start(t.Context(), host), "start receiver")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		assert.NoError(t, rec.Shutdown(ctx), "cleanup receiver after real daemon restart")
		if t.Failed() {
			for _, entry := range observed.All() {
				if entry.Level >= zapcore.WarnLevel || entry.Message == "Start osqueryd" || entry.Message == "configPlugin GenerateConfig is called" {
					t.Logf("[%s] %s", entry.Level, entry.Message)
				}
			}
			for _, event := range host.GetEvents() {
				t.Logf("receiver status: %s", event.Status())
			}
		}
	})

	require.Eventually(t, func() bool {
		resultMu.Lock()
		defer resultMu.Unlock()
		return lastPID != "" && socketPath != "" && lastResultTime > 0
	}, time.Minute, 100*time.Millisecond, "scheduled results must expose the live daemon PID and extension socket")
	resultMu.Lock()
	initialPID, initialSocket := lastPID, socketPath
	resultMu.Unlock()

	// The deferred profile query runs against the real daemon's osquery_schedule.
	require.Eventually(t, func() bool {
		resultMu.Lock()
		defer resultMu.Unlock()
		return len(profileScheduleIDs) > 0
	}, 30*time.Second, 100*time.Millisecond, "scheduled executions must publish a query_profile event")
	resultMu.Lock()
	assert.Equal(t, scheduleID, profileScheduleIDs[0], "the profile must carry the policy's schedule_id")
	resultMu.Unlock()

	// Remove the live logger registration to provoke native exit 78.
	cli, err := osquery.NewClient(initialSocket, 3*time.Second, osquery.MaxWaitTime(3*time.Second))
	require.NoError(t, err, "connect to the live daemon for logger fault injection")
	defer cli.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	extensions, err := cli.ExtensionsContext(ctx)
	require.NoError(t, err, "list live daemon extension registrations")

	var removed bool
	for id, extension := range extensions {
		if extension.Name != "osqextman" {
			continue
		}
		status, err := cli.DeregisterExtensionContext(ctx, id)
		require.NoError(t, err, "remove the live receiver's logger route")
		require.Zero(t, status.Code, "native extension deregistration must succeed")
		removed = true
		break
	}
	require.True(t, removed, "the receiver's osqextman registration must exist before fault injection")
	cli.Close()
	faultAt := time.Now().Unix()

	// Verify the exit, a replacement daemon, and newly executed results.
	require.Eventually(t, func() bool {
		for _, entry := range observed.FilterMessageSnippet("Failed to run osqueryd").All() {
			if strings.Contains(entry.Message, "exit status 78") {
				return true
			}
		}
		return false
	}, 30*time.Second, 100*time.Millisecond, "the real osqueryd process must exit with status 78 after its logger route fails")

	require.Eventually(t, func() bool {
		resultMu.Lock()
		defer resultMu.Unlock()
		return lastPID != initialPID && lastResultTime > faultAt
	}, time.Minute, 100*time.Millisecond, "the receiver must start a replacement daemon and publish newly executed results")

	for _, event := range host.GetEvents() {
		assert.NotEqual(t, componentstatus.StatusPermanentError, event.Status(), "exit 78 must not permanently fail the receiver")
		assert.NotEqual(t, componentstatus.StatusFatalError, event.Status(), "exit 78 must not fatally fail the receiver")
	}
	// The receiver's beat status does not reach the host on this branch, so
	// check that the restart went through the bounded backoff instead.
	assert.NotEmpty(t, observed.FilterMessageSnippet("Restart osqueryd after recoverable error in 1s").All(), "the first runtime exit must restart after the initial backoff")
}

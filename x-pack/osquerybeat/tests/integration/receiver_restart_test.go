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
	home := receiverTestHome(t)
	core, observed := observer.New(zapcore.DebugLevel)
	host := &mockActionHost{MockHost: &oteltest.MockHost{}}
	var resultMu sync.Mutex
	var lastPID, socketPath string
	var lastResultTime int64
	logConsumer, err := consumer.NewLogs(func(_ context.Context, logs plog.Logs) error {
		for _, resource := range logs.ResourceLogs().All() {
			for _, scope := range resource.ScopeLogs().All() {
				for _, record := range scope.LogRecords().All() {
					fields := record.Body().Map().AsRaw()
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
					"options": map[string]any{"thrift_timeout": 3, "schedule_splay_percent": 0},
					"schedule": map[string]any{"restart": map[string]any{
						"query": "SELECT info.pid, flags.value AS extension_socket " +
							"FROM osquery_info info CROSS JOIN osquery_flags flags WHERE flags.name = 'extensions_socket'",
						"interval": 1, "snapshot": true,
					}},
				},
			},
			map[string]any{
				"id": "restart-response", "type": "osquery",
				"data_stream": map[string]any{"dataset": "osquery_manager.action.responses"},
			},
		}},
	}}
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
	require.Eventually(t, func() bool {
		var degraded bool
		for _, event := range host.GetEvents() {
			if event.Status() == componentstatus.StatusRecoverableError {
				degraded = true
			}
			if degraded && event.Status() == componentstatus.StatusOK {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "the receiver must report degraded status and then return to running after restart readiness")
	for _, event := range host.GetEvents() {
		assert.NotEqual(t, componentstatus.StatusPermanentError, event.Status(), "exit 78 must not permanently fail the receiver")
		assert.NotEqual(t, componentstatus.StatusFatalError, event.Status(), "exit 78 must not fatally fail the receiver")
	}
}

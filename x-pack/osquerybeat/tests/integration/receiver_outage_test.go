// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// A consumer that holds a batch models an output outage without Elasticsearch.
// The snapshot exceeds queue capacity, so synchronous logger publication would
// block and outlast osqueryd's short thrift_timeout.
func TestReceiverOutputOutage(t *testing.T) {
	ensureOsquerydAvailable(t)
	for _, restoreOutput := range []bool{true, false} {
		name := "shutdown_while_output_unavailable"
		if restoreOutput {
			name = "collection_resumes_without_restart"
		}
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { oteltest.VerifyNoLeaks(t) })
			home := receiverTestHome(t)
			core, observed := observer.New(zapcore.DebugLevel)
			host := &mockActionHost{MockHost: &oteltest.MockHost{}}
			var unavailable atomic.Bool
			var delivered atomic.Int64
			var lastResultTime atomic.Int64
			var profiles, lastProfileTime, lastProfileExecutions atomic.Int64
			var blockedOnce, restoreOnce sync.Once
			blocked, restored := make(chan struct{}), make(chan struct{})
			restore := func() { restoreOnce.Do(func() { close(restored) }) }
			var pidMu sync.Mutex
			var lastPID string
			logConsumer, err := consumer.NewLogs(func(ctx context.Context, logs plog.Logs) error {
				if unavailable.Load() {
					blockedOnce.Do(func() { close(blocked) })
					select {
					case <-restored:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				for _, resource := range logs.ResourceLogs().All() {
					for _, scope := range resource.ScopeLogs().All() {
						for _, record := range scope.LogRecords().All() {
							fields := record.Body().Map().AsRaw()
							if profile, ok := fields["osquery_profile"].(map[string]any); ok {
								if executions, ok := profile["executions"].(int64); ok && executions > 0 && profile["source"] == "scheduled" && profile["query_name"] == "outage" {
									profiles.Add(1)
									lastProfileTime.Store(record.Timestamp().AsTime().Unix())
									lastProfileExecutions.Store(executions)
								}
							}
							if result, ok := fields["osquery"].(map[string]any); ok {
								if pid, ok := result["pid"]; ok {
									pidMu.Lock()
									lastPID = fmt.Sprint(pid)
									pidMu.Unlock()
									delivered.Add(1)
									if meta, ok := fields["osquery_meta"].(map[string]any); ok {
										if runTime, ok := meta["unix_time"].(int64); ok {
											lastResultTime.Store(runTime)
										}
									}
								}
							}
						}
					}
				}
				return nil
			})
			require.NoError(t, err, "create outage consumer")
			factory := osqreceiver.NewFactoryWithSettings(osqreceiver.Settings{Home: home})
			const values = "(SELECT 1 AS value UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 " +
				"UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8)"
			cfg := &osqreceiver.Config{Beatconfig: map[string]any{
				"path.home":                  home,
				"http.enabled":               false,
				"management.otel.enabled":    true,
				"queue.mem.events":           32,
				"queue.mem.flush.min_events": 1,
				"queue.mem.flush.timeout":    "0s",
				"osquerybeat": map[string]any{"inputs": []any{
					map[string]any{
						"id": "outage-result", "type": "osquery",
						"data_stream": map[string]any{"dataset": "osquery_manager.result"},
						"osquery": map[string]any{
							"options": map[string]any{"thrift_timeout": 3, "schedule_splay_percent": 0},
							"schedule": map[string]any{"outage": map[string]any{
								"query":    fmt.Sprintf("SELECT pid, a.value * 8 + b.value AS value FROM osquery_info CROSS JOIN %s a CROSS JOIN %s b", values, values),
								"interval": 1, "snapshot": true,
							}},
						},
					},
					map[string]any{
						"id": "outage-response", "type": "osquery",
						"data_stream": map[string]any{"dataset": "osquery_manager.action.responses"},
					},
					map[string]any{
						"id": "outage-profile", "type": "osquery",
						"data_stream": map[string]any{"dataset": "osquery_manager.query_profile"},
					},
				}},
			}}
			rec, err := factory.CreateLogs(t.Context(), makeReceiverSettings(factory, "outage", core), cfg, logConsumer)
			require.NoError(t, err, "create receiver")
			require.NoError(t, rec.Start(t.Context(), host), "start receiver")
			var shutdownStarted bool
			shutdownDone := make(chan error, 1)
			shutdown := func() {
				shutdownStarted = true
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					shutdownDone <- rec.Shutdown(ctx)
				}()
			}
			t.Cleanup(func() {
				restore()
				if !shutdownStarted {
					shutdown()
					select {
					case err := <-shutdownDone:
						assert.NoError(t, err, "cleanup receiver")
					case <-time.After(15 * time.Second):
						t.Error("receiver cleanup timed out")
					}
				}
				if t.Failed() {
					for _, entry := range observed.All() {
						if entry.Level >= zapcore.WarnLevel {
							t.Logf("[%s] %s", entry.Level, entry.Message)
						}
					}
				}
			})
			require.Eventually(t, func() bool { return delivered.Load() > 0 }, time.Minute, 100*time.Millisecond,
				"receiver must publish before the output outage")
			require.Eventually(t, func() bool { return profiles.Load() > 0 }, time.Minute, 100*time.Millisecond,
				"default profiling must publish a valid scheduled profile before the output outage")
			initialProfileExecutions := lastProfileExecutions.Load()
			pidMu.Lock()
			initialPID := lastPID
			pidMu.Unlock()
			require.NotEmpty(t, initialPID, "the result must identify the running osqueryd")
			require.Positive(t, lastResultTime.Load(), "the result must include its original execution time")
			unavailable.Store(true)
			select {
			case <-blocked:
			case <-time.After(15 * time.Second):
				t.Fatal("output consumer did not receive an outage batch")
			}
			// Hold every admitted slot longer than two thrift deadlines. Restoring
			// output earlier could hide the callback timeout and delayed failure.
			select {
			case <-time.After(8 * time.Second):
			case <-t.Context().Done():
				t.Fatal("test context cancelled during outage")
			}
			assert.Empty(t, observed.FilterMessageSnippet("Failed to run osqueryd").All(),
				"output backpressure must not terminate osqueryd")
			for _, event := range host.GetEvents() {
				assert.NotEqual(t, componentstatus.StatusPermanentError, event.Status(), "receiver must not permanently fail")
				assert.NotEqual(t, componentstatus.StatusFatalError, event.Status(), "receiver must not fatally fail")
			}
			if restoreOutput {
				recoveredAt := time.Now().Unix()
				restore()
				require.Eventually(t, func() bool { return lastResultTime.Load() > recoveredAt }, 20*time.Second, 100*time.Millisecond,
					"fresh scheduled results must arrive after output recovery")
				require.Eventually(t, func() bool {
					return lastProfileTime.Load() > recoveredAt && lastProfileExecutions.Load() > initialProfileExecutions
				}, 20*time.Second, 100*time.Millisecond,
					"fresh scheduled profiles must arrive after output recovery with profiling enabled by default")
				pidMu.Lock()
				assert.Equal(t, initialPID, lastPID, "osqueryd must remain alive through the outage")
				pidMu.Unlock()
			}
			shutdown()
			select {
			case err := <-shutdownDone:
				require.NoError(t, err, "receiver must shut down even while output remains unavailable")
			case <-time.After(12 * time.Second):
				restore()
				select {
				case <-shutdownDone:
				case <-time.After(15 * time.Second):
				}
				t.Fatal("receiver shutdown waited for output recovery")
			}
		})
	}
}

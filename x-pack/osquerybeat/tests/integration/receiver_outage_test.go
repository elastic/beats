// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/osqreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// A consumer that holds a batch models an output outage without Elasticsearch.
// The snapshot exceeds the pipeline queue, so the outage fills it and stops
// the scheduled worker; osqueryd's logger callback must keep returning within
// thrift_timeout while osquerybeat's scheduled queue holds the backlog.
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
			actions := &actionCapture{}
			host := &liveActionHost{MockHost: &oteltest.MockHost{}, ext: actions}
			var runs executionLog
			var unavailable atomic.Bool
			var delivered atomic.Int64
			var lastResultTime atomic.Int64
			var profiles, lastProfileTime, lastProfileExecutions atomic.Int64
			var blockedOnce, restoreOnce sync.Once
			blocked, restored := make(chan struct{}), make(chan struct{})
			restore := func() { restoreOnce.Do(func() { close(restored) }) }
			var pidMu sync.Mutex
			var lastPID string
			const liveActionID = "outage-live"
			// liveTimeoutErr is the text of beater.errLivePublishTimeout.
			const liveTimeoutErr = "timed out publishing live query results"
			var responseMu sync.Mutex
			var liveResponse map[string]any
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
							if fields["action_id"] == liveActionID && fields["completed_at"] != nil {
								responseMu.Lock()
								liveResponse = fields
								responseMu.Unlock()
							}
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
									var runTime int64
									if meta, ok := fields["osquery_meta"].(map[string]any); ok {
										if runTime, ok = meta["unix_time"].(int64); ok {
											lastResultTime.Store(runTime)
										}
									}
									runs.add(result, runTime)
								}
							}
						}
					}
				}
				return nil
			})
			require.NoError(t, err, "create outage consumer")
			factory := osqreceiver.NewFactoryWithSettings(osqreceiver.Settings{Home: home})
			const liveRows, outageRows = 8, 64
			const values = "(SELECT 1 AS value UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 " +
				"UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8)"
			cfg := &osqreceiver.Config{Beatconfig: map[string]any{
				"path.home":                  home,
				"http.enabled":               true,
				"http.host":                  "localhost",
				"http.port":                  0,
				"management.otel.enabled":    true,
				"queue.mem.events":           32,
				"queue.mem.flush.min_events": 1,
				"queue.mem.flush.timeout":    "0s",
				"osquerybeat": map[string]any{"inputs": []any{
					map[string]any{
						"id": "outage-result", "type": "osquery",
						"data_stream": map[string]any{"dataset": "osquery_manager.result"},
						"osquery": map[string]any{
							"options": map[string]any{"thrift_timeout": 30, "schedule_splay_percent": 0},
							// A 60 s outage queues about 4000 rows; room for all of them
							// makes any loss a failure rather than an expected overflow.
							// The live timeout ends well inside the outage, so the live
							// query always times out.
							"elastic_options": map[string]any{"publishing": map[string]any{
								"scheduled": map[string]any{"max_events": 100000},
								"live":      map[string]any{"timeout": "10s"},
							}},
							// Each run selects its own osquery_schedule execution count, so
							// every run carries a unique, consecutive number and a lost run
							// leaves a gap.
							"schedule": map[string]any{"outage": map[string]any{
								"query": fmt.Sprintf("SELECT i.pid, s.executions AS execution, a.value * 8 + b.value AS value "+
									"FROM osquery_info i CROSS JOIN osquery_schedule s CROSS JOIN %s a CROSS JOIN %s b "+
									"WHERE s.name = 'outage'", values, values),
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
			firstExecution, ok := runs.latest()
			require.True(t, ok, "results must carry the osquery_schedule execution count")
			monitorPort := monitoringPortsFromLogs(t, observed, 1, 30*time.Second)[0]
			if restoreOutput {
				require.Eventually(t, func() bool { return actions.handler() != nil }, 30*time.Second, 100*time.Millisecond,
					"receiver must register its live query action")
			}
			unavailable.Store(true)
			select {
			case <-blocked:
			case <-time.After(15 * time.Second):
				t.Fatal("output consumer did not receive an outage batch")
			}
			var liveRes map[string]any
			var liveErr error
			liveDone := make(chan struct{})
			if restoreOutput {
				// A queued scheduled result proves the pipeline is full, so the
				// live rows cannot all be handed to it before the timeout.
				require.Eventually(t, func() bool { return scheduledStats(t, monitorPort).Queue.Items > 0 }, 30*time.Second, 100*time.Millisecond,
					"scheduled results must queue behind the full pipeline before the live query")
				execute := actions.handler()
				go func() {
					defer close(liveDone)
					liveRes, liveErr = execute(t.Context(), map[string]any{
						"id":   liveActionID,
						"data": map[string]any{"query": "SELECT value FROM " + values},
					})
				}()
			}
			// Hold the output for two thrift deadlines. Restoring it earlier could
			// hide a logger callback that waits for the output.
			select {
			case <-time.After(60 * time.Second):
			case <-t.Context().Done():
				t.Fatal("test context cancelled during outage")
			}
			assert.Empty(t, observed.FilterMessageSnippet("Failed to run osqueryd").All(),
				"output backpressure must not terminate osqueryd")
			for _, event := range host.GetEvents() {
				assert.NotEqual(t, componentstatus.StatusPermanentError, event.Status(), "receiver must not permanently fail")
				assert.NotEqual(t, componentstatus.StatusFatalError, event.Status(), "receiver must not fatally fail")
			}
			dropped := scheduledStats(t, monitorPort).DroppedRows
			require.Contains(t, dropped, "overflow", "the scheduled queue must report overflow drops")
			assert.Zero(t, dropped["overflow"], "the outage backlog must fit in max_events without overflow drops")
			if restoreOutput {
				select {
				case <-liveDone:
				case <-time.After(30 * time.Second):
					t.Fatal("live query action did not return while the output was unavailable")
				}
				require.NoError(t, liveErr, "live query action reports failures in its result")
				assert.Equal(t, liveTimeoutErr, liveRes["error"],
					"a live query that outlasts live.timeout must report the publish timeout")
				count, ok := liveRes["count"].(int)
				require.True(t, ok, "a timed-out live query must report its published row count, got %v", liveRes)
				assert.Less(t, count, liveRows, "a timed-out live query must report only the rows it published")

				recoveredAt := time.Now().Unix()
				restore()
				require.Eventually(t, func() bool { return lastResultTime.Load() > recoveredAt }, 20*time.Second, 100*time.Millisecond,
					"fresh scheduled results must arrive after output recovery")
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					n, ok := runs.completeAfter(firstExecution, recoveredAt, outageRows)
					assert.True(c, ok, "scheduled execution %d must be published with all %d rows", n, outageRows)
				}, 20*time.Second, 100*time.Millisecond,
					"every scheduled execution during the outage must be published after recovery")
				require.Eventually(t, func() bool {
					return lastProfileTime.Load() > recoveredAt && lastProfileExecutions.Load() > initialProfileExecutions
				}, 20*time.Second, 100*time.Millisecond,
					"fresh scheduled profiles must arrive after output recovery with profiling enabled by default")
				pidMu.Lock()
				assert.Equal(t, initialPID, lastPID, "osqueryd must remain alive through the outage")
				pidMu.Unlock()
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					responseMu.Lock()
					defer responseMu.Unlock()
					require.NotNil(c, liveResponse, "no action response for %s yet", liveActionID)
					assert.Equal(c, liveTimeoutErr, liveResponse["error"], "the action response must carry the publish timeout")
				}, 20*time.Second, 100*time.Millisecond,
					"the timed-out live query's action response must be published after recovery")
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

// actionCapture is an action extension that keeps the receiver's live query
// handler so the test can run a live query.
type actionCapture struct {
	mu      sync.Mutex
	execute func(context.Context, map[string]any) (map[string]any, error)
}

func (*actionCapture) Start(context.Context, component.Host) error { return nil }
func (*actionCapture) Shutdown(context.Context) error              { return nil }

func (a *actionCapture) RegisterActionHandler(_ string, h func(context.Context, map[string]any) (map[string]any, error)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.execute = h
	return nil
}

func (a *actionCapture) UnregisterActionHandler(string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.execute = nil
}

func (a *actionCapture) handler() func(context.Context, map[string]any) (map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.execute
}

// liveActionHost is a status-tracking host that exposes ext as the action
// extension.
type liveActionHost struct {
	*oteltest.MockHost
	ext *actionCapture
}

func (h *liveActionHost) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{
		component.MustNewID("elastic_diagnostics"): h.ext,
	}
}

// executionLog records the distinct rows delivered for each osquery execution
// of the outage query, keyed by the execution count the query selects.
type executionLog struct {
	mu   sync.Mutex
	runs map[int64]*executionRows
}

type executionRows struct {
	values   map[string]struct{}
	unixTime int64
}

func (l *executionLog) add(result map[string]any, unixTime int64) {
	n, err := strconv.ParseInt(fmt.Sprint(result["execution"]), 10, 64)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.runs == nil {
		l.runs = map[int64]*executionRows{}
	}
	r := l.runs[n]
	if r == nil {
		r = &executionRows{values: map[string]struct{}{}}
		l.runs[n] = r
	}
	r.values[fmt.Sprint(result["value"])] = struct{}{}
	r.unixTime = unixTime
}

// latest returns the highest execution delivered so far.
func (l *executionLog) latest() (int64, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var last int64
	for n := range l.runs {
		last = max(last, n)
	}
	return last, len(l.runs) > 0
}

// completeAfter reports whether every execution from first on was delivered
// with all rows, up to one that ran after unixTime. Otherwise it returns the
// first missing or incomplete execution. A lost execution leaves a gap that
// never fills.
func (l *executionLog) completeAfter(first, unixTime int64, rows int) (int64, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for n := first; ; n++ {
		r := l.runs[n]
		if r == nil || len(r.values) < rows {
			return n, false
		}
		if r.unixTime > unixTime {
			return n, true
		}
	}
}

// scheduledPublishingStats is osquerybeat.publishing.scheduled from the
// receiver's stats endpoint.
type scheduledPublishingStats struct {
	DroppedRows map[string]uint64 `json:"dropped_rows"`
	Queue       struct {
		Items int64 `json:"items"`
	} `json:"queue"`
}

func scheduledStats(t *testing.T, port int) scheduledPublishingStats {
	t.Helper()
	// Without keep-alives no connection goroutines outlive the request.
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/stats", port)) //nolint:noctx // fine for tests
	require.NoError(t, err, "get receiver stats")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "receiver stats status")
	var stats struct {
		Osquerybeat struct {
			Publishing struct {
				Scheduled scheduledPublishingStats `json:"scheduled"`
			} `json:"publishing"`
		} `json:"osquerybeat"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats), "decode receiver stats")
	return stats.Osquerybeat.Publishing.Scheduled
}

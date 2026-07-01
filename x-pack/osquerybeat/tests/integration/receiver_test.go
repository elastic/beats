// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/osqreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// NOTE: TestNewReceiver, TestMultipleReceivers, and the "running input" sub-test
// of TestReceiverStatus live here rather than in osqreceiver/ because they require
// a live osqueryd binary. Unlike filebeat or metricbeat, osquerybeat cannot produce
// data without the external osqueryd process, so these tests carry the integration
// build tag and rely on ensureOsquerydAvailable (defined in otel_test.go) to skip
// when osqueryd is absent.

// makeOsqConfig returns a receiver Config for integration tests.
// pathHome must be unique per receiver to isolate osqueryd state.
// monitorPort is the TCP port for the beats HTTP monitoring server.
func makeOsqConfig(pathHome string, monitorPort int) *osqreceiver.Config {
	return &osqreceiver.Config{
		Beatconfig: map[string]any{
			"queue.mem.flush.timeout": "0s",
			"osquerybeat": map[string]any{
				"inputs": []any{
					map[string]any{
						"type": "osquery",
						"osquery": map[string]any{
							"schedule": map[string]any{
								"osquery_info": map[string]any{
									"query":    "SELECT * FROM osquery_info",
									"interval": 10,
								},
							},
						},
					},
				},
			},
			"http.enabled":            true,
			"http.host":               "localhost",
			"http.port":               monitorPort,
			"management.otel.enabled": true,
			"path.home":               pathHome,
		},
	}
}

// makeReceiverSettings builds receiver.Settings with an observed logger teed into
// sharedCore.  The otelcol.component.* context fields are added so that
// zapLogs.FilterField can separate logs from different receivers.
func makeReceiverSettings(factory receiver.Factory, name string, sharedCore zapcore.Core) receiver.Settings {
	id := component.NewIDWithName(factory.Type(), name)
	core := sharedCore.With([]zapcore.Field{
		zap.String("otelcol.component.id", id.String()),
		zap.String("otelcol.component.kind", "receiver"),
		zap.String("otelcol.signal", "logs"),
	})
	return receiver.Settings{
		ID: id,
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.New(core),
		},
	}
}

// startReceiver creates, starts, and returns a receiver together with the slice
// that accumulates its log records.
func startReceiver(
	t *testing.T,
	factory receiver.Factory,
	name string,
	cfg *osqreceiver.Config,
	sharedCore zapcore.Core,
	host component.Host,
) (receiver.Logs, *[]mapstr.M, *sync.Mutex) {
	t.Helper()

	set := makeReceiverSettings(factory, name, sharedCore)

	var mu sync.Mutex
	var logs []mapstr.M

	logConsumer, err := consumer.NewLogs(func(_ context.Context, ld plog.Logs) error {
		for _, rl := range ld.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					mu.Lock()
					logs = append(logs, lr.Body().Map().AsRaw())
					mu.Unlock()
				}
			}
		}
		return nil
	})
	require.NoError(t, err)

	rec, err := factory.CreateLogs(t.Context(), set, cfg, logConsumer)
	require.NoError(t, err)
	require.NoError(t, rec.Start(t.Context(), host))

	return rec, &logs, &mu
}

// monitoringGet performs a single HTTP GET to a monitoring endpoint using a
// short-lived connection (DisableKeepAlives) so that no persistent goroutines
// are left behind to trip VerifyNoLeaks.
func monitoringGet(t require.TestingT, port int) {
	client := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/stats", port)) //nolint:noctx // fine for tests
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewReceiver(t *testing.T) {
	ensureOsquerydAvailable(t)
	defer oteltest.VerifyNoLeaks(t)

	monitorPort := int(libbeattesting.MustAvailableTCP4Port(t))

	factory := osqreceiver.NewFactoryWithSettings(osqreceiver.Settings{Home: t.TempDir()})
	observedCore, zapLogs := observer.New(zapcore.DebugLevel)

	rec, logsPtr, mu := startReceiver(t,
		factory, "r1",
		makeOsqConfig(t.TempDir(), monitorPort),
		observedCore,
		componenttest.NewNopHost(),
	)

	t.Cleanup(func() {
		if t.Failed() {
			mu.Lock()
			n := len(*logsPtr)
			mu.Unlock()
			t.Logf("receiver produced %d events", n)
			for _, entry := range zapLogs.All() {
				t.Logf("[%s] %s", entry.Level, entry.Message)
			}
		}
	})

	// Phase 1: wait for the beat framework to start (metrics logging is the
	// first reliable signal that the receiver is up, even before osqueryd runs
	// its first query).
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		started := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s")
		assert.NotEmpty(c, started.All(), "metrics logging not started")
	}, 3*time.Minute, time.Second, "receiver framework did not start")

	// Phase 2: wait for osqueryd to execute the first scheduled query.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		mu.Lock()
		logs := *logsPtr
		mu.Unlock()

		require.NotEmptyf(c, logs, "expected at least one log event from osquerybeat")
		assert.Equal(c, "osquerybeat", logs[0].Flatten()["agent.type"],
			"expected agent.type to be 'osquerybeat'")

		processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors")
		assert.Empty(c, processorsLoaded.All(), "processors loaded but none expected")
	}, 2*time.Minute, time.Second, "expected receiver to produce events after starting")

	// Verify monitoring endpoint once, outside the poll loop.
	// http.Get inside EventuallyWithT creates a new persistent connection on every
	// iteration and leaves read/write goroutines behind that trip VerifyNoLeaks.
	monitoringGet(t, monitorPort)

	require.NoError(t, rec.Shutdown(t.Context()))
}

func TestMultipleReceivers(t *testing.T) {
	ensureOsquerydAvailable(t)
	defer oteltest.VerifyNoLeaks(t)

	factory := osqreceiver.NewFactoryWithSettings(osqreceiver.Settings{Home: t.TempDir()})

	port1 := int(libbeattesting.MustAvailableTCP4Port(t))
	port2 := int(libbeattesting.MustAvailableTCP4Port(t))

	// Shared core: all log lines from both receivers land in one ObservedLogs
	// so we can verify per-receiver isolation by filtering on otelcol.component.id.
	sharedCore, zapLogs := observer.New(zapcore.DebugLevel)
	host := componenttest.NewNopHost()

	r1, logs1, mu1 := startReceiver(t, factory, "r1", makeOsqConfig(t.TempDir(), port1), sharedCore, host)
	r2, logs2, mu2 := startReceiver(t, factory, "r2", makeOsqConfig(t.TempDir(), port2), sharedCore, host)

	t.Cleanup(func() {
		if t.Failed() {
			mu1.Lock()
			n1 := len(*logs1)
			mu1.Unlock()
			mu2.Lock()
			n2 := len(*logs2)
			mu2.Unlock()
			t.Logf("r1: %d events, r2: %d events", n1, n2)
			for _, entry := range zapLogs.All() {
				if entry.Level >= zapcore.WarnLevel {
					t.Logf("[%s] %s %v", entry.Level, entry.Message, entry.ContextMap())
				}
			}
		}
	})

	// Phase 1: wait for BOTH receivers to signal their beat framework has started.
	// Two concurrent osqueryd processes need time to initialise; this phase gives
	// a clear signal that the receivers are live before we wait for query events.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		r1Started := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").
			FilterField(zap.String("otelcol.component.id", "osquerybeatreceiver/r1"))
		r2Started := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").
			FilterField(zap.String("otelcol.component.id", "osquerybeatreceiver/r2"))
		assert.NotEmpty(c, r1Started.All(), "r1 metrics logging not started")
		assert.NotEmpty(c, r2Started.All(), "r2 metrics logging not started")
	}, 3*time.Minute, time.Second, "one or both receivers did not start their framework")

	// Phase 2: wait for at least one event from each receiver.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		mu1.Lock()
		l1 := *logs1
		mu1.Unlock()
		mu2.Lock()
		l2 := *logs2
		mu2.Unlock()

		require.NotEmptyf(c, l1, "r1: expected at least one log event")
		require.NotEmptyf(c, l2, "r2: expected at least one log event")

		// Verify each receiver has its own logger instance. Previously the
		// Beats logger was global, causing fields to be overwritten when
		// multiple receivers ran in the same process.
		r1StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(
			zap.String("otelcol.component.id", "osquerybeatreceiver/r1"))
		assert.Equalf(c, 1, r1StartLogs.Len(),
			"r1 should have exactly one Beat ID log entry")

		r2StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(
			zap.String("otelcol.component.id", "osquerybeatreceiver/r2"))
		assert.Equalf(c, 1, r2StartLogs.Len(),
			"r2 should have exactly one Beat ID log entry")
	}, 2*time.Minute, time.Second, "expected both receivers to produce events")

	// Verify monitoring endpoints once, outside the poll loop.
	monitoringGet(t, port1)
	monitoringGet(t, port2)

	require.NoError(t, r1.Shutdown(t.Context()))
	require.NoError(t, r2.Shutdown(t.Context()))
}

func TestReceiverStatus(t *testing.T) {
	ensureOsquerydAvailable(t)
	defer oteltest.VerifyNoLeaks(t)

	monitorPort := int(libbeattesting.MustAvailableTCP4Port(t))
	factory := osqreceiver.NewFactoryWithSettings(osqreceiver.Settings{Home: t.TempDir()})
	observedCore, zapLogs := observer.New(zapcore.DebugLevel)
	host := &oteltest.MockHost{}

	rec, _, _ := startReceiver(t, factory, "r1", makeOsqConfig(t.TempDir(), monitorPort), observedCore, host)

	t.Cleanup(func() {
		if t.Failed() {
			for _, entry := range zapLogs.All() {
				t.Logf("[%s] %s", entry.Level, entry.Message)
			}
		}
	})

	// Wait until the beat framework confirms osqueryd started. If the binary
	// check fails, the beat exits before reaching this log message and the
	// test times out here rather than giving a false-positive status result.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.NotEmpty(c, zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").All(),
			"metrics logging not started")
	}, 3*time.Minute, time.Second, "receiver framework did not start")

	// osqueryd is running; the final OTel status must be StatusOK.
	// Without osqueryd, StatusFailed would be the last event instead —
	// which is what the unit test's "early status ok" sub-test exposes.
	evt := host.GetEvent()
	require.NotNil(t, evt, "expected at least one status event")
	require.Equal(t, componentstatus.StatusOK, evt.Status(),
		"expected StatusOK as final status after osqueryd started; full event history: %v", host.GetEvents())

	require.NoError(t, rec.Shutdown(t.Context()))
}

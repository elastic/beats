// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"
)

// resetSystemBridgeForTest resets the package-level singleton state so tests
// don't interfere with each other.
func resetSystemBridgeForTest() {
	systemMu.Lock()
	defer systemMu.Unlock()
	if systemInst != nil {
		systemInst.shutdown()
	}
	systemInst = nil
	systemRefs = 0
}

func newTestSystemBridge(t *testing.T, reader *metric.ManualReader, statsReg *monitoring.Registry) *SystemRegistryBridge {
	t.Helper()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = provider
	b, err := newSystemRegistryBridge(settings, statsReg)
	require.NoError(t, err)
	return b
}

func setupTestRegistry(t *testing.T) *monitoring.Registry {
	t.Helper()
	reg := monitoring.NewRegistry()
	err := metricreport.SetupMetricsOptions(metricreport.MetricOptions{
		Logger:         logp.NewLogger("system-bridge-test"),
		Name:           "beat",
		SystemMetrics:  reg.GetOrCreateRegistry("system"),
		ProcessMetrics: reg.GetOrCreateRegistry("beat"),
	})
	require.NoError(t, err)
	return reg
}

func TestSystemBridgeMetrics(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := setupTestRegistry(t)
	bridge := newTestSystemBridge(t, reader, statsReg)

	rm := collectMetrics(t, reader)

	// Verify that system metrics registered by SetupMetricsOptions are
	// bridged. The exact set is platform-dependent (e.g. beat.handles.*
	// on Linux/Windows, system.load.* on non-Windows), so we check
	// cross-platform metrics that are always present.
	assert.NotNil(t, findMetricByName(rm, "beat.memstats.memory_alloc"))
	assert.NotNil(t, findMetricByName(rm, "beat.memstats.rss"))
	assert.NotNil(t, findMetricByName(rm, "beat.memstats.gc_next"))
	assert.NotNil(t, findMetricByName(rm, "beat.cpu.total.ticks"))
	assert.NotNil(t, findMetricByName(rm, "beat.runtime.goroutines"))
	assert.NotNil(t, findMetricByName(rm, "beat.info.uptime.ms"))

	bridge.shutdown()
}

func TestSystemBridgeNoReceiverAttribute(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := setupTestRegistry(t)
	bridge := newTestSystemBridge(t, reader, statsReg)

	rm := collectMetrics(t, reader)

	// Check int metric has no receiver attribute.
	rss := findMetricByName(rm, "beat.memstats.rss")
	require.NotNil(t, rss)
	gaugeDPs := getGaugeInt64DataPoints(rss)
	require.Len(t, gaugeDPs, 1)
	_, hasReceiver := gaugeDPs[0].Attributes.Value(attribute.Key("receiver"))
	assert.False(t, hasReceiver, "system metrics should not have 'receiver' attribute")

	// Check float metric has no receiver attribute.
	load1 := findMetricByName(rm, "system.load.1")
	require.NotNil(t, load1)
	floatGauge, ok := load1.Data.(metricdata.Gauge[float64])
	require.True(t, ok)
	require.Len(t, floatGauge.DataPoints, 1)
	_, hasReceiver = floatGauge.DataPoints[0].Attributes.Value(attribute.Key("receiver"))
	assert.False(t, hasReceiver, "system metrics should not have 'receiver' attribute")

	bridge.shutdown()
}

func TestSystemBridgeNilRegistry(t *testing.T) {
	reader := metric.NewManualReader()

	bridge := newTestSystemBridge(t, reader, nil)
	require.NotNil(t, bridge)

	// Collection should succeed without panicking.
	rm := collectMetrics(t, reader)
	assert.NotNil(t, rm)

	bridge.shutdown()
}

func TestSystemBridgeShutdown(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := setupTestRegistry(t)
	bridge := newTestSystemBridge(t, reader, statsReg)

	// Before shutdown, metric is observed.
	rm := collectMetrics(t, reader)
	require.NotNil(t, findMetricByName(rm, "beat.memstats.rss"))

	// After shutdown, callback is unregistered — closed flag prevents
	// further observations even if the SDK invokes the callback.
	bridge.shutdown()
	assert.True(t, bridge.closed)
}

func TestSystemBridgeAcquireRelease(t *testing.T) {
	resetSystemBridgeForTest()
	t.Cleanup(resetSystemBridgeForTest)

	settings := componenttest.NewNopTelemetrySettings()

	// First acquire creates the singleton.
	release1, err := AcquireSystemBridge(settings)
	require.NoError(t, err)
	require.NotNil(t, release1)

	systemMu.Lock()
	inst1 := systemInst
	refs1 := systemRefs
	systemMu.Unlock()
	require.NotNil(t, inst1)
	assert.Equal(t, 1, refs1)

	// Second acquire returns same instance.
	release2, err := AcquireSystemBridge(settings)
	require.NoError(t, err)

	systemMu.Lock()
	inst2 := systemInst
	refs2 := systemRefs
	systemMu.Unlock()
	assert.Same(t, inst1, inst2, "second acquire should return same instance")
	assert.Equal(t, 2, refs2)

	// Release first — still running.
	release1()

	systemMu.Lock()
	assert.NotNil(t, systemInst, "singleton should still be alive after first release")
	assert.Equal(t, 1, systemRefs)
	systemMu.Unlock()

	// Release second — shut down.
	release2()

	systemMu.Lock()
	assert.Nil(t, systemInst, "singleton should be nil after all releases")
	assert.Equal(t, 0, systemRefs)
	systemMu.Unlock()

	// Re-acquire creates a fresh instance.
	release3, err := AcquireSystemBridge(settings)
	require.NoError(t, err)

	systemMu.Lock()
	inst3 := systemInst
	systemMu.Unlock()
	assert.NotSame(t, inst1, inst3, "re-acquire should create a fresh instance")

	release3()
}

func TestSystemBridgeIdempotentRelease(t *testing.T) {
	resetSystemBridgeForTest()
	t.Cleanup(resetSystemBridgeForTest)

	settings := componenttest.NewNopTelemetrySettings()

	release, err := AcquireSystemBridge(settings)
	require.NoError(t, err)

	// Calling release twice should not panic.
	release()
	release()

	systemMu.Lock()
	assert.Nil(t, systemInst)
	assert.Equal(t, 0, systemRefs)
	systemMu.Unlock()
}

func TestSystemBridgeDoubleShutdown(t *testing.T) {
	reader := metric.NewManualReader()
	bridge := newTestSystemBridge(t, reader, nil)

	// Double shutdown should not panic.
	bridge.shutdown()
	bridge.shutdown()
}

func TestSystemBridgeNilMeterProvider(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.Logger = nil

	b, err := newSystemRegistryBridge(settings, nil)
	require.NoError(t, err)
	require.NotNil(t, b)
	b.shutdown()
}

func TestSystemBridgeLiveValues(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	beatReg := statsReg.GetOrCreateRegistry("beat")
	rss := monitoring.NewUint(beatReg, "memstats.rss")
	rss.Set(1000)

	bridge := newTestSystemBridge(t, reader, statsReg)

	// First collection sees initial value.
	rm := collectMetrics(t, reader)
	assert.Equal(t, int64(1000), getGaugeInt64Value(findMetricByName(rm, "beat.memstats.rss")))

	// Update the value in the registry.
	rss.Set(2000)

	// Second collection should see the updated value, proving the callback
	// reads live data from the registry rather than caching the snapshot
	// taken at construction time.
	rm = collectMetrics(t, reader)
	assert.Equal(t, int64(2000), getGaugeInt64Value(findMetricByName(rm, "beat.memstats.rss")))

	bridge.shutdown()
}

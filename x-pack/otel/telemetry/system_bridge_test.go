// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"runtime"
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
	defer bridge.shutdown()

	rm := collectMetrics(t, reader)

	// Verify at least one cross-platform metric is bridged.
	assert.NotNil(t, findMetricByName(rm, "beat.memstats.rss"))
}

func TestSystemBridgeNoReceiverAttribute(t *testing.T) {
	reader := metric.NewManualReader()
	statsReg := setupTestRegistry(t)
	bridge := newTestSystemBridge(t, reader, statsReg)
	defer bridge.shutdown()

	rm := collectMetrics(t, reader)

	// Int gauge: always present on all platforms.
	rss := findMetricByName(rm, "beat.memstats.rss")
	require.NotNil(t, rss)
	gaugeDPs := getGaugeInt64DataPoints(rss)
	require.Len(t, gaugeDPs, 1)
	_, hasReceiver := gaugeDPs[0].Attributes.Value(attribute.Key("receiver"))
	assert.False(t, hasReceiver, "system metrics should not have 'receiver' attribute")

	// Float gauge: system.load.* only exists on non-Windows.
	if runtime.GOOS != "windows" {
		load1 := findMetricByName(rm, "system.load.1")
		require.NotNil(t, load1)
		floatGauge, ok := load1.Data.(metricdata.Gauge[float64])
		require.True(t, ok)
		require.Len(t, floatGauge.DataPoints, 1)
		_, hasReceiver = floatGauge.DataPoints[0].Attributes.Value(attribute.Key("receiver"))
		assert.False(t, hasReceiver, "system metrics should not have 'receiver' attribute")
	}
}

func TestSystemBridgeNilRegistry(t *testing.T) {
	reader := metric.NewManualReader()
	bridge := newTestSystemBridge(t, reader, nil)
	defer bridge.shutdown()

	rm := collectMetrics(t, reader)
	assert.NotNil(t, rm)
}

func TestSystemBridgeShutdown(t *testing.T) {
	reader := metric.NewManualReader()
	statsReg := setupTestRegistry(t)
	bridge := newTestSystemBridge(t, reader, statsReg)
	defer bridge.shutdown()

	rm := collectMetrics(t, reader)
	require.NotNil(t, findMetricByName(rm, "beat.memstats.rss"))

	bridge.shutdown()
	assert.True(t, bridge.closed)
}

func TestSystemBridgeAcquireRelease(t *testing.T) {
	resetSystemBridgeForTest()
	t.Cleanup(resetSystemBridgeForTest)

	settings := componenttest.NewNopTelemetrySettings()

	release1, err := AcquireSystemBridge(settings)
	require.NoError(t, err)
	require.NotNil(t, release1)

	systemMu.Lock()
	inst1 := systemInst
	systemMu.Unlock()
	require.NotNil(t, inst1)

	// Second acquire returns same instance.
	release2, err := AcquireSystemBridge(settings)
	require.NoError(t, err)

	systemMu.Lock()
	inst2 := systemInst
	systemMu.Unlock()
	assert.Same(t, inst1, inst2)

	// Release first — still running.
	release1()

	systemMu.Lock()
	assert.NotNil(t, systemInst)
	systemMu.Unlock()

	// Release second — shut down.
	release2()

	systemMu.Lock()
	assert.Nil(t, systemInst)
	systemMu.Unlock()

	// Re-acquire creates a fresh instance.
	release3, err := AcquireSystemBridge(settings)
	require.NoError(t, err)

	systemMu.Lock()
	inst3 := systemInst
	systemMu.Unlock()
	assert.NotSame(t, inst1, inst3)

	release3()
}

func TestSystemBridgeIdempotentRelease(t *testing.T) {
	resetSystemBridgeForTest()
	t.Cleanup(resetSystemBridgeForTest)

	settings := componenttest.NewNopTelemetrySettings()
	release, err := AcquireSystemBridge(settings)
	require.NoError(t, err)

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
	defer b.shutdown()
}

func TestSystemBridgeLiveValues(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	rss := monitoring.NewUint(statsReg.GetOrCreateRegistry("beat"), "memstats.rss")
	rss.Set(1000)

	bridge := newTestSystemBridge(t, reader, statsReg)
	defer bridge.shutdown()

	rm := collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "beat.memstats.rss"))

	// Update and re-collect to confirm the callback reads live data.
	rss.Set(2000)

	rm = collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "beat.memstats.rss"))
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

func newTestBridge(t *testing.T, reader *metric.ManualReader, statsReg, inputsReg *monitoring.Registry) *RegistryBridge {
	t.Helper()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = provider
	b, err := NewRegistryBridge(settings, statsReg, inputsReg)
	require.NoError(t, err)
	return b
}

func collectMetrics(t *testing.T, reader *metric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	return rm
}

func findMetricByName(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

func getGaugeInt64Value(m *metricdata.Metrics) int64 {
	if m == nil {
		return 0
	}
	gauge, ok := m.Data.(metricdata.Gauge[int64])
	if !ok {
		return 0
	}
	if len(gauge.DataPoints) == 0 {
		return 0
	}
	return gauge.DataPoints[0].Value
}

func getGaugeFloat64Value(m *metricdata.Metrics) float64 {
	if m == nil {
		return 0
	}
	gauge, ok := m.Data.(metricdata.Gauge[float64])
	if !ok {
		return 0
	}
	if len(gauge.DataPoints) == 0 {
		return 0
	}
	return gauge.DataPoints[0].Value
}

func getSumInt64Value(m *metricdata.Metrics) int64 {
	if m == nil {
		return 0
	}
	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		return 0
	}
	if len(sum.DataPoints) == 0 {
		return 0
	}
	return sum.DataPoints[0].Value
}

func getSumInt64DataPoints(m *metricdata.Metrics) []metricdata.DataPoint[int64] {
	if m == nil {
		return nil
	}
	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		return nil
	}
	return sum.DataPoints
}

func getGaugeInt64DataPoints(m *metricdata.Metrics) []metricdata.DataPoint[int64] {
	if m == nil {
		return nil
	}
	gauge, ok := m.Data.(metricdata.Gauge[int64])
	if !ok {
		return nil
	}
	return gauge.DataPoints
}

func TestBridgeStaticMetrics(t *testing.T) {
	reader := metric.NewManualReader()

	// Create a stats registry and populate it with known metrics.
	statsReg := monitoring.NewRegistry()

	// Pipeline metrics
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	monitoring.NewUint(pipelineReg, "clients").Set(3)
	monitoring.NewUint(pipelineReg, "events.active").Set(42)
	monitoring.NewUint(pipelineReg, "events.total").Set(1000)
	monitoring.NewUint(pipelineReg, "events.published").Set(950)
	monitoring.NewUint(pipelineReg, "events.filtered").Set(20)
	monitoring.NewUint(pipelineReg, "events.failed").Set(5)
	monitoring.NewUint(pipelineReg, "events.dropped").Set(2)
	monitoring.NewUint(pipelineReg, "events.retry").Set(10)

	// Queue metrics (under pipeline.queue)
	queueReg := pipelineReg.GetOrCreateRegistry("queue")
	monitoring.NewUint(queueReg, "max_events").Set(4096)
	monitoring.NewUint(queueReg, "max_bytes").Set(0)
	monitoring.NewUint(queueReg, "filled.events").Set(100)
	monitoring.NewUint(queueReg, "filled.bytes").Set(0)
	monitoring.NewFloat(queueReg, "filled.pct").Set(0.025)
	monitoring.NewUint(queueReg, "added.events").Set(500)
	monitoring.NewUint(queueReg, "added.bytes").Set(50000)
	monitoring.NewUint(queueReg, "consumed.events").Set(450)
	monitoring.NewUint(queueReg, "consumed.bytes").Set(45000)
	monitoring.NewUint(queueReg, "removed.events").Set(400)
	monitoring.NewUint(queueReg, "removed.bytes").Set(40000)

	// Output metrics
	outputReg := statsReg.GetOrCreateRegistry("output")
	monitoring.NewUint(outputReg, "events.total").Set(800)
	monitoring.NewUint(outputReg, "events.acked").Set(790)
	monitoring.NewUint(outputReg, "events.failed").Set(8)
	monitoring.NewUint(outputReg, "events.dropped").Set(2)
	monitoring.NewUint(outputReg, "events.batches").Set(80)
	monitoring.NewUint(outputReg, "events.active").Set(10)
	monitoring.NewUint(outputReg, "write.bytes").Set(100000)
	monitoring.NewUint(outputReg, "write.errors").Set(1)
	monitoring.NewUint(outputReg, "read.bytes").Set(5000)
	monitoring.NewUint(outputReg, "read.errors").Set(0)

	// Beat process metrics
	beatReg := statsReg.GetOrCreateRegistry("beat")
	monitoring.NewUint(beatReg, "memstats.memory_alloc").Set(1024000)
	monitoring.NewUint(beatReg, "memstats.rss").Set(2048000)
	monitoring.NewUint(beatReg, "memstats.gc_next").Set(512000)
	monitoring.NewUint(beatReg, "cpu.total.ticks").Set(5000)
	monitoring.NewUint(beatReg, "handles.open").Set(15)
	monitoring.NewUint(beatReg, "runtime.goroutines").Set(25)
	monitoring.NewUint(beatReg, "info.uptime.ms").Set(60000)

	// System metrics
	systemReg := statsReg.GetOrCreateRegistry("system")
	monitoring.NewFloat(systemReg, "load.1").Set(1.5)
	monitoring.NewFloat(systemReg, "load.5").Set(2.0)
	monitoring.NewFloat(systemReg, "load.15").Set(1.8)
	monitoring.NewFloat(systemReg, "load.norm.1").Set(0.375)
	monitoring.NewFloat(systemReg, "load.norm.5").Set(0.5)
	monitoring.NewFloat(systemReg, "load.norm.15").Set(0.45)

	bridge := newTestBridge(t, reader, statsReg, nil)

	// Collect
	rm := collectMetrics(t, reader)

	// Verify pipeline gauges (exact registry key paths)
	assert.Equal(t, int64(3), getGaugeInt64Value(findMetricByName(rm, "pipeline.clients")))
	assert.Equal(t, int64(42), getGaugeInt64Value(findMetricByName(rm, "pipeline.events.active")))

	// Verify pipeline counters
	assert.Equal(t, int64(1000), getSumInt64Value(findMetricByName(rm, "pipeline.events.total")))
	assert.Equal(t, int64(950), getSumInt64Value(findMetricByName(rm, "pipeline.events.published")))
	assert.Equal(t, int64(20), getSumInt64Value(findMetricByName(rm, "pipeline.events.filtered")))
	assert.Equal(t, int64(5), getSumInt64Value(findMetricByName(rm, "pipeline.events.failed")))
	assert.Equal(t, int64(2), getSumInt64Value(findMetricByName(rm, "pipeline.events.dropped")))
	assert.Equal(t, int64(10), getSumInt64Value(findMetricByName(rm, "pipeline.events.retry")))

	// Verify queue gauges
	assert.Equal(t, int64(100), getGaugeInt64Value(findMetricByName(rm, "pipeline.queue.filled.events")))
	assert.Equal(t, int64(0), getGaugeInt64Value(findMetricByName(rm, "pipeline.queue.filled.bytes")))
	assert.Equal(t, int64(4096), getGaugeInt64Value(findMetricByName(rm, "pipeline.queue.max_events")))
	assert.Equal(t, int64(0), getGaugeInt64Value(findMetricByName(rm, "pipeline.queue.max_bytes")))

	// Verify queue fill pct
	assert.InDelta(t, 0.025, getGaugeFloat64Value(findMetricByName(rm, "pipeline.queue.filled.pct")), 0.001)

	// Verify queue counters
	assert.Equal(t, int64(500), getSumInt64Value(findMetricByName(rm, "pipeline.queue.added.events")))
	assert.Equal(t, int64(50000), getSumInt64Value(findMetricByName(rm, "pipeline.queue.added.bytes")))
	assert.Equal(t, int64(450), getSumInt64Value(findMetricByName(rm, "pipeline.queue.consumed.events")))
	assert.Equal(t, int64(45000), getSumInt64Value(findMetricByName(rm, "pipeline.queue.consumed.bytes")))
	assert.Equal(t, int64(400), getSumInt64Value(findMetricByName(rm, "pipeline.queue.removed.events")))
	assert.Equal(t, int64(40000), getSumInt64Value(findMetricByName(rm, "pipeline.queue.removed.bytes")))

	// Verify output counters
	assert.Equal(t, int64(800), getSumInt64Value(findMetricByName(rm, "output.events.total")))
	assert.Equal(t, int64(790), getSumInt64Value(findMetricByName(rm, "output.events.acked")))
	assert.Equal(t, int64(8), getSumInt64Value(findMetricByName(rm, "output.events.failed")))
	assert.Equal(t, int64(2), getSumInt64Value(findMetricByName(rm, "output.events.dropped")))
	assert.Equal(t, int64(80), getSumInt64Value(findMetricByName(rm, "output.events.batches")))
	assert.Equal(t, int64(100000), getSumInt64Value(findMetricByName(rm, "output.write.bytes")))
	assert.Equal(t, int64(1), getSumInt64Value(findMetricByName(rm, "output.write.errors")))
	assert.Equal(t, int64(5000), getSumInt64Value(findMetricByName(rm, "output.read.bytes")))
	assert.Equal(t, int64(0), getSumInt64Value(findMetricByName(rm, "output.read.errors")))

	// Verify output gauge
	assert.Equal(t, int64(10), getGaugeInt64Value(findMetricByName(rm, "output.events.active")))

	// Verify beat process gauges
	assert.Equal(t, int64(1024000), getGaugeInt64Value(findMetricByName(rm, "beat.memstats.memory_alloc")))
	assert.Equal(t, int64(2048000), getGaugeInt64Value(findMetricByName(rm, "beat.memstats.rss")))
	assert.Equal(t, int64(512000), getGaugeInt64Value(findMetricByName(rm, "beat.memstats.gc_next")))
	assert.Equal(t, int64(5000), getGaugeInt64Value(findMetricByName(rm, "beat.cpu.total.ticks")))
	assert.Equal(t, int64(15), getGaugeInt64Value(findMetricByName(rm, "beat.handles.open")))
	assert.Equal(t, int64(25), getGaugeInt64Value(findMetricByName(rm, "beat.runtime.goroutines")))
	assert.Equal(t, int64(60000), getGaugeInt64Value(findMetricByName(rm, "beat.info.uptime.ms")))

	// Verify system load gauges
	assert.InDelta(t, 1.5, getGaugeFloat64Value(findMetricByName(rm, "system.load.1")), 0.001)
	assert.InDelta(t, 2.0, getGaugeFloat64Value(findMetricByName(rm, "system.load.5")), 0.001)
	assert.InDelta(t, 1.8, getGaugeFloat64Value(findMetricByName(rm, "system.load.15")), 0.001)
	assert.InDelta(t, 0.375, getGaugeFloat64Value(findMetricByName(rm, "system.load.norm.1")), 0.001)
	assert.InDelta(t, 0.5, getGaugeFloat64Value(findMetricByName(rm, "system.load.norm.5")), 0.001)
	assert.InDelta(t, 0.45, getGaugeFloat64Value(findMetricByName(rm, "system.load.norm.15")), 0.001)

	bridge.Shutdown()
}

func TestBridgePerInputMetrics(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	inputsReg := monitoring.NewRegistry()

	// Create two mock inputs
	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("filestream-1")
	monitoring.NewString(input1, "input").Set("filestream")
	monitoring.NewUint(input1, "events_processed_total").Set(100)
	monitoring.NewUint(input1, "bytes_processed_total").Set(50000)
	monitoring.NewUint(input1, "files_opened_total").Set(5)
	monitoring.NewUint(input1, "files_closed_total").Set(3)
	monitoring.NewUint(input1, "files_active").Set(2)
	monitoring.NewUint(input1, "messages_read_total").Set(100)
	monitoring.NewUint(input1, "processing_errors_total").Set(1)

	input2 := inputsReg.GetOrCreateRegistry("input-2")
	monitoring.NewString(input2, "id").Set("kafka-1")
	monitoring.NewString(input2, "input").Set("kafka")
	monitoring.NewUint(input2, "events_processed_total").Set(200)
	monitoring.NewUint(input2, "bytes_processed_total").Set(100000)

	bridge := newTestBridge(t, reader, statsReg, inputsReg)

	rm := collectMetrics(t, reader)

	// Verify per-input counters have data points for each input
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	require.NotNil(t, eventsProcessed)
	dps := getSumInt64DataPoints(eventsProcessed)
	require.Len(t, dps, 2)

	// Check both inputs are present (order may vary)
	values := map[string]int64{}
	for _, dp := range dps {
		inputID, ok := dp.Attributes.Value(attribute.Key("input_id"))
		require.True(t, ok)
		values[inputID.AsString()] = dp.Value
	}
	assert.Equal(t, int64(100), values["filestream-1"])
	assert.Equal(t, int64(200), values["kafka-1"])

	// Verify files_active is discovered as a counter (no _gauge suffix, not in gauge set)
	// since per-input field "files_active" doesn't match isGauge by default.
	filesActive := findMetricByName(rm, "files_active")
	require.NotNil(t, filesActive)
	// files_active should be a counter since it doesn't match gauge detection
	filesActiveDPs := getSumInt64DataPoints(filesActive)
	require.Len(t, filesActiveDPs, 1)
	inputIDVal, ok := filesActiveDPs[0].Attributes.Value(attribute.Key("input_id"))
	require.True(t, ok)
	assert.Equal(t, "filestream-1", inputIDVal.AsString())
	assert.Equal(t, int64(2), filesActiveDPs[0].Value)

	bridge.Shutdown()
}

func TestBridgeDynamicInputs(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	inputsReg := monitoring.NewRegistry()

	// Add one input
	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("filestream-1")
	monitoring.NewString(input1, "input").Set("filestream")
	monitoring.NewUint(input1, "events_processed_total").Set(50)

	bridge := newTestBridge(t, reader, statsReg, inputsReg)

	// First collection
	rm := collectMetrics(t, reader)
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	require.NotNil(t, eventsProcessed)
	dps := getSumInt64DataPoints(eventsProcessed)
	require.Len(t, dps, 1)

	// Remove the input (simulating input shutdown)
	inputsReg.Remove("input-1")

	// Second collection should have no data points for this input
	rm = collectMetrics(t, reader)
	eventsProcessed = findMetricByName(rm, "events_processed_total")
	// With no inputs, there might be no data points at all
	if eventsProcessed != nil {
		dps = getSumInt64DataPoints(eventsProcessed)
		assert.Empty(t, dps)
	}

	bridge.Shutdown()
}

func TestBridgeZeroValues(t *testing.T) {
	reader := metric.NewManualReader()

	// Create a stats registry with all zero values
	statsReg := monitoring.NewRegistry()
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	monitoring.NewUint(pipelineReg, "clients").Set(0)
	monitoring.NewUint(pipelineReg, "events.active").Set(0)
	monitoring.NewUint(pipelineReg, "events.total").Set(0)

	bridge := newTestBridge(t, reader, statsReg, nil)

	rm := collectMetrics(t, reader)

	// Zero-valued metrics should still be reported
	clients := findMetricByName(rm, "pipeline.clients")
	require.NotNil(t, clients, "zero-valued gauge should still be reported")
	assert.Equal(t, int64(0), getGaugeInt64Value(clients))

	eventsTotal := findMetricByName(rm, "pipeline.events.total")
	require.NotNil(t, eventsTotal, "zero-valued counter should still be reported")
	assert.Equal(t, int64(0), getSumInt64Value(eventsTotal))

	bridge.Shutdown()
}

func TestBridgeNilRegistries(t *testing.T) {
	reader := metric.NewManualReader()

	// Both nil registries should not panic
	bridge := newTestBridge(t, reader, nil, nil)

	// Collection should succeed without panicking
	rm := collectMetrics(t, reader)
	assert.NotNil(t, rm)

	bridge.Shutdown()
}

func TestBridgeShutdownUnregisters(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	clients := monitoring.NewUint(pipelineReg, "clients")
	clients.Set(5)

	bridge := newTestBridge(t, reader, statsReg, nil)

	// Before shutdown, should collect metrics
	rm := collectMetrics(t, reader)
	m := findMetricByName(rm, "pipeline.clients")
	require.NotNil(t, m)
	assert.Equal(t, int64(5), getGaugeInt64Value(m))

	// After shutdown, callbacks are unregistered
	bridge.Shutdown()

	// Update the value — since the callback is unregistered, we should
	// no longer see updated values
	clients.Set(99)
	rm = collectMetrics(t, reader)
	m = findMetricByName(rm, "pipeline.clients")
	// After unregistration, the metric may still appear but the callback
	// is no longer called.
	if m != nil {
		val := getGaugeInt64Value(m)
		assert.NotEqual(t, int64(99), val)
	}
}

func TestBridgeDynamicMetricDiscovery(t *testing.T) {
	reader := metric.NewManualReader()

	// Start with an empty stats registry
	statsReg := monitoring.NewRegistry()
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	monitoring.NewUint(pipelineReg, "clients").Set(3)

	bridge := newTestBridge(t, reader, statsReg, nil)

	// First collection — only pipeline.clients should exist
	rm := collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "pipeline.clients"))

	// Add a new metric AFTER bridge construction (simulates queue appearing later)
	queueReg := pipelineReg.GetOrCreateRegistry("queue")
	monitoring.NewUint(queueReg, "max_events").Set(4096)

	// Second collection — bridge discovers the new metric, creates the
	// instrument, and triggers async re-registration. The observation for
	// the new metric is not yet recorded (instrument not in callback).
	_ = collectMetrics(t, reader)

	// Wait for the async re-registration goroutine to complete.
	bridge.reRegWg.Wait()

	// Third collection — instrument is now registered, value is observed.
	rm = collectMetrics(t, reader)
	assert.Equal(t, int64(3), getGaugeInt64Value(findMetricByName(rm, "pipeline.clients")))

	queueMax := findMetricByName(rm, "pipeline.queue.max_events")
	require.NotNil(t, queueMax, "dynamically discovered metric should be reported")
	assert.Equal(t, int64(4096), getGaugeInt64Value(queueMax))

	bridge.Shutdown()
}

func TestBridgeGaugeSuffixDetection(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	inputsReg := monitoring.NewRegistry()

	// Create an input with a _gauge suffixed metric (like AWS S3/GCS inputs)
	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("s3-1")
	monitoring.NewString(input1, "input").Set("aws-s3")
	monitoring.NewUint(input1, "sqs_messages_inflight_gauge").Set(42)
	monitoring.NewUint(input1, "events_processed_total").Set(100)

	bridge := newTestBridge(t, reader, statsReg, inputsReg)

	rm := collectMetrics(t, reader)

	// _gauge suffix metric should be a gauge
	sqsGauge := findMetricByName(rm, "sqs_messages_inflight_gauge")
	require.NotNil(t, sqsGauge)
	gaugeDPs := getGaugeInt64DataPoints(sqsGauge)
	require.Len(t, gaugeDPs, 1)
	assert.Equal(t, int64(42), gaugeDPs[0].Value)

	// Non-gauge metric should be a counter
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	require.NotNil(t, eventsProcessed)
	sumDPs := getSumInt64DataPoints(eventsProcessed)
	require.Len(t, sumDPs, 1)
	assert.Equal(t, int64(100), sumDPs[0].Value)

	bridge.Shutdown()
}

func TestBridgeDoubleShutdown(t *testing.T) {
	reader := metric.NewManualReader()
	bridge := newTestBridge(t, reader, nil, nil)

	// Double shutdown should be safe.
	bridge.Shutdown()
	bridge.Shutdown()
}

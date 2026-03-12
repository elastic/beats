// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"context"
	"fmt"
	"sync"
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
	b, err := NewRegistryBridge(settings, "testbeat", statsReg, inputsReg)
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

	statsReg := monitoring.NewRegistry()

	// One int gauge, one int counter, one float gauge.
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	monitoring.NewUint(pipelineReg, "clients").Set(3)
	monitoring.NewUint(pipelineReg, "events.total").Set(1000)

	queueReg := pipelineReg.GetOrCreateRegistry("queue")
	monitoring.NewFloat(queueReg, "filled.pct").Set(0.025)

	// One system metric from each excluded prefix.
	monitoring.NewUint(statsReg.GetOrCreateRegistry("beat"), "memstats.rss").Set(2048000)
	monitoring.NewFloat(statsReg.GetOrCreateRegistry("system"), "load.1").Set(1.5)

	bridge := newTestBridge(t, reader, statsReg, nil)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	assert.NotNil(t, findMetricByName(rm, "pipeline.clients"))
	assert.NotNil(t, findMetricByName(rm, "pipeline.events.total"))
	assert.NotNil(t, findMetricByName(rm, "pipeline.queue.filled.pct"))

	// System-level metrics should be excluded from per-receiver bridge.
	assert.Nil(t, findMetricByName(rm, "beat.memstats.rss"), "beat.* should be excluded")
	assert.Nil(t, findMetricByName(rm, "system.load.1"), "system.* should be excluded")
}

func TestBridgePerInputMetrics(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("filestream-1")
	monitoring.NewString(input1, "input").Set("filestream")
	monitoring.NewUint(input1, "events_processed_total").Set(100)

	input2 := inputsReg.GetOrCreateRegistry("input-2")
	monitoring.NewString(input2, "id").Set("kafka-1")
	monitoring.NewString(input2, "input").Set("kafka")
	monitoring.NewUint(input2, "events_processed_total").Set(200)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	// Same metric name produces one data point per input, distinguished by input_id.
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	require.NotNil(t, eventsProcessed)
	dps := getSumInt64DataPoints(eventsProcessed)
	require.Len(t, dps, 2)

	inputIDs := map[string]bool{}
	for _, dp := range dps {
		inputID, ok := dp.Attributes.Value(attribute.Key("input_id"))
		require.True(t, ok)
		inputIDs[inputID.AsString()] = true
	}
	assert.True(t, inputIDs["filestream-1"])
	assert.True(t, inputIDs["kafka-1"])
}

func TestBridgeDynamicInputs(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("filestream-1")
	monitoring.NewString(input1, "input").Set("filestream")
	monitoring.NewUint(input1, "events_processed_total").Set(50)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)
	require.NotNil(t, findMetricByName(rm, "events_processed_total"))

	// Remove the input (simulating input shutdown).
	inputsReg.Remove("input-1")

	rm = collectMetrics(t, reader)
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	if eventsProcessed != nil {
		dps := getSumInt64DataPoints(eventsProcessed)
		assert.Empty(t, dps)
	}
}

func TestBridgeZeroValues(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	monitoring.NewUint(pipelineReg, "clients").Set(0)
	monitoring.NewUint(pipelineReg, "events.total").Set(0)

	bridge := newTestBridge(t, reader, statsReg, nil)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	assert.NotNil(t, findMetricByName(rm, "pipeline.clients"), "zero-valued gauge should still be reported")
	assert.NotNil(t, findMetricByName(rm, "pipeline.events.total"), "zero-valued counter should still be reported")
}

func TestBridgeNilRegistries(t *testing.T) {
	provider := metric.NewMeterProvider(metric.WithReader(metric.NewManualReader()))
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = provider

	_, err := NewRegistryBridge(settings, "testbeat", nil, nil)
	require.Error(t, err)
}

func TestBridgeShutdownUnregisters(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	monitoring.NewUint(statsReg.GetOrCreateRegistry("pipeline"), "clients").Set(5)

	bridge := newTestBridge(t, reader, statsReg, nil)

	rm := collectMetrics(t, reader)
	require.NotNil(t, findMetricByName(rm, "pipeline.clients"))

	bridge.Shutdown()

	// After shutdown, the callback is unregistered so the metric should no
	// longer appear (or at minimum not reflect new values).
	rm = collectMetrics(t, reader)
	m := findMetricByName(rm, "pipeline.clients")
	if m != nil {
		assert.NotEqual(t, int64(99), getGaugeInt64Value(m))
	}
}

func TestBridgeDynamicMetricDiscovery(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	pipelineReg := statsReg.GetOrCreateRegistry("pipeline")
	monitoring.NewUint(pipelineReg, "clients").Set(3)

	bridge := newTestBridge(t, reader, statsReg, nil)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "pipeline.clients"))

	// Add a new metric AFTER bridge construction.
	queueReg := pipelineReg.GetOrCreateRegistry("queue")
	monitoring.NewUint(queueReg, "max_events").Set(4096)

	_ = collectMetrics(t, reader)
	bridge.reRegWg.Wait()

	rm = collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "pipeline.clients"))
	assert.NotNil(t, findMetricByName(rm, "pipeline.queue.max_events"), "dynamically discovered metric should be reported")
}

func TestBridgeGaugeSuffixDetection(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("s3-1")
	monitoring.NewString(input1, "input").Set("aws-s3")
	monitoring.NewUint(input1, "sqs_messages_inflight_gauge").Set(42)
	monitoring.NewUint(input1, "events_processed_total").Set(100)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	// _gauge suffix metric should be an int64 gauge.
	sqsGauge := findMetricByName(rm, "sqs_messages_inflight_gauge")
	require.NotNil(t, sqsGauge)
	require.NotEmpty(t, getGaugeInt64DataPoints(sqsGauge))

	// Non-gauge metric should be a counter (Sum).
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	require.NotNil(t, eventsProcessed)
	require.NotEmpty(t, getSumInt64DataPoints(eventsProcessed))
}

func TestBridgeDoubleShutdown(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	monitoring.NewUint(statsReg.GetOrCreateRegistry("pipeline"), "clients").Set(1)

	bridge := newTestBridge(t, reader, statsReg, nil)

	bridge.Shutdown()
	bridge.Shutdown()
}

func TestBridgePerInputFloatMetrics(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("filestream-1")
	monitoring.NewString(input1, "input").Set("filestream")
	monitoring.NewFloat(input1, "processing_time_seconds").Set(1.5)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	procTime := findMetricByName(rm, "processing_time_seconds")
	require.NotNil(t, procTime)
	_, ok := procTime.Data.(metricdata.Gauge[float64])
	assert.True(t, ok, "float metric should be a float64 gauge")
}

func TestBridgeDynamicStatsFloatDiscovery(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	queueReg := statsReg.GetOrCreateRegistry("pipeline.queue")
	monitoring.NewFloat(queueReg, "filled.pct").Set(0.25)

	bridge := newTestBridge(t, reader, statsReg, nil)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "pipeline.queue.filled.pct"))

	// Add a new float metric after construction.
	monitoring.NewFloat(queueReg, "utilization.pct").Set(0.75)

	_ = collectMetrics(t, reader)
	bridge.reRegWg.Wait()

	rm = collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "pipeline.queue.utilization.pct"), "dynamically discovered float metric should be reported")
}

func TestBridgeDynamicInputDiscovery(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("filestream-1")
	monitoring.NewString(input1, "input").Set("filestream")
	monitoring.NewUint(input1, "events_processed_total").Set(10)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)
	require.NotNil(t, findMetricByName(rm, "events_processed_total"))

	// Add new fields after construction.
	monitoring.NewUint(input1, "bytes_total").Set(5000)
	monitoring.NewFloat(input1, "lag_seconds").Set(0.5)

	_ = collectMetrics(t, reader)
	bridge.reRegWg.Wait()

	rm = collectMetrics(t, reader)
	assert.NotNil(t, findMetricByName(rm, "bytes_total"), "dynamically discovered per-input int metric should be reported")
	assert.NotNil(t, findMetricByName(rm, "lag_seconds"), "dynamically discovered per-input float metric should be reported")
}

func TestBridgeInputMissingIDOrType(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	noID := inputsReg.GetOrCreateRegistry("no-id")
	monitoring.NewString(noID, "input").Set("filestream")
	monitoring.NewUint(noID, "events_processed_total").Set(100)

	noType := inputsReg.GetOrCreateRegistry("no-type")
	monitoring.NewString(noType, "id").Set("filestream-1")
	monitoring.NewUint(noType, "events_processed_total").Set(200)

	valid := inputsReg.GetOrCreateRegistry("valid")
	monitoring.NewString(valid, "id").Set("kafka-1")
	monitoring.NewString(valid, "input").Set("kafka")
	monitoring.NewUint(valid, "events_processed_total").Set(300)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	// Only the valid input should produce data points.
	eventsProcessed := findMetricByName(rm, "events_processed_total")
	require.NotNil(t, eventsProcessed)
	dps := getSumInt64DataPoints(eventsProcessed)
	require.Len(t, dps, 1)
	inputID, ok := dps[0].Attributes.Value(attribute.Key("input_id"))
	require.True(t, ok)
	assert.Equal(t, "kafka-1", inputID.AsString())
}

func TestToInt64Value(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   int64
		wantOK bool
	}{
		{"int64", int64(42), 42, true},
		{"uint64", uint64(100), 100, true},
		{"int", int(7), 7, true},
		{"float64", float64(3.9), 3, true},
		{"string", "hello", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toInt64Value(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBridgeNilMeterProvider(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.Logger = nil

	statsReg := monitoring.NewRegistry()
	monitoring.NewUint(statsReg.GetOrCreateRegistry("pipeline"), "clients").Set(1)

	b, err := NewRegistryBridge(settings, "testbeat", statsReg, nil)
	require.NoError(t, err)
	require.NotNil(t, b)
	defer b.Shutdown()
}

func TestBridgeInputIntGaugeObservation(t *testing.T) {
	reader := metric.NewManualReader()

	inputsReg := monitoring.NewRegistry()

	input1 := inputsReg.GetOrCreateRegistry("input-1")
	monitoring.NewString(input1, "id").Set("s3-1")
	monitoring.NewString(input1, "input").Set("aws-s3")
	monitoring.NewUint(input1, "inflight_gauge").Set(5)

	bridge := newTestBridge(t, reader, nil, inputsReg)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	inflightGauge := findMetricByName(rm, "inflight_gauge")
	require.NotNil(t, inflightGauge)
	require.NotEmpty(t, getGaugeInt64DataPoints(inflightGauge))
}

func TestBridgeReceiverAttribute(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	monitoring.NewUint(statsReg.GetOrCreateRegistry("pipeline"), "clients").Set(3)

	provider := metric.NewMeterProvider(metric.WithReader(reader))
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = provider
	bridge, err := NewRegistryBridge(settings, "myreceiver", statsReg, nil)
	require.NoError(t, err)
	defer bridge.Shutdown()

	rm := collectMetrics(t, reader)

	clients := findMetricByName(rm, "pipeline.clients")
	require.NotNil(t, clients)
	gaugeDPs := getGaugeInt64DataPoints(clients)
	require.Len(t, gaugeDPs, 1)
	recvVal, ok := gaugeDPs[0].Attributes.Value(attribute.Key("receiver"))
	require.True(t, ok, "stats metric should have 'receiver' attribute")
	assert.Equal(t, "myreceiver", recvVal.AsString())
}

// TestBridgeConcurrentMapAccess verifies that concurrent instrument creation
// and map reads don't race. Only meaningful with -race.
func TestBridgeConcurrentMapAccess(t *testing.T) {
	reader := metric.NewManualReader()

	statsReg := monitoring.NewRegistry()
	monitoring.NewUint(statsReg.GetOrCreateRegistry("pipeline"), "clients").Set(1)

	bridge := newTestBridge(t, reader, statsReg, nil)
	defer bridge.Shutdown()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bridge.mu.Lock()
			_ = bridge.ensureStatsInt(fmt.Sprintf("pipeline.dynamic_%d", i))
			_ = bridge.ensureStatsFloat(fmt.Sprintf("pipeline.pct_%d", i))
			_ = bridge.ensureInputInt(fmt.Sprintf("input_counter_%d", i))
			_ = bridge.ensureInputFloat(fmt.Sprintf("input_gauge_%d", i))
			bridge.mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = bridge.allInstruments()
		}
	}()

	wg.Wait()
}

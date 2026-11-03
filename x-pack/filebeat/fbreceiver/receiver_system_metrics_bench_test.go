// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// BenchmarkNReceiverTelemetry measures what each additional receiver costs to
// start and to have its metrics collected, independent of any input work.
//
// Each receiver runs one filestream input over a path that matches nothing, so
// the measurement is the cost of the Beat itself, which is what every receiver
// pays. Receivers report to a real OTel SDK MeterProvider, and a collection
// (the periodic reader's tick) is driven by hand.
//
// Reported metrics (lower is better):
//   - startup-B/receiver : bytes allocated creating and starting one receiver
//   - collect-B/op       : bytes allocated by one collection across the fleet
//   - collect-allocs/op  : allocations made by one collection across the fleet
//
// ns/op is the time of one fleet-wide collection.
func BenchmarkNReceiverTelemetry(b *testing.B) {
	for _, n := range []int{1, 10, 50} {
		b.Run(fmt.Sprintf("receivers=%d", n), func(b *testing.B) {
			benchNReceiverTelemetry(b, n)
		})
	}
}

func benchNReceiverTelemetry(b *testing.B, nReceivers int) {
	homeDir := b.TempDir()
	dataDir := b.TempDir()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	b.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	host := &oteltest.MockHost{}
	factory := NewFactoryWithSettings(Settings{Home: homeDir, Data: dataDir})

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	receivers := make([]receiver.Logs, 0, nReceivers)
	for r := range nReceivers {
		settings := receiver.Settings{}
		settings.ID = component.NewIDWithName(factory.Type(), fmt.Sprintf("bench%d", r))
		settings.Logger = zap.NewNop()
		settings.MeterProvider = mp

		cfg := &Config{
			Beatconfig: map[string]any{
				"filebeat": map[string]any{
					"inputs": []map[string]any{{
						"type":    "filestream",
						"id":      fmt.Sprintf("telemetry-%d", r),
						"enabled": true,
						"paths":   []string{homeDir + fmt.Sprintf("/nomatch-%d/*.log", r)},
					}},
				},
				"logging":                 map[string]any{"level": "error"},
				"path.home":               homeDir,
				"path.data":               dataDir,
				"management.otel.enabled": true,
			},
		}

		rcvr, err := factory.CreateLogs(b.Context(), settings, cfg, consumertest.NewNop())
		require.NoError(b, err)
		require.NoError(b, rcvr.Start(b.Context(), host))
		receivers = append(receivers, rcvr)
	}
	b.Cleanup(func() {
		for _, rcvr := range receivers {
			_ = rcvr.Shutdown(context.Background())
		}
	})

	runtime.ReadMemStats(&after)
	startup := float64(after.TotalAlloc-before.TotalAlloc) / float64(nReceivers)

	var rm metricdata.ResourceMetrics
	// The bridge creates instruments for metrics it discovers after start on a
	// separate goroutine after a collection. Collecting again while that is in
	// flight can deadlock it against the SDK (the bridge holds its lock while
	// creating an instrument, the SDK holds its lock while running the bridge's
	// callback), so let it settle before measuring.
	for range 3 {
		require.NoError(b, reader.Collect(b.Context(), &rm))
		time.Sleep(time.Second)
	}

	b.ReportAllocs()
	runtime.ReadMemStats(&before)
	for b.Loop() {
		require.NoError(b, reader.Collect(b.Context(), &rm))
	}
	runtime.ReadMemStats(&after)

	b.ReportMetric(startup, "startup-B/receiver")
	b.ReportMetric(float64(after.TotalAlloc-before.TotalAlloc)/float64(b.N), "collect-B/op")
	b.ReportMetric(float64(after.Mallocs-before.Mallocs)/float64(b.N), "collect-allocs/op")
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"
	"runtime"
	"runtime/metrics"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

const (
	benchQueueSize    = 3200
	benchMinBatchSize = 1600
	benchNetworkDelay = 100 * time.Millisecond
	benchBatchTimeout = 10 * time.Second
	// benchWarmupBatches is the number of batches delivered before the timed
	// window starts. One batch is not enough: the beat queue, otelconsumer
	// goroutines, and Go's allocator all need a few cycles to reach steady state.
	benchWarmupBatches = 5
)

// readUserCPUSecs returns the cumulative user-mode CPU time consumed by this
// process, in seconds. Snapshot it before and after the timed window and
// subtract to get CPU spent during the benchmark.
func readUserCPUSecs() float64 {
	s := []metrics.Sample{{Name: "/cpu/classes/user:cpu-seconds"}}
	metrics.Read(s)
	if s[0].Value.Kind() == metrics.KindFloat64 {
		return s[0].Value.Float64()
	}
	return 0
}

// BenchmarkReceiverQueueInteraction measures end-to-end throughput of the
// filebeat receiver pipeline when backed by an exporterhelper batching queue
// configured to match the Elasticsearch exporter defaults used from the beat
// receiver (queue_size=3200, block_on_overflow=true, wait_for_result=true,
// batch min/max_size=1600 items, flush_timeout=10s).
//
// The pipeline under test is:
//
//	filebeat benchmark input
//	  → beat in-memory queue (flush.min_events=1600, flush.timeout=10s defaults)
//	  → otelconsumer
//	  → exporterhelper queue (items, queue_size=3200, wait_for_result=true)
//	      batcher (min/max=1600, flush_timeout=10s)
//	  → test sink (simulates 100ms network round-trip)
//
// Each b.N "op" corresponds to one batch delivery at the test sink.
// ns/op is the time to accumulate and deliver one 1600-event batch;
// events/s is the sustained throughput during the timed portion.
func BenchmarkReceiverQueueInteraction(b *testing.B) {
	tmpDir := b.TempDir()
	ctx := b.Context()

	batches := make(chan struct{}, 1024)
	var totalReceived atomic.Int64

	done := make(chan struct{})

	qCfg := exporterhelper.NewDefaultQueueConfig()
	qCfg.QueueSize = benchQueueSize
	qCfg.BlockOnOverflow = true
	qCfg.WaitForResult = true
	qCfg.NumConsumers = 1
	qCfg.Batch = configoptional.Some(exporterhelper.BatchConfig{
		FlushTimeout: benchBatchTimeout,
		Sizer:        exporterhelper.RequestSizerTypeItems,
		MinSize:      benchMinBatchSize,
		MaxSize:      benchMinBatchSize,
	})

	expSettings := exportertest.NewNopSettings(component.MustNewType("benchsink"))
	expSettings.Logger = zap.NewNop()

	sink, err := exporterhelper.NewLogs(ctx, expSettings, &struct{}{},
		func(_ context.Context, ld plog.Logs) error {
			time.Sleep(benchNetworkDelay)
			select {
			case batches <- struct{}{}:
				totalReceived.Add(int64(ld.LogRecordCount()))
			case <-done:
			}
			return nil
		},
		exporterhelper.WithQueue(configoptional.Some(qCfg)),
	)
	require.NoError(b, err)

	host := &oteltest.MockHost{}
	require.NoError(b, sink.Start(ctx, host))
	b.Cleanup(func() { _ = sink.Shutdown(context.Background()) })

	factory := NewFactoryWithSettings(Settings{Home: tmpDir})
	rcvrSettings := receiver.Settings{}
	rcvrSettings.ID = component.NewIDWithName(factory.Type(), "bench")
	rcvrSettings.Logger = zap.NewNop()

	cfg := &Config{
		Beatconfig: map[string]any{
			"filebeat": map[string]any{
				"inputs": []map[string]any{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "bench",
						// No count → unlimited; no eps → max throughput.
					},
				},
			},
			"logging":   map[string]any{"level": "error"},
			"path.home": tmpDir,
		},
	}

	rcvr, err := factory.CreateLogs(ctx, rcvrSettings, cfg, sink)
	require.NoError(b, err)
	require.NoError(b, rcvr.Start(ctx, host))
	b.Cleanup(func() { _ = rcvr.Shutdown(context.Background()) })

	// Warm up: wait for the first batch so the pipeline is running at steady
	// state before the timed portion begins.
	select {
	case <-batches:
	case <-time.After(30 * time.Second):
		b.Fatal("pipeline did not deliver the first batch within 30s")
	}

	cpuBefore := readUserCPUSecs()
	receivedAtStart := totalReceived.Load()
	b.ResetTimer()
	for b.Loop() {
		select {
		case <-batches:
		case <-ctx.Done():
			b.Fatal("context cancelled during benchmark")
		}
	}
	b.StopTimer()
	close(done)

	cpuSecs := readUserCPUSecs() - cpuBefore
	timedEvents := totalReceived.Load() - receivedAtStart
	elapsed := b.Elapsed().Seconds()
	b.ReportMetric(float64(timedEvents)/elapsed, "events/s")
	if timedEvents > 0 {
		b.ReportMetric(cpuSecs*1e9/float64(timedEvents), "ns_cpu/event")
	}
}

// BenchmarkNReceivers measures aggregate throughput when N concurrent filebeat
// receiver instances all share one batching sink. This exercises queue
// contention and shows whether per-receiver throughput degrades as concurrency
// grows.
//
// Additional reported metrics:
//   - heap_bytes/receiver: steady-state heap delta divided by receiver count
//   - goroutines: goroutine count at end of timed portion
func BenchmarkNReceivers(b *testing.B) {
	for _, n := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("receivers=%d", n), func(b *testing.B) {
			benchNReceivers(b, n)
		})
	}
}

func benchNReceivers(b *testing.B, nReceivers int) {
	ctx := b.Context()

	batches := make(chan struct{}, 1024)
	var totalReceived atomic.Int64
	done := make(chan struct{})

	qCfg := exporterhelper.NewDefaultQueueConfig()
	qCfg.QueueSize = benchQueueSize
	qCfg.BlockOnOverflow = true
	qCfg.WaitForResult = true
	qCfg.NumConsumers = 1
	qCfg.Batch = configoptional.Some(exporterhelper.BatchConfig{
		FlushTimeout: benchBatchTimeout,
		Sizer:        exporterhelper.RequestSizerTypeItems,
		MinSize:      benchMinBatchSize,
		MaxSize:      benchMinBatchSize,
	})

	expSettings := exportertest.NewNopSettings(component.MustNewType("benchsink"))
	expSettings.Logger = zap.NewNop()

	sink, err := exporterhelper.NewLogs(ctx, expSettings, &struct{}{},
		func(_ context.Context, ld plog.Logs) error {
			time.Sleep(benchNetworkDelay)
			select {
			case batches <- struct{}{}:
				totalReceived.Add(int64(ld.LogRecordCount()))
			case <-done:
			}
			return nil
		},
		exporterhelper.WithQueue(configoptional.Some(qCfg)),
	)
	require.NoError(b, err)

	host := &oteltest.MockHost{}
	require.NoError(b, sink.Start(ctx, host))
	b.Cleanup(func() { _ = sink.Shutdown(context.Background()) })

	factory := NewFactoryWithSettings(Settings{Home: b.TempDir()})

	for i := range nReceivers {
		rcvrDir := b.TempDir()
		rcvrSettings := receiver.Settings{}
		rcvrSettings.ID = component.NewIDWithName(factory.Type(), fmt.Sprintf("bench%d", i))
		rcvrSettings.Logger = zap.NewNop()

		cfg := &Config{
			Beatconfig: map[string]any{
				"filebeat": map[string]any{
					"inputs": []map[string]any{
						{
							"type":    "benchmark",
							"enabled": true,
							"message": "bench",
						},
					},
				},
				"logging":   map[string]any{"level": "error"},
				"path.home": rcvrDir,
			},
		}

		rcvr, err := factory.CreateLogs(ctx, rcvrSettings, cfg, sink)
		require.NoError(b, err)
		require.NoError(b, rcvr.Start(ctx, host))
		b.Cleanup(func() { _ = rcvr.Shutdown(context.Background()) })
	}

	// Warm up: drain benchWarmupBatches before measuring.
	warmup := time.After(30 * time.Second)
	for range benchWarmupBatches {
		select {
		case <-batches:
		case <-warmup:
			b.Fatal("pipeline did not complete warmup within 30s")
		}
	}

	// GC before the snapshot so ReadMemStats reflects the live set, not
	// objects that haven't been collected yet from the warmup phase.
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	cpuBefore := readUserCPUSecs()
	receivedAtStart := totalReceived.Load()
	b.ResetTimer()
	for b.Loop() {
		select {
		case <-batches:
		case <-ctx.Done():
			b.Fatal("context cancelled during benchmark")
		}
	}
	b.StopTimer()
	close(done)

	cpuSecs := readUserCPUSecs() - cpuBefore

	// GC again so memAfter reflects only what the running receivers retain,
	// not transient allocations from in-flight batches.
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	timedEvents := totalReceived.Load() - receivedAtStart
	elapsed := b.Elapsed().Seconds()
	b.ReportMetric(float64(timedEvents)/elapsed, "events/s")
	if timedEvents > 0 {
		b.ReportMetric(cpuSecs*1e9/float64(timedEvents), "ns_cpu/event")
	}
	if memAfter.HeapInuse >= memBefore.HeapInuse {
		b.ReportMetric(float64(memAfter.HeapInuse-memBefore.HeapInuse)/float64(nReceivers), "heap_bytes/receiver")
	}
	b.ReportMetric(float64(runtime.NumGoroutine()), "goroutines")
}

// BenchmarkMixedReceivers runs 1 heavy receiver (unlimited EPS) alongside N
// light receivers (each capped at 500 events/s). The heavy receiver keeps the
// shared downstream pipeline saturated; the benchmark verifies that the
// rate-limited producers still receive fair delivery — no light-receiver event
// should be starved even when the heavy receiver competes for the same queue.
//
// The batcher accumulates events from all receivers into 1600-item batches.
// Because WaitForResult=true, the heavy receiver's ConsumeLogs blocks when the
// queue is full, giving the light receivers slots each time a batch drains.
//
// Reported metrics:
//   - events/s:       aggregate throughput across all receivers
//   - heavy_events/s: throughput from the heavy receiver
//   - light_events/s: combined throughput from all light receivers
//   - light_pct:      percentage of delivered events from light receivers
//
// A non-zero light_pct confirms that light receivers are not starved.
func BenchmarkMixedReceivers(b *testing.B) {
	for _, nLight := range []int{1, 3, 7} {
		b.Run(fmt.Sprintf("light=%d", nLight), func(b *testing.B) {
			benchMixedReceivers(b, nLight)
		})
	}
}

func benchMixedReceivers(b *testing.B, nLight int) {
	const lightEPS = 500 // events/s per light receiver

	ctx := b.Context()

	batches := make(chan struct{}, 1024)
	var heavyReceived, lightReceived atomic.Int64
	done := make(chan struct{})

	qCfg := exporterhelper.NewDefaultQueueConfig()
	qCfg.QueueSize = benchQueueSize
	qCfg.BlockOnOverflow = true
	qCfg.WaitForResult = true
	qCfg.NumConsumers = 1
	qCfg.Batch = configoptional.Some(exporterhelper.BatchConfig{
		FlushTimeout: benchBatchTimeout,
		Sizer:        exporterhelper.RequestSizerTypeItems,
		MinSize:      benchMinBatchSize,
		MaxSize:      benchMinBatchSize,
	})

	expSettings := exportertest.NewNopSettings(component.MustNewType("benchsink"))
	expSettings.Logger = zap.NewNop()

	sink, err := exporterhelper.NewLogs(ctx, expSettings, &struct{}{},
		func(_ context.Context, ld plog.Logs) error {
			var heavy, light int64
			for _, rl := range ld.ResourceLogs().All() {
				for _, sl := range rl.ScopeLogs().All() {
					for _, lr := range sl.LogRecords().All() {
						if isHeavyRecord(lr.Body()) {
							heavy++
						} else {
							light++
						}
					}
				}
			}
			time.Sleep(benchNetworkDelay)
			select {
			case batches <- struct{}{}:
				heavyReceived.Add(heavy)
				lightReceived.Add(light)
			case <-done:
			}
			return nil
		},
		exporterhelper.WithQueue(configoptional.Some(qCfg)),
	)
	require.NoError(b, err)

	host := &oteltest.MockHost{}
	require.NoError(b, sink.Start(ctx, host))
	b.Cleanup(func() { _ = sink.Shutdown(context.Background()) })

	factory := NewFactoryWithSettings(Settings{Home: b.TempDir()})

	// Heavy receiver: no EPS cap, drives the pipeline to saturation.
	heavySettings := receiver.Settings{}
	heavySettings.ID = component.NewIDWithName(factory.Type(), "bench-heavy")
	heavySettings.Logger = zap.NewNop()
	heavyCfg := &Config{
		Beatconfig: map[string]any{
			"filebeat": map[string]any{
				"inputs": []map[string]any{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "heavy",
					},
				},
			},
			"logging":   map[string]any{"level": "error"},
			"path.home": b.TempDir(),
		},
	}
	heavyRcvr, err := factory.CreateLogs(ctx, heavySettings, heavyCfg, sink)
	require.NoError(b, err)
	require.NoError(b, heavyRcvr.Start(ctx, host))
	b.Cleanup(func() { _ = heavyRcvr.Shutdown(context.Background()) })

	// Light receivers: rate-capped so they don't fill the queue alone, but
	// they must make continuous forward progress regardless.
	for i := range nLight {
		lightSettings := receiver.Settings{}
		lightSettings.ID = component.NewIDWithName(factory.Type(), fmt.Sprintf("bench-light%d", i))
		lightSettings.Logger = zap.NewNop()
		lightCfg := &Config{
			Beatconfig: map[string]any{
				"filebeat": map[string]any{
					"inputs": []map[string]any{
						{
							"type":    "benchmark",
							"enabled": true,
							"message": "light",
							"eps":     lightEPS,
						},
					},
				},
				"logging":   map[string]any{"level": "error"},
				"path.home": b.TempDir(),
			},
		}
		lightRcvr, err := factory.CreateLogs(ctx, lightSettings, lightCfg, sink)
		require.NoError(b, err)
		require.NoError(b, lightRcvr.Start(ctx, host))
		b.Cleanup(func() { _ = lightRcvr.Shutdown(context.Background()) })
	}

	// Warm up: the heavy receiver fills batches quickly; drain benchWarmupBatches
	// so the light receivers have also had a chance to begin producing.
	warmup := time.After(30 * time.Second)
	for range benchWarmupBatches {
		select {
		case <-batches:
		case <-warmup:
			b.Fatal("pipeline did not complete warmup within 30s")
		}
	}

	cpuBefore := readUserCPUSecs()
	heavyAtStart := heavyReceived.Load()
	lightAtStart := lightReceived.Load()
	b.ResetTimer()
	for b.Loop() {
		select {
		case <-batches:
		case <-ctx.Done():
			b.Fatal("context cancelled during benchmark")
		}
	}
	b.StopTimer()
	close(done)

	cpuSecs := readUserCPUSecs() - cpuBefore
	elapsed := b.Elapsed().Seconds()
	timedHeavy := float64(heavyReceived.Load() - heavyAtStart)
	timedLight := float64(lightReceived.Load() - lightAtStart)
	timedTotal := timedHeavy + timedLight

	b.ReportMetric(timedTotal/elapsed, "events/s")
	b.ReportMetric(timedHeavy/elapsed, "heavy_events/s")
	b.ReportMetric(timedLight/elapsed, "light_events/s")
	if timedTotal > 0 {
		b.ReportMetric(100*timedLight/timedTotal, "light_pct")
		b.ReportMetric(cpuSecs*1e9/timedTotal, "ns_cpu/event")
	}
	b.ReportMetric(float64(runtime.NumGoroutine()), "goroutines")
}

// BenchmarkGetDebounce sweeps the slabqueue Get debounce window across a range
// of values for both a heavy-only (8 receivers, where the Publish-goroutine
// fan-out peaks) and a mixed (1 heavy + 3 light) scenario, so the best value —
// balancing throughput, goroutine count, and CPU per event — can be chosen from
// data rather than guessed. Run with:
//
//	go test -run '^$' -bench BenchmarkGetDebounce -benchtime=10s ./x-pack/filebeat/fbreceiver/
func BenchmarkGetDebounce(b *testing.B) {
	debounces := []time.Duration{
		0,
		250 * time.Microsecond,
		500 * time.Microsecond,
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
	}
	for _, d := range debounces {
		b.Run(fmt.Sprintf("heavy8/debounce=%s", d), func(b *testing.B) {
			prev := slabqueue.DefaultGetDebounce
			slabqueue.DefaultGetDebounce = d
			b.Cleanup(func() { slabqueue.DefaultGetDebounce = prev })
			benchNReceivers(b, 8)
		})
		b.Run(fmt.Sprintf("mixed3/debounce=%s", d), func(b *testing.B) {
			prev := slabqueue.DefaultGetDebounce
			slabqueue.DefaultGetDebounce = d
			b.Cleanup(func() { slabqueue.DefaultGetDebounce = prev })
			benchMixedReceivers(b, 3)
		})
	}
}

// isHeavyRecord returns true when the log record body carries the "heavy"
// message tag produced by the heavy benchmark receiver.
func isHeavyRecord(body pcommon.Value) bool {
	if body.Type() != pcommon.ValueTypeMap {
		return false
	}
	msg, ok := body.Map().Get("message")
	return ok && msg.Str() == "heavy"
}

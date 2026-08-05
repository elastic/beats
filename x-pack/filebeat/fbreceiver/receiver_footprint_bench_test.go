// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/filebeat/input/filestream"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// BenchmarkNReceiverIdleFootprint measures the steady-state resource footprint
// of N filebeatreceiver instances, each running one filestream input over its
// own small log file, all sharing a single path.data. That is the layout
// elastic-agent produces: one receiver per input stream, every receiver of a
// component pointed at the same BeatDataPath(comp.ID).
//
// Unlike BenchmarkNReceivers this does not measure throughput. It measures what
// each additional receiver costs once its file has been fully ingested and the
// input is idle (open and tailed), which is the state a real deployment spends
// almost all of its time in.
//
// Reported metrics (lower is better):
//   - goroutines/receiver : resident goroutines per additional receiver
//   - heap-B/receiver     : live heap bytes retained per receiver
//   - goroutines          : absolute goroutine count of the idle fleet
func BenchmarkNReceiverIdleFootprint(b *testing.B) {
	for _, n := range []int{1, 10, 50} {
		b.Run(fmt.Sprintf("receivers=%d", n), func(b *testing.B) {
			benchNReceiverFootprint(b, n, 1)
		})
	}
}

// BenchmarkNReceiverRegistryFootprint measures how registry state scales with
// the number of receivers sharing one path.data.
//
// It exists because readStates loads every key carrying the input type's prefix,
// not just the keys belonging to the input doing the loading. With a private
// in-memory view per receiver, N receivers each hold all N receivers' states —
// quadratic in the receiver count. Giving each receiver enough files that the
// state table dominates the heap makes that scaling visible.
//
// Reported metric:
//   - heap-B/file : live heap bytes per tracked file across the whole fleet.
//     Flat as receivers are added means the state table is shared; growing
//     linearly means each receiver holds a private copy.
func BenchmarkNReceiverRegistryFootprint(b *testing.B) {
	const filesPerReceiver = 300
	for _, n := range []int{2, 10} {
		b.Run(fmt.Sprintf("receivers=%d", n), func(b *testing.B) {
			benchNReceiverFootprint(b, n, filesPerReceiver)
		})
	}
}

func benchNReceiverFootprint(b *testing.B, nReceivers, filesPerReceiver int) {
	const linesPerFile = 50
	wantEvents := int64(nReceivers * filesPerReceiver * linesPerFile)

	var goroutinesPer, heapPer, heapPerFile, goroutines float64

	for b.Loop() {
		// A fresh data dir per iteration so every receiver re-reads its file
		// from offset 0 instead of resuming a registry from the last iteration.
		dataDir := b.TempDir()
		homeDir := b.TempDir()
		logDir := b.TempDir()

		// Each receiver gets its own subdirectory so its glob matches only its
		// own files, exactly as one receiver per input stream would be configured.
		paths := make([]string, nReceivers)
		for r := range nReceivers {
			paths[r] = writeBenchLogFiles(b, logDir, r, filesPerReceiver, linesPerFile)
		}

		host := &oteltest.MockHost{}
		factory := NewFactoryWithSettings(Settings{Home: homeDir, Data: dataDir})

		// First pass: ingest everything so the registry on disk ends up holding
		// every receiver's state, then shut the fleet down. This is what makes
		// the measured pass realistic — a receiver in a long-running deployment
		// almost always starts against a registry that already describes all of
		// its co-tenants, and that is the state each one loads into memory.
		sink := &countingLogsConsumer{}
		stopFleet := startFleet(b, factory, host, sink, paths, homeDir, dataDir)
		require.Eventually(b,
			func() bool { return sink.count.Load() >= wantEvents },
			180*time.Second, 50*time.Millisecond,
			"all receivers should fully ingest their files")
		stopFleet()

		runtime.GC()
		baseGoroutines := runtime.NumGoroutine()
		var base runtime.MemStats
		runtime.ReadMemStats(&base)

		// Second pass: restart against the warm registry and measure. The files
		// are already at EOF, so nothing is re-ingested; what is measured is what
		// the fleet retains simply by being up.
		restartSink := &countingLogsConsumer{}
		stopFleet = startFleet(b, factory, host, restartSink, paths, homeDir, dataDir)
		time.Sleep(fleetSettleTime)

		runtime.GC()
		runtime.GC()
		idleGoroutines := runtime.NumGoroutine()
		var idle runtime.MemStats
		runtime.ReadMemStats(&idle)

		dumpIdleGoroutines(b, nReceivers)

		// A GC between the two snapshots can leave the idle fleet holding less
		// than the baseline, so the difference is taken as signed rather than
		// underflowing uint64.
		var heapDelta float64
		if idle.HeapAlloc > base.HeapAlloc {
			heapDelta = float64(idle.HeapAlloc - base.HeapAlloc)
		}
		goroutinesPer = float64(idleGoroutines-baseGoroutines) / float64(nReceivers)
		heapPer = heapDelta / float64(nReceivers)
		heapPerFile = heapDelta / float64(nReceivers*filesPerReceiver)
		goroutines = float64(idleGoroutines)

		stopFleet()
	}

	b.ReportMetric(goroutinesPer, "goroutines/receiver")
	b.ReportMetric(heapPer, "heap-B/receiver")
	b.ReportMetric(goroutines, "goroutines")
	if filesPerReceiver > 1 {
		b.ReportMetric(heapPerFile, "heap-B/file")
	}
}

// fleetSettleTime is how long the restarted fleet is given to reach steady
// state: prospectors complete their first scan, harvesters open their files and
// park at EOF, and the publisher pipelines go quiet.
const fleetSettleTime = 10 * time.Second

// startFleet starts one receiver per path, each with its own filestream input
// but all sharing homeDir and dataDir, and returns a function that shuts them
// all down. Sharing dataDir is the point: elastic-agent gives every receiver of
// a component the same path.data.
func startFleet(
	b *testing.B,
	factory receiver.Factory,
	host component.Host,
	sink *countingLogsConsumer,
	paths []string,
	homeDir, dataDir string,
) func() {
	b.Helper()

	receivers := make([]receiver.Logs, 0, len(paths))
	for r, path := range paths {
		rcvrSettings := receiver.Settings{}
		rcvrSettings.ID = component.NewIDWithName(factory.Type(), fmt.Sprintf("bench%d", r))
		rcvrSettings.Logger = zap.NewNop()

		cfg := &Config{
			Beatconfig: map[string]any{
				"filebeat": map[string]any{
					"inputs": []map[string]any{
						{
							"type":    "filestream",
							"id":      fmt.Sprintf("footprint-%d", r),
							"enabled": true,
							"paths":   []string{path},
							// Everything else is left at its default, most
							// importantly the fingerprint file identity: the
							// footprint worth tracking is the one the default
							// configuration produces. writeBenchLogFiles keeps
							// every file above DefaultFingerprintSize so each
							// one is actually harvested.
							"prospector": map[string]any{
								"scanner": map[string]any{
									"check_interval": "100ms",
								},
							},
						},
					},
				},
				"logging":   map[string]any{"level": "error"},
				"path.home": homeDir,
				"path.data": dataDir,
				// Run under the OtelManager, as a receiver started by
				// elastic-agent does. It marks the beat as under agent, which
				// changes what the pipeline builds — the footprint is only
				// meaningful if it is measured in that mode.
				"management.otel.enabled": true,
			},
		}

		rcvr, err := factory.CreateLogs(b.Context(), rcvrSettings, cfg, sink)
		require.NoError(b, err)
		require.NoError(b, rcvr.Start(b.Context(), host))
		receivers = append(receivers, rcvr)
	}

	return func() {
		for _, rcvr := range receivers {
			shutdownCtx, cancel := context.WithTimeout(b.Context(), 60*time.Second)
			require.NoError(b, rcvr.Shutdown(shutdownCtx))
			cancel()
		}
	}
}

// dumpIdleGoroutines writes the idle fleet's goroutine profile to the path in
// FBRECEIVER_GOROUTINE_DUMP (with the receiver count appended), so the
// per-receiver goroutine cost can be attributed to the code that creates it.
// It is a no-op unless that variable is set.
func dumpIdleGoroutines(b *testing.B, nReceivers int) {
	b.Helper()

	dir := os.Getenv("FBRECEIVER_GOROUTINE_DUMP")
	if dir == "" {
		return
	}

	root, err := os.OpenRoot(dir)
	require.NoError(b, err)
	defer root.Close()

	f, err := root.Create(fmt.Sprintf("goroutines-%d.txt", nReceivers))
	require.NoError(b, err)
	defer f.Close()

	// debug=1 aggregates identical stacks with a count, which is what makes the
	// per-receiver breakdown readable.
	require.NoError(b, pprof.Lookup("goroutine").WriteTo(f, 1))
}

// writeBenchLogFiles writes count log files for one receiver into their own
// subdirectory and returns the glob matching them.
//
// The receiver and file numbers are part of every line so that no two files
// share a fingerprint, and each file is required to be larger than
// DefaultFingerprintSize — below that the default fingerprint identity cannot
// take a file, and it would simply never be harvested.
func writeBenchLogFiles(b *testing.B, dir string, receiver, count, lines int) string {
	b.Helper()

	rcvrDir := filepath.Join(dir, fmt.Sprintf("receiver-%d", receiver))
	require.NoError(b, os.MkdirAll(rcvrDir, 0o750))

	for n := range count {
		path := filepath.Join(rcvrDir, fmt.Sprintf("bench-%d.log", n))
		f, err := os.Create(path)
		require.NoError(b, err)

		written := 0
		for l := range lines {
			c, err := fmt.Fprintf(f, "receiver %d file %d line %d: a log line of a fairly typical length\n",
				receiver, n, l)
			require.NoError(b, err)
			written += c
		}
		require.NoError(b, f.Sync())
		require.NoError(b, f.Close())

		require.Greaterf(b, int64(written), filestream.DefaultFingerprintSize,
			"file %s is too small to be fingerprinted, increase linesPerFile", path)
	}

	return filepath.Join(rcvrDir, "*.log")
}

// countingLogsConsumer counts delivered log records without ever blocking, so
// the receivers reach idle instead of stalling on a slow sink.
type countingLogsConsumer struct{ count atomic.Int64 }

func (c *countingLogsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *countingLogsConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	c.count.Add(int64(ld.LogRecordCount()))
	return nil
}

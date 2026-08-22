// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package filestream

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// BenchmarkIdleResourceFootprint measures the steady-state goroutine count and
// live heap of the filestream input once a fleet of files has been read and is
// idle (open and tailed, but not being written to). This is the scenario where
// the run-to-EOF-then-park model wins over one-goroutine-per-open-file: idle
// files cost no goroutine and no live reader pipeline.
//
// Reported metrics (lower is better):
//   - goroutines/file : resident goroutines per open idle file
//   - heap-B/file      : live heap bytes retained per open idle file
//   - idle-heap-MiB    : total live heap retained by the idle fleet
func BenchmarkIdleResourceFootprint(b *testing.B) {
	for _, count := range []int{2000, 10000} {
		b.Run(fmt.Sprintf("logfile/%d_idle_files", count), func(b *testing.B) {
			benchmarkLogfileIdleFootprint(b, count, 5)
		})
		b.Run(fmt.Sprintf("cursor/%d_idle_sources", count), func(b *testing.B) {
			benchmarkCursorIdleFootprint(b, count)
		})
	}
}

func benchmarkLogfileIdleFootprint(b *testing.B, fileCount, linesPerFile int) {
	logger := logp.NewNopLogger()

	dir := b.TempDir()
	for range fileCount {
		generateFile(b, dir, linesPerFile)
	}
	wantEvents := int64(fileCount * linesPerFile)

	// Realistic tailing defaults: native identity (cheap), keep reading the file
	// (no close_on_eof), and the default close_inactive (5m) keeps idle files open
	// well past this short measurement.
	cfg := fmt.Sprintf(`
type: filestream
prospector.scanner.check_interval: 100ms
prospector.scanner.fingerprint.enabled: false
file_identity.native: ~
paths:
  - %s
`, filepath.Join(dir, "*"))
	c, err := conf.NewConfigWithYAML([]byte(cfg), cfg)
	require.NoError(b, err)

	var gPerFile, bytesPerFile, heapMiB float64
	for i := 0; i < b.N; i++ {
		// A fresh store each iteration so the fleet is re-read from offset 0.
		p := Plugin(logger, createTestStore(b))
		//nolint:errcheck // Close is a cleanup function and never returns an error
		b.Cleanup(p.Manager.(*loginp.InputManager).Close)
		input, err := p.Manager.Create(c)
		require.NoError(b, err)

		ctx, cancel := context.WithCancel(context.Background())
		v2ctx := v2.Context{
			ID:              "footprint",
			Name:            "filestream-test",
			Cancelation:     ctx,
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          logger,
		}

		runtime.GC()
		baseGoroutines := runtime.NumGoroutine()
		var base runtime.MemStats
		runtime.ReadMemStats(&base)

		var count int64
		var runErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			runErr = input.Run(v2ctx, &countingPipeline{count: &count})
		}()

		// Wait for the whole fleet to be ingested, then let it settle into idle.
		require.Eventually(b,
			func() bool { return atomic.LoadInt64(&count) >= wantEvents },
			120*time.Second, 50*time.Millisecond,
			"all files should be fully ingested")
		time.Sleep(3 * time.Second) // settle: harvesters reach EOF and park / back off

		runtime.GC()
		runtime.GC()
		idleGoroutines := runtime.NumGoroutine()
		var idle runtime.MemStats
		runtime.ReadMemStats(&idle)

		gPerFile = float64(idleGoroutines-baseGoroutines) / float64(fileCount)
		bytesPerFile = float64(idle.HeapAlloc-base.HeapAlloc) / float64(fileCount)
		heapMiB = float64(idle.HeapAlloc-base.HeapAlloc) / (1 << 20)

		cancel()
		select {
		case <-done:
		case <-time.After(60 * time.Second):
			b.Fatal("input did not stop after cancellation")
		}
		require.NoError(b, runErr)
	}

	b.ReportMetric(gPerFile, "goroutines/file")
	b.ReportMetric(bytesPerFile, "heap-B/file")
	b.ReportMetric(heapMiB, "idle-heap-MiB")
}

// countingPipeline counts published events without ever stopping the input, so
// the harvesters stay alive and idle after reaching EOF.
type countingPipeline struct{ count *int64 }

func (p *countingPipeline) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return &countingClient{p.count}, nil
}
func (p *countingPipeline) Connect() (beat.Client, error)    { return &countingClient{p.count}, nil }
func (p *countingPipeline) Disconnect(context.Context) error { return nil }

type countingClient struct{ count *int64 }

func (c *countingClient) Publish(beat.Event) { atomic.AddInt64(c.count, 1) }
func (c *countingClient) PublishAll(es []beat.Event) {
	atomic.AddInt64(c.count, int64(len(es)))
}
func (c *countingClient) Close() error { return nil }

// benchmarkCursorIdleFootprint measures the steady-state goroutine count and
// live heap of the cursor InputManager with sourceCount idle sources — sources
// that have started but are blocking, waiting for more data. This is the
// baseline cost of the traditional one-goroutine-per-source model: each source
// holds a goroutine for the lifetime of the input run, even while idle.
func benchmarkCursorIdleFootprint(b *testing.B, sourceCount int) {
	b.Helper()
	logger := logp.NewNopLogger()

	sources := make([]cursor.Source, sourceCount)
	for i := range sourceCount {
		sources[i] = cursorBenchSource(fmt.Sprintf("source-%d", i))
	}

	var gPerSource, bytesPerSource, heapMiB float64
	for range b.N {
		var started sync.WaitGroup
		started.Add(sourceCount)

		manager := &cursor.InputManager{
			Logger:              logger,
			StateStore:          createTestStore(b),
			Type:                "cursor-bench",
			DefaultCleanTimeout: time.Minute,
			Configure: func(_ *conf.C, _ *logp.Logger) ([]cursor.Source, cursor.Input, error) {
				return sources, &idleCursorInput{started: &started}, nil
			},
		}
		b.Cleanup(manager.Close)

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(b, err)

		ctx, cancel := context.WithCancel(context.Background())
		v2ctx := v2.Context{
			ID:              "cursor-footprint",
			Name:            "cursor-bench",
			Cancelation:     ctx,
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          logger,
		}

		runtime.GC()
		baseGoroutines := runtime.NumGoroutine()
		var base runtime.MemStats
		runtime.ReadMemStats(&base)

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = inp.Run(v2ctx, &countingPipeline{count: new(int64)})
		}()

		// Wait for all N source goroutines to start and settle.
		started.Wait()
		time.Sleep(500 * time.Millisecond)

		runtime.GC()
		runtime.GC()
		idleGoroutines := runtime.NumGoroutine()
		var idle runtime.MemStats
		runtime.ReadMemStats(&idle)

		gPerSource = float64(idleGoroutines-baseGoroutines) / float64(sourceCount)
		bytesPerSource = float64(idle.HeapAlloc-base.HeapAlloc) / float64(sourceCount)
		heapMiB = float64(idle.HeapAlloc-base.HeapAlloc) / (1 << 20)

		cancel()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			b.Fatal("cursor input did not stop after cancellation")
		}
	}

	b.ReportMetric(gPerSource, "goroutines/source")
	b.ReportMetric(bytesPerSource, "heap-B/source")
	b.ReportMetric(heapMiB, "idle-heap-MiB")
}

type cursorBenchSource string

func (s cursorBenchSource) Name() string { return string(s) }

type idleCursorInput struct{ started *sync.WaitGroup }

func (i *idleCursorInput) Name() string { return "cursor-bench" }

func (i *idleCursorInput) Test(_ cursor.Source, _ v2.TestContext) error { return nil }

func (i *idleCursorInput) Run(ctx v2.Context, _ cursor.Source, _ cursor.Cursor, _ cursor.Publisher) error {
	i.started.Done()
	<-ctx.Cancelation.Done()
	return nil
}

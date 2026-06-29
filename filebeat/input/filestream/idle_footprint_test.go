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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
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
	for _, fileCount := range []int{2000, 10000} {
		b.Run(fmt.Sprintf("%d_idle_files", fileCount), func(b *testing.B) {
			benchmarkIdleFootprint(b, fileCount, 5)
		})
	}
}

func benchmarkIdleFootprint(b *testing.B, fileCount, linesPerFile int) {
	logger := logp.NewNopLogger()

	dir := b.TempDir()
	for i := 0; i < fileCount; i++ {
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
		input, err := Plugin(logger, createTestStore(b)).Manager.Create(c)
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

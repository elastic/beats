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

package cursor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// BenchmarkIdleResourceFootprint measures the steady-state goroutine count and
// live heap of the cursor InputManager with a fleet of idle sources — sources
// that have started but are blocking, waiting for more data. This is the
// baseline cost of the traditional one-goroutine-per-source model: each source
// holds a goroutine for the lifetime of the input run, even while idle.
//
// Reported metrics (lower is better):
//   - goroutines/source : resident goroutines per idle source
//   - heap-B/source     : live heap bytes retained per idle source
//   - idle-heap-MiB     : total live heap retained by the idle fleet
func BenchmarkIdleResourceFootprint(b *testing.B) {
	for _, count := range []int{2000, 10000} {
		b.Run(fmt.Sprintf("input_manager/%d_idle_sources", count), func(b *testing.B) {
			benchmarkInputManagerIdleFootprint(b, count)
		})
	}
}

func benchmarkInputManagerIdleFootprint(b *testing.B, sourceCount int) {
	b.Helper()
	logger := logp.NewNopLogger()

	sources := make([]Source, sourceCount)
	for i := range sourceCount {
		sources[i] = benchSource(fmt.Sprintf("source-%d", i))
	}

	var gPerSource, bytesPerSource, heapMiB float64
	for range b.N {
		var started sync.WaitGroup
		started.Add(sourceCount)

		manager := &InputManager{
			Logger:              logger,
			StateStore:          makeBenchStateStore(b),
			Type:                "cursor-bench",
			DefaultCleanTimeout: time.Minute,
			Configure: func(_ *conf.C, _ *logp.Logger) ([]Source, Input, error) {
				return sources, &idleBenchInput{started: &started}, nil
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
			_ = inp.Run(v2ctx, &benchPipeline{count: new(int64)})
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

type benchSource string

func (s benchSource) Name() string { return string(s) }

type idleBenchInput struct{ started *sync.WaitGroup }

func (i *idleBenchInput) Name() string { return "cursor-bench" }

func (i *idleBenchInput) Test(_ Source, _ v2.TestContext) error { return nil }

func (i *idleBenchInput) Run(ctx v2.Context, _ Source, _ Cursor, _ Publisher) error {
	i.started.Done()
	<-ctx.Cancelation.Done()
	return nil
}

type benchPipeline struct{ count *int64 }

func (p *benchPipeline) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return &benchClient{p.count}, nil
}
func (p *benchPipeline) Connect() (beat.Client, error)    { return &benchClient{p.count}, nil }
func (p *benchPipeline) Disconnect(context.Context) error { return nil }

type benchClient struct{ count *int64 }

func (c *benchClient) Publish(beat.Event)              { atomic.AddInt64(c.count, 1) }
func (c *benchClient) PublishAll(es []beat.Event)      { atomic.AddInt64(c.count, int64(len(es))) }
func (c *benchClient) Close() error                    { return nil }

func makeBenchStateStore(b *testing.B) testStateStore {
	b.Helper()
	reg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
	s, err := reg.Get("bench")
	require.NoError(b, err)
	b.Cleanup(func() { _ = reg.Close() })
	return testStateStore{Store: s}
}

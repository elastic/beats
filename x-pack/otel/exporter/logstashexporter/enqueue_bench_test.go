// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter/internal"
	"github.com/elastic/elastic-agent-libs/logp"
)

// enqueueWorkPolling is a copy of the sleep-and-retry implementation that
// enqueueWork used before it became a blocking send. It only exists so that
// BenchmarkEnqueueWork can show the difference between the two.
func enqueueWorkPolling(ctx context.Context, queue chan *internal.Work, w *internal.Work) error {
	backoff := 5 * time.Millisecond
	maxBackoff := 250 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case queue <- w:
			return nil
		default:
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff = min(backoff*2, maxBackoff)
			}
		}
	}
}

// inOrderRelease models how the pipeline's queue releases batches: a batch can
// only be released once every older batch is done, so a finished batch sits
// unreleased (with everything attached to its events) behind an older one that
// is still waiting.
type inOrderRelease struct {
	mu         sync.Mutex
	done       map[int64]struct{}
	next       int64 // next sequence number to release
	unreleased int   // done but not released
	peak       int
}

func newInOrderRelease() *inOrderRelease {
	return &inOrderRelease{done: make(map[int64]struct{})}
}

func (r *inOrderRelease) markDone(seq int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done[seq] = struct{}{}
	r.unreleased++
	r.peak = max(r.peak, r.unreleased)
	for {
		if _, ok := r.done[r.next]; !ok {
			return
		}
		delete(r.done, r.next)
		r.next++
		r.unreleased--
	}
}

// BenchmarkEnqueueWork compares how enqueueWork admits batches to a single,
// slow Logstash worker when many ConsumeLogs calls compete for it, as with
// small batches arriving from several inputs.
//
// Each iteration pushes a fixed number of batches from many goroutines through
// a work queue drained by one worker. It reports:
//   - max-wait-ms and p99-wait-ms: how long a single batch waited to be admitted;
//   - peak-unreleased: the most batches that were finished but could not be
//     released because an older batch had not been admitted yet. In the
//     pipeline this is the number of batches (and their events' Private data)
//     held in memory behind a starved one.
func BenchmarkEnqueueWork(b *testing.B) {
	const (
		callers        = 64
		batchesPerIter = 4000
		serviceTime    = 100 * time.Microsecond
	)

	newQueue := func() (*logstashExporter, chan *internal.Work) {
		exp := &logstashExporter{
			logger:    logp.NewNopLogger(),
			workQueue: make(chan *internal.Work, runtime.NumCPU()),
			shutdown:  make(chan struct{}),
		}
		return exp, exp.workQueue
	}

	run := func(b *testing.B, enqueue func(context.Context, *logstashExporter, chan *internal.Work, *internal.Work) error) {
		var maxWait, p99Wait time.Duration
		var peak int
		for range b.N {
			exp, queue := newQueue()
			release := newInOrderRelease()
			seqOf := make(map[*internal.Work]int64, batchesPerIter)
			var seqMu sync.Mutex

			// The single worker.
			workerDone := make(chan struct{})
			go func() {
				defer close(workerDone)
				for range batchesPerIter {
					w := <-queue
					time.Sleep(serviceTime)
					seqMu.Lock()
					seq := seqOf[w]
					seqMu.Unlock()
					release.markDone(seq)
				}
			}()

			var issued atomic.Int64
			waits := make([][]time.Duration, callers)
			var wg sync.WaitGroup
			for c := range callers {
				wg.Go(func() {
					for {
						seq := issued.Add(1) - 1
						if seq >= batchesPerIter {
							return
						}
						w := internal.NewWork(nil)
						seqMu.Lock()
						seqOf[w] = seq
						seqMu.Unlock()
						start := time.Now()
						if err := enqueue(context.Background(), exp, queue, w); err != nil {
							b.Error(err)
							return
						}
						waits[c] = append(waits[c], time.Since(start))
					}
				})
			}
			wg.Wait()
			<-workerDone

			all := slices.Concat(waits...)
			slices.Sort(all)
			maxWait = max(maxWait, all[len(all)-1])
			p99Wait = max(p99Wait, all[len(all)*99/100])
			peak = max(peak, release.peak)
		}
		b.ReportMetric(float64(maxWait)/float64(time.Millisecond), "max-wait-ms")
		b.ReportMetric(float64(p99Wait)/float64(time.Millisecond), "p99-wait-ms")
		b.ReportMetric(float64(peak), "peak-unreleased")
		b.ReportMetric(float64(batchesPerIter), "batches/op")
	}

	b.Run("polling-sleep", func(b *testing.B) {
		run(b, func(ctx context.Context, _ *logstashExporter, q chan *internal.Work, w *internal.Work) error {
			return enqueueWorkPolling(ctx, q, w)
		})
	})
	b.Run("blocking-send", func(b *testing.B) {
		run(b, func(ctx context.Context, exp *logstashExporter, _ chan *internal.Work, w *internal.Work) error {
			return exp.enqueueWork(ctx, w)
		})
	})
}

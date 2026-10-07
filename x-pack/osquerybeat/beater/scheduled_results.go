// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	scheduledResultLimit      = 128
	scheduledResultBytesLimit = 8 << 20
)

type scheduledResult struct {
	result     QueryResult
	namespace  string
	query      QueryInfo
	generation uint64
	bytes      int
}

type scheduledResultMetrics struct {
	overflow, stale, shutdown *monitoring.Uint
	lastWarning               atomic.Int64
	log                       *logp.Logger
}

func newScheduledResultMetrics(reg *monitoring.Registry, log *logp.Logger) *scheduledResultMetrics {
	reg = reg.GetOrCreateRegistry("osquerybeat.scheduled_results")
	return &scheduledResultMetrics{
		overflow: monitoring.NewUint(reg, "dropped.overflow"),
		stale:    monitoring.NewUint(reg, "dropped.stale"),
		shutdown: monitoring.NewUint(reg, "dropped.shutdown"),
		log:      log,
	}
}

func (m *scheduledResultMetrics) drop(counter *monitoring.Uint) {
	counter.Inc()
	now := time.Now().UnixNano()
	last := m.lastWarning.Load()
	if now-last >= int64(time.Minute) && m.lastWarning.CompareAndSwap(last, now) {
		m.log.Warn("Scheduled osquery result dropped; see osquerybeat.scheduled_results.dropped monitoring counters")
	}
}

// scheduledResultWorker owns a bounded FIFO for one osquery run. Outstanding
// accounting includes the result being processed, so a stalled query cannot
// bypass either bound by being removed from the channel.
type scheduledResultWorker struct {
	ctx                context.Context
	cancel             context.CancelFunc
	done               chan struct{}
	pending            chan scheduledResult
	mu                 sync.Mutex
	count, bytes       int
	maxCount, maxBytes int
	metrics            *scheduledResultMetrics
}

func newScheduledResultWorker(ctx context.Context, count, bytes int, metrics *scheduledResultMetrics, handle func(context.Context, scheduledResult)) *scheduledResultWorker {
	ctx, cancel := context.WithCancel(ctx)
	w := &scheduledResultWorker{ctx: ctx, cancel: cancel, done: make(chan struct{}), pending: make(chan scheduledResult, count), maxCount: count, maxBytes: bytes, metrics: metrics}
	go func() {
		defer close(w.done)
		defer func() {
			w.mu.Lock()
			defer w.mu.Unlock()
			for {
				select {
				case res := <-w.pending:
					w.count--
					w.bytes -= res.bytes
					w.metrics.drop(w.metrics.shutdown)
				default:
					return
				}
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-w.pending:
				if ctx.Err() != nil {
					w.metrics.drop(w.metrics.shutdown)
				} else {
					handle(ctx, res)
				}
				w.release(res)
			}
		}
	}()
	return w
}

func (w *scheduledResultWorker) enqueue(res scheduledResult) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx.Err() != nil {
		w.metrics.drop(w.metrics.shutdown)
		return false
	}
	if w.count >= w.maxCount || res.bytes > w.maxBytes-w.bytes {
		w.metrics.drop(w.metrics.overflow)
		return false
	}
	w.count++
	w.bytes += res.bytes
	w.pending <- res // The outstanding count bounds this channel, including in-flight work.
	return true
}

func (w *scheduledResultWorker) release(res scheduledResult) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.count--
	w.bytes -= res.bytes
}

func (w *scheduledResultWorker) stop() { w.cancel(); <-w.done }

// resultSize estimates retained decoded storage, including map and row overhead,
// rather than treating a result document with millions of tiny fields as cheap.
func resultSize(res QueryResult) int {
	size := 256 + len(res.Name) + len(res.Action) + len(res.CalendarTime)
	for _, rows := range [][]map[string]string{res.Hits, res.DiffResults.Added, res.DiffResults.Removed} {
		size += 24 * len(rows)
		for _, row := range rows {
			size += 128
			for key, value := range row {
				size += 128 + len(key) + len(value)
			}
		}
	}
	return size
}

func (bt *osquerybeat) newScheduledLogger(ctx context.Context, cli scheduledQueryClient, configPlugin *ConfigPlugin) (*LoggerPlugin, func()) {
	metrics := bt.resultMetrics
	if metrics == nil {
		metrics = newScheduledResultMetrics(monitoring.NewRegistry(), bt.log)
	}
	worker := newScheduledResultWorker(ctx, scheduledResultLimit, scheduledResultBytesLimit, metrics, func(ctx context.Context, res scheduledResult) {
		bt.handleScheduledResult(ctx, cli, res, metrics)
	})
	plugin := NewLoggerPlugin(bt.log, func(res QueryResult) {
		ns, qi, generation, ok := configPlugin.lookupResultMetadata(res.Name)
		if !ok {
			metrics.drop(metrics.stale)
			return
		}
		worker.enqueue(scheduledResult{result: res, namespace: ns, query: qi, generation: generation, bytes: resultSize(res)})
	})
	plugin.maxResultBytes = scheduledResultBytesLimit
	plugin.dropOversized = func() { metrics.drop(metrics.overflow) }
	return plugin, worker.stop
}

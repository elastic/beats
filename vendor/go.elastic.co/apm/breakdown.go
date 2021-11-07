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

package apm // import "go.elastic.co/apm"

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"go.elastic.co/apm/model"
)

const (
	// breakdownMetricsLimit is the maximum number of breakdown metric
	// buckets to accumulate per reporting period. Metrics are broken
	// down by {transactionType, transactionName, spanType, spanSubtype}
	// tuples.
	breakdownMetricsLimit = 1000

	// appSpanType is the special span type associated with transactions,
	// for reporting transaction self-time.
	appSpanType = "app"

	// Breakdown metric names.
	transactionDurationCountMetricName  = "transaction.duration.count"
	transactionDurationSumMetricName    = "transaction.duration.sum.us"
	transactionBreakdownCountMetricName = "transaction.breakdown.count"
	spanSelfTimeCountMetricName         = "span.self_time.count"
	spanSelfTimeSumMetricName           = "span.self_time.sum.us"
)

type pad32 struct {
	// Zero-sized on 64-bit architectures, 4 bytes on 32-bit.
	_ [(unsafe.Alignof(uint64(0)) % 8) / 4]uintptr
}

var (
	breakdownMetricsLimitWarning = fmt.Sprintf(`
The limit of %d breakdown metricsets has been reached, no new metricsets will be created.
Try to name your transactions so that there are less distinct transaction names.`[1:],
		breakdownMetricsLimit,
	)
)

// spanTimingsKey identifies a span type and subtype, for use as the key in
// spanTimingsMap.
type spanTimingsKey struct {
	spanType    string
	spanSubtype string
}

// spanTiming records the number of times a {spanType, spanSubtype} pair
// has occurred (within the context of a transaction group), along with
// the sum of the span durations.
type spanTiming struct {
	duration int64
	count    uintptr
}

// spanTimingsMap records span timings for a transaction group.
type spanTimingsMap map[spanTimingsKey]spanTiming

// add accumulates the timing for a {spanType, spanSubtype} pair.
func (m spanTimingsMap) add(spanType, spanSubtype string, d time.Duration) {
	k := spanTimingsKey{spanType: spanType, spanSubtype: spanSubtype}
	timing := m[k]
	timing.count++
	timing.duration += int64(d)
	m[k] = timing
}

// reset resets m back to its initial zero state.
func (m spanTimingsMap) reset() {
	for k := range m {
		delete(m, k)
	}
}

// breakdownMetrics holds a pair of breakdown metrics maps. The "active" map
// accumulates new breakdown metrics, and is swapped with the "inactive" map
// just prior to when metrics gathering begins. When metrics gathering
// completes, the inactive map will be empty.
//
// breakdownMetrics may be written to concurrently by the tracer, and any
// number of other goroutines when a transaction cannot be enqueued.
type breakdownMetrics struct {
	enabled bool

	mu               sync.RWMutex
	active, inactive *breakdownMetricsMap
}

func newBreakdownMetrics() *breakdownMetrics {
	return &breakdownMetrics{
		active:   newBreakdownMetricsMap(),
		inactive: newBreakdownMetricsMap(),
	}
}

type breakdownMetricsMap struct {
	mu      sync.RWMutex
	entries int
	m       map[uint64][]*breakdownMetricsMapEntry
	space   []breakdownMetricsMapEntry
}

func newBreakdownMetricsMap() *breakdownMetricsMap {
	return &breakdownMetricsMap{
		m:     make(map[uint64][]*breakdownMetricsMapEntry),
		space: make([]breakdownMetricsMapEntry, breakdownMetricsLimit),
	}
}

type breakdownMetricsMapEntry struct {
	breakdownTiming
	breakdownMetricsKey
}

// breakdownMetricsKey identifies a transaction group, and optionally a
// spanTimingsKey, for recording transaction and span breakdown metrics.
type breakdownMetricsKey struct {
	transactionType string
	transactionName string
	spanTimingsKey
}

func (k breakdownMetricsKey) hash() uint64 {
	h := newFnv1a()
	h.add(k.transactionType)
	h.add(k.transactionName)
	if k.spanType != "" {
		h.add(k.spanType)
	}
	if k.spanSubtype != "" {
		h.add(k.spanSubtype)
	}
	return uint64(h)
}

// breakdownTiming holds breakdown metrics.
type breakdownTiming struct {
	// transaction holds the "transaction.duration" metric values.
	transaction spanTiming

	// Padding to ensure the span field below is 64-bit aligned.
	_ pad32

	// span holds the "span.self_time" metric values.
	span spanTiming

	// breakdownCount records the number of transactions for which we
	// have calculated breakdown metrics. If breakdown metrics are
	// enabled, this will be equal transaction.count.
	breakdownCount uintptr
}

func (lhs *breakdownTiming) accumulate(rhs breakdownTiming) {
	atomic.AddUintptr(&lhs.transaction.count, rhs.transaction.count)
	atomic.AddInt64(&lhs.transaction.duration, rhs.transaction.duration)
	atomic.AddUintptr(&lhs.span.count, rhs.span.count)
	atomic.AddInt64(&lhs.span.duration, rhs.span.duration)
	atomic.AddUintptr(&lhs.breakdownCount, rhs.breakdownCount)
}

// recordTransaction records breakdown metrics for td into m.
//
// recordTransaction returns true if breakdown metrics were
// completely recorded, and false if any metrics were not
// recorded due to the limit being reached.
func (m *breakdownMetrics) recordTransaction(td *TransactionData) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k := breakdownMetricsKey{
		transactionType: td.Type,
		transactionName: td.Name,
	}
	k.spanType = appSpanType

	var breakdownCount int
	var transactionSpanTiming spanTiming
	var transactionDuration = spanTiming{count: 1, duration: int64(td.Duration)}
	if td.breakdownMetricsEnabled {
		breakdownCount = 1
		endTime := td.timestamp.Add(td.Duration)
		transactionSelfTime := td.Duration - td.childrenTimer.finalDuration(endTime)
		transactionSpanTiming = spanTiming{count: 1, duration: int64(transactionSelfTime)}
	}

	if !m.active.record(k, breakdownTiming{
		transaction:    transactionDuration,
		breakdownCount: uintptr(breakdownCount),
		span:           transactionSpanTiming,
	}) {
		// We couldn't record the transaction's metricset, so we won't
		// be able to record spans for that transaction either.
		return false
	}

	ok := true
	for sk, timing := range td.spanTimings {
		k.spanTimingsKey = sk
		ok = ok && m.active.record(k, breakdownTiming{span: timing})
	}
	return ok
}

// record records a single breakdown metric, identified by k.
func (m *breakdownMetricsMap) record(k breakdownMetricsKey, bt breakdownTiming) bool {
	hash := k.hash()
	m.mu.RLock()
	entries, ok := m.m[hash]
	m.mu.RUnlock()
	var offset int
	if ok {
		for offset = range entries {
			if entries[offset].breakdownMetricsKey == k {
				// The append may reallocate the entries, but the
				// entries are pointers into m.activeSpace. Therefore,
				// entries' timings can safely be atomically incremented
				// without holding the read lock.
				entries[offset].breakdownTiming.accumulate(bt)
				return true
			}
		}
		offset++ // where to start searching with the write lock below
	}

	m.mu.Lock()
	entries, ok = m.m[hash]
	if ok {
		for i := range entries[offset:] {
			if entries[offset+i].breakdownMetricsKey == k {
				m.mu.Unlock()
				entries[offset+i].breakdownTiming.accumulate(bt)
				return true
			}
		}
	} else if m.entries >= breakdownMetricsLimit {
		m.mu.Unlock()
		return false
	}
	entry := &m.space[m.entries]
	*entry = breakdownMetricsMapEntry{
		breakdownTiming:     bt,
		breakdownMetricsKey: k,
	}
	m.m[hash] = append(entries, entry)
	m.entries++
	m.mu.Unlock()
	return true
}

// gather is called by builtinMetricsGatherer to gather breakdown metrics.
func (m *breakdownMetrics) gather(out *Metrics) {
	// Hold m.mu only long enough to swap m.active and m.inactive.
	// This will be blocked by metric updates, but that's OK; only
	// metrics gathering will be delayed. After swapping we do not
	// need to hold m.mu, since nothing concurrently accesses
	// m.inactive while the gatherer is iterating over it.
	m.mu.Lock()
	m.active, m.inactive = m.inactive, m.active
	m.mu.Unlock()

	for hash, entries := range m.inactive.m {
		for _, entry := range entries {
			if entry.transaction.count > 0 {
				out.transactionGroupMetrics = append(out.transactionGroupMetrics, &model.Metrics{
					Transaction: model.MetricsTransaction{
						Type: entry.transactionType,
						Name: entry.transactionName,
					},
					Samples: map[string]model.Metric{
						transactionDurationCountMetricName: {
							Value: float64(entry.transaction.count),
						},
						transactionDurationSumMetricName: {
							Value: durationMicros(time.Duration(entry.transaction.duration)),
						},
						transactionBreakdownCountMetricName: {
							Value: float64(entry.breakdownCount),
						},
					},
				})
			}
			if entry.span.count > 0 {
				out.transactionGroupMetrics = append(out.transactionGroupMetrics, &model.Metrics{
					Transaction: model.MetricsTransaction{
						Type: entry.transactionType,
						Name: entry.transactionName,
					},
					Span: model.MetricsSpan{
						Type:    entry.spanType,
						Subtype: entry.spanSubtype,
					},
					Samples: map[string]model.Metric{
						spanSelfTimeCountMetricName: {
							Value: float64(entry.span.count),
						},
						spanSelfTimeSumMetricName: {
							Value: durationMicros(time.Duration(entry.span.duration)),
						},
					},
				})
			}
			entry.breakdownMetricsKey = breakdownMetricsKey{} // release strings
		}
		delete(m.inactive.m, hash)
	}
	m.inactive.entries = 0
}

// childrenTimer tracks time spent by children of a transaction or span.
//
// childrenTimer is not goroutine-safe.
type childrenTimer struct {
	// active holds the number active children.
	active int

	// start holds the timestamp at which active went from zero to one.
	start time.Time

	// totalDuration holds the total duration of time periods in which
	// at least one child was active.
	totalDuration time.Duration
}

func (t *childrenTimer) childStarted(start time.Time) {
	t.active++
	if t.active == 1 {
		t.start = start
	}
}

func (t *childrenTimer) childEnded(end time.Time) {
	t.active--
	if t.active == 0 {
		t.totalDuration += end.Sub(t.start)
	}
}

func (t *childrenTimer) finalDuration(end time.Time) time.Duration {
	if t.active > 0 {
		t.active = 0
		t.totalDuration += end.Sub(t.start)
	}
	return t.totalDuration
}

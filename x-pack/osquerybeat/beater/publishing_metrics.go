// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"sync/atomic"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// Reasons for dropping a queued item, used as counter names.
const (
	dropOverflow = "overflow" // evicted from a full queue
	dropExpired  = "expired"  // older than max_age when the worker reached it
	dropStale    = "stale"    // no active policy or client publishes it
	dropShutdown = "shutdown" // osquerybeat is stopping
	dropTimeout  = "timeout"  // a live action stopped waiting for the output
)

// dropReasons lists the reasons each queue reports.
var dropReasons = map[string][]string{
	"scheduled": {dropOverflow, dropExpired, dropStale, dropShutdown},
	"live":      {dropStale, dropShutdown, dropTimeout},
}

// publishingMetrics reports one publishing queue under
// osquerybeat.publishing.<queue>. dropped.<reason> counts dropped items and
// dropped_rows.<reason> their rows; queue.items and queue.events are the
// current backlog.
type publishingMetrics struct {
	queue                string
	dropped, droppedRows map[string]*monitoring.Uint
	items, events        *monitoring.Int
	log                  *logp.Logger
	lastWarning          atomic.Int64 // Unix nanoseconds
}

func newPublishingMetrics(reg *monitoring.Registry, queue string, log *logp.Logger) *publishingMetrics {
	reg = reg.GetOrCreateRegistry("osquerybeat.publishing." + queue)
	m := &publishingMetrics{
		queue:       queue,
		dropped:     map[string]*monitoring.Uint{},
		droppedRows: map[string]*monitoring.Uint{},
		items:       monitoring.NewInt(reg, "queue.items"),
		events:      monitoring.NewInt(reg, "queue.events"),
		log:         log,
	}
	for _, reason := range dropReasons[queue] {
		m.dropped[reason] = monitoring.NewUint(reg, "dropped."+reason)
		m.droppedRows[reason] = monitoring.NewUint(reg, "dropped_rows."+reason)
	}
	return m
}

// drop counts one dropped item of rows rows.
func (m *publishingMetrics) drop(reason string, rows int) {
	m.dropped[reason].Inc()
	m.dropRows(reason, rows)
}

// dropRows counts rows dropped without dropping an item, such as an event
// the pipeline discards while closing.
func (m *publishingMetrics) dropRows(reason string, rows int) {
	m.droppedRows[reason].Add(uint64(rows))
	now := time.Now().UnixNano()
	last := m.lastWarning.Load()
	if now-last >= int64(time.Minute) && m.lastWarning.CompareAndSwap(last, now) {
		m.log.Warnf("Dropped osquery %s results; see the osquerybeat.publishing.%s.dropped.%s monitoring counter", m.queue, m.queue, reason)
	}
}

func (m *publishingMetrics) setQueue(items, events int) {
	m.items.Set(int64(items))
	m.events.Set(int64(events))
}

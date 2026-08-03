// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"sync"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

// histogramAssemblerMonitoring holds low-cardinality internal metrics for histogram
// assembly under the metricset registry at histogram_assembler.*.
//
// Pending gauges reflect in-memory assembler state (the pending map only). Unpublished
// flush events retained for retry are not included; they are accounted on shutdown via
// shutdown_dropped_total together with pending histograms.
type histogramAssemblerMonitoring struct {
	pending            *monitoring.Uint
	pendingBuckets     *monitoring.Uint
	quietFlushes       *monitoring.Uint
	hardTimeoutFlushes *monitoring.Uint
	partialFlushes     *monitoring.Uint
	capacityRejections *monitoring.Uint
	lateBuckets        *monitoring.Uint
	lateBucketsDropped *monitoring.Uint
	shutdownDropped    *monitoring.Uint
}

type histogramAssemblerMonitoringSlot struct {
	once sync.Once
	m    *histogramAssemblerMonitoring
}

var histogramAssemblerMonitoringSlots sync.Map // *monitoring.Registry -> *histogramAssemblerMonitoringSlot

func registerHistogramAssemblerMonitoring(msReg *monitoring.Registry) *histogramAssemblerMonitoring {
	if msReg == nil {
		return nil
	}
	slotVal, _ := histogramAssemblerMonitoringSlots.LoadOrStore(msReg, &histogramAssemblerMonitoringSlot{})
	slot, ok := slotVal.(*histogramAssemblerMonitoringSlot)
	if !ok {
		return nil
	}
	slot.once.Do(func() {
		reg := msReg.GetOrCreateRegistry("histogram_assembler")
		slot.m = &histogramAssemblerMonitoring{
			pending:            monitoring.NewUint(reg, "pending_gauge"),
			pendingBuckets:     monitoring.NewUint(reg, "pending_buckets_gauge"),
			quietFlushes:       monitoring.NewUint(reg, "quiet_flushes_total"),
			hardTimeoutFlushes: monitoring.NewUint(reg, "hard_timeout_flushes_total"),
			partialFlushes:     monitoring.NewUint(reg, "partial_flushes_total"),
			capacityRejections: monitoring.NewUint(reg, "capacity_rejections_total"),
			lateBuckets:        monitoring.NewUint(reg, "late_buckets_total"),
			lateBucketsDropped: monitoring.NewUint(reg, "late_buckets_dropped_total"),
			shutdownDropped:    monitoring.NewUint(reg, "shutdown_dropped_total"),
		}
	})
	return slot.m
}

func (m *histogramAssemblerMonitoring) setPending(histograms, buckets int) {
	if m == nil {
		return
	}
	// G115: pending counts come from map lengths and are therefore non-negative.
	m.pending.Set(uint64(histograms))     //nolint:gosec
	m.pendingBuckets.Set(uint64(buckets)) //nolint:gosec
}

func (m *histogramAssemblerMonitoring) observeLateBucket(dropped bool) {
	if m == nil {
		return
	}
	m.lateBuckets.Inc()
	if dropped {
		m.lateBucketsDropped.Inc()
	}
}

func (m *histogramAssemblerMonitoring) observeFlush(reason flushReason) {
	if m == nil {
		return
	}
	switch reason {
	case flushReasonQuiet:
		m.quietFlushes.Inc()
	case flushReasonHard:
		m.hardTimeoutFlushes.Inc()
		m.partialFlushes.Inc()
	}
}

func (m *histogramAssemblerMonitoring) observeCapacityRejection() {
	if m == nil {
		return
	}
	m.capacityRejections.Inc()
}

func (m *histogramAssemblerMonitoring) observeShutdownDropped(count uint64) {
	if m == nil || count == 0 {
		return
	}
	m.shutdownDropped.Add(count)
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

type inputMetrics struct {
	intervalExecutionTime     metrics.Sample   // histogram of the total time elapsed during an interval
	intervalPageExecutionTime metrics.Sample   // histogram of per page execution time during an interval
	intervalPages             metrics.Sample   // histogram of pages per interval
	intervals                 *monitoring.Uint // total number of intervals executed
	intervalErrs              *monitoring.Uint // total number of interval errors
}

func newInputMetrics(reg *monitoring.Registry) *inputMetrics {
	if reg == nil {
		return nil
	}

	out := &inputMetrics{
		intervals:                 monitoring.NewUint(reg, "httpjson_interval_total"),
		intervalErrs:              monitoring.NewUint(reg, "httpjson_interval_errors_total"),
		intervalExecutionTime:     metrics.NewUniformSample(1024),
		intervalPageExecutionTime: metrics.NewUniformSample(1024),
		intervalPages:             metrics.NewUniformSample(1024),
	}

	_ = adapter.GetGoMetrics(reg, "httpjson_interval_execution_time", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.intervalExecutionTime))
	_ = adapter.GetGoMetrics(reg, "httpjson_interval_pages_execution_time", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.intervalPageExecutionTime))
	_ = adapter.GetGoMetrics(reg, "httpjson_interval_pages", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.intervalPages))

	return out
}

func (m *inputMetrics) updateIntervalMetrics(err error, t time.Time) {
	if m == nil {
		return
	}
	m.intervals.Add(1)
	m.intervalExecutionTime.Update(time.Since(t).Nanoseconds())
	if err != nil {
		m.intervalErrs.Add(1)
	}
}

func (m *inputMetrics) updatePageExecutionTime(t time.Time) {
	if m == nil {
		return
	}
	m.intervalPageExecutionTime.Update(time.Since(t).Nanoseconds())
}

func (m *inputMetrics) updatePagesPerInterval(npages int64) {
	if m == nil {
		return
	}
	m.intervalPages.Update(npages)
}

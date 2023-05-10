// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"time"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/rcrowley/go-metrics"
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
		intervals:                 monitoring.NewUint(reg, "httpjson.interval.total"),
		intervalErrs:              monitoring.NewUint(reg, "httpjson.interval.errors"),
		intervalExecutionTime:     metrics.NewUniformSample(1024),
		intervalPageExecutionTime: metrics.NewUniformSample(1024),
		intervalPages:             metrics.NewUniformSample(1024),
	}

	_ = adapter.NewGoMetrics(reg, "httpjson.interval.execution_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.intervalExecutionTime))
	_ = adapter.NewGoMetrics(reg, "httpjson.interval.pages.execution_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.intervalPageExecutionTime))
	_ = adapter.NewGoMetrics(reg, "httpjson.interval.pages.total", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.intervalPages))

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

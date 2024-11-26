// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

type inputMetrics struct {
	intervalEvents metrics.Sample   // histogram of the total events per interval
	intervals      *monitoring.Uint // total number of intervals executed
	errs           *monitoring.Uint // total number of errors
}

func newInputMetrics(reg *monitoring.Registry) *inputMetrics {
	if reg == nil {
		return nil
	}

	out := &inputMetrics{
		intervals:      monitoring.NewUint(reg, "unifiedlogs_interval_total"),
		errs:           monitoring.NewUint(reg, "unifiedlogs_errors_total"),
		intervalEvents: metrics.NewUniformSample(1024),
	}

	_ = adapter.GetGoMetrics(reg, "unifiedlogs_interval_events", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.intervalEvents))

	return out
}

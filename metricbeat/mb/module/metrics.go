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

package module

import (
	"errors"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// inputMetrics belongs to one metricset instance, unlike the legacy aggregate stats.
type inputMetrics struct {
	success, failures, events                                *monitoring.Uint
	fetches, fetchErrors, partialErrors, consecutiveFailures *monitoring.Uint
	inProgress                                               *monitoring.Bool
	started, completed                                       *monitoring.Int
	duration, publishWait                                    metrics.Sample
}

func newInputMetrics(reg *monitoring.Registry, logger *logp.Logger) *inputMetrics {
	if reg == nil {
		return nil
	}
	m := &inputMetrics{
		success:             monitoring.NewUint(reg, "success"),
		failures:            monitoring.NewUint(reg, "failures"),
		events:              monitoring.NewUint(reg, "events"),
		fetches:             monitoring.NewUint(reg, "fetches_total"),
		fetchErrors:         monitoring.NewUint(reg, "fetch_errors_total"),
		partialErrors:       monitoring.NewUint(reg, "fetch_partial_errors_total"),
		consecutiveFailures: monitoring.NewUint(reg, "consecutive_failures"),
		inProgress:          monitoring.NewBool(reg, "fetch_in_progress"),
		started:             monitoring.NewInt(reg, "fetch_started_at"),
		completed:           monitoring.NewInt(reg, "last_fetch_completed_at"),
		duration:            metrics.NewUniformSample(256),
		publishWait:         metrics.NewUniformSample(256),
	}
	_ = adapter.GetGoMetrics(reg, "fetch_duration_ns", logger, adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(m.duration))
	_ = adapter.GetGoMetrics(reg, "publish_wait_ns", logger, adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(m.publishWait))
	return m
}

func (m *inputMetrics) beginFetch(now time.Time) {
	if m == nil {
		return
	}
	m.started.Set(now.UnixMilli())
	m.inProgress.Set(true)
}

func (m *inputMetrics) endFetch(start time.Time, err error) {
	if m == nil {
		return
	}
	now := time.Now()
	m.duration.Update(now.Sub(start).Nanoseconds())
	m.completed.Set(now.UnixMilli())
	m.fetches.Inc()
	switch {
	case err == nil:
		m.consecutiveFailures.Set(0)
	case errors.As(err, &mb.PartialMetricsError{}):
		m.partialErrors.Inc()
		m.consecutiveFailures.Set(0)
	default:
		m.fetchErrors.Inc()
		m.consecutiveFailures.Inc()
	}
	m.inProgress.Set(false)
}

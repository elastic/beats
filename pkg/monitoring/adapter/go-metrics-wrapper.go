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

package adapter

import (
	metrics "github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

// go-metrics wrapper interface required to unpack the original metric
type goMetricsWrapper interface {
	wrapped() any
}

// go-metrics wrappers
type (
	goMetricsCounter struct{ c metrics.Counter }

	goMetricsGauge        struct{ g metrics.Gauge }
	goMetricsGaugeFloat64 struct{ g metrics.GaugeFloat64 }

	goMetricsFuncGauge      struct{ g metrics.FunctionalGauge }
	goMetricsFuncGaugeFloat struct {
		g metrics.FunctionalGaugeFloat64
	}

	goMetricsHistogram             struct{ h metrics.Histogram }
	goMetricsClearOnVisitHistogram struct{ h *ClearOnVisitHistogram }

	goMetricsMeter struct{ m metrics.Meter }
)

// goMetricsWrap tries to wrap a metric for use with monitoring package.
func goMetricsWrap(metric any) (monitoring.Var, bool) {
	switch v := metric.(type) {
	case *metrics.StandardCounter:
		return goMetricsCounter{v}, true
	case *metrics.StandardGauge:
		return goMetricsGauge{v}, true
	case *metrics.StandardGaugeFloat64:
		return goMetricsGaugeFloat64{v}, true
	case metrics.FunctionalGauge:
		return goMetricsFuncGauge{v}, true
	case metrics.FunctionalGaugeFloat64:
		return goMetricsFuncGaugeFloat{v}, true
	case *metrics.StandardHistogram:
		return goMetricsHistogram{v}, true
	case *metrics.StandardMeter:
		return goMetricsMeter{v}, true
	case *ClearOnVisitHistogram:
		return goMetricsClearOnVisitHistogram{v}, true
	}
	return nil, false
}

func (w goMetricsCounter) wrapped() any { return w.c }
func (w goMetricsCounter) Get() int64   { return w.c.Count() }
func (w goMetricsCounter) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsGauge) wrapped() any { return w.g }
func (w goMetricsGauge) Get() int64   { return w.g.Value() }
func (w goMetricsGauge) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsGaugeFloat64) wrapped() any { return w.g }
func (w goMetricsGaugeFloat64) Get() float64 { return w.g.Value() }
func (w goMetricsGaugeFloat64) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnFloat(w.Get())
}

func (w goMetricsFuncGauge) wrapped() any { return w.g }
func (w goMetricsFuncGauge) Get() int64   { return w.g.Value() }
func (w goMetricsFuncGauge) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsFuncGaugeFloat) wrapped() any { return w.g }
func (w goMetricsFuncGaugeFloat) Get() float64 { return w.g.Value() }
func (w goMetricsFuncGaugeFloat) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnFloat(w.Get())
}

func (w goMetricsHistogram) wrapped() any { return w.h }
func (w goMetricsHistogram) Get() int64   { return w.h.Sum() }
func (w goMetricsHistogram) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnRegistryStart()
	defer vs.OnRegistryFinished()

	h := w.h.Snapshot()
	ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
	vs.OnKey("count")
	vs.OnInt(w.h.Count())
	vs.OnKey("min")
	vs.OnInt(w.h.Min())
	vs.OnKey("max")
	vs.OnInt(w.h.Max())
	vs.OnKey("mean")
	vs.OnFloat(w.h.Mean())
	vs.OnKey("stddev")
	vs.OnFloat(w.h.StdDev())
	vs.OnKey("median")
	vs.OnFloat(ps[0])
	vs.OnKey("p75")
	vs.OnFloat(ps[1])
	vs.OnKey("p95")
	vs.OnFloat(ps[2])
	vs.OnKey("p99")
	vs.OnFloat(ps[3])
	vs.OnKey("p999")
	vs.OnFloat(ps[4])
}

func (w goMetricsMeter) wrapped() any { return w.m }
func (w goMetricsMeter) Get() int64   { return w.m.Count() }
func (w goMetricsMeter) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsClearOnVisitHistogram) wrapped() any { return w.h }
func (w goMetricsClearOnVisitHistogram) Get() int64   { return w.h.Sum() }
func (w goMetricsClearOnVisitHistogram) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnRegistryStart()
	defer vs.OnRegistryFinished()

	h := w.h.Snapshot()
	w.h.Clear()
	ps := h.Percentiles([]float64{0.5, 0.99})
	vs.OnKey("count")
	vs.OnInt(h.Count())
	vs.OnKey("min")
	vs.OnInt(h.Min())
	vs.OnKey("max")
	vs.OnInt(h.Max())
	vs.OnKey("median")
	vs.OnFloat(ps[0])
	vs.OnKey("p99")
	vs.OnFloat(ps[1])

}

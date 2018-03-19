package adapter

import (
	metrics "github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/libbeat/monitoring"
)

// go-metrics wrapper interface required to unpack the original metric
type goMetricsWrapper interface {
	wrapped() interface{}
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

	goMetricsHistogram struct{ h metrics.Histogram }

	goMetricsMeter struct{ m metrics.Meter }
)

// goMetricsWrap tries to wrap a metric for use with monitoring package.
func goMetricsWrap(metric interface{}) (monitoring.Var, bool) {
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
	}
	return nil, false
}

func (w goMetricsCounter) wrapped() interface{} { return w.c }
func (w goMetricsCounter) Get() int64           { return w.c.Count() }
func (w goMetricsCounter) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsGauge) wrapped() interface{} { return w.g }
func (w goMetricsGauge) Get() int64           { return w.g.Value() }
func (w goMetricsGauge) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsGaugeFloat64) wrapped() interface{} { return w.g }
func (w goMetricsGaugeFloat64) Get() float64         { return w.g.Value() }
func (w goMetricsGaugeFloat64) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnFloat(w.Get())
}

func (w goMetricsFuncGauge) wrapped() interface{} { return w.g }
func (w goMetricsFuncGauge) Get() int64           { return w.g.Value() }
func (w goMetricsFuncGauge) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsFuncGaugeFloat) wrapped() interface{} { return w.g }
func (w goMetricsFuncGaugeFloat) Get() float64         { return w.g.Value() }
func (w goMetricsFuncGaugeFloat) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnFloat(w.Get())
}

func (w goMetricsHistogram) wrapped() interface{} { return w.h }
func (w goMetricsHistogram) Get() int64           { return w.h.Sum() }
func (w goMetricsHistogram) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

func (w goMetricsMeter) wrapped() interface{} { return w.m }
func (w goMetricsMeter) Get() int64           { return w.m.Count() }
func (w goMetricsMeter) Visit(_ monitoring.Mode, vs monitoring.Visitor) {
	vs.OnInt(w.Get())
}

package adapter

import (
	"fmt"
	"reflect"
	"sync"

	metrics "github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

// implement adapter for adding go-metrics based counters
// to monitoring

// GoMetricsRegistry wraps a monitoring.Registry for filtering and registering
// go-metrics based metrics with the monitoring package. GoMetricsRegistry implements
// the go-metrics.Registry interface.
//
// Note: with the go-metrics using `interface{}`, there is no guarantee
//       a variable satisfying any of go-metrics interfaces is returned.
//       It's recommended to not mix go-metrics with other metrics types
//       in the same namespace.
type GoMetricsRegistry struct {
	mutex sync.Mutex

	reg     *monitoring.Registry
	filters *metricFilters

	shadow metrics.Registry // store non-accepted metrics
}

// GetGoMetrics wraps an existing monitoring.Registry with `name` into a
// GoMetricsRegistry for using the registry with go-metrics.Registry.
// If the monitoring.Registry does not exist yet, a new one will be generated.
//
// Note: with users of go-metrics potentially removing any metric at runtime,
//       it's recommended to have the underlying registry being generated with
//       `monitoring.IgnorePublishExpvar`.
func GetGoMetrics(parent *monitoring.Registry, name string, filters ...MetricFilter) *GoMetricsRegistry {
	v := parent.Get(name)
	if v == nil {
		return NewGoMetrics(parent, name, filters...)
	}

	reg := v.(*monitoring.Registry)
	return &GoMetricsRegistry{
		reg:     reg,
		shadow:  metrics.NewRegistry(),
		filters: makeFilters(filters...),
	}
}

// NewGoMetrics creates and registers a new GoMetricsRegistry with the parent
// registry.
func NewGoMetrics(parent *monitoring.Registry, name string, filters ...MetricFilter) *GoMetricsRegistry {
	return &GoMetricsRegistry{
		reg:     parent.NewRegistry(name, monitoring.IgnorePublishExpvar),
		shadow:  metrics.NewRegistry(),
		filters: makeFilters(filters...),
	}
}

// Each only iterates the shadowed metrics, not registered to the monitoring package,
// as those metrics are owned by monitoring.Registry only.
func (r *GoMetricsRegistry) Each(f func(string, interface{})) {
	r.shadow.Each(f)
}

func (r *GoMetricsRegistry) find(name string) interface{} {
	st := r.findState(name)
	if st.action == actIgnore {
		return nil
	}

	return r.reg.Get(st.name)
}

// Get retrieves a registered metric by name. If the name is unknown, Get returns nil.
//
// Note: with the return values being `interface{}`, there is no guarantee
//       a variable satisfying any of go-metrics interfaces is returned.
//       It's recommended to not mix go-metrics with other metrics types in one
//       namespace.
func (r *GoMetricsRegistry) Get(name string) interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.get(name)
}

func (r *GoMetricsRegistry) get(name string) interface{} {
	m := r.find(name)
	if m == nil {
		return r.shadow.Get(name)
	}

	if w, ok := m.(goMetricsWrapper); ok {
		return w.wrapped()
	}

	return m
}

// GetOrRegister retries an existing metric via `Get` or registers a new one
// if the metric is unknown. For lazy instantiation metric can be a function.
func (r *GoMetricsRegistry) GetOrRegister(name string, metric interface{}) interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v := r.get(name)
	if v != nil {
		return v
	}

	return r.doRegister(name, metric)
}

// Register adds a new metric.
// An error is returned if the metric is already known.
func (r *GoMetricsRegistry) Register(name string, metric interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.get(name) != nil {
		return fmt.Errorf("metric '%v' already registered", name)
	}

	r.doRegister(name, metric)
	return nil
}

func (r *GoMetricsRegistry) doRegister(name string, metric interface{}) interface{} {
	if v := reflect.ValueOf(metric); v.Kind() == reflect.Func {
		metric = v.Call(nil)[0].Interface()
	}

	st := r.addState(name, metric)
	if st.action == actIgnore {
		return r.shadow.GetOrRegister(name, st.metric)
	}

	if st.action == actAccept {
		w, ok := goMetricsWrap(st.metric)
		if ok {
			r.reg.Add(st.name, w, st.mode)
		}
	}

	return st.metric
}

// RunHealthchecks is a noop, required to satisfy the metrics.Registry interface.
func (r *GoMetricsRegistry) RunHealthchecks() {}

// Unregister removes a metric.
func (r *GoMetricsRegistry) Unregister(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	st := r.rmState(name)
	r.reg.Remove(st.name)
	r.shadow.Unregister(name)
}

// UnregisterAll calls `Clear` on the underlying monitoring.Registry
func (r *GoMetricsRegistry) UnregisterAll() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.shadow.UnregisterAll()
	err := r.reg.Clear()
	if err != nil {
		logp.Err("Failed to clear registry: %v", err)
	}
}

func (r *GoMetricsRegistry) findState(name string) state {
	return r.stateWith(kndFind, name, nil)
}

func (r *GoMetricsRegistry) addState(name string, metric interface{}) state {
	return r.stateWith(kndAdd, name, metric)
}

func (r *GoMetricsRegistry) rmState(name string) state {
	return r.stateWith(kndRemove, name, nil)
}

func (r *GoMetricsRegistry) stateWith(k kind, name string, metric interface{}) state {
	return r.filters.apply(state{
		kind:   k,
		action: actIgnore,
		reg:    r.reg,
		name:   name,
		mode:   monitoring.Full,
		metric: metric,
	})
}

// GoMetricsRegistry MetricFilter used to convert all metrics not being
// accepted by the filters to be replace with a Noop-metric.
// This can be used to disable metrics in go-metrics users lazily generating
// metrics via GetOrRegister.
var GoMetricsNilify = withVarFilter(func(st state) state {
	if st.action != actIgnore {
		return st
	}

	switch st.metric.(type) {
	case *metrics.StandardCounter:
		st.metric = metrics.NilCounter{}
	case *metrics.StandardEWMA:
		st.metric = metrics.NilEWMA{}
	case *metrics.StandardGauge:
		st.metric = metrics.NilGauge{}
	case *metrics.StandardGaugeFloat64:
		st.metric = metrics.NilGaugeFloat64{}
	case *metrics.StandardHealthcheck:
		st.metric = metrics.NilHealthcheck{}
	case *metrics.StandardHistogram:
		st.metric = metrics.NilHistogram{}
	case *metrics.StandardMeter:
		st.metric = metrics.NilMeter{}
	case *metrics.StandardTimer:
		st.metric = metrics.NilTimer{}
	}

	return st
})

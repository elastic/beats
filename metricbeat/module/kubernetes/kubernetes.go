package kubernetes

import (
	dto "github.com/prometheus/client_model/go"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "kubernetes" module.
	if err := mb.Registry.AddModule("kubernetes", ModuleBuilder()); err != nil {
		panic(err)
	}
}

type Module interface {
	mb.Module
	StartSharedFetcher(prometheus p.Prometheus, period time.Duration)
	GetSharedFamilies() []*dto.MetricFamily
}

type module struct {
	mb.BaseModule
	lock sync.Mutex

	prometheus p.Prometheus

	families []*dto.MetricFamily
	running  atomic.Bool
	stateMetricsPeriod time.Duration
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	return func(base mb.BaseModule) (mb.Module, error) {
		m := module{
			BaseModule: base,
		}
		return &m, nil
	}
}

func (m *module) StartSharedFetcher(prometheus p.Prometheus, period time.Duration) {
	if m.prometheus == nil {
		m.prometheus = prometheus
	}
	go m.runStateMetricsFetcher(period)
}

func (m *module) SetSharedFamilies(families []*dto.MetricFamily) {
	m.lock.Lock()
	m.families = families
	m.lock.Unlock()
}

func (m *module) GetSharedFamilies() []*dto.MetricFamily {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.families
}

// run ensures that the module is running with the passed subscription
func (m *module) runStateMetricsFetcher(period time.Duration) {
	var ticker *time.Ticker
	quit := make(chan bool)
	if !m.running.CAS(false, true) {
		// Module is already running, just check if there is a smaller period to adjust.
		if period < m.stateMetricsPeriod {
			m.stateMetricsPeriod = period
			ticker.Stop()
			ticker = time.NewTicker(period)
		}
		return
	}
	ticker = time.NewTicker(period)

	defer func() { m.running.Store(false) }()

	families, err := m.prometheus.GetFamilies()
	if err != nil {
		// communicate the error
	}
	m.SetSharedFamilies(families)

	// use a ticker here
	for {
		select {
		case <- ticker.C:
			families, err := m.prometheus.GetFamilies()
			if err != nil {
				// communicate the error
			}
			m.SetSharedFamilies(families)
		case <- quit:
			ticker.Stop()
			return
			// quit properly
		}
	}
}

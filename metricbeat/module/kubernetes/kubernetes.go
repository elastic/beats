package kubernetes

import (
	"fmt"
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
	//RegisterStateListener(prometheus p.Prometheus, stateMetricsChan chan []*dto.MetricFamily)
	RegisterStateListener(prometheus p.Prometheus, period time.Duration)
	GetSharedFamilies() []*dto.MetricFamily
}

type module struct {
	mb.BaseModule
	lock sync.Mutex

	prometheus p.Prometheus

	families []*dto.MetricFamily
	running  atomic.Bool
	stateMetricsPeriod time.Duration
	state_listeners        []chan []*dto.MetricFamily
	kubelet_listeners       []chan []*dto.MetricFamily
}

// prometheus
func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	return func(base mb.BaseModule) (mb.Module, error) {
		m := module{
			BaseModule: base,
		}

		stateMetricsets := 1 // number of subscribed state_* metricsets, for this PoC is just 1
		m.state_listeners = make([] chan []*dto.MetricFamily, stateMetricsets)
		m.kubelet_listeners = make([] chan []*dto.MetricFamily, 5)

		return &m, nil
	}
}

func (m *module) RegisterStateListener(prometheus p.Prometheus, period time.Duration) {

	if m.prometheus == nil {
		m.prometheus = prometheus
	}

	//m.lock.Lock()
	//m.state_listeners = append(m.state_listeners, stateMetricsChan)
	//m.lock.Unlock()

	// TODO: start a global kube_state_metrics fetcher with a minimum interval set to
	//  the smallest period of the state_* metricsets
	go m.runStateMetricsFetcher(period)
}

//func (m *module) notifyStateMetricsListeners(families []*dto.MetricFamily) {
//	m.lock.Lock()
//	for _, lis := range m.state_listeners {
//		lis <- families
//	}
//	m.lock.Unlock()
//}

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
		// Module is already running, just check if there is a smaller period to attach.
		if period < m.stateMetricsPeriod {
			m.stateMetricsPeriod = period
			ticker.Stop()
			ticker = time.NewTicker(period)
		}
		return
	}
	ticker = time.NewTicker(period)

	defer func() { m.running.Store(false) }()

	fmt.Println("Getting Families")
	families, err := m.prometheus.GetFamilies()
	// fetch and notify
	if err != nil {
		// communicate the error
	}
	m.SetSharedFamilies(families)

	// use a ticker here
	for {
		select {
		case <- ticker.C:
			fmt.Println("Getting Families")
			families, err := m.prometheus.GetFamilies()
			// fetch and notify
			if err != nil {
				// communicate the error
			}
			m.SetSharedFamilies(families)
			//m.notifyStateMetricsListeners(families)
		case <- quit:
			ticker.Stop()
			return
			// quit properly
		}
	}
}

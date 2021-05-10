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

package kubernetes

import (
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"

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

	families           []*dto.MetricFamily
	running            atomic.Bool
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
		case <-ticker.C:
			families, err := m.prometheus.GetFamilies()
			if err != nil {
				// communicate the error
			}
			m.SetSharedFamilies(families)
		case <-quit:
			ticker.Stop()
			return
			// quit properly
		}
	}
}

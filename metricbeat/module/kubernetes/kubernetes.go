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
	GetSharedFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, error)
}

type module struct {
	mb.BaseModule
	lock sync.Mutex

	sharedFamilies     []*dto.MetricFamily
	lastFetchErr       error
	lastFetchTimestamp time.Time
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	return func(base mb.BaseModule) (mb.Module, error) {
		m := module{
			BaseModule: base,
		}
		return &m, nil
	}
}

func (m *module) GetSharedFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := time.Now()
	if m.lastFetchTimestamp.IsZero() || now.Sub(m.lastFetchTimestamp) > m.Config().Period {
		m.sharedFamilies, m.lastFetchErr = prometheus.GetFamilies()
		m.lastFetchTimestamp = now
	}

	return m.sharedFamilies, m.lastFetchErr
}

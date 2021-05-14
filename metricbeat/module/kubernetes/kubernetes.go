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
	"github.com/elastic/beats/v7/libbeat/logp"
	"sync"
	"time"
	"fmt"

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
	GetSharedFamilies(prometheus p.Prometheus, ms string) ([]*dto.MetricFamily, error)
}

type familiesCache struct {
	sharedFamilies     []*dto.MetricFamily
	lastFetchErr       error
	lastFetchTimestamp time.Time
	setter string
}

type cacheMap map[string]*familiesCache

type module struct {
	mb.BaseModule
	lock sync.Mutex

	fCache cacheMap
	logger  *logp.Logger
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	sharedFamiliesCache := make(cacheMap)
	return func(base mb.BaseModule) (mb.Module, error) {
		hash := fmt.Sprintf("%s%s", base.Config().Period, base.Config().Hosts)
		sharedFamiliesCache[hash] = &familiesCache{}
		m := module{
			BaseModule: base,
			logger : logp.NewLogger(fmt.Sprintf("debug (%s)", hash)),
			fCache: sharedFamiliesCache,
		}
		m.logger.Warn("Building module now with  ", base.Config())
		return &m, nil
	}
}

func (m *module) GetSharedFamilies(prometheus p.Prometheus, ms string) ([]*dto.MetricFamily, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := time.Now()
	hash := fmt.Sprintf("%s%s", m.BaseModule.Config().Period, m.BaseModule.Config().Hosts)
	fCache := m.fCache[hash]

	if ms != fCache.setter {
		m.logger.Warn("DIFF[ms!=cacheSetter]: ", ms, " != ", fCache.setter)
	}

	if fCache.lastFetchTimestamp.IsZero() || now.Sub(fCache.lastFetchTimestamp) > m.Config().Period {
		m.logger.Warn("FETCH families for ms: ", ms, ". Last setter was ", fCache.setter)
		fCache.sharedFamilies, fCache.lastFetchErr = prometheus.GetFamilies()
		fCache.lastFetchTimestamp = now
		fCache.setter = ms
	} else {
		m.logger.Warn("REUSE families for ms: ", ms, ". Last setter was ", fCache.setter)
	}

	return fCache.sharedFamilies, fCache.lastFetchErr
}

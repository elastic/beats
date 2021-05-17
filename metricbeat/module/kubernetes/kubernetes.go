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
	"fmt"
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

type familiesCache struct {
	sharedFamilies     []*dto.MetricFamily
	lastFetchErr       error
	lastFetchTimestamp time.Time
	lock               sync.Mutex
}

type cacheMap map[string]*familiesCache

type module struct {
	mb.BaseModule

	fCache cacheMap
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	sharedFamiliesCache := make(cacheMap)
	return func(base mb.BaseModule) (mb.Module, error) {
		hash := generateCacheHash(base.Config().Hosts)
		// NOTE: These entries will be never removed, this can be a leak if
		// metricbeat is used to monitor clusters dynamically created.
		// (https://github.com/elastic/beats/pull/25640#discussion_r633395213)
		sharedFamiliesCache[hash] = &familiesCache{}
		m := module{
			BaseModule: base,
			fCache:     sharedFamiliesCache,
		}
		return &m, nil
	}
}

func (m *module) GetSharedFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, error) {
	now := time.Now()

	hash := generateCacheHash(m.Config().Hosts)
	fCache := m.fCache[hash]

	fCache.lock.Lock()
	defer fCache.lock.Unlock()

	if fCache.lastFetchTimestamp.IsZero() || now.Sub(fCache.lastFetchTimestamp) > m.Config().Period {
		fCache.sharedFamilies, fCache.lastFetchErr = prometheus.GetFamilies()
		fCache.lastFetchTimestamp = now
	}

	return fCache.sharedFamilies, fCache.lastFetchErr
}

func generateCacheHash(host []string) string {
	return fmt.Sprintf("%s", host)
}

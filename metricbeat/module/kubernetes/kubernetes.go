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

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
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
}

type kubeStateMetricsCache struct {
	cacheMap map[uint64]*familiesCache
	lock     sync.Mutex
}

func (c *kubeStateMetricsCache) getCacheMapEntry(hash uint64) *familiesCache {
	c.lock.Lock()
	defer c.lock.Unlock()
	if _, ok := c.cacheMap[hash]; !ok {
		c.cacheMap[hash] = &familiesCache{}
	}
	return c.cacheMap[hash]
}

type module struct {
	mb.BaseModule

	kubeStateMetricsCache *kubeStateMetricsCache
	familiesCache         *familiesCache
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	kubeStateMetricsCache := &kubeStateMetricsCache{
		cacheMap: make(map[uint64]*familiesCache),
	}
	return func(base mb.BaseModule) (mb.Module, error) {
		hash, err := generateCacheHash(base.Config().Hosts)
		if err != nil {
			return nil, errors.Wrap(err, "error generating cache hash for kubeStateMetricsCache")
		}
		// NOTE: These entries will be never removed, this can be a leak if
		// metricbeat is used to monitor clusters dynamically created.
		// (https://github.com/elastic/beats/pull/25640#discussion_r633395213)
		familiesCache := kubeStateMetricsCache.getCacheMapEntry(hash)
		m := module{
			BaseModule:            base,
			kubeStateMetricsCache: kubeStateMetricsCache,
			familiesCache:         familiesCache,
		}
		return &m, nil
	}
}

func (m *module) GetSharedFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, error) {
	m.kubeStateMetricsCache.lock.Lock()
	defer m.kubeStateMetricsCache.lock.Unlock()

	now := time.Now()

	if m.familiesCache.lastFetchTimestamp.IsZero() || now.Sub(m.familiesCache.lastFetchTimestamp) > m.Config().Period {
		m.familiesCache.sharedFamilies, m.familiesCache.lastFetchErr = prometheus.GetFamilies()
		m.familiesCache.lastFetchTimestamp = now
	}

	return m.familiesCache.sharedFamilies, m.familiesCache.lastFetchErr
}

func generateCacheHash(host []string) (uint64, error) {
	id, err := hashstructure.Hash(host, nil)
	if err != nil {
		return 0, err
	}
	return id, nil
}

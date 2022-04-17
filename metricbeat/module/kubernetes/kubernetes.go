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

	"github.com/mitchellh/hashstructure"
	dto "github.com/prometheus/client_model/go"

	"github.com/menderesk/beats/v7/metricbeat/helper"
	p "github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "kubernetes" module.
	if err := mb.Registry.AddModule("kubernetes", ModuleBuilder()); err != nil {
		panic(err)
	}
}

type Module interface {
	mb.Module
	GetStateMetricsFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, error)
	GetKubeletStats(http *helper.HTTP) ([]byte, error)
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
	if _, ok := c.cacheMap[hash]; !ok {
		c.cacheMap[hash] = &familiesCache{}
	}
	return c.cacheMap[hash]
}

type statsCache struct {
	sharedStats        []byte
	lastFetchErr       error
	lastFetchTimestamp time.Time
}

type kubeletStatsCache struct {
	cacheMap map[uint64]*statsCache
	lock     sync.Mutex
}

func (c *kubeletStatsCache) getCacheMapEntry(hash uint64) *statsCache {
	if _, ok := c.cacheMap[hash]; !ok {
		c.cacheMap[hash] = &statsCache{}
	}
	return c.cacheMap[hash]
}

type module struct {
	mb.BaseModule

	kubeStateMetricsCache *kubeStateMetricsCache
	kubeletStatsCache     *kubeletStatsCache
	cacheHash             uint64
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	kubeStateMetricsCache := &kubeStateMetricsCache{
		cacheMap: make(map[uint64]*familiesCache),
	}
	kubeletStatsCache := &kubeletStatsCache{
		cacheMap: make(map[uint64]*statsCache),
	}
	return func(base mb.BaseModule) (mb.Module, error) {
		hash, err := generateCacheHash(base.Config().Hosts)
		if err != nil {
			return nil, fmt.Errorf("error generating cache hash for kubeStateMetricsCache: %w", err)
		}
		m := module{
			BaseModule:            base,
			kubeStateMetricsCache: kubeStateMetricsCache,
			kubeletStatsCache:     kubeletStatsCache,
			cacheHash:             hash,
		}
		return &m, nil
	}
}

func (m *module) GetStateMetricsFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, error) {
	m.kubeStateMetricsCache.lock.Lock()
	defer m.kubeStateMetricsCache.lock.Unlock()

	now := time.Now()
	// NOTE: These entries will be never removed, this can be a leak if
	// metricbeat is used to monitor clusters dynamically created.
	// (https://github.com/menderesk/beats/pull/25640#discussion_r633395213)
	familiesCache := m.kubeStateMetricsCache.getCacheMapEntry(m.cacheHash)

	if familiesCache.lastFetchTimestamp.IsZero() || now.Sub(familiesCache.lastFetchTimestamp) > m.Config().Period {
		familiesCache.sharedFamilies, familiesCache.lastFetchErr = prometheus.GetFamilies()
		familiesCache.lastFetchTimestamp = now
	}

	return familiesCache.sharedFamilies, familiesCache.lastFetchErr
}

func (m *module) GetKubeletStats(http *helper.HTTP) ([]byte, error) {
	m.kubeletStatsCache.lock.Lock()
	defer m.kubeletStatsCache.lock.Unlock()

	now := time.Now()

	// NOTE: These entries will be never removed, this can be a leak if
	// metricbeat is used to monitor clusters dynamically created.
	// (https://github.com/menderesk/beats/pull/25640#discussion_r633395213)
	statsCache := m.kubeletStatsCache.getCacheMapEntry(m.cacheHash)

	if statsCache.lastFetchTimestamp.IsZero() || now.Sub(statsCache.lastFetchTimestamp) > m.Config().Period {
		statsCache.sharedStats, statsCache.lastFetchErr = http.FetchContent()
		statsCache.lastFetchTimestamp = now
	}

	return statsCache.sharedStats, statsCache.lastFetchErr
}

func generateCacheHash(host []string) (uint64, error) {
	id, err := hashstructure.Hash(host, nil)
	if err != nil {
		return 0, err
	}
	return id, nil
}

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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"
	dto "github.com/prometheus/client_model/go"

	"github.com/elastic/beats/v7/metricbeat/helper"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
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
	GetKubeletSummary(http *helper.HTTP) (*Summary, error)
	GetMetricsRepo() *util.MetricsRepo
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
	sharedSummary      *Summary
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
	metricsRepo           *util.MetricsRepo
	cacheHash             uint64
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	kubeStateMetricsCache := &kubeStateMetricsCache{
		cacheMap: make(map[uint64]*familiesCache),
	}
	kubeletStatsCache := &kubeletStatsCache{
		cacheMap: make(map[uint64]*statsCache),
	}
	metricsRepo := util.NewMetricsRepo()
	return func(base mb.BaseModule) (mb.Module, error) {
		hash, err := generateCacheHash(base.Config().Hosts)
		if err != nil {
			return nil, fmt.Errorf("error generating cache hash for kubeStateMetricsCache: %w", err)
		}

		m := module{
			BaseModule:            base,
			kubeStateMetricsCache: kubeStateMetricsCache,
			kubeletStatsCache:     kubeletStatsCache,
			metricsRepo:           metricsRepo,
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
	// (https://github.com/elastic/beats/pull/25640#discussion_r633395213)
	familiesCache := m.kubeStateMetricsCache.getCacheMapEntry(m.cacheHash)

	if familiesCache.lastFetchTimestamp.IsZero() || now.Sub(familiesCache.lastFetchTimestamp) > m.Config().Period {
		familiesCache.sharedFamilies, familiesCache.lastFetchErr = prometheus.GetFamilies()
		familiesCache.lastFetchTimestamp = now
	}

	return familiesCache.sharedFamilies, familiesCache.lastFetchErr
}

func (m *module) GetKubeletStats(http *helper.HTTP) ([]byte, error) {
	summary, err := m.GetKubeletSummary(http)
	if err != nil {
		return nil, err
	}

	// Convert Summary back to JSON for backward compatibility
	return json.Marshal(summary)
}

// GetKubeletSummary returns the parsed kubernetes.Summary from kubelet stats endpoint
func (m *module) GetKubeletSummary(http *helper.HTTP) (*Summary, error) {
	m.kubeletStatsCache.lock.Lock()
	defer m.kubeletStatsCache.lock.Unlock()

	now := time.Now()

	// Generate cache key based on both module hash and HTTP URL to ensure
	// different URLs have different cache entries
	url := http.GetURI()
	cacheKey, err := generateURLCacheHash(m.cacheHash, url)
	if err != nil {
		return nil, fmt.Errorf("error generating cache key for URL %s: %w", url, err)
	}

	// NOTE: These entries will be never removed, this can be a leak if
	// metricbeat is used to monitor clusters dynamically created.
	// (https://github.com/elastic/beats/pull/25640#discussion_r633395213)
	statsCache := m.kubeletStatsCache.getCacheMapEntry(cacheKey)

	if statsCache.lastFetchTimestamp.IsZero() || now.Sub(statsCache.lastFetchTimestamp) > m.Config().Period {
		// Fetch raw data from HTTP endpoint
		rawData, fetchErr := http.FetchContent()
		if fetchErr != nil {
			statsCache.lastFetchErr = fetchErr
			statsCache.lastFetchTimestamp = now
			return nil, fetchErr
		}

		// Parse JSON into Summary structure
		var summary Summary
		if parseErr := json.Unmarshal(rawData, &summary); parseErr != nil {
			statsCache.lastFetchErr = fmt.Errorf("cannot unmarshal json response: %w", parseErr)
			statsCache.lastFetchTimestamp = now
			return nil, statsCache.lastFetchErr
		}

		// Cache the parsed structure
		statsCache.sharedSummary = &summary
		statsCache.lastFetchErr = nil
		statsCache.lastFetchTimestamp = now
	}

	return statsCache.sharedSummary, statsCache.lastFetchErr
}

func generateCacheHash(host []string) (uint64, error) {
	id, err := hashstructure.Hash(host, nil)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// generateURLCacheHash generates a cache key based on module hash and HTTP URL
func generateURLCacheHash(moduleHash uint64, url string) (uint64, error) {
	cacheData := struct {
		ModuleHash uint64
		URL        string
	}{
		ModuleHash: moduleHash,
		URL:        url,
	}

	id, err := hashstructure.Hash(cacheData, nil)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *module) GetMetricsRepo() *util.MetricsRepo {
	return m.metricsRepo
}

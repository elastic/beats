// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package containerd

import (
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"

	p "github.com/elastic/beats/v8/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

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

func init() {
	// Register the ModuleFactory function for the "containerd" module.
	if err := mb.Registry.AddModule("containerd", ModuleBuilder()); err != nil {
		panic(err)
	}
}

type Module interface {
	mb.Module
	GetContainerdMetricsFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, time.Time, error)
}

type familiesCache struct {
	sharedFamilies     []*dto.MetricFamily
	lastFetchErr       error
	lastFetchTimestamp time.Time
}

type containerdMetricsCache struct {
	cacheMap map[uint64]*familiesCache
	lock     sync.Mutex
}

func (c *containerdMetricsCache) getCacheMapEntry(hash uint64) *familiesCache {
	if _, ok := c.cacheMap[hash]; !ok {
		c.cacheMap[hash] = &familiesCache{}
	}
	return c.cacheMap[hash]
}

type module struct {
	mb.BaseModule

	containerdMetricsCache *containerdMetricsCache
	cacheHash              uint64
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	containerdMetricsCache := &containerdMetricsCache{
		cacheMap: make(map[uint64]*familiesCache),
	}

	return func(base mb.BaseModule) (mb.Module, error) {
		hash, err := generateCacheHash(base.Config().Hosts)
		if err != nil {
			return nil, errors.Wrap(err, "error generating cache hash for containerdMetricsCache")
		}
		m := module{
			BaseModule:             base,
			containerdMetricsCache: containerdMetricsCache,
			cacheHash:              hash,
		}
		return &m, nil
	}
}

func (m *module) GetContainerdMetricsFamilies(prometheus p.Prometheus) ([]*dto.MetricFamily, time.Time, error) {
	m.containerdMetricsCache.lock.Lock()
	defer m.containerdMetricsCache.lock.Unlock()

	now := time.Now()
	// NOTE: These entries will be never removed, this can be a leak if
	// metricbeat is used to monitor clusters dynamically created.
	// (https://github.com/elastic/beats/pull/25640#discussion_r633395213)
	familiesCache := m.containerdMetricsCache.getCacheMapEntry(m.cacheHash)

	if familiesCache.lastFetchTimestamp.IsZero() || now.Sub(familiesCache.lastFetchTimestamp) > m.Config().Period {
		familiesCache.sharedFamilies, familiesCache.lastFetchErr = prometheus.GetFamilies()
		familiesCache.lastFetchTimestamp = now
	}

	return familiesCache.sharedFamilies, familiesCache.lastFetchTimestamp, familiesCache.lastFetchErr
}

func generateCacheHash(host []string) (uint64, error) {
	id, err := hashstructure.Hash(host, nil)
	if err != nil {
		return 0, err
	}
	return id, nil
}

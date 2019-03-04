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

package util

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// PerfMetrics stores known metrics from Kubernetes nodes and containers
var PerfMetrics = NewPerfMetricsCache()

func init() {
	PerfMetrics.Start()
}

const defaultTimeout = 120 * time.Second

// NewPerfMetricsCache initializes and returns a new PerfMetricsCache
func NewPerfMetricsCache() *PerfMetricsCache {
	return &PerfMetricsCache{
		NodeMemAllocatable:   newValueMap(defaultTimeout),
		NodeCoresAllocatable: newValueMap(defaultTimeout),

		ContainerMemLimit:   newValueMap(defaultTimeout),
		ContainerCoresLimit: newValueMap(defaultTimeout),
	}
}

// PerfMetricsCache stores known metrics from Kubernetes nodes and containers
type PerfMetricsCache struct {
	NodeMemAllocatable   *valueMap
	NodeCoresAllocatable *valueMap

	ContainerMemLimit   *valueMap
	ContainerCoresLimit *valueMap
}

// Start cache workers
func (c *PerfMetricsCache) Start() {
	c.NodeMemAllocatable.Start()
	c.NodeCoresAllocatable.Start()
	c.ContainerMemLimit.Start()
	c.ContainerCoresLimit.Start()
}

// Stop cache workers
func (c *PerfMetricsCache) Stop() {
	c.NodeMemAllocatable.Stop()
	c.NodeCoresAllocatable.Stop()
	c.ContainerMemLimit.Stop()
	c.ContainerCoresLimit.Stop()
}

type valueMap struct {
	cache   *common.Cache
	timeout time.Duration
}

func newValueMap(timeout time.Duration) *valueMap {
	return &valueMap{
		cache:   common.NewCache(timeout, 0),
		timeout: timeout,
	}
}

// Get value
func (m *valueMap) Get(name string) float64 {
	return m.GetWithDefault(name, 0.0)
}

// Get value
func (m *valueMap) GetWithDefault(name string, def float64) float64 {
	v := m.cache.Get(name)
	if v, ok := v.(float64); ok {
		return v
	}
	return def
}

// Set value
func (m *valueMap) Set(name string, val float64) {
	m.cache.PutWithTimeout(name, val, m.timeout)
}

// Start cache workers
func (m *valueMap) Start() {
	m.cache.StartJanitor(m.timeout)
}

// Stop cache workers
func (m *valueMap) Stop() {
	m.cache.StopJanitor()
}

// ContainerUID creates an unique ID for from namespace, pod name and container name
func ContainerUID(namespace, pod, container string) string {
	return namespace + "/" + pod + "/" + container
}

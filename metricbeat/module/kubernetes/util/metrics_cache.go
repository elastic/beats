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

	"github.com/elastic/beats/v7/libbeat/common"
)

// NewPerfMetricsCache initializes and returns a new PerfMetricsCache
func NewPerfMetricsCache(timeout time.Duration) *PerfMetricsCache {
	return &PerfMetricsCache{
		NodeMemAllocatable:   newValueMap(timeout),
		NodeCoresAllocatable: newValueMap(timeout),

		ContainerMemLimit:   newValueMap(timeout),
		ContainerCoresLimit: newValueMap(timeout),
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

// Returns the maximum timeout of all the caches under PerfMetricsCache
func (c *PerfMetricsCache) GetTimeout() time.Duration {
	var ans time.Duration = 0

	nmATimeout := c.NodeMemAllocatable.GetTimeout()
	if nmATimeout > ans {
		ans = nmATimeout
	}

	ncATimeout := c.NodeCoresAllocatable.GetTimeout()
	if ncATimeout > ans {
		ans = ncATimeout
	}

	cmLTimeout := c.ContainerMemLimit.GetTimeout()
	if cmLTimeout > ans {
		ans = cmLTimeout
	}

	ccLTimeout := c.ContainerCoresLimit.GetTimeout()
	if ccLTimeout > ans {
		ans = ccLTimeout
	}
	return ans
}

// Set the timeout of all the caches under PerfMetricsCache, then Stop and Start all the cache janitors
func (c *PerfMetricsCache) SetOrUpdateTimeout(timeout time.Duration) {
	c.NodeMemAllocatable.SetTimeout(timeout)
	c.NodeCoresAllocatable.SetTimeout(timeout)
	c.ContainerMemLimit.SetTimeout(timeout)
	c.ContainerCoresLimit.SetTimeout(timeout)

	c.Stop()
	c.Start()
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

func (m *valueMap) GetTimeout() time.Duration {
	return m.timeout
}

func (m *valueMap) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

// ContainerUID creates an unique ID for from namespace, pod name and container name
func ContainerUID(namespace, pod, container string) string {
	return namespace + "/" + pod + "/" + container
}

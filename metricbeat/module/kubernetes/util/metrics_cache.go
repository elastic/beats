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
	"sync"
	"time"
)

// PerfMetrics stores known metrics from Kubernetes nodes and containers
var PerfMetrics = NewPerfMetricsCache()

const defaultTimeout = 120 * time.Second

var now = time.Now
var sleep = time.Sleep

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
	mutex                sync.RWMutex
	NodeMemAllocatable   *valueMap
	NodeCoresAllocatable *valueMap

	ContainerMemLimit   *valueMap
	ContainerCoresLimit *valueMap
}

func newValueMap(timeout time.Duration) *valueMap {
	return &valueMap{
		values:  map[string]value{},
		timeout: timeout,
	}
}

type valueMap struct {
	sync.RWMutex
	running bool
	timeout time.Duration
	values  map[string]value
}

type value struct {
	value   float64
	expires int64
}

// ContainerUID creates an unique ID for from namespace, pod name and container name
func ContainerUID(namespace, pod, container string) string {
	return namespace + "-" + pod + "-" + container
}

// Get value
func (m *valueMap) Get(name string) float64 {
	m.RLock()
	defer m.RUnlock()
	return m.values[name].value
}

// Get value
func (m *valueMap) GetWithDefault(name string, def float64) float64 {
	m.RLock()
	defer m.RUnlock()
	val, ok := m.values[name]
	if ok {
		return val.value
	}
	return def
}

// Set value
func (m *valueMap) Set(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	m.ensureCleanupWorker()
	m.values[name] = value{val, now().Add(m.timeout).Unix()}
}

func (m *valueMap) ensureCleanupWorker() {
	if !m.running {
		// Run worker to cleanup expired entries
		m.running = true
		go func() {
			for {
				sleep(m.timeout)
				m.Lock()
				now := now().Unix()
				for name, val := range m.values {
					if now > val.expires {
						delete(m.values, name)
					}
				}
				m.Unlock()
			}
		}()
	}
}

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

func newValueMap(timeout time.Duration) *valueMap {
	return newValueMapWithClock(timeout, clock{})
}

func newValueMapWithClock(timeout time.Duration, clock clock) *valueMap {
	m := valueMap{
		values:  map[string]*value{},
		timeout: timeout,
		time:    clock,
	}
	m.startWorkers()
	return &m
}

type clock struct {
	now   func() time.Time
	after func(time.Duration) <-chan time.Time
}

func (c *clock) Now() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *clock) After(d time.Duration) <-chan time.Time {
	if c.after != nil {
		return c.after(d)
	}
	return time.After(d)
}

type valueMap struct {
	sync.Mutex
	timeout time.Duration
	values  map[string]*value
	time    clock
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
	return m.getWithDefault(name, 0)
}

// Get value
func (m *valueMap) GetWithDefault(name string, def float64) float64 {
	return m.getWithDefault(name, def)
}

func (m *valueMap) getWithDefault(name string, def float64) float64 {
	m.Lock()
	defer m.Unlock()
	val, ok := m.values[name]
	if ok {
		m.renew(val)
		return val.value
	}
	return def
}

// Set value
func (m *valueMap) Set(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	v, ok := m.values[name]
	if ok {
		v.value = val
	} else {
		v = &value{value: val}
		m.values[name] = v
	}
	m.renew(v)
}

func (m *valueMap) renew(v *value) {
	v.expires = m.time.Now().Add(m.timeout).Unix()
}

func (m *valueMap) startWorkers() {
	go func() {
		for {
			now := <-m.time.After(m.timeout)
			m.Lock()
			for name, val := range m.values {
				if now.Unix() > val.expires {
					delete(m.values, name)
				}
			}
			m.Unlock()
		}
	}()
}

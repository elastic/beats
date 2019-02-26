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
	"context"
	"sync"
	"time"
)

// PerfMetrics stores known metrics from Kubernetes nodes and containers
var PerfMetrics = NewPerfMetricsCache()

const defaultTimeout = 120 * time.Second

var now = time.Now
var after = time.After

// NewPerfMetricsCache initializes and returns a new PerfMetricsCache
func NewPerfMetricsCache() *PerfMetricsCache {
	ctx := context.TODO()
	return &PerfMetricsCache{
		NodeMemAllocatable:   newValueMap(ctx, defaultTimeout),
		NodeCoresAllocatable: newValueMap(ctx, defaultTimeout),

		ContainerMemLimit:   newValueMap(ctx, defaultTimeout),
		ContainerCoresLimit: newValueMap(ctx, defaultTimeout),
	}
}

// PerfMetricsCache stores known metrics from Kubernetes nodes and containers
type PerfMetricsCache struct {
	NodeMemAllocatable   *valueMap
	NodeCoresAllocatable *valueMap

	ContainerMemLimit   *valueMap
	ContainerCoresLimit *valueMap
}

func newValueMap(ctx context.Context, timeout time.Duration) *valueMap {
	m := &valueMap{
		values:  map[string]*value{},
		timeout: timeout,
	}
	m.startWorkers(ctx)
	return m
}

type valueMap struct {
	sync.Mutex
	timeout time.Duration
	values  map[string]*value
}

type value struct {
	value   float64
	expires int64
}

func (v *value) renew(timeout time.Duration) {
	v.expires = now().Add(timeout).Unix()
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
		val.renew(m.timeout)
		return val.value
	}
	return def
}

// Set value
func (m *valueMap) Set(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	v := &value{value: val}
	v.renew(m.timeout)
	m.values[name] = v
}

func (m *valueMap) startWorkers(ctx context.Context) {
	go func() {
		for {
			var now time.Time
			select {
			case now = <-after(m.timeout):
			case <-ctx.Done():
				return
			}
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

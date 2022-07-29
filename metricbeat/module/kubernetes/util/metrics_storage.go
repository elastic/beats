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
	"strings"
	"sync"
)

type Metric int64

const (
	ContainerCoresLimitMetric Metric = iota
	ContainerMemoryLimitMetric
	NodeCoresAllocatableMetric
	NodeMemoryAllocatableMetric
)

type MetricPrefix int64

const (
	ContainerMetricPrefix MetricPrefix = iota
	NodeMetricPrefix
)

func (m Metric) String() string {
	switch m {
	case ContainerCoresLimitMetric:
		return "container.cores.limit"
	case ContainerMemoryLimitMetric:
		return "container.memory.limit"
	case NodeCoresAllocatableMetric:
		return "node.cores.allocatable"
	case NodeMemoryAllocatableMetric:
		return "node.memory.allocatable"
	}
	return "unknown"
}

func (mp MetricPrefix) String() string {
	switch mp {
	case ContainerMetricPrefix:
		return "container"
	case NodeMetricPrefix:
		return "node"
	}
	return "unknown"
}

type Metrics struct {
	entries map[Metric]float64
}

func NewMetrics() *Metrics {
	ans := &Metrics{
		entries: make(map[Metric]float64),
	}
	return ans
}

func (m *Metrics) Set(name Metric, value float64) {
	m.entries[name] = value
}

func (m *Metrics) Get(name Metric) (float64, bool) {
	ans, exists := m.entries[name]
	return ans, exists
}

func (m *Metrics) GetWithDefault(name Metric, defaultValue float64) (float64, bool) {
	ans, exists := m.Get(name)
	if !exists {
		return defaultValue, false
	}
	return ans, exists
}

func (m *Metrics) Delete(name Metric) {
	delete(m.entries, name)
}

func (m *Metrics) Clear() {
	for k := range m.entries {
		delete(m.entries, k)
	}
}

type MetricsStorage struct {
	sync.RWMutex
	metrics map[string]*Metrics
}

func NewMetricsStorage() *MetricsStorage {
	ans := &MetricsStorage{
		metrics: make(map[string]*Metrics),
	}
	return ans
}

func (ms *MetricsStorage) Clear() {
	ms.Lock()
	defer ms.Unlock()
	for k := range ms.metrics {
		delete(ms.metrics, k)
	}
}

func (ms *MetricsStorage) addMetrics(metricId string) *Metrics {
	ms.Lock()
	defer ms.Unlock()
	ms.metrics[metricId] = NewMetrics()
	return ms.metrics[metricId]
}

func (ms *MetricsStorage) getMetrics(metricId string) (*Metrics, bool) {
	ms.RLock()
	defer ms.RUnlock()
	ans, exists := ms.metrics[metricId]
	return ans, exists
}

func (ms *MetricsStorage) Get(metricId string, metricName Metric) (float64, bool) {
	metrics, exists := ms.getMetrics(metricId)
	if !exists {
		return -1, false
	}
	ans, exists := metrics.Get(metricName)
	return ans, exists
}

func (ms *MetricsStorage) GetWithDefault(metricId string, metricName Metric, defaultValue float64) (float64, bool) {
	metrics, exists := ms.getMetrics(metricId)
	if !exists {
		return defaultValue, false
	}
	return metrics.GetWithDefault(metricName, defaultValue)
}

func (ms *MetricsStorage) Set(metricId string, metricName Metric, metricValue float64) {
	metrics, exists := ms.getMetrics(metricId)
	if !exists {
		metrics = ms.addMetrics(metricId)
	}
	metrics.Set(metricName, metricValue)
}

func (ms *MetricsStorage) Delete(metricId string) {
	ms.Lock()
	defer ms.Unlock()
	delete(ms.metrics, metricId)
}

func GetMetricId(owner string, prefix MetricPrefix) string {
	metricPrefix := prefix.String()
	fields := []string{metricPrefix, owner}
	ans := strings.Join(fields, "/")

	return ans
}

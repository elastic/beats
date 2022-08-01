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

// Metric defines a enumeration for all possible metrics.
type Metric int64

const (
	ContainerCoresLimitMetric Metric = iota
	ContainerMemoryLimitMetric
	NodeCoresAllocatableMetric
	NodeMemoryAllocatableMetric
)

// MetricSource defines an optional prefix for MetricRepoID to distinguish metrics for container, nodes, etc.
type MetricSource int64

const (
	ContainerMetricSource MetricSource = iota
	NodeMetricSource
)

// Metrics stores a group of metrics in a dictionary of (name Metric, value float64). The name of a metric must be unique.
type Metrics struct {
	entries map[Metric]float64
}

// MetricsRepo stores a dictionary of (uid String, metrics Metrics). MetricsRepo is the top level object used to store/access metrics from the Kubernetes metricsets. Uid is typically made of a MetricSource and a name. Eg. a uid might be "container.metricbeat-abcd" where `container` is the MetricSource and `metricbeat-abcd` is the id of the container in Kubernetes). Access to entries in this dictionary is thread-safe by using a sync.RWMutex.
type MetricsRepo struct {
	sync.RWMutex
	metrics map[string]*Metrics
}

// Converts a Metric type into a string.
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

// Converts a MetricSource type into a string.
func (mp MetricSource) String() string {
	switch mp {
	case ContainerMetricSource:
		return "container"
	case NodeMetricSource:
		return "node"
	}
	return "unknown"
}

// NewMetrics initializes and returns a new Metrics.
func NewMetrics() *Metrics {
	ans := &Metrics{
		entries: make(map[Metric]float64),
	}
	return ans
}

// Sets the value of a metric given a name and a value. If a value for that metric is already present, it overwrites it.
func (m *Metrics) Set(name Metric, value float64) {
	m.entries[name] = value
}

// Returns the value of a metric by name and whether the entry already exists in Metrics.
func (m *Metrics) Get(name Metric) (float64, bool) {
	ans, exists := m.entries[name]
	return ans, exists
}

// Returns the value of a metric by name. If the metric doesn't exists, it returns the defaultValue provided instead.
func (m *Metrics) GetWithDefault(name Metric, defaultValue float64) float64 {
	ans, exists := m.Get(name)
	if !exists {
		return defaultValue
	}
	return ans
}

// Deletes a metric by name from Metrics.
func (m *Metrics) Delete(name Metric) {
	delete(m.entries, name)
}

// Deletes all entries from Metrics.
func (m *Metrics) Clear() {
	for k := range m.entries {
		delete(m.entries, k)
	}
}

// NewMetricsRepo initializes and returns a new MetricsRepo.
func NewMetricsRepo() *MetricsRepo {
	ans := &MetricsRepo{
		metrics: make(map[string]*Metrics),
	}
	return ans
}

// Deletes all entries from MetricsRepo
func (ms *MetricsRepo) Clear() {
	ms.Lock()
	defer ms.Unlock()
	for k := range ms.metrics {
		delete(ms.metrics, k)
	}
}

// Initialize an empty Metrics for a UID.
func (ms *MetricsRepo) initMetrics(uid string) *Metrics {
	ms.Lock()
	defer ms.Unlock()
	ms.metrics[uid] = NewMetrics()
	return ms.metrics[uid]
}

// Returns all the Metrics for a UID.
func (ms *MetricsRepo) getMetrics(uid string) (*Metrics, bool) {
	ms.RLock()
	defer ms.RUnlock()
	ans, exists := ms.metrics[uid]
	return ans, exists
}

// Returns the value of a metric for a (UID, metricName) in MetricsRepo and whether it already exist. If the metrics doesn't exists it return (-1, false).
func (ms *MetricsRepo) Get(uid string, metricName Metric) (float64, bool) {
	metrics, exists := ms.getMetrics(uid)
	if !exists {
		return -1, false
	}
	ans, exists := metrics.Get(metricName)
	return ans, exists
}

// Returns the value of a metric for a (UID, metricName) in MetricsRepo. If the metric doesn't exists, it returns the defaultValue provided instead.
func (ms *MetricsRepo) GetWithDefault(uid string, metricName Metric, defaultValue float64) float64 {
	metrics, exists := ms.getMetrics(uid)
	if !exists {
		return defaultValue
	}
	return metrics.GetWithDefault(metricName, defaultValue)
}

// Set the value of a metric for a (UID, metricName) in MetricsRepo.
func (ms *MetricsRepo) Set(uid string, metricName Metric, metricValue float64) {
	metrics, exists := ms.getMetrics(uid)
	if !exists {
		metrics = ms.initMetrics(uid)
	}
	metrics.Set(metricName, metricValue)
}

// Delete the value of a metric by UID in MetricsRepo.
func (ms *MetricsRepo) Delete(uid string) {
	ms.Lock()
	defer ms.Unlock()
	delete(ms.metrics, uid)
}

// Keys returns all the UIDs in MetricsRepo.
func (ms *MetricsRepo) Keys() []string {
	ms.Lock()
	defer ms.Unlock()
	ans := make([]string, 0, len(ms.metrics))
	for repoId := range ms.metrics {
		ans = append(ans, repoId)
	}
	return ans
}

// Returns a MetricRepoID used as key in MetricsRepo dictionary by combining a MetricSource and a name. Eg. a MetricRepoID might be "container.metricbeat-abcd" where `container` is the MetricSource and `metricbeat-abcd` is the id of the container in Kubernetes.
func GetMetricsRepoId(prefix MetricSource, name string) string {
	metricPrefix := prefix.String()
	fields := []string{metricPrefix, name}
	ans := strings.Join(fields, "/")

	return ans
}

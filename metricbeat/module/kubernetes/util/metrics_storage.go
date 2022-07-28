package util

import (
	"strings"
	"sync"
)

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

type MetricSet struct {
	sync.RWMutex
	metrics map[string]*Metrics
}

func NewMetricSet() *MetricSet {
	ans := &MetricSet{
		metrics: make(map[string]*Metrics),
	}
	return ans
}

func (ms *MetricSet) Clear() {
	ms.Lock()
	defer ms.Unlock()
	for k := range ms.metrics {
		delete(ms.metrics, k)
	}
}

func (ms *MetricSet) addMetrics(id string) *Metrics {
	ms.Lock()
	defer ms.Unlock()
	ms.metrics[id] = NewMetrics()
	return ms.metrics[id]
}

func (ms *MetricSet) getMetrics(id string) (*Metrics, bool) {
	ms.RLock()
	defer ms.RUnlock()
	ans, exists := ms.metrics[id]
	return ans, exists
}

func (ms *MetricSet) Get(id string, metricName Metric) (float64, bool) {
	metrics, exists := ms.getMetrics(id)
	if !exists {
		return -1, false
	}
	ans, exists := metrics.Get(metricName)
	return ans, exists
}

func (ms *MetricSet) GetWithDefault(id string, metricName Metric, defaultValue float64) (float64, bool) {
	metrics, exists := ms.getMetrics(id)
	if !exists {
		return defaultValue, false
	}
	return metrics.GetWithDefault(metricName, defaultValue)
}

func (ms *MetricSet) Set(id string, metricName Metric, metricValue float64) {
	metrics, exists := ms.getMetrics(id)
	if !exists {
		metrics = ms.addMetrics(id)
	}
	metrics.Set(metricName, metricValue)
}

func (ms *MetricSet) Delete(id string) {
	ms.Lock()
	defer ms.Unlock()
	delete(ms.metrics, id)
}

type MetricsStorage struct {
	metrics *MetricSet
}

func NewMetricsStorage() *MetricsStorage {
	ans := &MetricsStorage{
		metrics: NewMetricSet(),
	}
	return ans
}

func (s *MetricsStorage) Clear() {
	s.metrics.Clear()
}

type Metric int64

const (
	CONTAINER_CORES_LIMIT_METRIC Metric = iota
	CONTAINER_MEMORY_LIMIT_METRIC
	NODE_CORES_ALLOCATABLE_METRIC
	NODE_MEMORY_ALLOCATABLE_METRIC
)

func (m Metric) String() string {
	switch m {
	case CONTAINER_CORES_LIMIT_METRIC:
		return "container.cores.limit"
	case CONTAINER_MEMORY_LIMIT_METRIC:
		return "container.memory.limit"
	case NODE_CORES_ALLOCATABLE_METRIC:
		return "node.cores.allocatable"
	case NODE_MEMORY_ALLOCATABLE_METRIC:
		return "node.memory.allocatable"
	}
	return "unknown"
}

type MetricPrefix int64

const (
	CONTAINER_METRIC_PREFIX MetricPrefix = iota
	NODE_METRIC_PREFIX
)

func (mp MetricPrefix) String() string {
	switch mp {
	case CONTAINER_METRIC_PREFIX:
		return "container"
	case NODE_METRIC_PREFIX:
		return "node"
	}
	return "unknown"
}

func GetMetricOwner(owner string, prefix MetricPrefix) string {
	metricPrefix := prefix.String()
	fields := []string{metricPrefix, owner}
	ans := strings.Join(fields, "/")

	return ans
}

func (s *MetricsStorage) SetMetric(owner string, metricName Metric, metricValue float64) {
	s.metrics.Set(owner, metricName, metricValue)
}

func (s *MetricsStorage) Delete(owner string) {
	s.metrics.Delete(owner)
}

func (s *MetricsStorage) GetMetric(owner string, metricName Metric) (float64, bool) {
	return s.metrics.Get(owner, metricName)
}

func (s *MetricsStorage) GetMetricWithDefault(owner string, metricName Metric, defaultValue float64) (float64, bool) {
	return s.metrics.GetWithDefault(owner, metricName, defaultValue)
}

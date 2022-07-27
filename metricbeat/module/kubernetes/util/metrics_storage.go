package util

import (
	"sync"
)

type Metrics struct {
	entries map[int]float64
}

func NewMetrics() *Metrics {
	ans := &Metrics{
		entries: make(map[int]float64),
	}
	return ans
}

func (m *Metrics) Set(name int, value float64) {
	m.entries[name] = value
}

func (m *Metrics) Get(name int) (float64, bool) {
	ans, exists := m.entries[name]
	return ans, exists
}

func (m *Metrics) GetWithDefault(name int, defaultValue float64) (float64, bool) {
	ans, exists := m.Get(name)
	if !exists {
		return defaultValue, false
	}
	return ans, exists
}

func (m *Metrics) Delete(name int) {
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

func (ms *MetricSet) Get(id string, metricName int) (float64, bool) {
	metrics, exists := ms.getMetrics(id)
	if !exists {
		return -1, false
	}
	ans, exists := metrics.Get(metricName)
	return ans, exists
}

func (ms *MetricSet) GetWithDefault(id string, metricName int, defaultValue float64) (float64, bool) {
	metrics, exists := ms.getMetrics(id)
	if !exists {
		return -1, false
	}
	return metrics.GetWithDefault(metricName, defaultValue)
}

func (ms *MetricSet) Set(id string, metricName int, metricValue float64) {
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
	containerMetrics *MetricSet
	nodeMetrics      *MetricSet
}

func NewMetricsStorage() *MetricsStorage {
	ans := &MetricsStorage{
		containerMetrics: NewMetricSet(),
		nodeMetrics:      NewMetricSet(),
	}
	return ans
}

func (s *MetricsStorage) Clear() {
	s.containerMetrics.Clear()
	s.nodeMetrics.Clear()
}

// todo: replace with enums with enum type
const CONTAINER_CORES_LIMIT = 1
const CONTAINER_MEMORY_LIMIT = 2
const NODE_CORES_ALLOCATABLE = 3
const NODE_MEMORY_ALLOCATABLE = 4

func (s *MetricsStorage) DeleteNodeMetric(id string) {
	s.nodeMetrics.Delete(id)
}

func (s *MetricsStorage) SetNodeMetric(id string, metricName int, metricValue float64) {
	s.nodeMetrics.Set(id, metricName, metricValue)
}

func (s *MetricsStorage) GetNodeMetric(id string, metricName int) (float64, bool) {
	return s.nodeMetrics.Get(id, metricName)
}

func (s *MetricsStorage) GetNodeMetricWithDefault(id string, metricName int, defaultValue float64) (float64, bool) {
	return s.nodeMetrics.GetWithDefault(id, metricName, defaultValue)
}

func (s *MetricsStorage) DeleteContainerMetric(id string) {
	s.containerMetrics.Delete(id)
}

func (s *MetricsStorage) SetContainerMetric(id string, metricName int, metricValue float64) {
	s.containerMetrics.Set(id, metricName, metricValue)
}

func (s *MetricsStorage) GetContainerMetric(id string, metricName int) (float64, bool) {
	return s.containerMetrics.Get(id, metricName)
}

func (s *MetricsStorage) GetContainerMetricWithDefault(id string, metricName int, defaultValue float64) (float64, bool) {
	return s.containerMetrics.GetWithDefault(id, metricName, defaultValue)
}

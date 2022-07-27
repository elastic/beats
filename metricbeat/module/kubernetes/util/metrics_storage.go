package util

import (
	"fmt"
	"sync"
)

type Float64Metric struct {
	sync.RWMutex
	metric float64
}

func (m *Float64Metric) NewFloat64Metric() *Float64Metric {
	ans := &Float64Metric{}
	return ans
}

func (m *Float64Metric) Set(v float64) {
	m.Lock()
	defer m.Unlock()
	m.metric = v
}

func (m *Float64Metric) Get() float64 {
	m.RLock()
	defer m.RUnlock()
	return m.metric
}

type Metrics struct {
	// NodeMemAllocatable *Float64Metric
	// NodeCoresAllocatable *Float64Metric
	// ContainerMemLimit *Float64Metric
	// ContainerCoresLimit *Float64Metric
	entries map[int]*Float64Metric
}

func NewMetrics() *Metrics {
	entries := make(map[int]*Float64Metric)

	ans := &Metrics{
		entries: entries,
	}
	return ans
}

type MetricsStorage struct {
	metrics map[string]*Metrics
}

func NewMetricsStorage() *MetricsStorage {
	metrics := make(map[string]*Metrics)

	ans := &MetricsStorage{
		metrics: metrics,
	}
	return ans
}

func (s *MetricsStorage) Clear() {
	for k := range s.metrics {
		delete(s.metrics, k)
	}
}

const CONTAINER_CORES_LIMIT = 1
const CONTAINER_MEMORY_LIMIT = 2
const NODE_CORES_ALLOCATABLE = 3
const NODE_MEMORY_ALLOCATABLE = 4

func (s *MetricsStorage) Delete(id string) {
	// delete(s.metrics, id) // TODO: lock on metrics by id
}

func (s *MetricsStorage) Set(id string, metricName int, metricValue float64) error {
	metrics, exists := s.metrics[id]
	if !exists {
		s.metrics[id] = NewMetrics()
		metrics, exists = s.metrics[id]
		if !exists {
			return fmt.Errorf("Cannot create metrics for id: %s", id)
		}
	}
	metric, exists := metrics.entries[metricName]
	if !exists {
		metrics.entries[metricName] = metric.NewFloat64Metric()
		metric, exists = metrics.entries[metricName]
		if !exists {
			return fmt.Errorf("Cannot create metric for id: %s, name: %v", id, metricName)
		}
	}

	metric.Set(metricValue)

	return nil
}

func (s *MetricsStorage) Get(id string, metricName int) (*Float64Metric, error) {
	metrics, exists := s.metrics[id]
	if !exists {
		return nil, fmt.Errorf("Metrics not found for id: %s", id)
	}

	metric, exists := metrics.entries[metricName]
	if !exists {
		return nil, fmt.Errorf("Metric not found for id: %s, name: %v", id, metricName)
	}

	return metric, nil
}

func (s *MetricsStorage) GetWithDefault(id string, metricName int, defaultValue float64) (float64) {
	metricValue, err := s.Get(id, metricName)
	if err != nil {
		return defaultValue
	}
	return metricValue.Get()
}

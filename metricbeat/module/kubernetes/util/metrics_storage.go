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
	entries map[string]*Float64Metric
}

func NewMetrics() *Metrics {
	entries := make(map[string]*Float64Metric)

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

const CONTAINER_CORES_LIMIT = "ContainerCoresLimit"

func (s *MetricsStorage) Set(id, metricName string, metricValue float64) error {
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
			return fmt.Errorf("Cannot create metric for id: %s, name: %s", id, metricName)
		}
	}

	metric.Set(metricValue)

	return nil
}

func (s *MetricsStorage) Get(id, metricName string) (*Float64Metric, error) {
	metrics, exists := s.metrics[id]
	if !exists {
		return nil, fmt.Errorf("Metrics not found for id: %s", id)
	}

	metric, exists := metrics.entries[metricName]
	if !exists {
		return nil, fmt.Errorf("Metric not found for id: %s, name: %s", id, metricName)
	}

	return metric, nil
}

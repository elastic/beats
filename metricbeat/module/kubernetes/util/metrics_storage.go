package util

import (
	"errors"
	"fmt"
	"sync"
)

type Float64Metric struct {
	sync.RWMutex
	metric float64
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

type MetricsEntry struct {
	// NodeMemAllocatable *Float64Metric
	// NodeCoresAllocatable *Float64Metric
	// ContainerMemLimit *Float64Metric
	// ContainerCoresLimit *Float64Metric
	Entries map[string]*Float64Metric
}

type MetricsStorage struct {
	Metrics map[string]*MetricsEntry
}

func NewMetricsStorage() *MetricsStorage {
	ans := &MetricsStorage{}
	return ans
}

const CONTAINER_CORES_LIMIT = "ContainerCoresLimit"

func (s *MetricsStorage) Set(cuid, metricName string, metricValue float64) error {
	metric, err := s.Get(cuid, metricName)
	if err != nil {
		return err
	}

	metric.Set(metricValue)

	return nil
}


func (s *MetricsStorage) Get(cuid, metricName string) (*Float64Metric, error) {
	metrics, exists := s.Metrics[cuid]
	if !exists {
		return nil, fmt.Errorf("Container id not found: %s", cuid)
	}

	metric, exists := metrics.Entries[metricName]
	if !exists {
		return nil, fmt.Errorf("Metric not found: %s", metricName)
	}

	return metric, nil
}

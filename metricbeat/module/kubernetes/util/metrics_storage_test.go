package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExampleTestSuite struct {
	suite.Suite
	Cuid string
	MetricName int
	AnotherMetricName int
	MetricValue float64
	// Storage *MetricsStorage
	MetricSet *MetricSet
}

func (s *ExampleTestSuite) SetupTest() {
	ns := "namespace"
	pod := "pod"
	container := "container"
	s.Cuid = ContainerUID(ns, pod, container)
	s.MetricName = CONTAINER_CORES_LIMIT
	s.AnotherMetricName = NODE_CORES_ALLOCATABLE
	s.MetricValue = 0.2
	// s.Storage = NewMetricsStorage()
	s.MetricSet = NewMetricSet()
}

func (s *ExampleTestSuite) TestNotFoundSet() {
	s.MetricSet.Clear()
	s.MetricSet.Set(s.Cuid, s.MetricName, s.MetricValue)

	s.assertGetMetric(s.MetricSet, s.Cuid, s.MetricName, s.MetricValue)
}

func (s *ExampleTestSuite) TestSetChange() {
	s.MetricSet.Clear()
	s.MetricSet.Set(s.Cuid, s.MetricName, s.MetricValue)

	s.assertGetMetric(s.MetricSet, s.Cuid, s.MetricName, s.MetricValue)

	changedMetricValue := 0.4

	s.MetricSet.Set(s.Cuid, s.MetricName, changedMetricValue)

	s.assertGetMetric(s.MetricSet, s.Cuid, s.MetricName, changedMetricValue)
}

func (s *ExampleTestSuite) TestIdNotFoundGet() {
	s.MetricSet.Clear()

	_, exists := s.MetricSet.Get(s.Cuid, s.MetricName)
	s.False(exists)
}

func (s *ExampleTestSuite) TestMetricNotFoundGet() {
	s.MetricSet.Clear()

	s.MetricSet.Set(s.Cuid, s.MetricName, s.MetricValue)
	s.assertGetMetric(s.MetricSet, s.Cuid, s.MetricName, s.MetricValue)

	_, exists := s.MetricSet.Get(s.Cuid, s.AnotherMetricName)
	s.False(exists)
}

func (s *ExampleTestSuite) assertGetMetric(metricSet *MetricSet, id string, name int, expectedValue float64) {
	value, exists := s.MetricSet.Get(s.Cuid, s.MetricName)
	s.True(exists)
	s.Equal(expectedValue, value)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}

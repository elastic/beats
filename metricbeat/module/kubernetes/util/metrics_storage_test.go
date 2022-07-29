package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExampleTestSuite struct {
	suite.Suite
	MetricId          string
	MetricName        Metric
	AnotherMetricName Metric
	MetricValue       float64
	MetricsStorage    *MetricsStorage
}

func (s *ExampleTestSuite) SetupTest() {
	ns := "namespace"
	pod := "pod"
	container := "container"
	s.MetricId = ContainerUID(ns, pod, container)
	s.MetricName = ContainerCoresLimitMetric
	s.AnotherMetricName = NodeCoresAllocatableMetric
	s.MetricValue = 0.2
	s.MetricsStorage = NewMetricsStorage()
}

func (s *ExampleTestSuite) TestNotFoundSet() {
	s.MetricsStorage.Clear()
	s.MetricsStorage.Set(s.MetricId, s.MetricName, s.MetricValue)

	s.assertGetMetric(s.MetricsStorage, s.MetricId, s.MetricName, s.MetricValue)
}

func (s *ExampleTestSuite) TestSetChange() {
	s.MetricsStorage.Clear()
	s.MetricsStorage.Set(s.MetricId, s.MetricName, s.MetricValue)

	s.assertGetMetric(s.MetricsStorage, s.MetricId, s.MetricName, s.MetricValue)

	changedMetricValue := 0.4

	s.MetricsStorage.Set(s.MetricId, s.MetricName, changedMetricValue)

	s.assertGetMetric(s.MetricsStorage, s.MetricId, s.MetricName, changedMetricValue)
}

func (s *ExampleTestSuite) TestIdNotFoundGet() {
	s.MetricsStorage.Clear()

	_, exists := s.MetricsStorage.Get(s.MetricId, s.MetricName)
	s.False(exists)
}

func (s *ExampleTestSuite) TestMetricNotFoundGet() {
	s.MetricsStorage.Clear()

	s.MetricsStorage.Set(s.MetricId, s.MetricName, s.MetricValue)
	s.assertGetMetric(s.MetricsStorage, s.MetricId, s.MetricName, s.MetricValue)

	_, exists := s.MetricsStorage.Get(s.MetricId, s.AnotherMetricName)
	s.False(exists)
}

func (s *ExampleTestSuite) assertGetMetric(metricsStorage *MetricsStorage, id string, name Metric, expectedValue float64) {
	value, exists := s.MetricsStorage.Get(s.MetricId, s.MetricName)
	s.True(exists)
	s.Equal(expectedValue, value)
}

func (s *ExampleTestSuite) TestContainerUID() {
	s.Equal("a/b/c", ContainerUID("a", "b", "c"))
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}

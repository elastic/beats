package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExampleTestSuite struct {
	suite.Suite
	Cuid string
	MetricName int
	MetricValue float64
	Storage *MetricsStorage
}

func (s *ExampleTestSuite) SetupTest() {
	ns := "namespace"
	pod := "pod"
	container := "container"
	s.Cuid = ContainerUID(ns, pod, container)
	s.MetricName = CONTAINER_CORES_LIMIT
	s.MetricValue = 0.2
	s.Storage = NewMetricsStorage()
}

func (s *ExampleTestSuite) TestNotFoundSet() {
	err := s.Storage.Set(s.Cuid, s.MetricName, s.MetricValue)
	s.Nil(err)

	metric, err := s.Storage.Get(s.Cuid, s.MetricName)
	s.Nil(err)
	s.NotNil(metric)

	value := metric.Get()
	s.Equal(s.MetricValue, value)
}

func (s *ExampleTestSuite) TestSetChange() {
	err := s.Storage.Set(s.Cuid, s.MetricName, s.MetricValue)
	s.Nil(err)

	s.assertGetMetric(s.Storage, s.Cuid, s.MetricName, s.MetricValue)

	changedMetricValue := 0.4

	err = s.Storage.Set(s.Cuid, s.MetricName, changedMetricValue)
	s.Nil(err)

	s.assertGetMetric(s.Storage, s.Cuid, s.MetricName, changedMetricValue)
}

func (s *ExampleTestSuite) TestIdNotFoundGet() {
	value, err := s.Storage.Get(s.Cuid, s.MetricName)
	s.NotNil(err)
	s.Nil(value)

	s.Equal("Metrics not found for id: namespace/pod/container", err.Error())
}

func (s *ExampleTestSuite) TestMetricNotFoundGet() {
	err := s.Storage.Set(s.Cuid, NODE_MEMORY_ALLOCATABLE, s.MetricValue)
	s.Nil(err)

	value, err := s.Storage.Get(s.Cuid, s.MetricName)
	s.NotNil(err)
	s.Nil(value)
}

func (s *ExampleTestSuite) assertGetMetric(storage *MetricsStorage, id string, name int, expectedValue float64) {
	metric, err := storage.Get(id, name)
	s.Nil(err)
	s.NotNil(metric)

	value := metric.Get()
	s.Equal(expectedValue, value)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}

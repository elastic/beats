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
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExampleTestSuite struct {
	suite.Suite
	RepoId            string
	MetricName        Metric
	AnotherMetricName Metric
	MetricValue       float64
	MetricsRepo       *MetricsRepo
}

func (s *ExampleTestSuite) SetupTest() {
	ns := "namespace"
	pod := "pod"
	container := "container"
	s.RepoId = GetMetricsRepoId(ContainerMetricSource, ContainerUID(ns, pod, container))
	s.MetricName = ContainerCoresLimitMetric
	s.AnotherMetricName = NodeCoresAllocatableMetric
	s.MetricValue = 0.2
	s.MetricsRepo = NewMetricsRepo()
}

func (s *ExampleTestSuite) TestNotFoundSet() {
	s.MetricsRepo.Clear()
	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)
}

func (s *ExampleTestSuite) TestSetChange() {
	s.MetricsRepo.Clear()
	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	changedMetricValue := 0.4

	s.MetricsRepo.Set(s.RepoId, s.MetricName, changedMetricValue)

	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, changedMetricValue)
}

func (s *ExampleTestSuite) TestIdNotFoundGet() {
	s.MetricsRepo.Clear()

	_, exists := s.MetricsRepo.Get(s.RepoId, s.MetricName)
	s.False(exists)
}

func (s *ExampleTestSuite) TestMetricNotFoundGet() {
	s.MetricsRepo.Clear()

	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	_, exists := s.MetricsRepo.Get(s.RepoId, s.AnotherMetricName)
	s.False(exists)
}

func (s *ExampleTestSuite) assertGetMetric(metricsRepo *MetricsRepo, id string, name Metric, expectedValue float64) {
	value, exists := s.MetricsRepo.Get(s.RepoId, s.MetricName)
	s.True(exists)
	s.Equal(expectedValue, value)
}

func (s *ExampleTestSuite) TestContainerUID() {
	s.Equal("a/b/c", ContainerUID("a", "b", "c"))
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}

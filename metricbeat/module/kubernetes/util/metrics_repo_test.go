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

type MetricsRepoTestSuite struct {
	suite.Suite
	RepoId            string
	AnotherRepoId     string
	MetricName        Metric
	AnotherMetricName Metric
	MetricValue       float64
	MetricsRepo       *MetricsRepo
}

func (s *MetricsRepoTestSuite) SetupTest() {
	ns := "namespace"
	pod := "pod"
	container := "container"
	s.RepoId = GetMetricsRepoId(ContainerMetricSource, ContainerUID(ns, pod, container))
	s.AnotherRepoId = GetMetricsRepoId(NodeMetricSource, ContainerUID(ns, pod, container))
	s.MetricName = ContainerCoresLimitMetric
	s.AnotherMetricName = NodeCoresAllocatableMetric
	s.MetricValue = 0.2
	s.MetricsRepo = NewMetricsRepo()
}

func (s *MetricsRepoTestSuite) TestSet() {
	s.MetricsRepo.Clear()
	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)
}

func (s *MetricsRepoTestSuite) TestSetOverwrite() {
	s.MetricsRepo.Clear()
	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	changedMetricValue := 0.4

	s.MetricsRepo.Set(s.RepoId, s.MetricName, changedMetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, changedMetricValue)
}

func (s *MetricsRepoTestSuite) TestSetMultipleMetrics() {
	s.MetricsRepo.Clear()
	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	s.MetricsRepo.Set(s.RepoId, s.AnotherMetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.AnotherMetricName, s.MetricValue)
}

func (s *MetricsRepoTestSuite) TestSetMultipleRepoIds() {
	s.MetricsRepo.Clear()
	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	s.MetricsRepo.Set(s.AnotherRepoId, s.AnotherMetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 2)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)
	s.assertGetMetric(s.MetricsRepo, s.AnotherRepoId, s.AnotherMetricName, s.MetricValue)
}

func (s *MetricsRepoTestSuite) TestGetNotFound() {
	s.MetricsRepo.Clear()

	_, exists := s.MetricsRepo.Get(s.RepoId, s.MetricName)

	s.assertKeysLen(s.MetricsRepo, 0)
	s.False(exists)
}

func (s *MetricsRepoTestSuite) TestGetWithDefaultNotFound() {
	s.MetricsRepo.Clear()

	ans := s.MetricsRepo.GetWithDefault(s.RepoId, s.MetricName, 0.1)

	s.assertKeysLen(s.MetricsRepo, 0)
	s.Equal(0.1, ans)
}

func (s *MetricsRepoTestSuite) TestGetAnotherMetric() {
	s.MetricsRepo.Clear()

	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	_, exists := s.MetricsRepo.Get(s.RepoId, s.AnotherMetricName)
	s.False(exists)
}

func (s *MetricsRepoTestSuite) TestGetWithDefaultAnotherMetric() {
	s.MetricsRepo.Clear()

	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	ans := s.MetricsRepo.GetWithDefault(s.RepoId, s.AnotherMetricName, 0.1)
	s.Equal(0.1, ans)
	s.assertKeysLen(s.MetricsRepo, 1)
}

func (s *MetricsRepoTestSuite) TestDeleteNotFound() {
	s.MetricsRepo.Clear()

	s.MetricsRepo.Delete(s.RepoId)
	s.assertKeysLen(s.MetricsRepo, 0)
}

func (s *MetricsRepoTestSuite) TestDelete() {
	s.MetricsRepo.Clear()

	s.MetricsRepo.Set(s.RepoId, s.MetricName, s.MetricValue)

	s.assertKeysLen(s.MetricsRepo, 1)
	s.assertGetMetric(s.MetricsRepo, s.RepoId, s.MetricName, s.MetricValue)

	s.MetricsRepo.Delete(s.RepoId)

	s.assertKeysLen(s.MetricsRepo, 0)
}

func (s *MetricsRepoTestSuite) assertGetMetric(metricsRepo *MetricsRepo, id string, name Metric, expectedValue float64) {
	value, exists := s.MetricsRepo.Get(s.RepoId, s.MetricName)
	s.True(exists)
	s.Equal(expectedValue, value)
}

func (s *MetricsRepoTestSuite) assertKeysLen(metricsRepo *MetricsRepo, expectedKeysLen int) {
	keys := s.MetricsRepo.Keys()
	s.Equal(expectedKeysLen, len(keys))
}

func (s *MetricsRepoTestSuite) TestContainerUID() {
	s.Equal("a/b/c", ContainerUID("a", "b", "c"))
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsRepoTestSuite))
}

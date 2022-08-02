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
	NodeName               string
	AnotherNodeName        string
	PodId                  PodId
	AnotherPodId           PodId
	ContainerName          string
	AnotherContainerName   string
	ContainerMetric        *ContainerMetrics
	AnotherContainerMetric *ContainerMetrics
	MetricValue            float64
	MetricsRepo            *MetricsRepo
}

func (s *MetricsRepoTestSuite) SetupTest() {
	s.MetricsRepo = NewMetricsRepo()

	s.NodeName = "node"
	s.AnotherNodeName = "anotherNode"

	s.PodId = NewPodId("namespace", "pod")
	s.AnotherPodId = NewPodId("namespace", "pod2")

	s.ContainerName = "container"
	s.AnotherContainerName = "container2"

	s.ContainerMetric = NewContainerMetrics()
	s.ContainerMetric.CoresLimit = NewFloat64Metric(0.2)

	s.AnotherContainerMetric = NewContainerMetrics()
	s.AnotherContainerMetric.CoresLimit = NewFloat64Metric(0.3)
	s.AnotherContainerMetric.MemoryLimit = NewFloat64Metric(50)
}

func (s *MetricsRepoTestSuite) TestNodeNames() {
	s.MetricsRepo.DeleteAll()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.PodId, s.ContainerName, s.ContainerMetric)

	nodeNames := s.MetricsRepo.NodeNames()
	s.Equal(2, len(nodeNames))
	s.Equal(s.NodeName, nodeNames[0])
	s.Equal(s.AnotherNodeName, nodeNames[1])
}

func (s *MetricsRepoTestSuite) TestPodNames() {
	s.MetricsRepo.DeleteAll()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.AnotherPodId, s.ContainerName, s.AnotherContainerMetric)

	podNames := s.MetricsRepo.PodNames(s.NodeName)
	s.Equal(1, len(podNames))
	s.Equal(s.PodId, podNames[0])

	anotherPodNames := s.MetricsRepo.PodNames(s.AnotherNodeName)
	s.Equal(2, len(anotherPodNames))
	s.Equal(s.PodId, anotherPodNames[0])
	s.Equal(s.AnotherPodId, anotherPodNames[1])
}

func (s *MetricsRepoTestSuite) TestSetContainerMetrics() {
	s.MetricsRepo.DeleteAll()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))

	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsOverwrite() {
	s.MetricsRepo.DeleteAll()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsSamePod() {
	s.MetricsRepo.DeleteAll()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.AnotherContainerName, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.AnotherContainerName))

	s.Equal(1, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsMultiplePods() {
	s.MetricsRepo.DeleteAll()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.AnotherPodId, s.ContainerName, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.AnotherPodId, s.ContainerName))

	s.Equal(2, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsMultipleNodes() {
	s.MetricsRepo.DeleteAll()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.AnotherPodId, s.ContainerName, s.AnotherContainerMetric)

	s.Equal(2, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.AnotherNodeName, s.AnotherPodId, s.ContainerName))

	s.Equal(1, len(s.MetricsRepo.PodNames(s.NodeName)))
	s.Equal(1, len(s.MetricsRepo.PodNames(s.AnotherNodeName)))
}

func (s *MetricsRepoTestSuite) TestGetContainerMetricsNotFound() {
	s.MetricsRepo.DeleteAll()

	ans := GetMetric(s.MetricsRepo, s.NodeName, s.AnotherPodId, s.ContainerName)

	s.Equal(0, len(s.MetricsRepo.NodeNames()))
	s.Nil(ans.CoresLimit)
	s.Nil(ans.MemoryLimit)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsRepoTestSuite))
}

func addContainerMetric(metricsRepo *MetricsRepo, nodeName string, podId PodId, containerName string, containerMetric *ContainerMetrics) {
	nodeStore, _ := metricsRepo.Add(nodeName)
	podStore, _ := nodeStore.Add(podId)
	containerMetrics, _ := podStore.Add(containerName)
	containerMetrics.Set(containerMetric)
}

func GetMetric(metricsRepo *MetricsRepo, nodeName string, podId PodId, containerName string) *ContainerMetrics {
	nodeStore := metricsRepo.Get(nodeName)
	podStore := nodeStore.Get(podId)
	containerMetrics := podStore.Get(containerName)
	return containerMetrics
}

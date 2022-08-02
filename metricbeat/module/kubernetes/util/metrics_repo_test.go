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
	ContainerId            ContainerId
	AnotherContainerId     ContainerId
	SecondPodContainerId   ContainerId
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
	s.ContainerId = NewContainerId(s.PodId, "container")
	s.AnotherContainerId = NewContainerId(s.PodId, "container2")

	s.AnotherPodId = NewPodId("namespace", "pod2")
	s.SecondPodContainerId = NewContainerId(s.AnotherPodId, "container")

	s.ContainerMetric = NewContainerMetrics()
	s.ContainerMetric.CoresLimit = NewFloat64Metric(0.2)

	s.AnotherContainerMetric = NewContainerMetrics()
	s.AnotherContainerMetric.CoresLimit = NewFloat64Metric(0.3)
	s.AnotherContainerMetric.MemoryLimit = NewFloat64Metric(50)
}

func (s *MetricsRepoTestSuite) TestNodeNames() {
	s.MetricsRepo.DeleteAllNodes()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)
	s.MetricsRepo.SetContainerMetrics(s.AnotherNodeName, s.ContainerId, s.ContainerMetric)

	nodeNames := s.MetricsRepo.NodeNames()
	s.Equal(2, len(nodeNames))
	s.Equal(s.NodeName, nodeNames[0])
	s.Equal(s.AnotherNodeName, nodeNames[1])
}

func (s *MetricsRepoTestSuite) TestPodNames() {
	s.MetricsRepo.DeleteAllNodes()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)
	s.MetricsRepo.SetContainerMetrics(s.AnotherNodeName, s.ContainerId, s.ContainerMetric)
	s.MetricsRepo.SetContainerMetrics(s.AnotherNodeName, s.SecondPodContainerId, s.AnotherContainerMetric)

	podNames := s.MetricsRepo.PodNames(s.NodeName)
	s.Equal(1, len(podNames))
	s.Equal(s.PodId, podNames[0])

	anotherPodNames := s.MetricsRepo.PodNames(s.AnotherNodeName)
	s.Equal(2, len(anotherPodNames))
	s.Equal(s.PodId, anotherPodNames[0])
	s.Equal(s.AnotherPodId, anotherPodNames[1])
}

func (s *MetricsRepoTestSuite) TestSetContainerMetrics() {
	s.MetricsRepo.DeleteAllNodes()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsOverwrite() {
	s.MetricsRepo.DeleteAllNodes()

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId))

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.AnotherContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsSamePod() {
	s.MetricsRepo.DeleteAllNodes()

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)
	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.AnotherContainerId, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId))
	s.Equal(s.AnotherContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.AnotherContainerId))

	s.Equal(1, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsMultiplePods() {
	s.MetricsRepo.DeleteAllNodes()

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)
	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.SecondPodContainerId, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId))
	s.Equal(s.AnotherContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.SecondPodContainerId))

	s.Equal(2, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsMultipleNodes() {
	s.MetricsRepo.DeleteAllNodes()

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)
	s.MetricsRepo.SetContainerMetrics(s.AnotherNodeName, s.SecondPodContainerId, s.AnotherContainerMetric)

	s.Equal(2, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId))
	s.Equal(s.AnotherContainerMetric, s.MetricsRepo.GetContainerMetrics(s.AnotherNodeName, s.SecondPodContainerId))

	s.Equal(1, len(s.MetricsRepo.PodNames(s.NodeName)))
	s.Equal(1, len(s.MetricsRepo.PodNames(s.AnotherNodeName)))
}

func (s *MetricsRepoTestSuite) TestGetContainerMetricsNotFound() {
	s.MetricsRepo.DeleteAllNodes()

	ans := s.MetricsRepo.GetContainerMetrics(s.NodeName, s.ContainerId)

	s.Equal(0, len(s.MetricsRepo.NodeNames()))
	s.Nil(ans.CoresLimit)
	s.Nil(ans.MemoryLimit)
}

func (s *MetricsRepoTestSuite) TestDeleteNodeNotFound() {
	s.MetricsRepo.DeleteAllNodes()
	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	s.MetricsRepo.DeleteNode(s.NodeName)
	s.Equal(0, len(s.MetricsRepo.NodeNames()))
}

func (s *MetricsRepoTestSuite) TestDeleteNode() {
	s.MetricsRepo.DeleteAllNodes()
	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)
	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(1, len(s.MetricsRepo.PodNames(s.NodeName)))

	s.MetricsRepo.DeleteNode(s.NodeName)
	s.Equal(0, len(s.MetricsRepo.NodeNames()))
	s.Equal(0, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func (s *MetricsRepoTestSuite) TestDeletePodNotFound() {
	s.MetricsRepo.DeleteAllNodes()
	s.Equal(0, len(s.MetricsRepo.NodeNames()))
	s.Equal(0, len(s.MetricsRepo.PodNames(s.NodeName)))

	s.MetricsRepo.DeletePod(s.NodeName, s.ContainerId.PodId)
	s.Equal(0, len(s.MetricsRepo.NodeNames()))
	s.Equal(0, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func (s *MetricsRepoTestSuite) TestDeletePod() {
	s.MetricsRepo.DeleteAllNodes()

	s.MetricsRepo.SetContainerMetrics(s.NodeName, s.ContainerId, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(1, len(s.MetricsRepo.PodNames(s.NodeName)))

	s.MetricsRepo.DeletePod(s.NodeName, s.ContainerId.PodId)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(0, len(s.MetricsRepo.PodNames(s.NodeName)))
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsRepoTestSuite))
}

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
	NodeMetric             *NodeMetrics
	AnotherNodeMetric      *NodeMetrics
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

	s.NodeMetric = NewNodeMetrics()
	s.NodeMetric.CoresAllocatable = NewFloat64Metric(0.001)

	s.AnotherNodeMetric = NewNodeMetrics()
	s.AnotherNodeMetric.CoresAllocatable = NewFloat64Metric(0.002)
	s.AnotherNodeMetric.MemoryAllocatable = NewFloat64Metric(60)
}

func (s *MetricsRepoTestSuite) TestCloneContainerMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	newContainerMetric := s.ContainerMetric.Clone()
	s.Equal(s.ContainerMetric, newContainerMetric)
	s.True(s.ContainerMetric != newContainerMetric)

	anotherNewContainerMetric := s.AnotherContainerMetric.Clone()
	s.Equal(s.AnotherContainerMetric, anotherNewContainerMetric)
	s.True(s.AnotherContainerMetric != anotherNewContainerMetric)
}

func (s *MetricsRepoTestSuite) TestCloneNodeMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	newNodeMetric := s.NodeMetric.Clone()
	s.Equal(s.NodeMetric, newNodeMetric)
	s.True(s.NodeMetric != newNodeMetric)

	anotherNewNodeMetric := s.AnotherNodeMetric.Clone()
	s.Equal(s.AnotherNodeMetric, anotherNewNodeMetric)
	s.True(s.AnotherNodeMetric != anotherNewNodeMetric)
}

func (s *MetricsRepoTestSuite) TestNodeNames() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.PodId, s.ContainerName, s.ContainerMetric)

	nodeNames := s.MetricsRepo.NodeNames()
	s.Equal(2, len(nodeNames))
	s.Contains(nodeNames, s.NodeName)
	s.Contains(nodeNames, s.AnotherNodeName)
}

func (s *MetricsRepoTestSuite) TestPodNames() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.AnotherPodId, s.ContainerName, s.AnotherContainerMetric)

	nodeStore := s.MetricsRepo.GetNodeStore(s.NodeName)
	podNames := nodeStore.PodIds()
	s.Equal(1, len(podNames))
	s.Contains(podNames, s.PodId)

	anotherNodeStore := s.MetricsRepo.GetNodeStore(s.AnotherNodeName)
	anotherPodNames := anotherNodeStore.PodIds()
	s.Equal(2, len(anotherPodNames))
	s.Contains(anotherPodNames, s.PodId)
	s.Contains(anotherPodNames, s.AnotherPodId)
}

func (s *MetricsRepoTestSuite) TestContainerNames() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.AnotherContainerName, s.ContainerMetric)

	nodeNames := s.MetricsRepo.NodeNames()
	s.Equal(1, len(nodeNames))

	nodeStore := s.MetricsRepo.GetNodeStore(s.NodeName)
	podStore := nodeStore.GetPodStore(s.PodId)
	containerNames := podStore.ContainerNames()
	s.Equal(2, len(containerNames))
	s.Contains(containerNames, s.ContainerName)
	s.Contains(containerNames, s.AnotherContainerName)
}

func (s *MetricsRepoTestSuite) TestAddNodeStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, created := s.MetricsRepo.AddNodeStore(s.NodeName)
	s.True(created)

	sameNodeStore, created := s.MetricsRepo.AddNodeStore(s.NodeName)
	s.False(created)

	s.Equal(nodeStore, sameNodeStore)
	s.True(nodeStore == sameNodeStore)

	anotherNodeStore, created := s.MetricsRepo.AddNodeStore(s.AnotherNodeName)
	s.True(created)

	s.NotEqual(nodeStore, anotherNodeStore)
	s.True(nodeStore != anotherNodeStore)
}

func (s *MetricsRepoTestSuite) TestGetNodeStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, created := s.MetricsRepo.AddNodeStore(s.NodeName)
	s.True(created)

	sameNodeStore := s.MetricsRepo.GetNodeStore(s.NodeName)
	s.Equal(nodeStore, sameNodeStore)
	s.True(nodeStore == sameNodeStore)

	anotherNodeStore := s.MetricsRepo.GetNodeStore(s.AnotherNodeName)
	s.NotEqual(nodeStore, anotherNodeStore)
	s.True(nodeStore != anotherNodeStore)
}

func (s *MetricsRepoTestSuite) TestDeleteNodeStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	_, created := s.MetricsRepo.AddNodeStore(s.NodeName)
	s.True(created)

	anotherNodeStore, created := s.MetricsRepo.AddNodeStore(s.AnotherNodeName)
	s.True(created)

	s.Equal(2, len(s.MetricsRepo.NodeNames()))

	s.MetricsRepo.DeleteNodeStore(s.NodeName)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))

	sameAnotherNodeStore := s.MetricsRepo.GetNodeStore(s.AnotherNodeName)
	s.Equal(anotherNodeStore, sameAnotherNodeStore)
	s.True(anotherNodeStore == sameAnotherNodeStore)
}

func (s *MetricsRepoTestSuite) TestAddPodStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	podStore, created := nodeStore.AddPodStore(s.PodId)
	s.True(created)

	samePodStore, created := nodeStore.AddPodStore(s.PodId)
	s.False(created)

	s.Equal(podStore, samePodStore)
	s.True(podStore == samePodStore)

	anotherPodStore, created := nodeStore.AddPodStore(s.AnotherPodId)
	s.True(created)

	s.NotEqual(podStore, anotherPodStore)
	s.True(podStore != anotherPodStore)
}

func (s *MetricsRepoTestSuite) TestGetPodStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	podStore, created := nodeStore.AddPodStore(s.PodId)
	s.True(created)

	samePodStore := nodeStore.GetPodStore(s.PodId)
	s.Equal(podStore, samePodStore)
	s.True(podStore == samePodStore)

	anotherPodStore := nodeStore.GetPodStore(s.AnotherPodId)
	s.NotEqual(podStore, anotherPodStore)
	s.True(podStore != anotherPodStore)
}

func (s *MetricsRepoTestSuite) TestDeletePodStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	_, created := nodeStore.AddPodStore(s.PodId)
	s.True(created)

	anotherPodStore, created := nodeStore.AddPodStore(s.AnotherPodId)
	s.True(created)

	s.Equal(2, len(nodeStore.PodIds()))

	nodeStore.DeletePodStore(s.PodId)
	s.Equal(1, len(nodeStore.PodIds()))

	sameAnotherPodStore := nodeStore.GetPodStore(s.AnotherPodId)
	s.Equal(anotherPodStore, sameAnotherPodStore)
	s.True(anotherPodStore == sameAnotherPodStore)
}

func (s *MetricsRepoTestSuite) TestAddContainerStore() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	podStore, _ := nodeStore.AddPodStore(s.PodId)
	containerStore, created := podStore.AddContainerStore(s.ContainerName)
	s.True(created)

	sameContainerStore, created := podStore.AddContainerStore(s.ContainerName)
	s.False(created)

	s.Equal(containerStore, sameContainerStore)
	s.True(containerStore == sameContainerStore)

	anotherContainerStore, created := podStore.AddContainerStore(s.AnotherContainerName)
	s.True(created)

	s.NotEqual(containerStore, anotherContainerStore)
	s.True(containerStore != anotherContainerStore)
}

func (s *MetricsRepoTestSuite) TestGetContainerMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	podStore, _ := nodeStore.AddPodStore(s.PodId)
	containerStore, created := podStore.AddContainerStore(s.ContainerName)
	s.True(created)

	sameContainerStore := podStore.GetContainerStore(s.ContainerName)
	s.Equal(containerStore, sameContainerStore)
	s.True(containerStore == sameContainerStore)

	anotherContainerStore := podStore.GetContainerStore(s.AnotherContainerName)
	s.NotEqual(containerStore, anotherContainerStore)
	s.True(containerStore != anotherContainerStore)
}

func (s *MetricsRepoTestSuite) TestSetContainerMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.Equal(0, len(s.MetricsRepo.NodeNames()))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))

	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsOverwrite() {
	s.MetricsRepo.DeleteAllNodeStore()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsSamePod() {
	s.MetricsRepo.DeleteAllNodeStore()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.AnotherContainerName, s.AnotherContainerMetric)

	s.Equal(1, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.AnotherContainerName))

	nodeStore := s.MetricsRepo.GetNodeStore(s.NodeName)
	s.Equal(1, len(nodeStore.PodIds()))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsMultiplePods() {
	s.MetricsRepo.DeleteAllNodeStore()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.AnotherPodId, s.ContainerName, s.AnotherContainerMetric)

	nodeStore := s.MetricsRepo.GetNodeStore(s.NodeName)
	s.Equal(2, len(nodeStore.PodIds()))
}

func (s *MetricsRepoTestSuite) TestSetContainerMetricsMultipleNodes() {
	s.MetricsRepo.DeleteAllNodeStore()

	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, s.ContainerMetric)
	addContainerMetric(s.MetricsRepo, s.AnotherNodeName, s.AnotherPodId, s.ContainerName, s.AnotherContainerMetric)

	s.Equal(2, len(s.MetricsRepo.NodeNames()))
	s.Equal(s.ContainerMetric, GetMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName))
	s.Equal(s.AnotherContainerMetric, GetMetric(s.MetricsRepo, s.AnotherNodeName, s.AnotherPodId, s.ContainerName))

	nodeStore := s.MetricsRepo.GetNodeStore(s.NodeName)
	s.Equal(1, len(nodeStore.PodIds()))

	anotherNodeStore := s.MetricsRepo.GetNodeStore(s.AnotherNodeName)
	s.Equal(1, len(anotherNodeStore.PodIds()))
}

func (s *MetricsRepoTestSuite) TestGetContainerMetricsNotFound() {
	s.MetricsRepo.DeleteAllNodeStore()

	ans := GetMetric(s.MetricsRepo, s.NodeName, s.AnotherPodId, s.ContainerName)

	s.Equal(0, len(s.MetricsRepo.NodeNames()))
	s.Nil(ans.CoresLimit)
	s.Nil(ans.MemoryLimit)
}

func TestMetricsRepoTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsRepoTestSuite))
}

func addContainerMetric(metricsRepo *MetricsRepo, nodeName string, podId PodId, containerName string, containerMetric *ContainerMetrics) {
	nodeStore, _ := metricsRepo.AddNodeStore(nodeName)
	podStore, _ := nodeStore.AddPodStore(podId)
	containerStore, _ := podStore.AddContainerStore(containerName)
	containerStore.SetContainerMetrics(containerMetric)
}

func GetMetric(metricsRepo *MetricsRepo, nodeName string, podId PodId, containerName string) *ContainerMetrics {
	nodeStore := metricsRepo.GetNodeStore(nodeName)
	podStore := nodeStore.GetPodStore(podId)
	containerStore := podStore.GetContainerStore(containerName)
	return containerStore.metrics
}

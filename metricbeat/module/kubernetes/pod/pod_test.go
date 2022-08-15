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

//go:build !integration
// +build !integration

package pod

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/suite"
)

const testFile = "../_meta/test/stats_summary.json"
const testFileWithMultipleContainers = "../_meta/test/stats_summary_multiple_containers.json"

type PodTestSuite struct {
	suite.Suite
	MetricsRepo          *util.MetricsRepo
	NodeName             string
	Namespace            string
	PodName              string
	ContainerName        string
	AnotherContainerName string
	PodId                util.PodId
	Logger               *logp.Logger
}

func (s *PodTestSuite) SetupTest() {
	s.MetricsRepo = util.NewMetricsRepo()
	s.NodeName = "gke-beats-default-pool-a5b33e2e-hdww"
	s.Namespace = "default"
	s.PodName = "nginx-deployment-2303442956-pcqfc"
	s.ContainerName = "nginx"
	s.AnotherContainerName = "sidecar"

	s.PodId = util.NewPodId(s.Namespace, s.PodName)

	s.Logger = logp.NewLogger("kubernetes.pod")
}

func (s *PodTestSuite) ReadTestFile(testFile string) []byte {
	f, err := os.Open(testFile)
	s.NoError(err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	s.NoError(err, "cannot read test file "+testFile)

	return body
}

func (s *PodTestSuite) TestEventMapping() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(s.MetricsRepo, s.NodeName, nodeMetrics)

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, containerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		// calculated pct fields:
		"cpu.usage.nanocores": 11263994,
		"cpu.usage.node.pct":  0.005631997,
		"cpu.usage.limit.pct": 0.005631997,

		"memory.usage.bytes":           1462272,
		"memory.usage.node.pct":        0.01,
		"memory.usage.limit.pct":       0.1,
		"memory.working_set.limit.pct": 0.09943977591036414,
	}

	s.RunMetricsTests(events, cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithZeroNodeMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeMetrics := util.NewNodeMetrics()
	addNodeMetric(s.MetricsRepo, s.NodeName, nodeMetrics)

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, containerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		"cpu.usage.nanocores": 11263994,

		"memory.usage.bytes":           1462272,
		"memory.working_set.limit.pct": 0.09943977591036414,
	}

	s.RunMetricsTests(events, cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithNoNodeMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, containerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		"cpu.usage.nanocores": 11263994,

		"memory.usage.bytes":           1462272,
		"memory.usage.limit.pct":       0.1,
		"memory.working_set.limit.pct": 0.09943977591036414,
	}

	s.RunMetricsTests(events, cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithMultipleContainers() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(s.MetricsRepo, s.NodeName, nodeMetrics)

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, containerMetrics)

	body := s.ReadTestFile(testFileWithMultipleContainers)  // NOTE: different test file
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		// Following comments explain what is the difference with the test `TestEventMapping`
		"cpu.usage.nanocores": 22527988,    // 2x usage since 2 container
		"cpu.usage.node.pct":  0.011263994, // 2x usage since 2 container
		"cpu.usage.limit.pct": 0.011263994, // same value as `cpu.usage.node.pct` since `podCoreLimit` = 2x nodeCores = `nodeCores` (capped value)

		"memory.usage.bytes":           2924544,              // 2x since 2 containers
		"memory.usage.node.pct":        0.02,                 // 2x usage since 2 containers
		"memory.usage.limit.pct":       0.02,                 // same value as `cpu.usage.node.pct` since 2 containers but only 1 with limit, podMemLimit = containerMemLimit + nodeLimit > nodeLimit = nodeLimit (capped value)
		"memory.working_set.limit.pct": 0.019887955182072828, // similar concept to `memory.usage.limit.pct`. 2x usage but denominator 10x since nodeLimit = 10x containerMemLimit
	}

	s.RunMetricsTests(events, cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithMultipleContainersWithAllMemLimits() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(s.MetricsRepo, s.NodeName, nodeMetrics)

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.ContainerName, containerMetrics)

	anotherContainerMetrics := util.NewContainerMetrics()
	anotherContainerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(s.MetricsRepo, s.NodeName, s.PodId, s.AnotherContainerName, containerMetrics)

	body := s.ReadTestFile(testFileWithMultipleContainers) // NOTE: different test file
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		// Following comments explain what is the difference with the test `TestEventMapping
		"cpu.usage.nanocores": 22527988,    // 2x usage since 2 container
		"cpu.usage.node.pct":  0.011263994, // 2x usage since 2 container
		"cpu.usage.limit.pct": 0.011263994, // same value as `cpu.usage.node.pct` since `podCoreLimit` = 2x nodeCores = `nodeCores` (capped value)

		"memory.usage.bytes":           2924544,             // 2x since 2 containers
		"memory.usage.node.pct":        0.02,                // 2x usage since 2 containers
		"memory.usage.limit.pct":       0.1,                 // 2x usage / 2x limit = same value
		"memory.working_set.limit.pct": 0.09943977591036414, // 2x usage / 2x limit = same value
	}

	s.RunMetricsTests(events, cpuMemoryTestCases)
}

func (s *PodTestSuite) testValue(event mapstr.M, field string, expected interface{}) {
	data, err := event.GetValue(field)
	s.NoError(err, "Could not read field "+field)
	s.EqualValues(expected, data, "Wrong value for field "+field)
}

func addContainerMetric(metricsRepo *util.MetricsRepo, nodeName string, podId util.PodId, containerName string, containerMetric *util.ContainerMetrics) {
	nodeStore, _ := metricsRepo.AddNodeStore(nodeName)
	podStore, _ := nodeStore.AddPodStore(podId)
	containerStore, _ := podStore.AddContainerStore(containerName)
	containerStore.SetContainerMetrics(containerMetric)
}

func addNodeMetric(metricsRepo *util.MetricsRepo, nodeName string, nodeMetrics *util.NodeMetrics) {
	nodeStore, _ := metricsRepo.AddNodeStore(nodeName)
	nodeStore.SetNodeMetrics(nodeMetrics)
}

func (s *PodTestSuite) basicTests(events []mapstr.M, err error) {
	s.NoError(err, "error mapping "+testFile)

	s.Len(events, 1, "got wrong number of events")

	basicTestCases := map[string]interface{}{
		"name": "nginx-deployment-2303442956-pcqfc",
		"uid":  "beabc196-2456-11e7-a3ad-42010a840235",

		"network.rx.bytes":  107056,
		"network.rx.errors": 0,
		"network.tx.bytes":  72447,
		"network.tx.errors": 0,
	}

	s.RunMetricsTests(events, basicTestCases)
}

func (s *PodTestSuite) RunMetricsTests(events []mapstr.M, testCases map[string]interface{}) {
	for k, v := range testCases {
		s.testValue(events[0], k, v)
	}
}

func TestPodTestSuite(t *testing.T) {
	suite.Run(t, new(PodTestSuite))
}

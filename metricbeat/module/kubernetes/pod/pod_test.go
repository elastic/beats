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

package pod

import (
	"io"
	"os"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/suite"
)

// both these two files are used in tests to compare expected result
const testFile = "../_meta/test/stats_summary.json"
const testFileWithMultipleContainers = "../_meta/test/stats_summary_multiple_containers.json"

type PodTestSuite struct {
	suite.Suite
	MetricsRepo             *util.MetricsRepo
	NodeName                string
	Namespace               string
	PodName                 string
	ContainerName           string
	AnotherContainerName    string
	PodId                   util.PodId
	Logger                  *logp.Logger
	NodeMetrics             *util.NodeMetrics
	ContainerMetrics        *util.ContainerMetrics
	AnotherContainerMetrics *util.ContainerMetrics
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

	s.NodeMetrics = util.NewNodeMetrics()
	s.NodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	s.NodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)

	s.ContainerMetrics = util.NewContainerMetrics()
	s.ContainerMetrics.CoresLimit = util.NewFloat64Metric(0.5)
	s.ContainerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	s.ContainerMetrics.CoresRequest = util.NewFloat64Metric(0.25)
	s.ContainerMetrics.MemoryRequest = util.NewFloat64Metric(7311360)

	s.AnotherContainerMetrics = util.NewContainerMetrics()
	s.AnotherContainerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
}

func (s *PodTestSuite) ReadTestFile(testFile string) []byte {
	f, err := os.Open(testFile)
	s.NoError(err, "cannot open test file "+testFile)

	body, err := io.ReadAll(f)
	s.NoError(err, "cannot read test file "+testFile)

	return body
}

func (s *PodTestSuite) TestEventMapping() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.addNodeMetric(s.NodeMetrics)
	s.addContainerMetric(s.ContainerName, s.ContainerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		// calculated pct fields:
		"cpu.usage.nanocores":   11263994,
		"cpu.usage.node.pct":    0.005631997,
		"cpu.usage.limit.pct":   0.022527988,
		"cpu.limit.cores":       0.5,
		"cpu.request.cores":     0.25,
		"cpu.usage.request.pct": 0.04505597600000000,

		"memory.usage.bytes":             1462272,
		"memory.usage.node.pct":          0.01,
		"memory.usage.limit.pct":         0.1,
		"memory.working_set.limit.pct":   0.09943977591036414,
		"memory.limit.bytes":             14622720.0,
		"memory.request.bytes":           7311360.0,
		"memory.usage.request.pct":       0.2,
		"memory.working_set.request.pct": 0.19887955182072828,
	}

	s.RunMetricsTests(events[0], cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithZeroNodeMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	nodeMetrics := util.NewNodeMetrics()
	s.addNodeMetric(nodeMetrics)

	s.addContainerMetric(s.ContainerName, s.ContainerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		"cpu.usage.nanocores": 11263994,
		"cpu.usage.limit.pct": 0.022527988,

		"memory.usage.bytes":           1462272,
		"memory.usage.limit.pct":       0.1,
		"memory.working_set.limit.pct": 0.09943977591036414,
	}

	s.RunMetricsTests(events[0], cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithNoNodeMetrics() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.addContainerMetric(s.ContainerName, s.ContainerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		"cpu.usage.nanocores": 11263994,
		"cpu.usage.limit.pct": 0.022527988,

		"memory.usage.bytes":           1462272,
		"memory.usage.limit.pct":       0.1,
		"memory.working_set.limit.pct": 0.09943977591036414,
	}

	s.RunMetricsTests(events[0], cpuMemoryTestCases)
}

func (s *PodTestSuite) TestEventMappingWithMultipleContainers_NodeAndOneContainerLimits() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.addNodeMetric(s.NodeMetrics)
	s.addContainerMetric(s.ContainerName, s.ContainerMetrics)

	body := s.ReadTestFile(testFileWithMultipleContainers) // NOTE: different test file
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		// Following comments explain what is the difference with the test `TestEventMapping`
		"cpu.usage.nanocores": 22527988,    // 2x usage since 2 container
		"cpu.usage.node.pct":  0.011263994, // 2x usage since 2 container
		// "cpu.usage.limit.pct" is not reported, since AnotherCntainer does not contain CoresLimit

		"memory.usage.bytes":    2924544, // 2x since 2 containers
		"memory.usage.node.pct": 0.02,    // 2x usage since 2 containers
		// "memory.usage.limit.pct" is not reported, since AnotherContainer metrics were not added
		// "memory.working_set.limit.pct" is not reported, since AnotherContainer metrics were not added
	}

	s.RunMetricsTests(events[0], cpuMemoryTestCases)
}

// Scenario:
// Node metrics are defined,
// Pod contains 2 containers:
// - nginx with both cpu and memory limits and requests defined
// - sidecar with memory limit defined but no requests
func (s *PodTestSuite) TestEventMappingWithMultipleContainers_AllMemLimits() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.addNodeMetric(s.NodeMetrics)
	s.addContainerMetric(s.ContainerName, s.ContainerMetrics)
	s.addContainerMetric(s.AnotherContainerName, s.AnotherContainerMetrics)

	body := s.ReadTestFile(testFileWithMultipleContainers) // NOTE: different test file
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		// Following comments explain what is the difference with the test `TestEventMapping
		"cpu.usage.nanocores": 22527988,    // 2x usage since 2 container
		"cpu.usage.node.pct":  0.011263994, // 2x usage since 2 container
		// "cpu.usage.limit.pct" is not reported, since AnotherContainer does not contain CoresLimit
		// "cpu.request.cores" and "cpu.usage.request.pct" are not reported, since AnotherContainer has no CoresRequest

		"memory.usage.bytes":           2924544,             // 2x since 2 containers
		"memory.usage.node.pct":        0.02,                // 2x usage since 2 containers
		"memory.usage.limit.pct":       0.1,                 // 2x usage / 2x limit = same value
		"memory.working_set.limit.pct": 0.09943977591036414, // 2x usage / 2x limit = same value
		"memory.limit.bytes":           29245440.0,          // 2x since both containers have memory limits
		// "memory.request.bytes" and related pct fields are not reported, since AnotherContainer has no MemoryRequest
	}

	s.RunMetricsTests(events[0], cpuMemoryTestCases)

	// "cpu.limit.cores" is not reported since AnotherContainer has no CoresLimit
	s.assertFieldAbsent(events[0], "cpu.limit.cores")

	// Verify request fields are absent when not all containers have requests
	s.assertFieldAbsent(events[0], "cpu.request.cores")
	s.assertFieldAbsent(events[0], "cpu.usage.request.pct")
	s.assertFieldAbsent(events[0], "memory.request.bytes")
	s.assertFieldAbsent(events[0], "memory.usage.request.pct")
	s.assertFieldAbsent(events[0], "memory.working_set.request.pct")
}

func (s *PodTestSuite) testValue(event mapstr.M, field string, expected interface{}) {
	data, err := event.GetValue(field)
	s.NoError(err, "Could not read field "+field)
	s.EqualValues(expected, data, "Wrong value for field "+field)
}

func (s *PodTestSuite) assertFieldAbsent(event mapstr.M, field string) {
	_, err := event.GetValue(field)
	s.Error(err, "Expected field "+field+" to be absent, but it was present")
}

func (s *PodTestSuite) addContainerMetric(containerName string, containerMetric *util.ContainerMetrics) {
	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	podStore, _ := nodeStore.AddPodStore(s.PodId)
	containerStore, _ := podStore.AddContainerStore(containerName)
	containerStore.SetContainerMetrics(containerMetric)
}

func (s *PodTestSuite) addNodeMetric(nodeMetrics *util.NodeMetrics) {
	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
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

	s.RunMetricsTests(events[0], basicTestCases)
}

func (s *PodTestSuite) RunMetricsTests(events mapstr.M, testCases map[string]interface{}) {
	for k, v := range testCases {
		s.testValue(events, k, v)
	}
}

func TestPodTestSuite(t *testing.T) {
	suite.Run(t, new(PodTestSuite))
}

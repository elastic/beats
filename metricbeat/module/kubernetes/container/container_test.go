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

package container

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// this file is used for the tests to compare expected result
const testFile = "../_meta/test/stats_summary.json"

type ContainerTestSuite struct {
	suite.Suite
	MetricsRepo      *util.MetricsRepo
	NodeName         string
	Namespace        string
	PodName          string
	ContainerName    string
	PodId            util.PodId
	Logger           *logp.Logger
	NodeMetrics      *util.NodeMetrics
	ContainerMetrics *util.ContainerMetrics
}

func (s *ContainerTestSuite) SetupTest() {
	s.MetricsRepo = util.NewMetricsRepo()
	s.NodeName = "gke-beats-default-pool-a5b33e2e-hdww"
	s.Namespace = "default"
	s.PodName = "nginx-deployment-2303442956-pcqfc"
	s.ContainerName = "nginx"

	s.PodId = util.NewPodId(s.Namespace, s.PodName)

	s.Logger = logp.NewLogger("kubernetes.container")

	s.NodeMetrics = util.NewNodeMetrics()
	s.NodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	s.NodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)

	s.ContainerMetrics = util.NewContainerMetrics()
	s.ContainerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
}

func (s *ContainerTestSuite) ReadTestFile(testFile string) []byte {
	f, err := os.Open(testFile)
	s.NoError(err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	s.NoError(err, "cannot read test file "+testFile)

	return body
}

func (s *ContainerTestSuite) TestEventMapping() {
	s.MetricsRepo.DeleteAllNodeStore()

	s.addNodeMetric(s.NodeMetrics)
	s.addContainerMetric(s.ContainerName, s.ContainerMetrics)

	body := s.ReadTestFile(testFile)
	events, err := eventMapping(body, s.MetricsRepo, s.Logger)

	s.basicTests(events, err)

	cpuMemoryTestCases := map[string]interface{}{
		"cpu.usage.core.ns":   43959424,
		"cpu.usage.nanocores": 11263994,

		"memory.available.bytes":  0,
		"memory.usage.bytes":      1462272,
		"memory.rss.bytes":        1409024,
		"memory.workingset.bytes": 1454080,
		"memory.pagefaults":       841,
		"memory.majorpagefaults":  0,

		// calculated pct fields:
		"cpu.usage.node.pct":          0.005631997,
		"cpu.usage.limit.pct":         0.005631997,
		"memory.usage.node.pct":       0.01,
		"memory.usage.limit.pct":      0.1,
		"memory.workingset.limit.pct": 0.09943977591036414,
	}

	s.RunMetricsTests(events[0], cpuMemoryTestCases)

	containerEcsFields := ecsfields(events[0], s.Logger)
	testEcs := map[string]interface{}{
		"cpu.usage":    0.005631997,
		"memory.usage": 0.01,
		"name":         "nginx",
	}
	s.RunMetricsTests(containerEcsFields, testEcs)
}

func (s *ContainerTestSuite) testValue(event mapstr.M, field string, expected interface{}) {
	data, err := event.GetValue(field)
	s.NoError(err, "Could not read field "+field)
	s.EqualValues(expected, data, "Wrong value for field "+field)
}

func (s *ContainerTestSuite) addContainerMetric(containerName string, containerMetric *util.ContainerMetrics) {
	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	podStore, _ := nodeStore.AddPodStore(s.PodId)
	containerStore, _ := podStore.AddContainerStore(containerName)
	containerStore.SetContainerMetrics(containerMetric)
}

func (s *ContainerTestSuite) addNodeMetric(nodeMetrics *util.NodeMetrics) {
	nodeStore, _ := s.MetricsRepo.AddNodeStore(s.NodeName)
	nodeStore.SetNodeMetrics(nodeMetrics)
}

func (s *ContainerTestSuite) basicTests(events []mapstr.M, err error) {
	s.NoError(err, "error mapping "+testFile)

	s.Len(events, 1, "got wrong number of events")

	basicTestCases := map[string]interface{}{
		"logs.available.bytes": int64(98727014400),
		"logs.capacity.bytes":  int64(101258067968),
		"logs.used.bytes":      28672,
		"logs.inodes.count":    6258720,
		"logs.inodes.free":     6120096,
		"logs.inodes.used":     138624,

		"name": "nginx",

		"rootfs.available.bytes": int64(98727014400),
		"rootfs.capacity.bytes":  int64(101258067968),
		"rootfs.used.bytes":      61440,
		"rootfs.inodes.used":     21,
	}

	s.RunMetricsTests(events[0], basicTestCases)
}

func (s *ContainerTestSuite) RunMetricsTests(event mapstr.M, testCases map[string]interface{}) {
	for k, v := range testCases {
		s.testValue(event, k, v)
	}
}

func TestContainerTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerTestSuite))
}

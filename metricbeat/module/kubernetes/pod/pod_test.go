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

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const testFile = "../_meta/test/stats_summary.json"
const testFileWithMultipleContainers = "../_meta/test/stats_summary_multiple_containers.json"

func TestEventMapping(t *testing.T) {
	logger := logp.NewLogger("kubernetes.pod")

	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	metricsRepo := util.NewMetricsRepo()

	nodeName := "gke-beats-default-pool-a5b33e2e-hdww"

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(metricsRepo, nodeName, nodeMetrics)

	namespace := "default"
	podName := "nginx-deployment-2303442956-pcqfc"
	podId := util.NewPodId(namespace, podName)
	containerName := "nginx"

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(metricsRepo, nodeName, podId, containerName, containerMetrics)

	events, err := eventMapping(body, metricsRepo, logger)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"name": "nginx-deployment-2303442956-pcqfc",
		"uid":  "beabc196-2456-11e7-a3ad-42010a840235",

		"network.rx.bytes":  107056,
		"network.rx.errors": 0,
		"network.tx.bytes":  72447,
		"network.tx.errors": 0,

		// calculated pct fields:
		"cpu.usage.nanocores": 11263994,
		"cpu.usage.node.pct":  0.005631997,
		"cpu.usage.limit.pct": 0.005631997,

		"memory.usage.bytes":           1462272,
		"memory.usage.node.pct":        0.01,
		"memory.usage.limit.pct":       0.1,
		"memory.working_set.limit.pct": 0.09943977591036414,
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}
}

func TestEventMappingWithMultipleContainers(t *testing.T) {
	logger := logp.NewLogger("kubernetes.pod")

	f, err := os.Open(testFileWithMultipleContainers)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	metricsRepo := util.NewMetricsRepo()

	nodeName := "gke-beats-default-pool-a5b33e2e-hdww"

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(metricsRepo, nodeName, nodeMetrics)

	namespace := "default"
	podName := "nginx-deployment-2303442956-pcqfc"
	podId := util.NewPodId(namespace, podName)
	containerName := "nginx"

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(metricsRepo, nodeName, podId, containerName, containerMetrics)

	events, err := eventMapping(body, metricsRepo, logger)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"name": "nginx-deployment-2303442956-pcqfc",
		"uid":  "beabc196-2456-11e7-a3ad-42010a840235",

		"network.rx.bytes":  107056,
		"network.rx.errors": 0,
		"network.tx.bytes":  72447,
		"network.tx.errors": 0,

		// calculated pct fields:
		// Following comments explain what is the difference with the test `TestEventMapping`
		"cpu.usage.nanocores": 22527988,    // 2x usage since 2 container
		"cpu.usage.node.pct":  0.011263994, // 2x usage since 2 container
		"cpu.usage.limit.pct": 0.011263994, // same value as `cpu.usage.node.pct` since `podCoreLimit` = 2x nodeCores = `nodeCores` (capped value)

		"memory.usage.bytes":           2924544,              // 2x since 2 containers
		"memory.usage.node.pct":        0.02,                 // 2x usage since 2 containers
		"memory.usage.limit.pct":       0.02,                 // same value as `cpu.usage.node.pct` since 2 containers but only 1 with limit, podMemLimit = containerMemLimit + nodeLimit > nodeLimit = nodeLimit (capped value)
		"memory.working_set.limit.pct": 0.019887955182072828, // similar concept to `memory.usage.limit.pct`. 2x usage but denominator 10x since nodeLimit = 10x containerMemLimit
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}
}

func TestEventMappingWithMultipleContainersWithAllMemLimits(t *testing.T) {
	logger := logp.NewLogger("kubernetes.pod")

	f, err := os.Open(testFileWithMultipleContainers)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	metricsRepo := util.NewMetricsRepo()

	nodeName := "gke-beats-default-pool-a5b33e2e-hdww"

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(metricsRepo, nodeName, nodeMetrics)

	namespace := "default"
	podName := "nginx-deployment-2303442956-pcqfc"
	podId := util.NewPodId(namespace, podName)
	containerName := "nginx"

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(metricsRepo, nodeName, podId, containerName, containerMetrics)

	anotherContainerName := "sidecar"

	anotherContainerMetrics := util.NewContainerMetrics()
	anotherContainerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(metricsRepo, nodeName, podId, anotherContainerName, anotherContainerMetrics)

	events, err := eventMapping(body, metricsRepo, logger)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"name": "nginx-deployment-2303442956-pcqfc",
		"uid":  "beabc196-2456-11e7-a3ad-42010a840235",

		"network.rx.bytes":  107056,
		"network.rx.errors": 0,
		"network.tx.bytes":  72447,
		"network.tx.errors": 0,

		// calculated pct fields:
		// Following comments explain what is the difference with the test `TestEventMapping
		"cpu.usage.nanocores": 22527988,    // 2x usage since 2 container
		"cpu.usage.node.pct":  0.011263994, // 2x usage since 2 container
		"cpu.usage.limit.pct": 0.011263994, // same value as `cpu.usage.node.pct` since `podCoreLimit` = 2x nodeCores = `nodeCores` (capped value)

		"memory.usage.bytes":           2924544,             // 2x since 2 containers
		"memory.usage.node.pct":        0.02,                // 2x usage since 2 containers
		"memory.usage.limit.pct":       0.1,                 // 2x usage / 2x limit = same value
		"memory.working_set.limit.pct": 0.09943977591036414, // 2x usage / 2x limit = same value
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}
}

func testValue(t *testing.T, event mapstr.M, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, expected, data, "Wrong value for field "+field)
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

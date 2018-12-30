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

// +build !integration

package pod

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"
)

const testFile = "../_meta/test/stats_summary.json"

func TestEventMapping(t *testing.T) {
	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	cache := util.NewPerfMetricsCache()
	cache.NodeCoresAllocatable.Set("gke-beats-default-pool-a5b33e2e-hdww", 2)
	cache.NodeMemAllocatable.Set("gke-beats-default-pool-a5b33e2e-hdww", 146227200)
	cache.ContainerMemLimit.Set(util.ContainerUID("default", "nginx-deployment-2303442956-pcqfc", "nginx"), 14622720)

	events, err := eventMapping(body, cache)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"name": "nginx-deployment-2303442956-pcqfc",

		"network.rx.bytes":  107056,
		"network.rx.errors": 0,
		"network.tx.bytes":  72447,
		"network.tx.errors": 0,

		// calculated pct fields:
		"cpu.usage.nanocores": 11263994,
		"cpu.usage.node.pct":  0.005631997,
		"cpu.usage.limit.pct": 0.005631997,

		"memory.usage.bytes":     1462272,
		"memory.usage.node.pct":  0.01,
		"memory.usage.limit.pct": 0.1,
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, expected, data, "Wrong value for field "+field)
}

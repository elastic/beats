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

package node

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

const testFile = "../_meta/test/stats_summary.json"

func TestEventMapping(t *testing.T) {
	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	event, err := eventMapping(body)
	assert.NoError(t, err, "error mapping "+testFile)

	testCases := map[string]interface{}{
		"cpu.usage.core.ns":   int64(4189523881380),
		"cpu.usage.nanocores": 18691146,

		"memory.available.bytes":  1768316928,
		"memory.usage.bytes":      int64(2764943360),
		"memory.rss.bytes":        2150400,
		"memory.workingset.bytes": 2111090688,
		"memory.pagefaults":       131567,
		"memory.majorpagefaults":  103,

		"name": "gke-beats-default-pool-a5b33e2e-hdww",

		"fs.available.bytes": int64(98727014400),
		"fs.capacity.bytes":  int64(101258067968),
		"fs.used.bytes":      int64(2514276352),
		"fs.inodes.used":     138624,
		"fs.inodes.free":     uint64(18446744073709551615),
		"fs.inodes.count":    6258720,

		"network.rx.bytes":  1115133198,
		"network.rx.errors": 0,
		"network.tx.bytes":  812729002,
		"network.tx.errors": 0,

		"runtime.imagefs.available.bytes": int64(98727014400),
		"runtime.imagefs.capacity.bytes":  int64(101258067968),
		"runtime.imagefs.used.bytes":      860204379,
	}

	for k, v := range testCases {
		testValue(t, event, k, v)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, value interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, data, value, "Wrong value for field "+field)
}

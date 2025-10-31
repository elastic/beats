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

package node

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// this file is used for the tests to compare expected result
const testFile = "../_meta/test/stats_summary.json"

type NodeTestSuite struct {
	suite.Suite
	Logger *logp.Logger
}

func (s *NodeTestSuite) SetupTest() {
	s.Logger = logp.NewLogger("kubernetes.node")
}

func (s *NodeTestSuite) ReadTestFile(testFile string) []byte {
	f, err := os.Open(testFile)
	s.NoError(err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	s.NoError(err, "cannot read test file "+testFile)

	return body
}

func (s *NodeTestSuite) TestEventMapping() {
	body := s.ReadTestFile(testFile)
	event, err := eventMapping(body, s.Logger)

	s.basicTests(event, err)
}

func (s *NodeTestSuite) testValue(event mapstr.M, field string, expected interface{}) {
	data, err := event.GetValue(field)
	s.NoError(err, "Could not read field "+field)
	s.EqualValues(expected, data, "Wrong value for field "+field)
}

func (s *NodeTestSuite) basicTests(event mapstr.M, err error) {
	s.NoError(err, "error mapping "+testFile)

	basicTestCases := map[string]interface{}{
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

	s.RunMetricsTests(event, basicTestCases)
}

func (s *NodeTestSuite) RunMetricsTests(event mapstr.M, testCases map[string]interface{}) {
	for k, v := range testCases {
		s.testValue(event, k, v)
	}
}

func TestNodeTestSuite(t *testing.T) {
	suite.Run(t, new(NodeTestSuite))
}

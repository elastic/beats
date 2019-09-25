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

package add_nomad_metadata

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

const allocID = "43205e0e-3d55-f561-83cb-bed15e23b862"

func TestLogsPathMatcherEmpty(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	path := "/var/lib/nomad"
	expectedResult := ""
	executeTest(t, cfgLogsPath, path, expectedResult)
}

func TestLogsPathMatcherWithAllocation(t *testing.T) {
	cfgLogsPath := "/appdata/nomad/alloc/"
	path := "/appdata/nomad/alloc/43205e0e-3d55-f561-83cb-bed15e23b862/alloc/logs/teb-booking-gateway-prod.stdout.94"
	executeTest(t, cfgLogsPath, path, allocID)
}

func executeTest(t *testing.T, cfgLogsPath string, source string, expectedResult string) {
	var cfg = common.NewConfig()
	if cfgLogsPath != "" {
		cfg.SetString("logs_path", -1, cfgLogsPath)
	}

	logMatcher, err := newLogsPathMatcher(*cfg)
	assert.Nil(t, err)

	input := common.MapStr{
		"log": common.MapStr{
			"file": common.MapStr{
				"path": source,
			},
		},
	}

	output := logMatcher.MetadataIndex(input)

	assert.Equal(t, expectedResult, output)
}

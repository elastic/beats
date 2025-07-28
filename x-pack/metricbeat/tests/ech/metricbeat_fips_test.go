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

//go:build ech && requirefips

package fips

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/testing/go-ech"
)

const metricbeatFIPSConfig = `
metricbeat.modules:
  - module: system
    enabled: true
    period: 5s
    metricsets:
      - cpu
      - memory
path.home: %s
logging.to_files: false
logging.to_stderr: true
output:
  elasticsearch:
    hosts:
      - ${ES_HOST}
    protocol: https
    username: ${ES_USER}
    password: ${ES_PASS}
`

// TestMetricbeatFIPSSmoke starts a FIPS compatible binary and ensures that data ends up in an (https) ES cluster.
func TestMetricbeatFIPSSmoke(t *testing.T) {
	ech.VerifyEnvVars(t)
	ech.VerifyFIPSBinary(t, "../../metricbeat")

	ech.RunSmokeTest(t,
		"metricbeat",
		"../../metricbeat",
		fmt.Sprintf(metricbeatFIPSConfig, t.TempDir()),
	)
}

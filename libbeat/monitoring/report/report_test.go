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

package report

import (
	"testing"

	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestMergeHosts(t *testing.T) {
	tests := map[string]struct {
		outCfg      *conf.C
		reporterCfg *conf.C
		expectedCfg *conf.C
	}{
		"no_hosts": {
			expectedCfg: newConfigWithHosts(),
		},
		"only_reporter_hosts": {
			reporterCfg: newConfigWithHosts("r1", "r2"),
			expectedCfg: newConfigWithHosts("r1", "r2"),
		},
		"only_output_hosts": {
			outCfg:      newConfigWithHosts("o1", "o2"),
			expectedCfg: newConfigWithHosts("o1", "o2"),
		},
		"equal_hosts": {
			outCfg:      newConfigWithHosts("o1", "o2"),
			reporterCfg: newConfigWithHosts("r1", "r2"),
			expectedCfg: newConfigWithHosts("r1", "r2"),
		},
		"more_output_hosts": {
			outCfg:      newConfigWithHosts("o1", "o2"),
			reporterCfg: newConfigWithHosts("r1"),
			expectedCfg: newConfigWithHosts("r1"),
		},
		"more_reporter_hosts": {
			outCfg:      newConfigWithHosts("o1"),
			reporterCfg: newConfigWithHosts("r1", "r2"),
			expectedCfg: newConfigWithHosts("r1", "r2"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mergedCfg := conf.MustNewConfigFrom(map[string]interface{}{})
			err := mergeHosts(mergedCfg, test.outCfg, test.reporterCfg)
			require.NoError(t, err)

			require.Equal(t, test.expectedCfg, mergedCfg)
		})
	}
}

func newConfigWithHosts(hosts ...string) *conf.C {
	if len(hosts) == 0 {
		return conf.MustNewConfigFrom(map[string][]string{})
	}
	return conf.MustNewConfigFrom(map[string][]string{"hosts": hosts})
}

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

package outputs

import (
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHostsNumWorkers(t *testing.T) {
	tests := map[string]struct {
		hwc                hostWorkerCfg
		expectedNumWorkers int
	}{
		"worker_set":  {hwc: hostWorkerCfg{Worker: 17}, expectedNumWorkers: 17},
		"workers_set": {hwc: hostWorkerCfg{Workers: 23}, expectedNumWorkers: 23},
		"both_set":    {hwc: hostWorkerCfg{Worker: 17, Workers: 23}, expectedNumWorkers: 17},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expectedNumWorkers, test.hwc.NumWorkers())
		})
	}
}

func TestReadHostList(t *testing.T) {
	tests := map[string]struct {
		cfg           map[string]interface{}
		expectedHosts []string
	}{
		"one_host_no_worker_set": {
			cfg: map[string]interface{}{
				"hosts": []string{"foo.bar"},
			},
			expectedHosts: []string{"foo.bar"},
		},
		"one_host_worker_set": {
			cfg: map[string]interface{}{
				"hosts":  []string{"foo.bar"},
				"worker": 3,
			},
			expectedHosts: []string{"foo.bar", "foo.bar", "foo.bar"},
		},
		"one_host_workers_set": {
			cfg: map[string]interface{}{
				"hosts":   []string{"foo.bar"},
				"workers": 2,
			},
			expectedHosts: []string{"foo.bar", "foo.bar"},
		},
		"one_host_worker_workers_both_set": {
			cfg: map[string]interface{}{
				"hosts":   []string{"foo.bar"},
				"worker":  3,
				"workers": 2,
			},
			expectedHosts: []string{"foo.bar", "foo.bar", "foo.bar"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(test.cfg)
			require.NoError(t, err)

			hosts, err := ReadHostList(cfg)
			require.NoError(t, err)
			require.Equal(t, test.expectedHosts, hosts)
		})
	}
}

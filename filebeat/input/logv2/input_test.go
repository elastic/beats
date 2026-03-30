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

package logv2

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRunAsFilestream(t *testing.T) {
	testCases := map[string]struct {
		cfg        *config.C
		expectErr  bool
		expected   bool
		underAgent bool
	}{
		"simplest log input config": {
			underAgent: true,
			expected:   false,
			cfg: config.MustNewConfigFrom(map[string]any{
				"paths": []string{"/var/log.log"},
			}),
		},
		"log input invalid config": {
			// empty config is always invalid
			cfg:       config.NewConfig(),
			expectErr: true,
		},
		"invalid 'run_as_filestream'": {
			underAgent: true,
			cfg: config.MustNewConfigFrom(map[string]any{
				"paths":             []string{"/var/log.log"},
				"run_as_filestream": 42,
			}),
			expectErr: true,
		},
		"no filestream id": {
			underAgent: true,
			cfg: config.MustNewConfigFrom(map[string]any{
				"paths":             []string{"/var/log.log"},
				"run_as_filestream": true,
			}),
			expectErr: true,
		},
		"not under Elastic Agent": {
			underAgent: false,
			cfg: config.MustNewConfigFrom(map[string]any{
				"paths": []string{"/var/log.log"},
			}),
			expectErr: false,
			expected:  false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			underAgent := management.UnderAgent()
			t.Cleanup(func() {
				management.SetUnderAgent(underAgent)
			})
			management.SetUnderAgent(tc.underAgent)

			got, err := runAsFilestream(logp.NewNopLogger(), tc.cfg)
			if err != nil && !tc.expectErr {
				t.Errorf("did not expect an error: %s", err)
			}

			if got != tc.expected {
				t.Errorf("expecting 'runAsFilestream' to return %t, got %t", tc.expected, got)
			}
		})
	}
}

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

package filestream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestCreateProspector(t *testing.T) {
	t.Run("SetIgnoreInactiveSince", func(t *testing.T) {
		testCases := map[string]struct {
			ignore_inactive_since string
		}{
			"ignore_inactive_since set to since_last_start": {
				ignore_inactive_since: "since_last_start",
			},
			"ignore_inactive_since set to since_first_start": {
				ignore_inactive_since: "since_first_start",
			},
			"ignore_inactive_since not set": {
				ignore_inactive_since: "",
			},
		}
		for name, test := range testCases {
			test := test
			t.Run(name, func(t *testing.T) {
				c := config{
					IgnoreInactive: ignoreInactiveSettings[test.ignore_inactive_since],
				}
				p, _ := newProspector(c)
				fileProspector := p.(*fileProspector)
				assert.Equal(t, fileProspector.ignoreInactiveSince, ignoreInactiveSettings[test.ignore_inactive_since])
			})
		}
	})
	t.Run("file watcher and file identity compatibility", func(t *testing.T) {
		cases := []struct {
			name   string
			cfgStr string
			err    string
		}{
			{
				name: "returns no error for a fully default config",
				cfgStr: `
paths: ['some']
`,
			},
			{
				name: "returns no error when fingerprint and identity is configured",
				cfgStr: `
paths: ['some']
file_identity.fingerprint: ~
prospector.scanner.fingerprint.enabled: true
`,
			},
			{
				name: "returns no error when fingerprint and other identity is configured",
				cfgStr: `
paths: ['some']
file_identity.path: ~
prospector.scanner.fingerprint.enabled: true
`,
			},
			{
				name: "returns error when fingerprint is disabled but fingerprint identity is configured",
				cfgStr: `
paths: ['some']
file_identity.fingerprint: ~
prospector.scanner.fingerprint.enabled: false
`,
				err: "fingerprint file identity can be used only when fingerprint is enabled in the scanner",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c, err := conf.NewConfigWithYAML([]byte(tc.cfgStr), tc.cfgStr)
				require.NoError(t, err)

				cfg := defaultConfig()
				err = c.Unpack(&cfg)
				require.NoError(t, err)

				_, err = newProspector(cfg)
				if tc.err == "" {
					require.NoError(t, err)
					return
				}

				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			})
		}
	})
}

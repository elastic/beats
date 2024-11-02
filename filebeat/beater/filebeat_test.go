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

package beater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/filebeat/config"
	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/require"
)

type inputEntry struct {
	ID string `config:"id"`
}

func TestFetchInputConfiguration(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "config1.yml"), []byte(`
- type: filestream
  id: external-1
  paths:
    - "/some"
- type: filestream
  id: external-2
  paths:
    - "/another"
`), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config2.yml.disabled"), []byte(`
- type: filestream
  id: disabled
  paths:
    - "/some"
`), 0777)
	require.NoError(t, err)

	cases := []struct {
		name       string
		configFile string
		expected   []inputEntry
	}{
		{
			name: "loads mixed configuration",
			configFile: `
filebeat.config.inputs:
  enabled: true
  path: ` + dir + `/*.yml
filebeat.inputs:
  - type: filestream
    id: internal
    paths:
      - "/another"
output.console:
  enabled: true
`,
			expected: []inputEntry{
				{
					ID: "internal",
				},
				{
					ID: "external-1",
				},
				{
					ID: "external-2",
				},
			},
		},
		{
			name: "loads only internal configuration",
			configFile: `
filebeat.inputs:
  - type: filestream
    id: internal
    paths:
      - "/another"
output.console:
  enabled: true
`,
			expected: []inputEntry{
				{
					ID: "internal",
				},
			},
		},
		{
			name: "loads only external configuration",
			configFile: `
filebeat.config.inputs:
  enabled: true
  path: ` + dir + `/*.yml
output.console:
  enabled: true
`,
			expected: []inputEntry{
				{
					ID: "external-1",
				},
				{
					ID: "external-2",
				},
			},
		},
		{
			name: "loads nothing",
			configFile: `
filebeat.config.inputs:
  enabled: true
  path: ` + dir + `/*.nothing
output.console:
  enabled: true
`,
			expected: []inputEntry{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawConfig, err := conf.NewConfigFrom(tc.configFile)
			require.NoError(t, err)

			cfg := struct {
				Filebeat config.Config `config:"filebeat"`
			}{
				Filebeat: config.DefaultConfig,
			}
			err = rawConfig.Unpack(&cfg)
			require.NoError(t, err)

			inputs, err := fetchInputConfiguration(&cfg.Filebeat)
			require.NoError(t, err)

			actual := []inputEntry{}

			for _, i := range inputs {
				var entry inputEntry
				err := i.Unpack(&entry)
				require.NoError(t, err)
				actual = append(actual, entry)
			}

			require.Equal(t, tc.expected, actual)
		})
	}
}

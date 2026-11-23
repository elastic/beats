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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/beats/v7/pkg/config"
)

func TestTranslateConfig(t *testing.T) {
	const noEventLogs = "at least one event log must be configured as part of event_logs"

	tests := []struct {
		name             string
		in               string
		want             map[string]any
		wantRegistryFile string
		wantOverwrite    bool
		wantEventLogs    int
		wantErr          string
	}{
		{
			name: "event logs become winlog inputs",
			in: `
event_logs:
- name: Security
  id: sec
  processors:
  - drop_fields:
      fields: [message]
  tags: [a, b]
  fields:
    env: prod
  index: my-index
  ignore_older: 72h
- name: Application
`,
			want: map[string]any{
				"inputs": []any{
					map[string]any{
						"type":         "winlog",
						"name":         "Security",
						"id":           "sec",
						"processors":   []any{map[string]any{"drop_fields": map[string]any{"fields": []any{"message"}}}},
						"tags":         []any{"a", "b"},
						"fields":       map[string]any{"env": "prod"},
						"index":        "my-index",
						"ignore_older": "72h",
					},
					map[string]any{
						"type": "winlog",
						"name": "Application",
					},
				},
			},
			wantRegistryFile: ".winlogbeat.yml",
			wantEventLogs:    2,
		},
		{
			name: "registry file and flush",
			in: `
event_logs:
- name: Security
registry_file: state/x.yml
registry_flush: 10s
`,
			want: map[string]any{
				"inputs":   []any{map[string]any{"type": "winlog", "name": "Security"}},
				"registry": map[string]any{"flush": "10s"},
			},
			wantRegistryFile: "state/x.yml",
			wantEventLogs:    1,
		},
		{
			name: "no registry settings",
			in: `
event_logs:
- name: Security
`,
			want: map[string]any{
				"inputs": []any{map[string]any{"type": "winlog", "name": "Security"}},
			},
			wantRegistryFile: ".winlogbeat.yml",
			wantEventLogs:    1,
		},
		{
			name: "other keys copied",
			in: `
event_logs:
- name: Security
shutdown_timeout: 30s
overwrite_pipelines: true
`,
			want: map[string]any{
				"inputs":              []any{map[string]any{"type": "winlog", "name": "Security"}},
				"shutdown_timeout":    "30s",
				"overwrite_pipelines": true,
			},
			wantRegistryFile: ".winlogbeat.yml",
			wantOverwrite:    true,
			wantEventLogs:    1,
		},
		{
			name:    "empty event logs",
			in:      "event_logs: []",
			wantErr: noEventLogs,
		},
		{
			name:    "missing event logs",
			in:      "shutdown_timeout: 30s",
			wantErr: noEventLogs,
		},
		{
			name: "disabled event log dropped",
			in: `
event_logs:
- name: Security
- name: Application
  enabled: false
`,
			want: map[string]any{
				"inputs": []any{map[string]any{"type": "winlog", "name": "Security"}},
			},
			wantRegistryFile: ".winlogbeat.yml",
			wantEventLogs:    1,
		},
		{
			name: "registry path kept",
			in: `
event_logs:
- name: Security
registry_file: a.yml
registry.path: custom
`,
			want: map[string]any{
				"inputs":   []any{map[string]any{"type": "winlog", "name": "Security"}},
				"registry": map[string]any{"path": "custom"},
			},
			wantRegistryFile: "a.yml",
			wantEventLogs:    1,
		},
		{
			name: "type overwritten",
			in: `
event_logs:
- name: Security
  type: something
`,
			want: map[string]any{
				"inputs": []any{map[string]any{"type": "winlog", "name": "Security"}},
			},
			wantRegistryFile: ".winlogbeat.yml",
			wantEventLogs:    1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			old, err := conf.NewConfigWithYAML([]byte(test.in), "test")
			require.NoError(t, err, "failed to parse test config")

			got, err := translateConfig(old)
			if test.wantErr != "" {
				require.Error(t, err, "expected an error")
				assert.Contains(t, err.Error(), test.wantErr, "unexpected error text")
				return
			}
			require.NoError(t, err, "unexpected error")

			var fb map[string]any
			require.NoError(t, got.Filebeat.Unpack(&fb), "failed to unpack Filebeat config")
			assert.Equal(t, test.want, fb, "unexpected Filebeat config")
			assert.Equal(t, test.wantRegistryFile, got.RegistryFile, "unexpected RegistryFile")
			assert.Equal(t, test.wantOverwrite, got.OverwritePipelines, "unexpected OverwritePipelines")
			assert.Len(t, got.EventLogs, test.wantEventLogs, "unexpected number of EventLogs")
		})
	}
}

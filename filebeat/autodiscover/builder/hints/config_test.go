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

package hints

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestUnpackCopiesDefault(t *testing.T) {
	userCfg := conf.MustNewConfigFrom(mapstr.M{
		"default_config": mapstr.M{
			"type": "container",
			"paths": []string{
				"/var/log/containers/*${data.kubernetes.container.id}.log",
			},
		},
	})

	cfg1 := defaultConfig()
	assert.NoError(t, userCfg.Unpack(&cfg1))

	cfg2 := defaultConfig()
	assert.NoError(t, userCfg.Unpack(&cfg2))

	assert.NotEqual(t, cfg1.DefaultConfig, cfg2.DefaultConfig)
}

func TestDefaultConfigContainsCloseRemovedFalse(t *testing.T) {
	cfg := defaultConfig()
	closeRemoved, err := cfg.DefaultConfig.Bool("close.on_state_change.removed", -1)
	if err != nil {
		t.Fatalf("cannot get 'close.on_state_change.removed': %s", err)
	}

	// 'close.on_state_change.removed' to prevent missing log lines at the
	// end of files when using autodiscover, this is specially common on
	// Kubernetes.
	if closeRemoved {
		t.Fatalf("'close.on_state_change.removed' must be false")
	}
}

func TestInputAllowListConfig(t *testing.T) {
	tests := []struct {
		name        string
		userConfig  mapstr.M
		wantEnabled bool
		wantTypes   []string
	}{
		{
			name:       "empty",
			userConfig: mapstr.M{},
		},
		{
			name: "disabled",
			userConfig: mapstr.M{
				"input_allow_list": mapstr.M{
					"enabled": false,
					"types":   []string{"httpjson"},
				},
			},
			wantTypes: []string{"httpjson"},
		},
		{
			name: "enabled with omitted types",
			userConfig: mapstr.M{
				"input_allow_list": mapstr.M{
					"enabled": true,
				},
			},
			wantEnabled: true,
			wantTypes:   []string{"log", "filestream", "container"},
		},
		{
			name: "enabled with empty types",
			userConfig: mapstr.M{
				"input_allow_list": mapstr.M{
					"enabled": true,
					"types":   []string{},
				},
			},
			wantEnabled: true,
			wantTypes:   []string{"log", "filestream", "container"},
		},
		{
			name: "enabled with custom types",
			userConfig: mapstr.M{
				"input_allow_list": mapstr.M{
					"enabled": true,
					"types":   []string{"httpjson", "cel"},
				},
			},
			wantEnabled: true,
			wantTypes:   []string{"httpjson", "cel"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig()
			userConfig := conf.MustNewConfigFrom(test.userConfig)

			assert.NoError(t, userConfig.Unpack(&cfg), "input allow list config should unpack")
			assert.Equal(t, test.wantEnabled, cfg.InputAllowList.Enabled,
				"input allow list enabled state should match")
			assert.Equal(t, test.wantTypes, cfg.InputAllowList.Types,
				"input allow list types should match")
		})
	}
}

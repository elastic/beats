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

package ilm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

func TestConfig_Unpack(t *testing.T) {
	testdata := []struct {
		input common.MapStr
		cfg   Config
		err   string
		name  string
	}{
		{name: "empty config", input: nil, cfg: Config{}, err: "rollover_alias must be set"},
		{name: "ilm disabled", input: common.MapStr{"enabled": false}, cfg: Config{
			Enabled: ModeDisabled, Pattern: DefaultPattern, Policy: PolicyCfg{Name: DefaultPolicyName}}},
		{name: "default with rollover_alias", input: common.MapStr{"rollover_alias": "testbeat"}, cfg: Config{
			Enabled: ModeAuto, RolloverAlias: "testbeat", Pattern: DefaultPattern, Policy: PolicyCfg{Name: DefaultPolicyName}}},
		{name: "ilm enabled", input: common.MapStr{"rollover_alias": "testbeat", "enabled": "True", "pattern": "01"}, cfg: Config{
			Enabled: ModeEnabled, RolloverAlias: "testbeat", Pattern: "01", Policy: PolicyCfg{Name: DefaultPolicyName}}},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(td.input)
			require.NoError(t, err)
			var ilm Config
			err = cfg.Unpack(&ilm)
			if td.err == "" {
				require.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), td.err)
			}
			assert.Equal(t, td.cfg, ilm)

		})
	}
}

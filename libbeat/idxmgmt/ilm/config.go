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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
)

// Config is used for unpacking a config.C.
type Config struct {
	Enabled    bool                     `config:"enabled"`
	PolicyName fmtstr.EventFormatString `config:"policy_name"`
	PolicyFile string                   `config:"policy_file"`

	// CheckExists can disable the check for an existing policy. Check required
	// read_ilm privileges.  If check is disabled the policy will only be
	// installed if Overwrite is enabled.
	CheckExists bool `config:"check_exists"`

	// Enable always overwrite policy mode. This required manage_ilm privileges.
	Overwrite bool `config:"overwrite"`
}

// DefaultPolicy defines the default policy to be used if no custom policy is
// configured.
// By default the policy contains not warm, cold, or delete phase.
// The index is configured to rollover every 50GB or after 30d.
var DefaultPolicy = common.MapStr{
	"policy": common.MapStr{
		"phases": common.MapStr{
			"hot": common.MapStr{
				"actions": common.MapStr{
					"rollover": common.MapStr{
						"max_size": "50gb",
						"max_age":  "30d",
					},
				},
			},
		},
	},
}

//Validate verifies that expected config options are given and valid
func (cfg *Config) Validate() error {
	return nil
}

func defaultConfig(info beat.Info) Config {
	policyFmt := fmtstr.MustCompileEvent(info.Beat)

	return Config{
		Enabled:     true,
		PolicyName:  *policyFmt,
		PolicyFile:  "",
		CheckExists: true,
	}
}

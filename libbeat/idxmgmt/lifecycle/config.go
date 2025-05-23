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

package lifecycle

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Config is used for unpacking a config.C.
type Config struct {
	Enabled bool `config:"enabled"`
	// PolicyName, used by ILM
	PolicyName fmtstr.EventFormatString `config:"policy_name"`
	PolicyFile string                   `config:"policy_file"`
	// used only for testing
	policyRaw *Policy

	// CheckExists can disable the check for an existing policy. This check
	// requires read_ilm privileges. If CheckExists is set to false, the policy
	// will not be installed, even if Overwrite is enabled.
	CheckExists bool `config:"check_exists"`

	// Enable always overwrite policy mode. This required manage_ilm privileges.
	Overwrite bool `config:"overwrite"`
}

// DSLNameConfig just stores the datastream name for the DSL policy
// as this is the only config value that differs between ILM and DSL
type DSLNameConfig struct {
	DataStreamPattern fmtstr.EventFormatString `config:"data_stream_pattern"`
}

func DefaultDSLName() DSLNameConfig {
	return DSLNameConfig{
		DataStreamPattern: *fmtstr.MustCompileEvent("%{[beat.name]}-%{[beat.version]}"),
	}
}

// LifecycleConfig maps all possible ILM/DSL config values present in a config
type LifecycleConfig struct {
	ILM Config `config:"setup.ilm"`
	DSL Config `config:"setup.dsl"`
}

// RawConfig half-unpacks the policy config, allowing us to tell if a user has explicitly
// enabled a given config value
type RawConfig struct {
	ILM          *config.C `config:"setup.ilm"`
	DSL          *config.C `config:"setup.dsl"`
	TemplateName string    `config:"setup.template.name"`
}

// DefaultILMPolicy defines the default policy to be used if no custom policy is
// configured.
// By default the policy contains not warm, cold, or delete phase.
// The index is configured to rollover every 50GB or after 30d.
var DefaultILMPolicy = mapstr.M{
	"policy": mapstr.M{
		"phases": mapstr.M{
			"hot": mapstr.M{
				"actions": mapstr.M{
					"rollover": mapstr.M{
						"max_primary_shard_size": "50gb",
						"max_age":                "30d",
					},
				},
			},
		},
	},
}

// DefaultDSLPolicy defines the default policy to be used for DSL if
// no custom policy is configured
var DefaultDSLPolicy = mapstr.M{
	"data_retention": "7d",
}

// Validate verifies that expected config options are given and valid
func (cfg *Config) Validate() error {
	return nil
}

func DefaultILMConfig(info beat.Info) LifecycleConfig {
	return LifecycleConfig{
		ILM: Config{
			Enabled:     true,
			PolicyName:  *fmtstr.MustCompileEvent(info.Beat),
			PolicyFile:  "",
			CheckExists: true,
		},
		DSL: Config{
			Enabled:     false,
			PolicyName:  *fmtstr.MustCompileEvent("%{[beat.name]}-%{[beat.version]}"),
			CheckExists: true,
		},
	}
}

func DefaultDSLConfig(info beat.Info) LifecycleConfig {
	return LifecycleConfig{
		ILM: Config{
			Enabled:     false,
			PolicyName:  *fmtstr.MustCompileEvent(info.Beat),
			PolicyFile:  "",
			CheckExists: true,
		},
		DSL: Config{
			Enabled:     true,
			PolicyName:  *fmtstr.MustCompileEvent("%{[beat.name]}-%{[beat.version]}"),
			PolicyFile:  "",
			CheckExists: true,
		},
	}
}

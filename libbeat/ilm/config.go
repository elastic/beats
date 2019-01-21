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
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/fmtstr"

	"github.com/elastic/beats/libbeat/common"
)

//Config holds all config options supported for ILM configurations
type Config struct {
	Enabled       Mode      `config:"enabled"`
	RolloverAlias string    `config:"rollover_alias"`
	Pattern       string    `config:"pattern"`
	Policy        PolicyCfg `config:"policy"`
}

//PolicyCfg holds config options for ILM policies
type PolicyCfg struct {
	Name string `config:"name"`
	Path string `config:"path"`
}

//DefaultILMConfig sets default values
func DefaultILMConfig() Config {
	return Config{
		Enabled: ModeAuto,
		Pattern: DefaultPattern,
		Policy:  PolicyCfg{Name: DefaultPolicyName},
	}
}

//DefaultPattern used for ilm aliases
const DefaultPattern = "000001"

//DefaultPolicyName used for creating ilm aliases
var DefaultPolicyName = "beatDefaultPolicy"

//Mode is used for enumerating the ilm mode.
type Mode uint8

const (
	//ModeAuto enum 'auto'
	ModeAuto Mode = iota
	//ModeEnabled enum 'true'
	ModeEnabled
	//ModeDisabled enum 'false'
	ModeDisabled
)

//Unpack creates enumeration value true, false or auto
func (m *Mode) Unpack(in string) error {
	switch strings.ToLower(in) {
	case "auto":
		*m = ModeAuto
	case "true":
		*m = ModeEnabled
	case "false":
		*m = ModeDisabled
	default:
		return fmt.Errorf("ilm.enabled` mode '%v' is invalid (try auto, true, false)", in)
	}

	return nil
}

//Validate verifies that expected config options are given and valid
func (cfg *Config) Validate() error {
	if cfg.RolloverAlias == "" && cfg.Enabled != ModeDisabled {
		return fmt.Errorf("rollover_alias must be set when ILM is not disabled")
	}
	return nil
}

//Unpack sets the Config instance to the given values
func (cfg *Config) Unpack(c *common.Config) error {
	type tmpCfg Config
	var tmp = tmpCfg(DefaultILMConfig())
	if err := c.Unpack(&tmp); err != nil {
		return err
	}
	*cfg = Config(tmp)
	return nil
}

func (cfg *Config) prepare(info beat.Info) error {
	alias := cfg.RolloverAlias
	policyName := cfg.Policy.Name
	if policyName == "" {
		policyName = DefaultPolicyName
	}

	event := &beat.Event{
		Fields: common.MapStr{
			"beat": common.MapStr{
				"name":    info.IndexPrefix,
				"type":    info.IndexPrefix,
				"version": info.Version,
			},
			"agent": common.MapStr{
				"name":    info.IndexPrefix,
				"type":    info.IndexPrefix,
				"version": info.Version,
			},
			"observer": common.MapStr{
				"name":    info.IndexPrefix,
				"type":    info.IndexPrefix,
				"version": info.Version,
			},
		},
		Timestamp: time.Now(),
	}

	aliasFormatter, err := fmtstr.CompileEvent(alias)
	if err != nil {
		return err
	}
	alias, err = aliasFormatter.Run(event)
	if err != nil {
		return err
	}
	cfg.RolloverAlias = alias

	policyNameFormatter, err := fmtstr.CompileEvent(policyName)
	if err != nil {
		return err
	}
	policyName, err = policyNameFormatter.Run(event)
	if err != nil {
		return err
	}
	cfg.Policy.Name = policyName

	return nil
}

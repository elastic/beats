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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/fmtstr"

	"github.com/elastic/beats/libbeat/common"
)

//Configs holds a collection of Config entries
type Configs []Config

//Config holds all config options supported for ILM configurations
type Config struct {
	Enabled       string    `config:"enabled"`
	RolloverAlias string    `config:"rollover_alias"`
	Pattern       string    `config:"pattern"`
	Policy        policyCfg `config:"policy"`
}

type policyCfg struct {
	Name string `config:"name"`
	Path string `config:"path"`
}

//Unpack sets the Config instance to the given values
func (cfg *Config) Unpack(c *common.Config) error {
	type tmpCfg Config
	var tmp = tmpCfg(*defaultILMConfig())
	if err := c.Unpack(&tmp); err != nil {
		return err
	}
	*cfg = Config(tmp)
	return nil
}

//Validate checks if the given configuration is valid
func (cfg *Config) Validate() error {
	if _, err := strconv.ParseBool(cfg.Enabled); err != nil && !cfg.EnabledAuto() {
		return fmt.Errorf("validation error for ilm config section: `ilm.enabled: %s` is invalid", cfg.Enabled)
	}

	if !cfg.EnabledFalse() && cfg.RolloverAlias == "" {
		return errors.New("validation error for ilm config section: ilm.rollover_alias must be set when ilm is not disabled")
	}
	return nil
}

//EnabledFalse indicates if ILM is disabled
func (cfg *Config) EnabledFalse() bool {
	if e, err := strconv.ParseBool(cfg.Enabled); err == nil {
		return !e
	}
	return false
}

//EnabledAuto indicates if ILM is set to `auto`. If true,
//ILM is enabled by default if the configured output can handle it.
func (cfg *Config) EnabledAuto() bool {
	return strings.ToLower(cfg.Enabled) == "auto"
}

func (cfg *Config) prepare(info beat.Info) error {
	alias := cfg.RolloverAlias
	policyName := cfg.Policy.Name
	if policyName == "" {
		policyName = defaultPolicyName
	}

	event := &beat.Event{
		Fields: common.MapStr{
			"agent": common.MapStr{
				"name":    info.Name,
				"version": info.Version,
			},
			"observer": common.MapStr{
				"name":    info.Name,
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

func defaultILMConfig() *Config {
	return &Config{
		Enabled: "auto",
		Pattern: defaultPattern,
		Policy:  policyCfg{Name: defaultPolicyName},
	}
}

const defaultPattern = "000001"

var defaultPolicyName = "beatDefaultPolicy"

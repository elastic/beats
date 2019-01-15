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

package template

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
)

//Configs holds a collection of Config entries
type Configs []Config

//Config holds all the information around templates that can be configured
type Config struct {
	AppendFields common.Fields `config:"append_fields"`
	Enabled      bool          `config:"enabled"`
	Overwrite    bool          `config:"overwrite"`

	Name    string `config:"name"`
	Pattern string `config:"pattern"`
	Fields  string `config:"fields"`
	Module  string `config:"module"`
	JSON    struct {
		Enabled bool   `config:"enabled"`
		Path    string `config:"path"`
		Name    string `config:"name"`
	} `config:"json"`

	Settings Settings `config:"settings"`
}

//Settings holds information around index and _source for the template
type Settings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

//UpdateILM fetches relevant information from ILM config and
//adapts the template config accordingly.
func (cfg *Config) UpdateILM(config ilm.Config) bool {
	if cfg.JSON.Enabled {
		return false
	}
	cfg.Pattern = fmt.Sprintf("%s*", config.RolloverAlias)
	cfg.Settings.Index["lifecycle"] = map[string]interface{}{
		"rollover_alias": config.RolloverAlias,
		"name":           config.Policy.Name,
	}
	return true
}

var (
	// Defaults used in the template
	defaultDateDetection         = false
	defaultTotalFieldsLimit      = 10000
	defaultNumberOfRoutingShards = 30
)

func defaultTemplateCfg() Config {
	return Config{
		Enabled:   true,
		Overwrite: false,
		Fields:    "",
	}
}

//Unpack sets the Config instance to the given values
func (cfg *Config) Unpack(c *common.Config) error {
	type tmpConfig Config
	var tmp tmpConfig
	tmp = tmpConfig(defaultTemplateCfg())
	if err := c.Unpack(&tmp); err != nil {
		return err
	}

	*cfg = Config(tmp)
	return nil
}

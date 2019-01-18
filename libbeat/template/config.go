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
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

//Config holds all the information around templates that can be configured
type Config struct {
	AppendFields common.Fields `config:"append_fields"`
	Enabled      bool          `config:"enabled"`
	Overwrite    bool          `config:"overwrite"`

	Name    string `config:"name"`
	Pattern string `config:"pattern"`
	Fields  string `config:"fields"`
	JSON    JSON   `config:"json"`

	Settings Settings `config:"settings"`
}

type JSON struct {
	Enabled bool   `config:"enabled"`
	Path    string `config:"path"`
	Name    string `config:"name"`
}

//Settings holds information around index and _source for the template
type Settings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

var (
	// Defaults used in the template
	defaultDateDetection         = false
	defaultTotalFieldsLimit      = 10000
	defaultNumberOfRoutingShards = 30
)

func DefaultTemplateCfg() Config {
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
	tmp = tmpConfig(DefaultTemplateCfg())
	if err := c.Unpack(&tmp); err != nil {
		return err
	}
	if tmp.Pattern == "" {
		tmp.Pattern = fmt.Sprintf("%s*", tmp.Name)
	}

	*cfg = Config(tmp)
	return nil
}

func (cfg *Config) Validate() error {
	if cfg.Name == "" {
		return errors.New("template configuration requires a name")
	}
	return nil
}

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

package index

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/template"
)

// ESClient is a subset of the Elasticsearch client API capable of
// loading the templates and ILM related setup.
type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

//Configs holds a collection of Config entries
type Configs []Config

//Config supports the new configuration format around indices, templates and ILM
type Config struct {
	Index     string                 `config:"index"`
	ILM       ilm.Config             `config:"ilm"`
	Template  template.Config        `config:"template"`
	Condition map[string]interface{} `config:"condition"`
}

//CompatibleIndexCfg returns a configuration that is compatible with the deprecated output.elasticsearch.index format
func (i *Configs) CompatibleIndexCfg(client ESClient) (string, *common.Config, error) {
	ilmEnabled := ilm.EnabledFor(client)

	var idxName string
	var defaultIdxName string
	var cfgs []common.MapStr
	for _, entry := range *i {

		//set ilm.rollover_alias
		if ilmEnabled && entry.ILM.Enabled != ilm.ModeDisabled {
			idxName = entry.ILM.RolloverAlias
		} else {
			idxName = entry.Index
		}
		if entry.Condition == nil {
			defaultIdxName = idxName
			continue
		}

		cfg := map[string]interface{}{"index": idxName}
		for k, v := range entry.Condition {
			cfg[k] = v
		}

		cfgs = append(cfgs, cfg)
	}

	indices, err := common.NewConfigFrom(cfgs)
	if err != nil {
		return "", nil, err
	}
	return defaultIdxName, indices, nil
}

//Unpack implements logic how to unpack the configuration entries
func (i *Configs) Unpack(c *common.Config) error {
	var entries []Config
	if err := c.Unpack(&entries); err != nil {
		return err
	}

	*i = entries
	return nil
}

//Validate the index configuration settings
func (i *Configs) Validate() error {
	if i == nil {
		return nil
	}
	var defaultNames = 0
	for _, cfg := range *i {
		if cfg.Condition == nil {
			defaultNames++
		}
		if cfg.ILM.Enabled != ilm.ModeEnabled && cfg.Index == "" {
			return errors.New("indices entries must have set `index` when `ilm` is not disabled.")
		}
	}
	if defaultNames != 1 {
		return errors.New("exactly one indices option is required to be set without a condition")
	}
	return nil
}

//DefaultConfigs creates default configuration for `indices` setting
//can be overwritten as it is a global variable
var DefaultConfig = Config{
	Index: "%{[agent.name]}-%{[agent.version]}-%{+yyyy.MM.dd}",
	ILM: ilm.Config{
		Enabled:       ilm.ModeAuto,
		RolloverAlias: "%{[agent.name]}-%{[agent.version]}-000001",
		Pattern:       "000001",
		Policy:        ilm.PolicyCfg{Name: ilm.DefaultPolicyName}, //TODO: change when policy handling is changed
	},
	Template: template.Config{
		Enabled: true,
		Name:    "%{[agent.name]}-%{[agent.version]}",
		Pattern: "%{[agent.name]}-%{[agent.version]}*",
	},
}

//DeprecatedTemplateConfigs creates a new Indices configuration out of the deprecated template configuration.
func DeprecatedTemplateConfigs(cfg *common.Config) (Configs, error) {
	var tmplCfg = DefaultConfig.Template
	if name, err := cfg.String("name", -1); err != nil || name == "" {
		if err := cfg.SetString("name", -1, DefaultConfig.Template.Name); err != nil {
			return nil, err
		}
	}
	if err := cfg.Unpack(&tmplCfg); err != nil {
		return nil, err
	}
	return Configs{{Template: tmplCfg, ILM: ilm.Config{Enabled: ilm.ModeDisabled}}}, nil
}

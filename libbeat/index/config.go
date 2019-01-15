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
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/template"
)

//Configs holds a collection of Config entries
type Configs []Config

//Config supports the new configuration format around indices, templates and ILM
type Config struct {
	Name        string                 `config:"name"`
	ILMCfg      ilm.Config             `config:"ilm"`
	TemplateCfg template.Config        `config:"template"`
	Condition   map[string]interface{} `config:"condition"`
}

//CompatibleIndexCfg returns a configuration that is compatible with the deprecated output.elasticsearch.index format
func (i *Configs) CompatibleIndexCfg(client ESClient) (string, *common.Config, error) {
	ilmEnabled := ilm.EnabledFor(client)

	var idxName string
	var defaultIdxName string
	var cfgs []common.Config
	for _, entry := range *i {
		//set ilm.rollover_alias
		if ilmEnabled && !entry.ILMCfg.EnabledFalse() {
			idxName = entry.ILMCfg.RolloverAlias
		} else {
			idxName = entry.Name
		}
		if entry.Condition != nil {
			defaultIdxName = idxName
		}

		cfg := map[string]interface{}{"index": idxName}
		for k, v := range entry.Condition {
			cfg[k] = v
		}
		c, err := common.NewConfigFrom(cfg)
		if err != nil {
			return "", nil, err
		}
		cfgs = append(cfgs, *c)
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
	}
	if defaultNames != 1 {
		return errors.New("exactly one indices option is requierd to be set without a condition")
	}
	return nil
}

//LoadTemplates takes care of loading all configured templates to Elasticsearch,
//respecting ILM settings.
func (i *Configs) LoadTemplates(info beat.Info, client ESClient) (bool, error) {
	l, err := template.NewESLoader(client, info)
	if err != nil {
		return false, err
	}
	return i.loadTemplates(l, ilm.EnabledFor(client))
}

//PrintTemplates takes care of loading all configured templates to stdout,
//respecting ILM settings.
func (i *Configs) PrintTemplates(info beat.Info) (bool, error) {
	l, err := template.NewStdoutLoader(info)
	if err != nil {
		return false, err
	}
	return i.loadTemplates(l, true)
}

func (i *Configs) loadTemplates(l *template.Loader, ilmEnabled bool) (bool, error) {
	var loaded, loadedAny bool
	var err error

	for _, cfg := range *i {
		if ilmEnabled && !cfg.ILMCfg.EnabledFalse() {
			if updated := cfg.TemplateCfg.UpdateILM(cfg.ILMCfg); !updated {
				return loadedAny, fmt.Errorf("setup failed for index %s, cannot use ILM and template.json settings together", cfg.Name)
			}
		}

		if loaded, err = l.Load(cfg.TemplateCfg); err != nil {
			loadedAny = loadedAny || loaded
			return loadedAny, err
		}
		loadedAny = loadedAny || loaded
	}
	return loadedAny, nil
}

//LoadILMPolicies takes care of loading configured ILM policies to Elasticsearch
func (i *Configs) LoadILMPolicies(info beat.Info, client ESClient) (bool, error) {
	l, err := ilm.NewESLoader(client, info)
	if err != nil {
		return false, err
	}

	return i.loadILM(info, l.LoadPolicy)
}

//PrintILMPolicies prints configured ILM policies to stdout
func (i *Configs) PrintILMPolicies(info beat.Info) (bool, error) {
	l, err := ilm.NewStdoutLoader(info)
	if err != nil {
		return false, err
	}

	return i.loadILM(info, l.LoadPolicy)
}

//LoadILMWriteAliases takes care of loading configured ILM aliases to Elasticsearch
func (i *Configs) LoadILMWriteAliases(info beat.Info, client ESClient) (bool, error) {
	l, err := ilm.NewESLoader(client, info)
	if err != nil {
		return false, err
	}

	return i.loadILM(info, l.LoadWriteAlias)
}

func (i *Configs) loadILM(info beat.Info, f func(ilm.Config) (bool, error)) (bool, error) {
	var loaded, loadedAny bool
	var err error
	for _, cfg := range *i {
		if loaded, err = f(cfg.ILMCfg); err != nil {
			loadedAny = loadedAny || loaded
			return loadedAny, err
		}
		loadedAny = loadedAny || loaded
	}
	return loadedAny, nil
}

//DeprecatedTemplateConfigs creates a new Indices configuration out of the deprecated template configuration.
func DeprecatedTemplateConfigs(templateCfg *common.Config) (*Configs, error) {
	var tmplCfg template.Config
	if err := templateCfg.Unpack(&tmplCfg); err != nil {
		return nil, err
	}
	return &Configs{{TemplateCfg: tmplCfg, ILMCfg: ilm.Config{Enabled: "false"}}}, nil
}

// ESClient is a subset of the Elasticsearch client API capable of
// loading the templates and ILM related setup.
type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

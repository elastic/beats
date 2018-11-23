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
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/template"
)

type IndexConfigs []IndexConfig

type IndexConfig struct {
	Name        string                  `config:"name"`
	ILMCfg      ilm.ILMConfig           `config:"ilm"`
	TemplateCfg template.TemplateConfig `config:"template"`
}

type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

//LoadTemplates takes care of loading all configured templates to Elasticsearch,
//respecting ILM settings.
func (cfgs *IndexConfigs) LoadTemplates(client ESClient, info beat.Info) error {
	ilmEnabled := ilm.EnabledFor(client)
	var err error
	var l *template.Loader
	for _, cfg := range *cfgs {
		if ilmEnabled && !cfg.ILMCfg.EnabledFalse() {
			cfg.TemplateCfg.UpdateILM(cfg.ILMCfg)
		}
		l, err = template.NewLoader(cfg.TemplateCfg, client, info)
		if err != nil {
			return err
		}
		if err = l.Load(); err != nil {
			return err
		}
	}
	return nil
}

//LoadILMPolicies takes care of loading configured policies to Elasticsearch
func (cfgs *IndexConfigs) LoadILMPolicies(client ESClient, info beat.Info) error {
	return cfgs.loadILM(client, info, (*ilm.Loader).LoadPolicy)
}

//LoadIlmWriteAliases takes care of loading required aliases to Elasticsearch
func (cfgs *IndexConfigs) LoadILMWriteAliases(client ESClient, info beat.Info) error {
	return cfgs.loadILM(client, info, (*ilm.Loader).LoadWriteAlias)
}

func (cfgs *IndexConfigs) loadILM(client ESClient, info beat.Info, f func(*ilm.Loader) (bool, error)) error {
	ilmEnabled := ilm.EnabledFor(client)

	var err error
	var l *ilm.Loader
	for _, cfg := range *cfgs {
		l, err = ilm.NewLoader(cfg.ILMCfg, client, ilmEnabled, info)
		if err != nil {
			return err
		}
		if l == nil {
			//nothing to load
			continue
		}

		if _, err = f(l); err != nil {
			return err
		}
	}
	return nil
}

//DeprecatedConfigs creates a new Indices configuration from the deprecated template configuration.
func DeprecatedConfigs(c *common.Config) (IndexConfigs, error) {
	var tmplCfg template.TemplateConfig
	if err := c.Unpack(&tmplCfg); err != nil {
		return nil, err
	}
	cfgs := IndexConfigs{IndexConfig{TemplateCfg: tmplCfg, ILMCfg: ilm.ILMConfig{Enabled: "false"}}}
	return cfgs, nil
}

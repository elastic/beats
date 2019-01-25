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

package instance

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/template"
)

type indexSupport struct {
	ilm         ilm.Supporter
	info        beat.Info
	templateCfg template.TemplateConfig
}

type indexManager struct {
	support *indexSupport
	ilm     ilm.Manager

	client    *elasticsearch.Client
	fields    []byte
	migration bool
}

func newIndexSupport(
	info beat.Info,
	settings Settings,
	config *beatConfig,
	configRoot *common.Config,
) (*indexSupport, error) {
	ilmFactory := settings.ILM
	if ilmFactory == nil {
		ilmFactory = ilm.DefaultSupport
	}

	ilm, err := ilmFactory(info, configRoot)
	if err != nil {
		return nil, err
	}

	tmplCfg, err := unpackTemplateConfig(config.Template)
	if err != nil {
		return nil, err
	}

	return &indexSupport{
		ilm:         ilm,
		info:        info,
		templateCfg: tmplCfg,
	}, nil
}

func (s *indexSupport) Enabled() bool {
	return s.templateCfg.Enabled || (s.ilm.Mode() != ilm.ModeDisabled)
}

func (s *indexSupport) Manager(
	client *elasticsearch.Client,
	fields []byte,
	migration bool,
) indexManager {
	ilm := s.ilm.Manager(ilm.ESClientHandler(client))
	return indexManager{
		support:   s,
		ilm:       ilm,
		client:    client,
		fields:    fields,
		migration: migration,
	}
}

func (m *indexManager) Setup(template, policy bool) error {
	return m.load(template, policy)
}

func (m *indexManager) Load() error {
	return m.load(false, false)
}

func (m *indexManager) load(forceTemplate, forcePolicy bool) error {
	withILM, err := m.ilm.Enabled()
	if err != nil {
		return err
	}

	if withILM {
		if err := m.ilm.EnsurePolicy(forcePolicy); err != nil {
			return err
		}
		logp.Info("ILM policy successfully loaded.")
	}

	if m.support.templateCfg.Enabled {
		tmplCfg := m.support.templateCfg
		if withILM {
			ilmSettings := m.support.ilm.Template()
			tmplCfg, err = applyILMSettings(tmplCfg, ilmSettings)
			if err != nil {
				return err
			}
		}

		if forceTemplate {
			tmplCfg.Overwrite = true
		}

		loader, err := template.NewLoader(tmplCfg, m.client, m.support.info, m.fields, m.migration)
		if err != nil {
			return fmt.Errorf("Error creating Elasticsearch template loader: %v", err)
		}

		err = loader.Load()
		if err != nil {
			return fmt.Errorf("Error loading Elasticsearch template: %v", err)
		}

		logp.Info("Template successfully loaded.")
	}

	if withILM {
		if err := m.ilm.EnsureAlias(); err != nil {
			return err
		}
		logp.Info("Write alias successfully generated.")
	}

	return nil
}

func unpackTemplateConfig(cfg *common.Config) (template.TemplateConfig, error) {
	if cfg == nil {
		cfg = common.NewConfig()
	}

	config := template.DefaultConfig
	err := cfg.Unpack(&config)
	return config, err
}

func applyILMSettings(
	tmpl template.TemplateConfig,
	settings ilm.TemplateSettings,
) (template.TemplateConfig, error) {
	if !tmpl.Enabled {
		return tmpl, nil
	}

	alias := settings.Alias
	if alias == "" {
		return tmpl, errors.New("no ilm rollover alias configured")
	}

	policy := settings.PolicyName
	if policy == "" {
		return tmpl, errors.New("no ilm policy name configured")
	}

	tmpl.Name = alias
	logp.Info("Set setup.template.name to '%s' as ILM is enabled.", alias)

	tmpl.Pattern = fmt.Sprintf("%s-*", alias)
	logp.Info("Set setup.template.pattern to '%s' as ILM is enabled.", tmpl.Pattern)

	// rollover_alias and lifecycle.name can't be configured and will be overwritten

	// init/copy index settings
	idxSettings := tmpl.Settings.Index
	if idxSettings == nil {
		idxSettings = map[string]interface{}{}
	} else {
		tmp := make(map[string]interface{}, len(idxSettings))
		for k, v := range idxSettings {
			tmp[k] = v
		}
		idxSettings = tmp
	}
	tmpl.Settings.Index = idxSettings

	// init/copy index.lifecycle settings
	var lifecycle map[string]interface{}
	if ifcLifecycle := idxSettings["lifecycle"]; ifcLifecycle == nil {
		lifecycle = map[string]interface{}{}
	} else if tmp, ok := ifcLifecycle.(map[string]interface{}); ok {
		lifecycle = make(map[string]interface{}, len(tmp))
		for k, v := range tmp {
			lifecycle[k] = v
		}
	} else {
		return tmpl, errors.New("settings.index.lifecycle must be an object")
	}
	idxSettings["lifecycle"] = lifecycle

	// add rollover_alias and name to index.lifecycle settings
	if _, exists := lifecycle["rollover_alias"]; !exists {
		logp.Info("Set settings.index.lifecycle.rollover_alias in template to %s as ILM is enabled.", alias)
		lifecycle["rollover_alias"] = alias
	}
	if _, exists := lifecycle["name"]; !exists {
		logp.Info("Set settings.index.lifecycle.name in template to %s as ILM is enabled.", policy)
		lifecycle["name"] = policy
	}

	return tmpl, nil
}

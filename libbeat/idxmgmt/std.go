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

package idxmgmt

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/template"
)

type indexSupport struct {
	log          *logp.Logger
	ilm          ilm.Supporter
	info         beat.Info
	migration    bool
	templateCfg  template.TemplateConfig
	defaultIndex string

	st indexState
}

type indexState struct {
	withILM atomic.Bool
}

type indexManager struct {
	support *indexSupport
	ilm     ilm.Manager

	clientHandler ClientHandler
	assets        Asseter
}

type indexSelector outil.Selector

type ilmIndexSelector struct {
	index outil.Selector
	alias outil.Selector
	st    *indexState
}

func newIndexSupport(
	log *logp.Logger,
	info beat.Info,
	ilmFactory ilm.SupportFactory,
	tmplConfig *common.Config,
	ilmConfig *common.Config,
	migration bool,
) (*indexSupport, error) {
	if ilmFactory == nil {
		ilmFactory = ilm.DefaultSupport
	}

	ilm, err := ilmFactory(log, info, ilmConfig)
	if err != nil {
		return nil, err
	}

	tmplCfg, err := unpackTemplateConfig(tmplConfig)
	if err != nil {
		return nil, err
	}

	return &indexSupport{
		log:          log,
		ilm:          ilm,
		info:         info,
		templateCfg:  tmplCfg,
		migration:    migration,
		defaultIndex: fmt.Sprintf("%v-%v-%%{+yyyy.MM.dd}", info.IndexPrefix, info.Version),
	}, nil
}

func (s *indexSupport) Enabled() bool {
	return s.templateCfg.Enabled || (s.ilm.Mode() != ilm.ModeDisabled)
}

func (s *indexSupport) Manager(
	clientHandler ClientHandler,
	assets Asseter,
) Manager {
	ilm := s.ilm.Manager(clientHandler)
	return &indexManager{
		support:       s,
		ilm:           ilm,
		clientHandler: clientHandler,
		assets:        assets,
	}
}

func (s *indexSupport) BuildSelector(cfg *common.Config) (outputs.IndexSelector, error) {
	var err error
	log := s.log

	// we construct our own configuration object based on the available settings
	// in cfg and defaultIndex. The configuration object provided must not be
	// modified.
	selCfg := common.NewConfig()
	if cfg.HasField("indices") {
		sub, err := cfg.Child("indices", -1)
		if err != nil {
			return nil, err
		}
		selCfg.SetChild("indices", -1, sub)
	}

	var indexName string
	if cfg.HasField("index") {
		indexName, err = cfg.String("index", -1)
		if err != nil {
			return nil, err
		}
	}

	var alias string
	mode := s.ilm.Mode()
	if mode != ilm.ModeDisabled {
		alias = s.ilm.Alias().Name
		log.Infof("Set %v to '%s' as ILM is enabled.", cfg.PathOf("index"), alias)
	}
	if mode == ilm.ModeEnabled {
		indexName = alias
	}

	// no index name configuration found yet -> define default index name based on
	// beat.Info provided to the indexSupport on during setup.
	if indexName == "" {
		indexName = s.defaultIndex
	}

	selCfg.SetString("index", -1, indexName)
	buildSettings := outil.Settings{
		Key:              "index",
		MultiKey:         "indices",
		EnableSingleOnly: true,
		FailEmpty:        mode != ilm.ModeEnabled,
	}

	indexSel, err := outil.BuildSelectorFromConfig(selCfg, buildSettings)
	if err != nil {
		return nil, err
	}

	if mode != ilm.ModeAuto {
		return indexSelector(indexSel), nil
	}

	selCfg.SetString("index", -1, alias)
	aliasSel, err := outil.BuildSelectorFromConfig(selCfg, buildSettings)
	return &ilmIndexSelector{
		index: indexSel,
		alias: aliasSel,
		st:    &s.st,
	}, nil
}

func (m *indexManager) Setup(loadTemplate, loadILM LoadMode) error {
	var err error
	log := m.support.log

	withILM := m.support.st.withILM.Load()
	if !withILM {
		withILM, err = m.ilm.Enabled()
		if err != nil {
			return err
		}
	}
	if loadILM == LoadModeUnset {
		if withILM {
			loadILM = LoadModeEnabled
			log.Info("Auto ILM enable success.")
		} else {
			loadILM = LoadModeDisabled
		}
	}

	if withILM && loadILM.Enabled() {
		// mark ILM as enabled in indexState if withILM is true
		m.support.st.withILM.CAS(false, true)

		// install ilm policy
		policyCreated, err := m.ilm.EnsurePolicy(loadILM == LoadModeForce)
		if err != nil {
			return err
		}
		log.Info("ILM policy successfully loaded.")

		// The template should be updated if a new policy is created.
		if policyCreated && loadTemplate.Enabled() {
			loadTemplate = LoadModeForce
		}

		// create alias
		if err := m.ilm.EnsureAlias(); err != nil {
			if ilm.ErrReason(err) != ilm.ErrAliasAlreadyExists {
				return err
			}
			log.Info("Write alias exists already")
		} else {
			log.Info("Write alias successfully generated.")
		}
	}

	// create and install template
	if m.support.templateCfg.Enabled && loadTemplate.Enabled() {
		tmplCfg := m.support.templateCfg

		if withILM {
			ilm := m.support.ilm
			tmplCfg, err = applyILMSettings(log, tmplCfg, ilm.Policy(), ilm.Alias())
			if err != nil {
				return err
			}
		}

		if loadTemplate == LoadModeForce {
			tmplCfg.Overwrite = true
		}

		fields := m.assets.Fields(m.support.info.Beat)

		err = m.clientHandler.Load(tmplCfg, m.support.info, fields, m.support.migration)
		if err != nil {
			return fmt.Errorf("error loading template: %v", err)
		}

		log.Info("Loaded index template.")
	}

	return nil
}

func (s *ilmIndexSelector) Select(evt *beat.Event) (string, error) {
	if idx := getEventCustomIndex(evt); idx != "" {
		return idx, nil
	}

	if s.st.withILM.Load() {
		idx, err := s.alias.Select(evt)
		return idx, err
	}

	idx, err := s.index.Select(evt)
	return idx, err
}

func (s indexSelector) Select(evt *beat.Event) (string, error) {
	if idx := getEventCustomIndex(evt); idx != "" {
		return idx, nil
	}
	return outil.Selector(s).Select(evt)
}

func getEventCustomIndex(evt *beat.Event) string {
	if len(evt.Meta) == 0 {
		return ""
	}

	if tmp := evt.Meta["alias"]; tmp != nil {
		if alias, ok := tmp.(string); ok {
			return alias
		}
	}

	if tmp := evt.Meta["index"]; tmp != nil {
		if idx, ok := tmp.(string); ok {
			ts := evt.Timestamp.UTC()
			return fmt.Sprintf("%s-%d.%02d.%02d",
				idx, ts.Year(), ts.Month(), ts.Day())
		}
	}

	return ""
}

func unpackTemplateConfig(cfg *common.Config) (config template.TemplateConfig, err error) {
	config = template.DefaultConfig()
	if cfg != nil {
		err = cfg.Unpack(&config)
	}
	return config, err
}

func applyILMSettings(
	log *logp.Logger,
	tmpl template.TemplateConfig,
	policy ilm.Policy,
	alias ilm.Alias,
) (template.TemplateConfig, error) {
	if !tmpl.Enabled {
		return tmpl, nil
	}

	if alias.Name == "" {
		return tmpl, errors.New("no ilm rollover alias configured")
	}

	if policy.Name == "" {
		return tmpl, errors.New("no ilm policy name configured")
	}

	tmpl.Name = alias.Name
	if log != nil {
		log.Infof("Set setup.template.name to '%s' as ILM is enabled.", alias)
	}

	tmpl.Pattern = fmt.Sprintf("%s-*", alias.Name)
	if log != nil {
		log.Infof("Set setup.template.pattern to '%s' as ILM is enabled.", tmpl.Pattern)
	}

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
		log.Infof("Set settings.index.lifecycle.rollover_alias in template to %s as ILM is enabled.", alias)
		lifecycle["rollover_alias"] = alias.Name
	}
	if _, exists := lifecycle["name"]; !exists {
		log.Infof("Set settings.index.lifecycle.name in template to %s as ILM is enabled.", policy)
		lifecycle["name"] = policy.Name
	}

	return tmpl, nil
}

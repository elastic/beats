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
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/template"
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

type indexSelector struct {
	sel      outil.Selector
	beatInfo beat.Info
}

type ilmIndexSelector struct {
	index    outil.Selector
	alias    outil.Selector
	st       *indexState
	beatInfo beat.Info
}

type componentType uint8

//go:generate stringer -linecomment -type componentType
const (
	componentTemplate componentType = iota //template
	componentILM                           //ilm
)

type feature struct {
	component                componentType
	enabled, overwrite, load bool
}

func newFeature(c componentType, enabled, overwrite bool, mode LoadMode) feature {
	if mode == LoadModeUnset && !enabled {
		mode = LoadModeDisabled
	}
	if mode >= LoadModeOverwrite {
		overwrite = true
	}
	if mode == LoadModeForce {
		enabled = true
	}
	load := mode.Enabled() && enabled
	return feature{component: c, enabled: enabled, overwrite: overwrite, load: load}
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

	ilmSupporter, err := ilmFactory(log, info, ilmConfig)
	if err != nil {
		return nil, err
	}

	tmplCfg, err := unpackTemplateConfig(tmplConfig)
	if err != nil {
		return nil, err
	}

	return &indexSupport{
		log:          log,
		ilm:          ilmSupporter,
		info:         info,
		templateCfg:  tmplCfg,
		migration:    migration,
		defaultIndex: fmt.Sprintf("%v-%v-%%{+yyyy.MM.dd}", info.IndexPrefix, info.Version),
	}, nil
}

func (s *indexSupport) Enabled() bool {
	return s.enabled(componentTemplate) || s.enabled(componentILM)
}

func (s *indexSupport) enabled(c componentType) bool {
	switch c {
	case componentTemplate:
		return s.templateCfg.Enabled
	case componentILM:
		return s.ilm.Mode() != ilm.ModeDisabled
	}
	return false
}

func (s *indexSupport) Manager(
	clientHandler ClientHandler,
	assets Asseter,
) Manager {
	return &indexManager{
		support:       s,
		ilm:           s.ilm.Manager(clientHandler),
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
		Case:             outil.SelectorLowerCase,
	}

	indexSel, err := outil.BuildSelectorFromConfig(selCfg, buildSettings)
	if err != nil {
		return nil, err
	}

	if mode != ilm.ModeAuto {
		return indexSelector{indexSel, s.info}, nil
	}

	selCfg.SetString("index", -1, alias)
	aliasSel, err := outil.BuildSelectorFromConfig(selCfg, buildSettings)
	return &ilmIndexSelector{
		index: indexSel,
		alias: aliasSel,
		st:    &s.st,
	}, nil
}

func (m *indexManager) VerifySetup(loadTemplate, loadILM LoadMode) (bool, string) {
	ilmComponent := newFeature(componentILM, m.support.enabled(componentILM), m.support.ilm.Overwrite(), loadILM)

	templateComponent := newFeature(componentTemplate, m.support.enabled(componentTemplate),
		m.support.templateCfg.Overwrite, loadTemplate)

	if ilmComponent.load && !templateComponent.load {
		return false, "Loading ILM policy and write alias without loading template " +
			"is not recommended. Check your configuration."
	}

	if templateComponent.load && !ilmComponent.load && ilmComponent.enabled {
		return false, "Loading template with ILM settings whithout loading ILM " +
			"policy and alias can lead to issues and is not recommended. " +
			"Check your configuration."
	}

	var warn string
	if !ilmComponent.load {
		warn += "ILM policy and write alias loading not enabled.\n"
	} else if !ilmComponent.overwrite {
		warn += "Overwriting ILM policy is disabled. Set `setup.ilm.overwrite: true` for enabling.\n"
	}
	if !templateComponent.load {
		warn += "Template loading not enabled.\n"
	}
	return warn == "", warn
}

//
func (m *indexManager) Setup(loadTemplate, loadILM LoadMode) error {
	log := m.support.log

	withILM, err := m.setupWithILM()
	if err != nil {
		return err
	}
	if withILM && loadILM.Enabled() {
		log.Info("Auto ILM enable success.")
	}

	ilmComponent := newFeature(componentILM, withILM, m.support.ilm.Overwrite(), loadILM)
	templateComponent := newFeature(componentTemplate, m.support.enabled(componentTemplate),
		m.support.templateCfg.Overwrite, loadTemplate)

	if ilmComponent.load {
		// install ilm policy
		policyCreated, err := m.ilm.EnsurePolicy(ilmComponent.overwrite)
		if err != nil {
			return err
		}

		// The template should be updated if a new policy is created.
		if policyCreated && templateComponent.enabled {
			templateComponent.overwrite = true
		}
	}

	if templateComponent.load {
		tmplCfg := m.support.templateCfg
		tmplCfg.Overwrite, tmplCfg.Enabled = templateComponent.overwrite, templateComponent.enabled

		if ilmComponent.enabled {
			tmplCfg, err = applyILMSettings(log, tmplCfg, m.support.ilm.Policy(), m.support.ilm.Alias())
			if err != nil {
				return err
			}
		}
		fields := m.assets.Fields(m.support.info.Beat)
		err = m.clientHandler.Load(tmplCfg, m.support.info, fields, m.support.migration)
		if err != nil {
			return fmt.Errorf("error loading template: %v", err)
		}

		log.Info("Loaded index template.")
	}

	if ilmComponent.load {
		err := m.ilm.EnsureAlias()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *indexManager) setupWithILM() (bool, error) {
	var err error
	withILM := m.support.st.withILM.Load()
	if !withILM {
		withILM, err = m.ilm.CheckEnabled()
		if err != nil {
			return false, err
		}
		if withILM {
			// mark ILM as enabled in indexState
			m.support.st.withILM.CAS(false, true)
		}
	}
	return withILM, nil
}

func (s *ilmIndexSelector) Select(evt *beat.Event) (string, error) {
	if idx := getEventCustomIndex(evt, s.beatInfo); idx != "" {
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
	if idx := getEventCustomIndex(evt, s.beatInfo); idx != "" {
		return idx, nil
	}
	return s.sel.Select(evt)
}

func getEventCustomIndex(evt *beat.Event, beatInfo beat.Info) string {
	if len(evt.Meta) == 0 {
		return ""
	}

	if alias, err := events.GetMetaStringValue(*evt, events.FieldMetaAlias); err == nil {
		return strings.ToLower(alias)
	}

	if idx, err := events.GetMetaStringValue(*evt, events.FieldMetaIndex); err == nil {
		ts := evt.Timestamp.UTC()
		return fmt.Sprintf("%s-%d.%02d.%02d",
			strings.ToLower(idx), ts.Year(), ts.Month(), ts.Day())
	}

	// This is functionally identical to Meta["alias"], returning the overriding
	// metadata as the index name if present. It is currently used by Filebeat
	// to send the index for particular inputs to formatted string templates,
	// which are then expanded by a processor to the "raw_index" field.
	if idx, err := events.GetMetaStringValue(*evt, events.FieldMetaRawIndex); err == nil {
		return strings.ToLower(idx)
	}

	return ""
}

func unpackTemplateConfig(cfg *common.Config) (config template.TemplateConfig, err error) {
	return template.Unpack(cfg)
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

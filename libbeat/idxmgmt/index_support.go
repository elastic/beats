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
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/lifecycle"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/template"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type indexSupport struct {
	log          *logp.Logger
	ilm          lifecycle.Supporter
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
	ilm     lifecycle.Manager

	clientHandler ClientHandler
	assets        Asseter
}

type indexSelector struct {
	sel      outil.Selector
	beatInfo beat.Info
}

type componentType uint8

//go:generate stringer -linecomment -type componentType
const (
	componentTemplate componentType = iota //template
	componentILM                           //ilm
)

// feature determines what an index management feature is, and how it should be handled during setup
type feature struct {
	component                componentType
	enabled, overwrite, load bool
}

// newFeature creates a feature config object from a list of settings,
// returning a central object we use to determine how to perform setup for the feature
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

// creates a supporter that can perform setup and management actions for index support features such as ILM, index templates, etc
func newIndexSupport(
	log *logp.Logger,
	info beat.Info,
	ilmFactory lifecycle.SupportFactory,
	tmplConfig *config.C,
	lifecyclesEnabled bool,
	migration bool,
) (*indexSupport, error) {
	if ilmFactory == nil {
		ilmFactory = lifecycle.DefaultSupport
	}

	ilmSupporter, err := ilmFactory(log, info, lifecyclesEnabled)
	if err != nil {
		return nil, fmt.Errorf("error creating lifecycle supporter: %w", err)
	}

	tmplCfg, err := unpackTemplateConfig(info, tmplConfig)
	if err != nil {
		return nil, fmt.Errorf("error unpacking template config: %w", err)
	}

	return &indexSupport{
		log:          log,
		ilm:          ilmSupporter,
		info:         info,
		templateCfg:  tmplCfg,
		migration:    migration,
		defaultIndex: fmt.Sprintf("%v-%v", info.IndexPrefix, info.Version),
	}, nil
}

// Enabled returns true if some configured index management features are enabled
func (s *indexSupport) Enabled() bool {
	return s.enabled(componentTemplate) || s.enabled(componentILM)
}

// enabled checks if the given component is enabled in the config
func (s *indexSupport) enabled(c componentType) bool {
	switch c {
	case componentTemplate:
		return s.templateCfg.Enabled
	case componentILM:
		return s.ilm.Enabled()
	}
	return false
}

// Manager returns an indexManager object that
// can be used to perform the actual setup functions for the provided index management features
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

// BuildSelector creates an index selector
func (s *indexSupport) BuildSelector(cfg *config.C) (outputs.IndexSelector, error) {
	var err error
	// we construct our own configuration object based on the available settings
	// in cfg and defaultIndex. The configuration object provided must not be
	// modified.
	selCfg := config.NewConfig()
	if cfg.HasField("indices") {
		sub, err := cfg.Child("indices", -1)
		if err != nil {
			return nil, fmt.Errorf("error getting child value 'indices' in config: %w", err)
		}
		err = selCfg.SetChild("indices", -1, sub)
		if err != nil {
			return nil, fmt.Errorf("error setting child 'indices': %w", err)
		}
	}

	var indexName string
	if cfg.HasField("index") {
		indexName, err = cfg.String("index", -1)
		if err != nil {
			return nil, fmt.Errorf("error getting config string 'index': %w", err)
		}
	}

	// no index name configuration found yet -> define default index name based on
	// beat.Info provided to the indexSupport on during setup.
	if indexName == "" {
		indexName = s.defaultIndex
	}

	err = selCfg.SetString("index", -1, indexName)
	if err != nil {
		return nil, fmt.Errorf("error setting 'index' in selector cfg: %w", err)
	}
	buildSettings := outil.Settings{
		Key:              "index",
		MultiKey:         "indices",
		EnableSingleOnly: true,
		FailEmpty:        !s.ilm.Enabled(),
		Case:             outil.SelectorLowerCase,
	}

	indexSel, err := outil.BuildSelectorFromConfig(selCfg, buildSettings)
	if err != nil {
		return nil, err
	}

	return indexSelector{indexSel, s.info}, nil
}

// VerifySetup verifies the given feature setup, will return an error string if it detects something suspect
func (m *indexManager) VerifySetup(loadTemplate, loadLifecycle LoadMode) (bool, string) {
	ilmComponent := newFeature(componentILM, m.support.enabled(componentILM), m.clientHandler.Overwrite(), loadLifecycle)

	templateComponent := newFeature(componentTemplate, m.support.enabled(componentTemplate),
		m.support.templateCfg.Overwrite, loadTemplate)

	if ilmComponent.load && !templateComponent.load {
		return false, "Loading lifecycle policy without loading template is not recommended. Check your configuration."
	}

	if templateComponent.load && !ilmComponent.load && ilmComponent.enabled {
		return false, "Loading template with ILM settings whithout loading ILM " +
			"policy can lead to issues and is not recommended. " +
			"Check your configuration."
	}

	var warn string
	if !ilmComponent.load {
		warn += "lifecycle policy loading not enabled.\n"
	} else if !ilmComponent.overwrite {
		if m.clientHandler.Mode() == lifecycle.DSL {
			warn += "Overwriting lifecycle policy is disabled. Set `setup.dsl.overwrite: true` to overwrite.\n"
		} else {
			warn += "Overwriting lifecycle policy is disabled. Set `setup.ilm.overwrite: true` to overwrite.\n"
		}

	}
	if !templateComponent.load {
		warn += "Template loading not enabled.\n"
	}
	// remove last newline so we don't get weird formatting when this is printed to the console
	warn = strings.TrimSuffix(warn, "\n")
	return warn == "", warn
}

// Setup performs ILM/DSL and index template setup
func (m *indexManager) Setup(loadTemplate, loadILM LoadMode) error {
	log := m.support.log

	withILM, err := m.setupWithILM()
	if err != nil {
		return err
	}
	if withILM {
		log.Info("Auto lifecycle enable success.")
	}

	// create feature objects for ILM and template setup
	ilmComponent := newFeature(componentILM, withILM, m.clientHandler.Overwrite(), loadILM)
	templateComponent := newFeature(componentTemplate, m.support.enabled(componentTemplate),
		m.support.templateCfg.Overwrite, loadTemplate)

	if m.clientHandler.Mode() == lifecycle.DSL {
		log.Info("setting up DSL")
	}

	// on DSL, the template load will create the lifecycle policy
	// this is because the DSL API directly references the datastream,
	// so the datastream must be created first under DSL
	// If we're writing to a file, it doesn't matter
	if ilmComponent.load && (m.clientHandler.Mode() == lifecycle.ILM || !m.clientHandler.IsElasticsearch()) {
		// install ilm policy
		policyCreated, err := m.ilm.EnsurePolicy(ilmComponent.overwrite)
		if err != nil {
			return fmt.Errorf("EnsurePolicy failed during ILM setup: %w", err)
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
			tmplCfg, err = applyLifecycleSettingsToTemplate(log, tmplCfg, m.clientHandler)
			if err != nil {
				return fmt.Errorf("error applying ILM settings: %w", err)
			}
		}
		fields := m.assets.Fields(m.support.info.Beat)
		err = m.clientHandler.Load(tmplCfg, m.support.info, fields, m.support.migration)
		if err != nil {
			return fmt.Errorf("error loading template: %w", err)
		}

		log.Info("Loaded index template.")
	}

	return nil
}

// setupWithILM returns true if setup with ILM is expected
// will return false if we're currently talking to a serverless ES instance
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

	if idx, err := events.GetMetaStringValue(*evt, events.FieldMetaIndex); err == nil {
		return strings.ToLower(idx)
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

func unpackTemplateConfig(info beat.Info, cfg *config.C) (config template.TemplateConfig, err error) {
	config = template.DefaultConfig(info)

	if cfg != nil {
		err = cfg.Unpack(&config)
	}
	return config, err
}

// applies the specified ILM policy to the provided template, returns a struct of the template config
func applyLifecycleSettingsToTemplate(
	log *logp.Logger,
	tmpl template.TemplateConfig,
	policymgr lifecycle.ClientHandler,
) (template.TemplateConfig, error) {
	if !tmpl.Enabled {
		return tmpl, nil
	}

	if policymgr.PolicyName() == "" {
		return tmpl, errors.New("no policy name configured")
	}

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

	if policymgr.Mode() == lifecycle.ILM {
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

		if _, exists := lifecycle["name"]; !exists {
			log.Infof("Set settings.index.lifecycle.name in template to %s as ILM is enabled.", policymgr.PolicyName())
			lifecycle["name"] = policymgr.PolicyName()
		}
	} else {
		// when we're in DSL mode, this is what actually creates the policy
		tmpl.Settings.Lifecycle = policymgr.Policy().Body
	}

	return tmpl, nil
}

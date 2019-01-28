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
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/template"
)

// SupportFactory is used to provide custom index management support to libbeat.
type SupportFactory func(*logp.Logger, beat.Info, *common.Config) (Supporter, error)

// Supporter provides index management and configuration related services
// throughout libbeat.
// The BuildSelector is used by the output to create an IndexSelector. The
// index selector will report the per event index name to be used.
// A manager instantiated via Supporter is responsible for instantiating/configuring
// the index throughout the Elastic Stack.
type Supporter interface {
	Enabled() bool
	ILM() ilm.Supporter
	TemplateConfig(withILM bool) (template.TemplateConfig, error)
	Manager(client ESClient, assets Asseter) Manager
	BuildSelector(cfg *common.Config) (outputs.IndexSelector, error)
}

// Asseter provides access to beats assets required to load the template.
type Asseter interface {
	Fields(name string) []byte
}

// ESClient defines the minimal interface required for the index manager to
// prepare an index.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

// Manager is used to initialize indices, ILM policies, and aliases within the
// Elastic Stack.
type Manager interface {
	Setup(template, policy bool) error
}

// DefaultSupport initializes the default index management support used by most Beats.
func DefaultSupport(log *logp.Logger, info beat.Info, configRoot *common.Config) (Supporter, error) {
	factory := MakeDefaultSupport(nil)
	return factory(log, info, configRoot)
}

// MakeDefaultSupport creates some default index management support, with a
// custom ILM support implementation.
func MakeDefaultSupport(ilmSupport ilm.SupportFactory) SupportFactory {
	if ilmSupport == nil {
		ilmSupport = ilm.DefaultSupport
	}

	return func(log *logp.Logger, info beat.Info, configRoot *common.Config) (Supporter, error) {
		cfg := struct {
			ILM       *common.Config `config:"setup.ilm"`
			Template  *common.Config `config:"setup.template"`
			Migration *common.Config `config:"migration"`
		}{}
		if configRoot != nil {
			if err := configRoot.Unpack(&cfg); err != nil {
				return nil, err
			}
		}

		if log == nil {
			log = logp.NewLogger("index-management")
		} else {
			log = log.Named("index-management")
		}

		return newIndexSupport(log, info, ilmSupport, cfg.Template, cfg.ILM, cfg.Migration.Enabled())
	}
}

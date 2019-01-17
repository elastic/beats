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
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/template"
)

//Loader offers load methods needed in combination with the `indices` configuration
type Loader struct {
	esClient       ESClient
	beatInfo       beat.Info
	ilmEnabled     bool
	templateLoader *template.Loader
	ilmLoader      *ilm.Loader
}

//NewESLoader returns instance to load indices related data to ES
func NewESLoader(client ESClient, info beat.Info) (*Loader, error) {
	templateLoader, err := template.NewESLoader(client, info)
	if err != nil {
		return nil, err
	}
	ilmLoader, err := ilm.NewESLoader(client, info)
	if err != nil {
		return nil, err
	}
	return &Loader{
		esClient:       client,
		beatInfo:       info,
		ilmEnabled:     ilm.EnabledFor(client),
		templateLoader: templateLoader,
		ilmLoader:      ilmLoader,
	}, nil
}

//NewStdoutLoader returns instance to print indices related data to stdout
func NewStdoutLoader(info beat.Info) (*Loader, error) {
	templateLoader, err := template.NewStdoutLoader(info)
	if err != nil {
		return nil, err
	}
	ilmLoader, err := ilm.NewStdoutLoader(info)
	if err != nil {
		return nil, err
	}

	return &Loader{
		beatInfo:       info,
		ilmEnabled:     true,
		templateLoader: templateLoader,
		ilmLoader:      ilmLoader,
	}, nil
}

//LoadTemplates takes care of loading all configured templates to the configured output Elasticsearch or stdout
func (l *Loader) LoadTemplates(cfg []Config) (loaded int, noop int, failures int, loadErrors error) {
	var err error
	var errs []error
	var success bool
	for _, cfg := range cfg {
		if l.ilmEnabled && cfg.ILM.Enabled != ilm.ModeDisabled {
			if updated := cfg.Template.UpdateILM(cfg.ILM); !updated {
				errs = append(errs, errors.Wrapf(err, "mixing template.json and ilm is not allowed for %s", cfg.Template.Name))
				failures++
				continue
			}
		}

		success, err = l.templateLoader.Load(cfg.Template)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to load template %s", cfg.Template.Name))
			failures++
			continue
		}
		if !success {
			noop++
			continue
		}
		loaded++
	}
	return loaded, noop, failures, multierr.Combine(errs...)
}

//LoadILMPolicies takes care of loading configured ILM policies to the configured output Elasticsearch or stdout
func (l *Loader) LoadILMPolicies(cfg []Config) (loaded int, noop int, failures int, loadErrors error) {
	return l.loadILM(cfg, l.ilmLoader.LoadPolicy, func(cfg Config, err error) error {
		return errors.Wrapf(err, "failed to load ilm policy for %s", cfg.ILM.Policy.Name)
	})
}

//LoadILMWriteAliases takes care of loading configured ILM aliases to Elasticsearch
func (l *Loader) LoadILMWriteAliases(cfg []Config) (loaded int, noop int, failures int, loadErrors error) {
	return l.loadILM(cfg, l.ilmLoader.LoadWriteAlias, func(cfg Config, err error) error {
		return errors.Wrapf(err, "failed to load ilm write alias for %s", cfg.ILM.RolloverAlias)
	})
}

func (l *Loader) loadILM(
	cfg []Config,
	f func(ilm.Config) (bool, error),
	errF func(cfg Config, err error) error) (loaded int, noop int, failures int, loadErrors error) {

	var err error
	var errs []error
	var success bool
	for _, c := range cfg {
		if success, err = f(c.ILM); err != nil {
			errs = append(errs, errF(c, err))
			failures++
			continue
		}
		if !success {
			noop++
			continue
		}
		loaded++
	}
	return loaded, noop, failures, multierr.Combine(errs...)
}

// ESClient is a subset of the Elasticsearch client API capable of
// loading the templates and ILM related setup.
type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

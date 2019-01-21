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

	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/template"
)

//LoadTemplates takes care of loading all configured templates to the configured output Elasticsearch or stdout
func LoadTemplates(loader template.Loader, configs []Config) (loaded int, noop int, failures int, loadErrors error) {
	var err error
	var errs []error
	var success bool
	for _, cfg := range configs {
		success, err = loader.Load(cfg.Template, cfg.ILM)
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
func LoadILMPolicies(loader ilm.Loader, configs []Config) (int, int, int, error) {

	return loadILM(configs, loader.LoadPolicy, func(cfg Config, err error) error {
		return errors.Wrapf(err, "failed to load ilm policy for %s", cfg.ILM.Policy.Name)
	})
}

//LoadILMWriteAliases takes care of loading configured ILM aliases to Elasticsearch
func LoadILMWriteAliases(loader ilm.Loader, configs []Config) (int, int, int, error) {
	return loadILM(configs, loader.LoadWriteAlias, func(cfg Config, err error) error {
		return errors.Wrapf(err, "failed to load ilm write alias for %s", cfg.ILM.RolloverAlias)
	})
}

func loadILM(
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

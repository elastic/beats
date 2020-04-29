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

package mb

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
)

// LightMetricSet contains the definition of a non-registered metric set
type LightMetricSet struct {
	Name    string
	Module  string
	Default bool `config:"default"`
	Input   struct {
		Module    string      `config:"module" validate:"required"`
		MetricSet string      `config:"metricset" validate:"required"`
		Defaults  interface{} `config:"defaults"`
	} `config:"input" validate:"required"`
	Processors processors.PluginConfig `config:"processors"`
}

// Registration obtains a metric set registration for this light metric set, this registration
// contains a metric set factory that reprocess metric set creation taking into account the
// light metric set defaults
func (m *LightMetricSet) Registration(r *Register) (MetricSetRegistration, error) {
	registration, err := r.metricSetRegistration(m.Input.Module, m.Input.MetricSet)
	if err != nil {
		return registration, errors.Wrapf(err,
			"failed to start light metricset '%s/%s' using '%s/%s' metricset as input",
			m.Module, m.Name,
			m.Input.Module, m.Input.MetricSet)
	}

	originalFactory := registration.Factory
	registration.IsDefault = m.Default

	// Light modules factory has to override defaults and reproduce builder
	// functionality with the resulting configuration, it does:
	// - Override defaults
	// - Call module factory if registered (it wouldn't have been called
	//   if light module is really a registered mixed module)
	// - Call host parser if defined (it would have already been called
	//   without the light module defaults)
	// - Finally, call the original factory for the registered metricset
	registration.Factory = func(base BaseMetricSet) (MetricSet, error) {
		// Override default config on base module and metricset
		base.name = m.Name
		baseModule, err := m.baseModule(base.module)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create base module for light module '%s', using base module '%s'", m.Module, base.module.Name())
		}
		base.module = baseModule

		// Run module factory if registered, it will be called once per
		// metricset, but it should be idempotent
		moduleFactory := r.moduleFactory(m.Input.Module)
		if moduleFactory != nil {
			module, err := moduleFactory(*baseModule)
			if err != nil {
				return nil, errors.Wrapf(err, "module factory for module '%s' failed while creating light metricset '%s/%s'", m.Input.Module, m.Module, m.Name)
			}
			base.module = module
		}

		// At this point host parser was already run, we need to run this again
		// with the overriden defaults
		if registration.HostParser != nil {
			host := m.useHostURISchemeIfPossible(base.host, base.hostData.URI)
			base.hostData, err = registration.HostParser(base.module, host)
			if err != nil {
				return nil, errors.Wrapf(err, "host parser failed on light metricset factory for '%s/%s'", m.Module, m.Name)
			}
			base.host = base.hostData.Host
		}

		return originalFactory(base)
	}

	return registration, nil
}

// useHostURISchemeIfPossible method parses given URI to extract protocol scheme and prepend it to the host.
// It prevents from skipping protocol scheme (e.g. https) while executing HostParser.
func (m *LightMetricSet) useHostURISchemeIfPossible(host, uri string) string {
	u, err := url.ParseRequestURI(uri)
	if err == nil {
		if u.Scheme != "" {
			return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}
	}
	return host
}

// baseModule does the configuration overrides in the base module configuration
// taking into account the light metric set default configurations
func (m *LightMetricSet) baseModule(from Module) (*BaseModule, error) {
	// Initialize config using input defaults as raw config
	rawConfig, err := common.NewConfigFrom(m.Input.Defaults)
	if err != nil {
		return nil, errors.Wrap(err, "invalid input defaults")
	}

	// Copy values from user configuration
	if err = from.UnpackConfig(rawConfig); err != nil {
		return nil, errors.Wrap(err, "failed to copy values from user configuration")
	}

	// Create the base module
	baseModule, err := newBaseModuleFromConfig(rawConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create base module")
	}
	baseModule.name = m.Module

	return &baseModule, nil
}

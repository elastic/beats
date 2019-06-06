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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

const (
	moduleYML   = "module.yml"
	manifestYML = "manifest.yml"
)

// LightModulesRegistry reads module definitions from files
type LightModulesRegistry struct {
	paths []string
}

// NewLightModulesRegistry creates a new LightModulesRegistry
func NewLightModulesRegistry(paths ...string) *LightModulesRegistry {
	return &LightModulesRegistry{
		paths: paths,
	}
}

func (r *LightModulesRegistry) Modules() ([]string, error) {
	return r.listModules()
}

func (r *LightModulesRegistry) DefaultMetricSets(moduleName string) ([]string, error) {
	module, found, err := r.readModule(moduleName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get default metricsets for module '%s'", moduleName)
	}
	if !found {
		return nil, nil
	}
	var metricsets []string
	for _, ms := range module.MetricSets {
		if ms.Default {
			metricsets = append(metricsets, ms.Name)
		}
	}
	return metricsets, nil
}

func (r *LightModulesRegistry) MetricSets(moduleName string) ([]string, error) {
	module, found, err := r.readModule(moduleName)
	if err != nil || !found {
		return nil, errors.Wrapf(err, "failed to get metricsets for module '%s'", moduleName)
	}
	metricsets := make([]string, 0, len(module.MetricSets))
	for _, ms := range module.MetricSets {
		metricsets = append(metricsets, ms.Name)
	}
	return metricsets, nil
}

func (r *LightModulesRegistry) MetricSetRegistration(parent *Register, module, name string) (MetricSetRegistration, bool, error) {
	lightModule, found, err := r.readModule(module)
	if err != nil || !found {
		return MetricSetRegistration{}, found, err
	}

	ms, found := lightModule.MetricSets[name]
	if !found {
		return MetricSetRegistration{}, false, nil
	}

	registration, err := ms.Registration(parent)
	return registration, true, err
}

type lightModuleConfig struct {
	Name       string   `config:"name"`
	MetricSets []string `config:"metricsets"`
}

type LightModule struct {
	Name       string
	MetricSets map[string]LightMetricSet
}

type LightMetricSet struct {
	Name    string
	Module  string
	Default bool `config:"default"`
	Input   struct {
		Module    string      `config:"module" validate:"required"`
		MetricSet string      `config:"metricset" validate:"required"`
		Defaults  interface{} `config:"defaults"`
	} `config:"input" validate:"required"`
}

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
			return nil, errors.Wrap(err, "failed to create base module for light module")
		}
		base.module = baseModule

		// Run module factory if registered, it will be called once per
		// metricset, but it should be idempotent
		moduleFactory := r.moduleFactory(m.Input.Module)
		if moduleFactory != nil {
			m, err := moduleFactory(*baseModule)
			if err != nil {
				return nil, err
			}
			base.module = m
		}

		// At this point host parser was already run, we need to run this again
		// with the overriden defaults
		if registration.HostParser != nil {
			base.hostData, err = registration.HostParser(base.module, base.host)
			if err != nil {
				return nil, err
			}
			base.host = base.hostData.Host
		}

		return originalFactory(base)
	}

	return registration, nil
}

func (m *LightMetricSet) baseModule(from Module) (*BaseModule, error) {
	baseModule := BaseModule{
		name:   m.Module,
		config: from.Config(),
	}
	var err error
	// Set defaults
	if baseModule.rawConfig, err = common.NewConfigFrom(m.Input.Defaults); err != nil {
		return nil, errors.Wrap(err, "invalid input defaults")
	}
	// Copy values from user configuration
	if err = from.UnpackConfig(baseModule.rawConfig); err != nil {
		return nil, errors.Wrap(err, "failed to copy values from user configuration")
	}
	// Update module configuration
	if err = baseModule.UnpackConfig(&baseModule.config); err != nil {
		return nil, errors.Wrap(err, "failed to set module configuration")
	}
	return &baseModule, nil
}

func (r *LightModulesRegistry) readModule(moduleName string) (*LightModule, bool, error) {
	modulePath, found := r.findModulePath(moduleName)
	if !found {
		return nil, false, nil
	}

	moduleConfig, err := r.readModuleConfig(modulePath)
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to load light module '%s' definition", moduleName)
	}

	metricSets := make(map[string]LightMetricSet)
	for _, metricSet := range moduleConfig.MetricSets {
		manifestPath := filepath.Join(filepath.Dir(modulePath), metricSet, manifestYML)

		metricSetConfig, err := r.readMetricSetConfig(manifestPath)
		if err != nil {
			return nil, true, errors.Wrapf(err, "failed to load light metricset '%s/%s' definition", moduleName, metricSet)
		}
		metricSetConfig.Name = metricSet
		metricSetConfig.Module = moduleName

		metricSets[metricSet] = metricSetConfig
	}

	return &LightModule{Name: moduleName, MetricSets: metricSets}, true, nil
}

func (r *LightModulesRegistry) findModulePath(moduleName string) (string, bool) {
	for _, dir := range r.paths {
		p := filepath.Join(dir, moduleName, moduleYML)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func (r *LightModulesRegistry) readModuleConfig(modulePath string) (*lightModuleConfig, error) {
	c, err := common.LoadFile(modulePath)
	if err != nil {
		return nil, err
	}

	var moduleConfig lightModuleConfig
	if err = c.Unpack(&moduleConfig); err != nil {
		return nil, errors.Wrapf(err, "failed to parse light module definition from '%s'", modulePath)
	}
	return &moduleConfig, nil
}

func (r *LightModulesRegistry) readMetricSetConfig(manifestPath string) (ms LightMetricSet, err error) {
	c, err := common.LoadFile(manifestPath)
	if err != nil {
		return ms, err
	}

	if err := c.Unpack(&ms); err != nil {
		return ms, errors.Wrapf(err, "failed to parse metricset manifest from '%s'", manifestPath)
	}
	return
}

func (r *LightModulesRegistry) listModules() ([]string, error) {
	modules := make(map[string]bool)
	for _, dir := range r.paths {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list modules on path '%s'", dir)
		}
		for _, f := range files {
			modulePath := filepath.Join(dir, f.Name(), moduleYML)
			if _, err := os.Stat(modulePath); os.IsNotExist(err) {
				continue
			}
			modules[f.Name()] = true
		}
	}

	list := make([]string, 0, len(modules))
	for m := range modules {
		list = append(list, m)
	}
	return list, nil
}

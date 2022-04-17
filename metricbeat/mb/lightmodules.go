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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/processors"
)

const (
	moduleYML   = "module.yml"
	manifestYML = "manifest.yml"
)

// LightModulesSource loads module definitions from files in the provided paths
type LightModulesSource struct {
	paths []string
	log   *logp.Logger
}

// NewLightModulesSource creates a new LightModulesSource
func NewLightModulesSource(paths ...string) *LightModulesSource {
	return &LightModulesSource{
		paths: paths,
		log:   logp.NewLogger("registry.lightmodules"),
	}
}

// Modules lists the light modules available on the configured paths
func (s *LightModulesSource) Modules() ([]string, error) {
	return s.moduleNames()
}

// HasModule checks if there is a light module with the given name
func (s *LightModulesSource) HasModule(moduleName string) bool {
	names, err := s.moduleNames()
	if err != nil {
		s.log.Errorf("Failed to get list of light module names: %v", err)
		return false
	}
	for _, name := range names {
		if name == moduleName {
			return true
		}
	}
	return false
}

// DefaultMetricSets list the default metricsets for a given module
func (s *LightModulesSource) DefaultMetricSets(r *Register, moduleName string) ([]string, error) {
	module, err := s.loadModule(r, moduleName)
	if err != nil {
		return nil, errors.Wrapf(err, "getting default metricsets for module '%s'", moduleName)
	}
	var metricsets []string
	for _, ms := range module.MetricSets {
		if ms.Default {
			metricsets = append(metricsets, ms.Name)
		}
	}
	return metricsets, nil
}

// MetricSets list the available metricsets for a given module
func (s *LightModulesSource) MetricSets(r *Register, moduleName string) ([]string, error) {
	module, err := s.loadModule(r, moduleName)
	if err != nil {
		return nil, errors.Wrapf(err, "getting metricsets for module '%s'", moduleName)
	}
	metricsets := make([]string, 0, len(module.MetricSets))
	for _, ms := range module.MetricSets {
		metricsets = append(metricsets, ms.Name)
	}
	return metricsets, nil
}

// HasMetricSet checks if the given metricset exists
func (s *LightModulesSource) HasMetricSet(moduleName, metricSetName string) bool {
	modulePath, found := s.findModulePath(moduleName)
	if !found {
		return false
	}

	moduleConfig, err := s.loadModuleConfig(modulePath)
	if err != nil {
		s.log.Errorf("Failed to load module config for module '%s': %v", moduleName, err)
		return false
	}

	for _, name := range moduleConfig.MetricSets {
		if name == metricSetName {
			return true
		}
	}
	return false
}

// MetricSetRegistration obtains a registration for a light metric set
func (s *LightModulesSource) MetricSetRegistration(register *Register, moduleName, metricSetName string) (MetricSetRegistration, error) {
	lightModule, err := s.loadModule(register, moduleName)
	if err != nil {
		return MetricSetRegistration{}, errors.Wrapf(err, "loading module '%s'", moduleName)
	}

	ms, found := lightModule.MetricSets[metricSetName]
	if !found {
		return MetricSetRegistration{}, fmt.Errorf("metricset '%s/%s' not found", moduleName, metricSetName)
	}

	return ms.Registration(register)
}

// ModulesInfo returns a string representation of this source, with a list of known metricsets
func (s *LightModulesSource) ModulesInfo(r *Register) string {
	var metricSets []string
	modules, err := s.Modules()
	if err != nil {
		s.log.Errorf("Failed to list modules: %s", err)
	}
	for _, module := range modules {
		moduleMetricSets, err := s.MetricSets(r, module)
		if err != nil {
			s.log.Errorf("Failed to list light metricsets for module %s: %v", module, err)
		}
		for _, name := range moduleMetricSets {
			metricSets = append(metricSets, fmt.Sprintf("%s/%s", module, name))
		}
	}

	return fmt.Sprintf("LightModules:[%s]", strings.Join(metricSets, ", "))
}

type lightModuleConfig struct {
	Name       string   `config:"name"`
	MetricSets []string `config:"metricsets"`
}

// ProcessorsForMetricSet returns processors defined for the light metricset.
func (s *LightModulesSource) ProcessorsForMetricSet(r *Register, moduleName string, metricSetName string) (*processors.Processors, error) {
	module, err := s.loadModule(r, moduleName)
	if err != nil {
		return nil, errors.Wrapf(err, "reading processors for metricset '%s' in module '%s'", metricSetName, moduleName)
	}
	metricSet, ok := module.MetricSets[metricSetName]
	if !ok {
		return nil, fmt.Errorf("unknown metricset '%s' in module '%s'", metricSetName, moduleName)
	}
	return processors.New(metricSet.Processors)
}

// LightModule contains the definition of a light module
type LightModule struct {
	Name       string
	MetricSets map[string]LightMetricSet
}

func (s *LightModulesSource) loadModule(register *Register, moduleName string) (*LightModule, error) {
	modulePath, found := s.findModulePath(moduleName)
	if !found {
		return nil, fmt.Errorf("module '%s' not found", moduleName)
	}

	moduleConfig, err := s.loadModuleConfig(modulePath)
	if err != nil {
		return nil, errors.Wrapf(err, "loading light module '%s' definition", moduleName)
	}

	metricSets, err := s.loadMetricSets(register, filepath.Dir(modulePath), moduleConfig.Name, moduleConfig.MetricSets)
	if err != nil {
		return nil, errors.Wrapf(err, "loading metric sets for light module '%s'", moduleName)
	}

	return &LightModule{Name: moduleName, MetricSets: metricSets}, nil
}

func (s *LightModulesSource) findModulePath(moduleName string) (string, bool) {
	for _, dir := range s.paths {
		candidate := filepath.Join(dir, moduleName, moduleYML)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func (s *LightModulesSource) loadModuleConfig(modulePath string) (*lightModuleConfig, error) {
	config, err := common.LoadFile(modulePath)
	if err != nil {
		return nil, errors.Wrapf(err, "loading module configuration from '%s'", modulePath)
	}

	var moduleConfig lightModuleConfig
	if err = config.Unpack(&moduleConfig); err != nil {
		return nil, errors.Wrapf(err, "parsing light module definition from '%s'", modulePath)
	}
	return &moduleConfig, nil
}

func (s *LightModulesSource) loadMetricSets(register *Register, moduleDirPath, moduleName string, metricSetNames []string) (map[string]LightMetricSet, error) {
	metricSets := make(map[string]LightMetricSet)
	for _, metricSet := range metricSetNames {
		if moduleMetricSets, exists := register.metricSets[moduleName]; exists {
			if _, exists := moduleMetricSets[metricSet]; exists {
				continue
			}
		}

		manifestPath := filepath.Join(moduleDirPath, metricSet, manifestYML)

		metricSetConfig, err := s.loadMetricSetConfig(manifestPath)
		if err != nil {
			return nil, errors.Wrapf(err, "loading light metricset '%s'", metricSet)
		}
		metricSetConfig.Name = metricSet
		metricSetConfig.Module = moduleName

		metricSets[metricSet] = metricSetConfig
	}
	return metricSets, nil
}

func (s *LightModulesSource) loadMetricSetConfig(manifestPath string) (ms LightMetricSet, err error) {
	config, err := common.LoadFile(manifestPath)
	if err != nil {
		return ms, errors.Wrapf(err, "loading metricset manifest from '%s'", manifestPath)
	}

	if err := config.Unpack(&ms); err != nil {
		return ms, errors.Wrapf(err, "parsing metricset manifest from '%s'", manifestPath)
	}
	return
}

func (s *LightModulesSource) moduleNames() ([]string, error) {
	modules := make(map[string]bool)
	for _, dir := range s.paths {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			s.log.Debugf("Light modules directory '%d' doesn't exist", dir)
			continue
		}
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, errors.Wrapf(err, "listing modules on path '%s'", dir)
		}
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			modulePath := filepath.Join(dir, f.Name(), moduleYML)
			if _, err := os.Stat(modulePath); os.IsNotExist(err) {
				continue
			}
			modules[f.Name()] = true
		}
	}

	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	return names, nil
}

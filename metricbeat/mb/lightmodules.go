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
	"sort"

	"github.com/pkg/errors"
)

// LigthModulesRegistry wraps
type LightModulesRegistry struct {
	ModulesRegistry

	paths []string
}

// NewLightModulesRegistry creates a new LightModulesRegistry
func NewLightModulesRegistry(paths ...string) *LightModulesRegistry {
	return &LightModulesRegistry{
		ModulesRegistry: NewRegister(),
		paths:           paths,
	}
}

func (r *LightModulesRegistry) DefaultMetricSets(module string) ([]string, error) {
	if stringInSlice(module, r.parent().Modules()) {
		return r.parent().DefaultMetricSets(module)
	}
	module, err := r.readModule(module)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get default metricsets for module '%s'", module)
	}
	var metricsets []string
	for i, ms := range module.Metricsets {
		if ms.Default {
			metricsets = append(metricsets, ms)
		}
	}
	if len(metricsets) == 0 {
		return nil, fmt.Errorf("no default metricset exists for module '%s'", module)
	}
	return metricsets, nil
}

func (r *LightModulesRegistry) Modules() []string {
	registeredModules := r.parent().Modules()
	modules := r.listModules()
	return append(registeredModules, modules...)
}

func (r *LightModulesRegistry) MetricSets(module string) []string {
	if stringInSlice(module, r.parent().Modules()) {
		return r.parent().MetricSets(module)
	}
	module, err := r.readModule(module)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get metricsets for module '%s'", module)
	}
	metricsets := make([]string, len(module.MetricSets))
	for i := range module.Metricsets {
		metricset[i] = module.MetricSets[i].Name
	}
	return metricsets
}

func (r *LightModulesRegistry) String() string {
	var metricsets []string
	modules := r.listModules()
	for _, moduleName := range modules {
		module, err := r.readModule(module)
		if err != nil {
			continue
		}
		for _, metricset := range module.MetricSets {
			metricsets = append(metricsets, fmt.Sprintf("%s/%s", moduleName, metricset.Name))
		}
	}
	sort.Strings(metricsets)
	parent := r.parent().String()
	if len(modules) == 0 {
		return parent
	}
	return fmt.Sprintf("%s, Light Modules [MetricSets:[%s]]",
		parent,
		strings.Join(metricsets, ", "),
	)
}

func (r *LightModulesRegistry) moduleFactory(name string) ModuleFactory {
	return r.parent().moduleFactory(name)
}

func (r *LightModulesRegistry) metricSetRegistration(module, name string) (MetricSetRegistration, error) {
	return r.parent().metricSetRegistration(module, name)
}

func (r *LightModulesRegistry) parent() ModulesRegistry {
	return r.ModulesRegistry
}

func stringInSlice(s string, slice []string) bool {
	for i := range slice {
		if s == slice[i] {
			return true
		}
	}
	return false
}

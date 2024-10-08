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

package module

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// ConfiguredModules returns a list of all configured modules, including anyone present under dynamic config settings.
func ConfiguredModules(registry *mb.Register, modulesData []*conf.C, configModulesData *conf.C, moduleOptions []Option) ([]*Wrapper, error) {
	var modules []*Wrapper //nolint:prealloc //can't be preallocated

	for _, moduleCfg := range modulesData {
		module, err := NewWrapper(moduleCfg, registry, moduleOptions...)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	// Add dynamic modules
	if configModulesData.Enabled() {
		config := cfgfile.DefaultDynamicConfig
		if err := configModulesData.Unpack(&config); err != nil {
			return nil, err
		}

		modulesManager, err := cfgfile.NewGlobManager(config.Path, ".yml", ".disabled")
		if err != nil {
			return nil, fmt.Errorf("initialization error: %w", err)
		}

		for _, file := range modulesManager.ListEnabled() {
			confs, err := cfgfile.LoadList(file.Path)
			if err != nil {
				return nil, fmt.Errorf("error loading config files: %w", err)
			}
			for _, conf := range confs {
				m, err := NewWrapper(conf, registry, moduleOptions...)
				if err != nil {
					return nil, fmt.Errorf("module initialization error: %w", err)
				}
				modules = append(modules, m)
			}
		}
	}
	return modules, nil
}

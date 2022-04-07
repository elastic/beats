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
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/cfgfile"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// ConfiguredModules returns a list of all configured modules, including anyone present under dynamic config settings.
func ConfiguredModules(modulesData []*common.Config, configModulesData *common.Config, moduleOptions []Option) ([]*Wrapper, error) {
	var modules []*Wrapper

	for _, moduleCfg := range modulesData {
		module, err := NewWrapper(moduleCfg, mb.Registry, moduleOptions...)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	// Add dynamic modules
	if configModulesData.Enabled() {
		config := cfgfile.DefaultDynamicConfig
		configModulesData.Unpack(&config)

		modulesManager, err := cfgfile.NewGlobManager(config.Path, ".yml", ".disabled")
		if err != nil {
			return nil, errors.Wrap(err, "initialization error")
		}

		for _, file := range modulesManager.ListEnabled() {
			confs, err := cfgfile.LoadList(file.Path)
			if err != nil {
				return nil, errors.Wrap(err, "error loading config files")
			}
			for _, conf := range confs {
				m, err := NewWrapper(conf, mb.Registry, moduleOptions...)
				if err != nil {
					return nil, errors.Wrap(err, "module initialization error")
				}
				modules = append(modules, m)
			}
		}
	}
	return modules, nil
}

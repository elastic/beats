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
	"errors"

	"github.com/menderesk/beats/v7/libbeat/plugin"

	"github.com/menderesk/beats/v7/metricbeat/mb"
)

type modulePlugin struct {
	name       string
	factory    mb.ModuleFactory
	metricsets map[string]mb.MetricSetFactory
}

const pluginKey = "metricbeat.module"

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(modulePlugin)
		if !ok {
			return errors.New("plugin does not match metricbeat module plugin type")
		}

		if p.factory != nil {
			if err := mb.Registry.AddModule(p.name, p.factory); err != nil {
				return err
			}
		}

		for name, factory := range p.metricsets {
			if err := mb.Registry.AddMetricSet(p.name, name, factory); err != nil {
				return err
			}
		}

		return nil
	})
}

func Plugin(
	module string,
	factory mb.ModuleFactory,
	metricsets map[string]mb.MetricSetFactory,
) map[string][]interface{} {
	return plugin.MakePlugin(pluginKey, modulePlugin{module, factory, metricsets})
}

func MetricSetsPlugin(
	module string,
	metricsets map[string]mb.MetricSetFactory,
) map[string][]interface{} {
	return Plugin(module, nil, metricsets)
}

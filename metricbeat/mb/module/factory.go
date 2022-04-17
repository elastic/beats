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
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/cfgfile"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// Factory creates new Runner instances from configuration objects.
// It is used to register and reload modules.
type Factory struct {
	beatInfo beat.Info
	options  []Option
}

// NewFactory creates new Reloader instance for the given config
func NewFactory(beatInfo beat.Info, options ...Option) *Factory {
	return &Factory{
		beatInfo: beatInfo,
		options:  options,
	}
}

// Create creates a new metricbeat module runner reporting events to the passed pipeline.
func (r *Factory) Create(p beat.PipelineConnector, c *common.Config) (cfgfile.Runner, error) {
	module, metricSets, err := mb.NewModule(c, mb.Registry)
	if err != nil {
		return nil, err
	}

	var runners []Runner
	for _, metricSet := range metricSets {
		wrapper, err := NewWrapperForMetricSet(module, metricSet, r.options...)
		if err != nil {
			return nil, err
		}

		connector, err := NewConnector(r.beatInfo, p, c)
		if err != nil {
			return nil, err
		}

		err = connector.UseMetricSetProcessors(mb.Registry, module.Name(), metricSet.Name())
		if err != nil {
			return nil, err
		}

		client, err := connector.Connect()
		if err != nil {
			return nil, err
		}
		runners = append(runners, NewRunner(client, wrapper))
	}

	return newRunnerGroup(runners), nil
}

// CheckConfig checks if a config is valid or not
func (r *Factory) CheckConfig(config *common.Config) error {
	_, err := NewWrapper(config, mb.Registry, r.options...)
	if err != nil {
		return err
	}

	return nil
}

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
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

// Factory creates new Runner instances from configuration objects.
// It is used to register and reload modules.
type Factory struct {
	options []Option
}

// NewFactory creates new Reloader instance for the given config
func NewFactory(options ...Option) *Factory {
	return &Factory{
		options: options,
	}
}

// Create creates a new metricbeat module runner reporting events to the passed pipeline.
func (r *Factory) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	var errs multierror.Errors

	connector, err := NewConnector(p, c, meta)
	if err != nil {
		errs = append(errs, err)
	}
	w, err := NewWrapper(c, mb.Registry, r.options...)
	if err != nil {
		errs = append(errs, err)
	}

	if err := errs.Err(); err != nil {
		return nil, err
	}

	client, err := connector.Connect()
	if err != nil {
		return nil, err
	}

	mr := NewRunner(client, w)
	return mr, nil
}

// CheckConfig checks if a config is valid or not
func (r *Factory) CheckConfig(config *common.Config) error {
	_, err := NewWrapper(config, mb.Registry, r.options...)
	if err != nil {
		return err
	}

	return nil
}
